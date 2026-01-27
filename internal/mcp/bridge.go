package mcp

import (
	"github.com/go-deepseek/deepseek/request"
)

// ToDeepSeekTools converts MCP tools to DeepSeek tool format.
// This allows DeepSeek model to use MCP tools via function calling.
func ToDeepSeekTools(mcpTools []Tool) []request.Tool {
	tools := make([]request.Tool, 0, len(mcpTools))

	for _, t := range mcpTools {
		// Ensure properties is never nil (DeepSeek requires empty object, not null)
		properties := t.InputSchema.Properties
		if properties == nil {
			properties = make(map[string]interface{})
		}

		// Convert MCP InputSchema to DeepSeek parameters format
		params := map[string]interface{}{
			"type":       "object",
			"properties": properties,
		}

		if len(t.InputSchema.Required) > 0 {
			params["required"] = t.InputSchema.Required
		}

		tool := request.Tool{
			Type: "function",
			Function: &request.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		}
		tools = append(tools, tool)
	}

	return tools
}
