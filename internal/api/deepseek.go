package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/request"
	"github.com/notexe/cli-chat/internal/config"
)

// deepseekMessage extends request.Message to include ToolCalls field
// which the SDK's request.Message is missing but the API requires
type deepseekMessage struct {
	Role       string             `json:"role"`
	Content    string             `json:"content"`
	Name       string             `json:"name,omitempty"`
	ToolCallId string             `json:"tool_call_id,omitempty"`
	ToolCalls  []deepseekToolCall `json:"tool_calls,omitempty"`
}

type deepseekToolCall struct {
	Id       string               `json:"id"`
	Type     string               `json:"type"`
	Function deepseekToolFunction `json:"function"`
}

type deepseekToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// deepseekChatRequest is our custom request struct that supports tool_calls in messages
type deepseekChatRequest struct {
	Model       string            `json:"model"`
	Messages    []deepseekMessage `json:"messages"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature *float32          `json:"temperature,omitempty"`
	Stream      bool              `json:"stream"`
	Tools       *[]request.Tool   `json:"tools,omitempty"`
}

// deepseekChatResponse mirrors the API response structure
type deepseekChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Message      struct {
			Role      string             `json:"role"`
			Content   string             `json:"content"`
			ToolCalls []deepseekToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// deepseekErrorResponse for parsing API errors
type deepseekErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// DeepSeekProvider implements Provider for DeepSeek API.
type DeepSeekProvider struct {
	client deepseek.Client
	config config.DeepSeekConfig
}

// NewDeepSeekProvider creates a new DeepSeek provider.
func NewDeepSeekProvider(cfg config.DeepSeekConfig) (*DeepSeekProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is required")
	}

	client, err := deepseek.NewClient(cfg.APIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create DeepSeek client: %w", err)
	}

	return &DeepSeekProvider{
		client: client,
		config: cfg,
	}, nil
}

// SendMessage sends a message to DeepSeek API and returns the response.
func (p *DeepSeekProvider) SendMessage(ctx context.Context, req MessageRequest) (*MessageResponse, error) {
	// Check if any message has tool calls - if so, we need direct HTTP call
	hasToolCalls := false
	for _, msg := range req.Messages {
		if len(msg.ToolCalls) > 0 {
			hasToolCalls = true
			break
		}
	}

	if hasToolCalls {
		return p.sendMessageWithToolCalls(ctx, req)
	}

	return p.sendMessageSDK(ctx, req)
}

// sendMessageSDK uses the DeepSeek SDK for simple messages without tool calls
func (p *DeepSeekProvider) sendMessageSDK(ctx context.Context, req MessageRequest) (*MessageResponse, error) {
	messages := make([]*request.Message, 0, len(req.Messages)+1)

	if req.System != "" {
		messages = append(messages, &request.Message{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		m := &request.Message{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallId: msg.ToolCallID,
		}
		messages = append(messages, m)
	}

	var temp *float32
	if req.Temperature > 0 {
		t := float32(req.Temperature)
		temp = &t
	}

	chatReq := &request.ChatCompletionsRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: temp,
		Stream:      false,
	}

	// Add tools if provided
	if len(req.Tools) > 0 {
		chatReq.Tools = &req.Tools
	}

	resp, err := p.client.CallChatCompletionsChat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek API request failed: %w", err)
	}

	var content string
	var toolCalls []ToolCall

	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content

		// Extract tool calls from response
		for _, tc := range resp.Choices[0].Message.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:        tc.Id,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}

	response := &MessageResponse{
		Content:    content,
		StopReason: resp.Choices[0].FinishReason,
		Usage: Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
		ToolCalls: toolCalls,
	}

	return response, nil
}

// sendMessageWithToolCalls uses direct HTTP for messages containing tool calls
// This is needed because the SDK's request.Message doesn't support tool_calls field
func (p *DeepSeekProvider) sendMessageWithToolCalls(ctx context.Context, req MessageRequest) (*MessageResponse, error) {
	messages := make([]deepseekMessage, 0, len(req.Messages)+1)

	if req.System != "" {
		messages = append(messages, deepseekMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		m := deepseekMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallId: msg.ToolCallID,
		}

		// Convert tool calls if present
		if len(msg.ToolCalls) > 0 {
			m.ToolCalls = make([]deepseekToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				m.ToolCalls[i] = deepseekToolCall{
					Id:   tc.ID,
					Type: "function",
					Function: deepseekToolFunction{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}
		}

		messages = append(messages, m)
	}

	var temp *float32
	if req.Temperature > 0 {
		t := float32(req.Temperature)
		temp = &t
	}

	chatReq := deepseekChatRequest{
		Model:       req.Model,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: temp,
		Stream:      false,
	}

	if len(req.Tools) > 0 {
		chatReq.Tools = &req.Tools
	}

	// Make direct HTTP request
	resp, err := p.doHTTPRequest(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek API request failed: %w", err)
	}

	var content string
	var toolCalls []ToolCall

	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content

		for _, tc := range resp.Choices[0].Message.ToolCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:        tc.Id,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}

	response := &MessageResponse{
		Content:    content,
		StopReason: resp.Choices[0].FinishReason,
		Usage: Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
		ToolCalls: toolCalls,
	}

	return response, nil
}

// doHTTPRequest makes a direct HTTP call to the DeepSeek API
func (p *DeepSeekProvider) doHTTPRequest(ctx context.Context, chatReq deepseekChatRequest) (*deepseekChatResponse, error) {
	baseURL := p.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	url := fmt.Sprintf("%s/chat/completions", baseURL)

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.config.APIKey))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	client := &http.Client{
		Timeout: time.Duration(p.config.Timeout) * time.Second,
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp deepseekErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("%s", errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error: %s (status %d)", string(respBody), resp.StatusCode)
	}

	var chatResp deepseekChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &chatResp, nil
}

// Name returns the provider name.
func (p *DeepSeekProvider) Name() string {
	return "deepseek"
}

// Close releases resources (no-op for DeepSeek).
func (p *DeepSeekProvider) Close() error {
	return nil
}
