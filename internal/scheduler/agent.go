package scheduler

import (
	"context"
	"fmt"
	"strings"

	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/mcp"
)

const maxAgentRounds = 10

// RunAgenticPrompt runs a stateless agentic tool-calling loop:
// send prompt → if tool_calls: execute via MCP, append results, re-send → until final text.
func RunAgenticPrompt(
	ctx context.Context,
	provider api.Provider,
	mcpMgr *mcp.Manager,
	systemPrompt string,
	userPrompt string,
	model string,
	maxTokens int,
	temperature float64,
) (string, error) {
	messages := []api.Message{
		{Role: "user", Content: userPrompt},
	}

	for round := 0; round < maxAgentRounds; round++ {
		req := api.MessageRequest{
			Messages:    messages,
			System:      systemPrompt,
			Model:       model,
			MaxTokens:   maxTokens,
			Temperature: temperature,
		}

		if mcpMgr != nil {
			req.Tools = mcpMgr.GetDeepSeekTools()
		}

		resp, err := provider.SendMessage(ctx, req)
		if err != nil {
			return "", fmt.Errorf("API request failed (round %d): %w", round, err)
		}

		// No tool calls — we have the final answer
		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		// Add assistant message with tool calls
		messages = append(messages, api.Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		// Execute each tool call and collect results
		for _, tc := range resp.ToolCalls {
			result, err := mcpMgr.CallTool(ctx, tc.Name, tc.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			// Truncate large results
			const maxToolResultSize = 32000
			if len(result) > maxToolResultSize {
				result = result[:maxToolResultSize] + "\n\n[... truncated]"
			}

			messages = append(messages, api.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	// Collect any text from the conversation as fallback
	var parts []string
	for _, m := range messages {
		if m.Role == "assistant" && m.Content != "" {
			parts = append(parts, m.Content)
		}
	}

	if len(parts) > 0 {
		return strings.Join(parts, "\n"), nil
	}

	return "", fmt.Errorf("agent reached max rounds (%d) without a final answer", maxAgentRounds)
}
