package api

import (
	"fmt"

	"github.com/notexe/cli-chat/internal/config"
)

// NewProvider creates a Provider based on the configuration.
func NewProvider(cfg *config.ProviderConfig) (Provider, error) {
	switch cfg.Type {
	case config.ProviderDeepSeek:
		return NewDeepSeekProvider(cfg.DeepSeek)

	case config.ProviderOllama:
		return NewOllamaProvider(cfg.Ollama)

	default:
		return nil, fmt.Errorf("unknown provider type: %s (supported: %s, %s)",
			cfg.Type, config.ProviderDeepSeek, config.ProviderOllama)
	}
}
