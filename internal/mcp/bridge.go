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

// GetAskUserTool returns the ask_user tool definition for interactive questions
func GetAskUserTool() request.Tool {
	return request.Tool{
		Type: "function",
		Function: &request.ToolFunction{
			Name:        "ask_user",
			Description: "Present interactive multiple-choice questions to the user. Use this when you want the user to choose from specific options or clarify their preferences.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"questions": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"question": map[string]interface{}{
									"type":        "string",
									"description": "The question to ask the user",
								},
								"header": map[string]interface{}{
									"type":        "string",
									"description": "Short label for the question (1-3 words)",
								},
								"options": map[string]interface{}{
									"type": "array",
									"items": map[string]interface{}{
										"type": "object",
										"properties": map[string]interface{}{
											"label": map[string]interface{}{
												"type":        "string",
												"description": "The option text",
											},
											"description": map[string]interface{}{
												"type":        "string",
												"description": "Optional explanation of the option",
											},
										},
										"required": []string{"label"},
									},
									"description": "2-5 options for the user to choose from",
								},
								"multiSelect": map[string]interface{}{
									"type":        "boolean",
									"description": "Allow multiple selections (default: false)",
								},
							},
							"required": []string{"question", "options"},
						},
					},
				},
				"required": []string{"questions"},
			},
		},
	}
}
