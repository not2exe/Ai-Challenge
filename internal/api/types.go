package api

import "github.com/go-deepseek/deepseek/request"

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	TokenCount int        `json:"token_count,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // For tool responses
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`   // For assistant tool requests
}

type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type MessageRequest struct {
	Messages    []Message      `json:"messages"`
	System      string         `json:"system,omitempty"`
	Model       string         `json:"model"`
	MaxTokens   int            `json:"max_tokens"`
	Temperature float64        `json:"temperature"`
	Tools       []request.Tool `json:"tools,omitempty"` // MCP tools converted to DeepSeek format
}

type MessageResponse struct {
	Content    string     `json:"content"`
	StopReason string     `json:"stop_reason"`
	Usage      Usage      `json:"usage"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"` // Tools the model wants to call
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
