package repl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/ui"
)

// helpSearchPrompt is the system prompt for /help queries that use code index results.
const helpSearchPrompt = `You are a project assistant. The user asked a question about the codebase using the /help command.
Below are search results from the project's code index, grouped by priority.

Your task:
- Answer the question based ONLY on the provided search results
- Results labeled "Documentation index" have HIGHEST priority — prefer them over code results
- Results labeled "Code index" are supplementary — use them only if docs don't cover the question
- Show relevant code fragments in your answer when appropriate
- Point out style patterns, conventions, and architectural rules you can see
- If the snippets don't contain enough info, say so honestly
- Answer in the same language as the user's question
- Be concise and practical

CITATION REQUIREMENTS — MANDATORY:
Search results include citation IDs [1], [2], etc. and source file paths.
1. Reference sources inline using [N] format (e.g., "The handler is in REPL [1]")
2. Include a "Sources:" section at the END listing all referenced files with paths and line numbers
3. Format: "Sources:\n[1] path/to/file.go:10-25\n[2] another/file.go:100-150"
This lets the user click on file paths in the terminal to navigate directly to the code.`

// handleHelpQuery searches the code index and asks the AI to answer based on results.
func (r *REPL) handleHelpQuery(ctx context.Context, query string) error {
	if r.mcpManager == nil || !r.mcpManager.HasCodeIndexTools() {
		r.displayInfo("Code index not available. Make sure mcp-codeindex server is configured and running.\nUse /help without arguments to see available commands.")
		return nil
	}

	// Detect project root from git or CWD
	projectRoot := detectProjectRoot()

	// Phase 1: Search documentation index (docs/.codeindex) — highest priority
	r.status.Show("Searching documentation...")

	docsDir := filepath.Join(projectRoot, "docs")
	docsIndexDir := filepath.Join(docsDir, ".codeindex")
	var docsResult string

	if _, err := os.Stat(docsIndexDir); err == nil {
		// docs/.codeindex exists — search it
		result, err := r.searchIndex(ctx, query, docsDir, 5, 0.2, 1000)
		if err == nil {
			docsResult = result
		}
	} else if _, err := os.Stat(docsDir); err == nil {
		// docs/ exists but no index — create it
		r.status.Show("Indexing documentation...")
		indexArgs, _ := json.Marshal(map[string]interface{}{
			"path": docsDir,
		})
		if _, err := r.mcpManager.CallTool(ctx, "index_directory", string(indexArgs)); err == nil {
			result, err := r.searchIndex(ctx, query, docsDir, 5, 0.2, 1000)
			if err == nil {
				docsResult = result
			}
		}
	}

	// Phase 2: Search main code index (.codeindex) — second priority
	r.status.Show("Searching code index...")

	mainIndexDir := filepath.Join(projectRoot, ".codeindex")
	var codeResult string

	if _, err := os.Stat(mainIndexDir); err == nil {
		// .codeindex exists — search it
		result, err := r.searchIndex(ctx, query, "", 5, 0.3, 600)
		if err == nil {
			codeResult = result
		}
	} else {
		// No main index — create it
		r.status.Show("Indexing project...")
		indexArgs, _ := json.Marshal(map[string]interface{}{
			"path": projectRoot,
		})
		if _, err := r.mcpManager.CallTool(ctx, "index_directory", string(indexArgs)); err == nil {
			result, err := r.searchIndex(ctx, query, "", 5, 0.3, 600)
			if err == nil {
				codeResult = result
			}
		}
	}

	// Combine results with priority labels
	var combined strings.Builder
	if isValidResult(docsResult) {
		combined.WriteString("=== Documentation index (HIGHEST PRIORITY) ===\n")
		combined.WriteString(docsResult)
		combined.WriteString("\n\n")
	}
	if isValidResult(codeResult) {
		combined.WriteString("=== Code index ===\n")
		combined.WriteString(codeResult)
	}

	searchResult := combined.String()
	if strings.TrimSpace(searchResult) == "" {
		r.status.Hide()
		r.displayInfo(fmt.Sprintf("No results found for: %s\nTry a different query or check that the project is indexed.", query))
		return nil
	}

	// Phase 3: Send to AI
	r.status.Show("Generating answer...")

	prompt := fmt.Sprintf("Question: %s\n\n%s", query, searchResult)

	req := api.MessageRequest{
		Model:       r.session.GetModelName(),
		MaxTokens:   r.session.GetMaxTokens(),
		Temperature: r.session.GetTemperature(),
		Messages: []api.Message{
			{Role: "system", Content: helpSearchPrompt},
			{Role: "user", Content: prompt},
		},
	}

	start := time.Now()
	response, err := r.provider.SendMessage(ctx, req)
	duration := time.Since(start)
	if err != nil {
		r.status.Hide()
		return fmt.Errorf("API request failed: %w", err)
	}

	r.status.Hide()

	fmt.Println()
	fmt.Println(r.formatter.FormatAssistantMessage(response.Content))

	if r.config.UI.ShowTokenCount {
		fmt.Println(r.formatter.FormatTokenUsage(response.Usage, ui.TokenUsageOptions{
			Duration: duration,
			Model:    r.config.Model.Name,
		}))
	}
	fmt.Println()

	return nil
}

// searchIndex performs a semantic search, optionally at a specific index path.
func (r *REPL) searchIndex(ctx context.Context, query string, indexPath string, topK int, minSim float64, maxLen int) (string, error) {
	args := map[string]interface{}{
		"query":              query,
		"top_k":              topK,
		"min_similarity":     minSim,
		"max_content_length": maxLen,
	}
	if indexPath != "" {
		args["index_path"] = indexPath
	}

	argsJSON, _ := json.Marshal(args)
	return r.mcpManager.CallTool(ctx, "semantic_search", string(argsJSON))
}

// isValidResult checks if a search result contains actual content.
func isValidResult(result string) bool {
	return result != "" && result != "No results found" && result != "No results found." && result != "[]"
}

// detectProjectRoot finds the project root (git root or CWD).
func detectProjectRoot() string {
	// Try git root
	if out, err := execGitCommand("rev-parse", "--show-toplevel"); err == nil {
		return strings.TrimSpace(out)
	}
	// Fallback to CWD
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return "."
}

// execGitCommand runs a git command and returns stdout.
func execGitCommand(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
