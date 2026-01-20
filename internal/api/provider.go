package api

import "context"

// Provider defines the interface for AI chat providers.
// Implementations include DeepSeek API and Ollama local models.
type Provider interface {
	// SendMessage sends a message request and returns the response.
	SendMessage(ctx context.Context, req MessageRequest) (*MessageResponse, error)

	// Name returns the provider name (e.g., "deepseek", "ollama").
	Name() string

	// Close releases any resources held by the provider.
	Close() error
}
