// Command review is a headless AI agent for PR code review.
//
// It gets a PR diff via `gh` CLI, starts an mcp-codeindex server as a subprocess,
// and runs an agent loop where DeepSeek uses semantic_search to gather RAG context
// from the project's code indexes before writing a structured review.
//
// Usage:
//
//	./review --pr 42
//	./review --pr 42 --codeindex ./mcp-codeindex --model deepseek-chat
//	./review --diff-file /tmp/pr.diff   # skip gh, use local diff file
//
// Environment:
//
//	DEEPSEEK_API_KEY   Required. DeepSeek API key.
//	OLLAMA_URL         Ollama API URL (default: http://localhost:11434)
//	OLLAMA_MODEL       Embedding model (default: nomic-embed-text)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/config"
	"github.com/notexe/cli-chat/internal/mcp"
)

const reviewSystemPrompt = `You are an expert code reviewer. You have access to semantic_search tool to find relevant documentation and code from the project's indexes.

SEARCH STRATEGY:
1. First, call index_stats to check available indexes.
2. Search in docs index first (use index_path with docs directory if available) for project conventions, architecture decisions, and documentation.
3. Then search in code index (project root) for related code, similar patterns, existing implementations.
4. Make multiple targeted searches based on what you see in the diff — search for function names, module names, patterns you see.
5. Do NOT search for generic terms. Be specific: use class names, function names, module paths from the diff.

REVIEW OUTPUT (in Russian):
1. **Краткое резюме** — что делает этот PR (1-2 предложения)
2. **Найденные проблемы** — конкретные баги, логические ошибки, уязвимости. Укажи файл и строку из diff.
3. **Потенциальные баги** — edge cases, race conditions, ошибки обработки nil/null
4. **Стиль и качество кода** — нарушения конвенций проекта (используй docs для проверки), дублирование, неоптимальные решения
5. **Советы по улучшению** — конкретные предложения с примерами кода

RULES:
- Be specific: reference file names and line numbers from the diff
- If you found relevant project conventions in docs — cite them
- If no issues found, say so explicitly — don't invent problems
- Focus on real problems, not nitpicks
- Compare changes against existing patterns found via semantic_search
- Format output as Markdown`

// maxToolRounds limits the agent loop to prevent infinite tool calling.
const maxToolRounds = 10

// maxToolResultSize limits individual tool result size to prevent context overflow.
const maxToolResultSize = 32000

func main() {
	prNumber := flag.String("pr", "", "PR number (uses gh CLI to get diff)")
	diffFile := flag.String("diff-file", "", "Path to diff file (alternative to --pr)")
	codeindexBin := flag.String("codeindex", "./mcp-codeindex", "Path to mcp-codeindex binary")
	model := flag.String("model", "deepseek-chat", "DeepSeek model name")
	maxTokens := flag.Int("max-tokens", 4096, "Max tokens for response")
	temperature := flag.Float64("temperature", 0.3, "Temperature for generation")
	outputFile := flag.String("output", "", "Write review to file (default: stdout only)")
	flag.Parse()

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		fatal("DEEPSEEK_API_KEY environment variable is required")
	}

	// Get the diff
	diff := getDiff(*prNumber, *diffFile)
	if strings.TrimSpace(diff) == "" {
		fatal("Empty diff — nothing to review")
	}

	// Get PR info if available
	prTitle := ""
	prBody := ""
	if *prNumber != "" {
		prTitle = ghExec("pr", "view", *prNumber, "--json", "title", "-q", ".title")
		prBody = ghExec("pr", "view", *prNumber, "--json", "body", "-q", ".body")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log("Interrupted, shutting down...")
		cancel()
		os.Exit(1)
	}()

	// Create DeepSeek provider
	provider, err := api.NewDeepSeekProvider(config.DeepSeekConfig{
		APIKey:  apiKey,
		BaseURL: "https://api.deepseek.com",
		Timeout: 120,
	})
	if err != nil {
		fatal("Failed to create DeepSeek provider: %v", err)
	}
	defer provider.Close()

	// Start mcp-codeindex as subprocess
	mcpManager := mcp.NewManager()

	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	env := []string{"OLLAMA_URL=" + ollamaURL}
	if ollamaModel := os.Getenv("OLLAMA_MODEL"); ollamaModel != "" {
		env = append(env, "OLLAMA_MODEL="+ollamaModel)
	}

	log("Starting mcp-codeindex server: %s", *codeindexBin)
	initCtx, initCancel := context.WithTimeout(ctx, 30*time.Second)
	err = mcpManager.AddServer(initCtx, mcp.ServerConfig{
		Name:    "codeindex",
		Command: *codeindexBin,
		Env:     env,
	})
	initCancel()
	if err != nil {
		fatal("Failed to start mcp-codeindex: %v\nMake sure the binary exists at: %s", err, *codeindexBin)
	}
	defer mcpManager.Close()

	counts := mcpManager.ServerToolCount()
	log("mcp-codeindex connected: %d tools available", counts["codeindex"])

	// Build user message
	userMessage := buildUserMessage(prTitle, prBody, diff)

	// Run agent loop
	review := runAgentLoop(ctx, provider, mcpManager, *model, *maxTokens, *temperature, userMessage)

	if review == "" {
		fatal("Agent returned empty review")
	}

	// Output
	result := formatReviewOutput(review)
	fmt.Println(result)

	if *outputFile != "" {
		if err := os.WriteFile(*outputFile, []byte(result), 0644); err != nil {
			log("Warning: failed to write output file: %v", err)
		} else {
			log("Review written to %s", *outputFile)
		}
	}
}

func getDiff(prNumber, diffFilePath string) string {
	if diffFilePath != "" {
		data, err := os.ReadFile(diffFilePath)
		if err != nil {
			fatal("Failed to read diff file: %v", err)
		}
		return string(data)
	}

	if prNumber == "" {
		fatal("Either --pr or --diff-file is required")
	}

	return ghExec("pr", "diff", prNumber)
}

func buildUserMessage(title, body, diff string) string {
	var sb strings.Builder

	sb.WriteString("Please review this Pull Request.\n\n")

	if title != "" {
		sb.WriteString("## PR Title\n")
		sb.WriteString(strings.TrimSpace(title))
		sb.WriteString("\n\n")
	}

	if body != "" {
		sb.WriteString("## PR Description\n")
		sb.WriteString(strings.TrimSpace(body))
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Diff\n```diff\n")
	sb.WriteString(diff)
	sb.WriteString("\n```")

	return sb.String()
}

func runAgentLoop(
	ctx context.Context,
	provider api.Provider,
	mcpManager *mcp.Manager,
	model string,
	maxTokens int,
	temperature float64,
	userMessage string,
) string {
	tools := mcpManager.GetDeepSeekTools()
	messages := []api.Message{
		{Role: "user", Content: userMessage},
	}

	for round := 0; round < maxToolRounds; round++ {
		req := api.MessageRequest{
			Messages:    messages,
			System:      reviewSystemPrompt,
			Model:       model,
			MaxTokens:   maxTokens,
			Temperature: temperature,
			Tools:       tools,
		}

		log("Sending request to DeepSeek (round %d)...", round+1)
		resp, err := provider.SendMessage(ctx, req)
		if err != nil {
			fatal("DeepSeek API request failed: %v", err)
		}

		log("Response: %d chars, %d tool calls (tokens: in=%d, out=%d)",
			len(resp.Content), len(resp.ToolCalls),
			resp.Usage.InputTokens, resp.Usage.OutputTokens)

		// No tool calls — final answer
		if len(resp.ToolCalls) == 0 {
			return resp.Content
		}

		// Add assistant message with tool calls
		messages = append(messages, api.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call
		for _, tc := range resp.ToolCalls {
			log("  Tool: %s(%s)", tc.Name, truncate(tc.Arguments, 100))

			result, err := mcpManager.CallTool(ctx, tc.Name, tc.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
				log("  Error: %v", err)
			} else {
				log("  Result: %d chars", len(result))
			}

			// Truncate large results
			if len(result) > maxToolResultSize {
				result = result[:maxToolResultSize] + "\n\n[... truncated — result too large]"
			}

			messages = append(messages, api.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	log("Warning: reached max tool rounds (%d), returning last content", maxToolRounds)
	// Return whatever content we have from the last response
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if last.Role == "assistant" && last.Content != "" {
			return last.Content
		}
	}
	return "Review could not be completed: exceeded maximum tool call rounds."
}

func formatReviewOutput(review string) string {
	return "## AI Code Review\n\n" + review + "\n\n---\n*Reviewed by DeepSeek AI with RAG context from project indexes*"
}

// ghExec runs a gh CLI command and returns stdout.
func ghExec(args ...string) string {
	cmd := exec.Command("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}
		fatal("gh %s failed: %v\n%s", strings.Join(args, " "), err, stderr)
	}
	return string(out)
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func log(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[review] "+format+"\n", args...)
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[review] ERROR: "+format+"\n", args...)
	os.Exit(1)
}
