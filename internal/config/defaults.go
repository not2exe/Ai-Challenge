package config

import (
	"github.com/knadh/koanf/providers/confmap"
)

func DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"provider": "deepseek",
		"deepseek": map[string]interface{}{
			"api_key":  "",
			"base_url": "https://api.deepseek.com",
			"timeout":  120,
		},
		"ollama": map[string]interface{}{
			"base_url": "http://localhost:11434",
			"timeout":  120,
		},
		// Deprecated: kept for backwards compatibility
		"api": map[string]interface{}{
			"key":      "",
			"base_url": "https://api.deepseek.com",
			"timeout":  120,
		},
		"model": map[string]interface{}{
			"name":           "deepseek-chat",
			"max_tokens":     8192,
			"temperature":    1.0,
			"system_prompt":  "You are a helpful AI assistant. Provide clear, concise, and accurate responses.",
			"context_window": 0, // 0 means use default for model
		},
		"context": map[string]interface{}{
			"summarize_at":   0.70, // Summarize when context reaches 70%
			"target_after":   0.40, // Target 40% after summarization
			"auto_summarize": true, // Enable auto-summarization
		},
		"session": map[string]interface{}{
			"max_history":  50,
			"save_history": false,
			"history_file": "~/.cli-chat/history.json",
		},
		"ui": map[string]interface{}{
			"show_token_count": true,
			"colored_output":   true,
			"show_timestamps":  false,
		},
	}
}

func NewDefaultProvider() *confmap.Confmap {
	return confmap.Provider(DefaultConfig(), ".")
}

func GetDefaultConfigPath() string {
	return "~/.cli-chat/config.yaml"
}
