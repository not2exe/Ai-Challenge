package api

import (
	"context"
	"fmt"

	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/request"
	"github.com/notexe/cli-chat/internal/config"
)

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

// Name returns the provider name.
func (p *DeepSeekProvider) Name() string {
	return "deepseek"
}

// Close releases resources (no-op for DeepSeek).
func (p *DeepSeekProvider) Close() error {
	return nil
}
