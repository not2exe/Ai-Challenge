package api

import (
	"context"
	"fmt"

	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/request"
	"github.com/notexe/cli-chat/internal/config"
)

type Client struct {
	client deepseek.Client
	config *config.APIConfig
}

func NewClient(cfg *config.APIConfig) (*Client, error) {
	if cfg.Key == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client, err := deepseek.NewClient(cfg.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create DeepSeek client: %w", err)
	}

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

func (c *Client) SendMessage(ctx context.Context, req MessageRequest) (*MessageResponse, error) {
	messages := make([]*request.Message, 0, len(req.Messages)+1)

	if req.System != "" {
		messages = append(messages, &request.Message{
			Role:    "system",
			Content: req.System,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, &request.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
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

	resp, err := c.client.CallChatCompletionsChat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek API request failed: %w", err)
	}

	var content string
	if len(resp.Choices) > 0 {
		content = resp.Choices[0].Message.Content
	}

	response := &MessageResponse{
		Content:    content,
		StopReason: resp.Choices[0].FinishReason,
		Usage: Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	return response, nil
}
