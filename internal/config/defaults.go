package config

import (
	"github.com/knadh/koanf/providers/confmap"
)

func DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"api": map[string]interface{}{
			"key":      "",
			"base_url": "https://api.deepseek.com",
			"timeout":  120,
		},
		"model": map[string]interface{}{
			"name":          "deepseek-chat",
			"max_tokens":    2048,
			"temperature":   1.0,
			"system_prompt": "You are a helpful AI assistant. Provide clear, concise, and accurate responses.",
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
