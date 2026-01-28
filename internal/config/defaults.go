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
		"mcp": map[string]interface{}{
			"enabled":     true,
			"config_file": "~/.cli-chat/mcp.json",
			"servers": map[string]interface{}{
				"filesystem": map[string]interface{}{
					"command": "npx",
					"args":    []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
					"env":     map[string]string{},
				},
			},
		},
		"scheduler": map[string]interface{}{
			"enabled":  false,
			"interval": 3600,
			"prompt_template": "Use list_reminders to get all reminders. Then use get_due_reminders to check which ones are overdue. Respond with ONLY the HTML below, nothing else. No intro, no explanation.\n\nIf there are no reminders at all, respond with exactly: NO_REMINDERS\n\nOtherwise use this exact HTML format (Telegram supported tags only):\n\n<b>📋 Reminder Summary</b>\n\n🔴 <b>Due/Overdue:</b>\n• <b>Title</b> [PRIORITY] — ⏰ overdue by Xh Ym\n  <i>Description</i>\n  Deadline: DATE\nOr: None\n\n🟡 <b>Pending:</b>\n• <b>Title</b> [PRIORITY] — due DATE\n  <i>Description</i>\nOr: None\n\n✅ <b>Completed:</b>\n• <s>Title</s>\nOr: None\n\nUse 🔴 HIGH, 🟡 MEDIUM, 🟢 LOW for priority labels. Show deadline as a readable date. Only use Telegram HTML tags: <b> <i> <s> <code> <pre>.",
			"system_prompt":   "You output ONLY valid Telegram HTML. No introductions, no thinking, no commentary. Only use these HTML tags: <b> <i> <s> <code> <pre>. Never use <br> or <p> — use newlines instead. Your entire output is sent directly to Telegram as-is.",
			"telegram": map[string]interface{}{
				"bot_token": "",
				"chat_id":   "",
			},
		},
	}
}

func NewDefaultProvider() *confmap.Confmap {
	return confmap.Provider(DefaultConfig(), ".")
}

func GetDefaultConfigPath() string {
	return "~/.cli-chat/config.yaml"
}

func GetDefaultMCPConfigPath() string {
	return "~/.cli-chat/mcp.json"
}
