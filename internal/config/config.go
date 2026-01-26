package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Provider type constants (duplicated from api package to avoid import cycle)
const (
	ProviderDeepSeek = "deepseek"
	ProviderOllama   = "ollama"
)

type Config struct {
	Provider string         `koanf:"provider"`
	DeepSeek DeepSeekConfig `koanf:"deepseek"`
	Ollama   OllamaConfig   `koanf:"ollama"`
	Model    ModelConfig    `koanf:"model"`
	Session  SessionConfig  `koanf:"session"`
	UI       UIConfig       `koanf:"ui"`
	Context  ContextConfig  `koanf:"context"`
	MCP      MCPConfig      `koanf:"mcp"`

	// Deprecated: Use DeepSeek config instead. Kept for backwards compatibility.
	API APIConfig `koanf:"api"`
}

type MCPConfig struct {
	Enabled bool              `koanf:"enabled"`
	Servers []MCPServerConfig `koanf:"servers"`
}

type MCPServerConfig struct {
	Name    string   `koanf:"name"`
	Command string   `koanf:"command"`
	Args    []string `koanf:"args"`
	Env     []string `koanf:"env"` // Environment variables like GITHUB_TOKEN=xxx
}

type DeepSeekConfig struct {
	APIKey  string `koanf:"api_key"`
	BaseURL string `koanf:"base_url"`
	Timeout int    `koanf:"timeout"`
}

type OllamaConfig struct {
	BaseURL string `koanf:"base_url"`
	Timeout int    `koanf:"timeout"`
}

// APIConfig is kept for backwards compatibility with old config files.
type APIConfig struct {
	Key     string `koanf:"key"`
	BaseURL string `koanf:"base_url"`
	Timeout int    `koanf:"timeout"`
}

type ModelConfig struct {
	Name          string  `koanf:"name"`
	MaxTokens     int     `koanf:"max_tokens"`
	Temperature   float64 `koanf:"temperature"`
	SystemPrompt  string  `koanf:"system_prompt"`
	ContextWindow int     `koanf:"context_window"` // Override default context window (0 = use model default)
}

type ContextConfig struct {
	SummarizeAt   float64 `koanf:"summarize_at"`   // Threshold percentage to trigger summarization (0.70 = 70%)
	TargetAfter   float64 `koanf:"target_after"`   // Target percentage after summarization (0.40 = 40%)
	AutoSummarize bool    `koanf:"auto_summarize"` // Enable automatic summarization
}

type SessionConfig struct {
	MaxHistory  int    `koanf:"max_history"`
	SaveHistory bool   `koanf:"save_history"`
	HistoryFile string `koanf:"history_file"`
}

type UIConfig struct {
	ShowTokenCount bool `koanf:"show_token_count"`
	ColoredOutput  bool `koanf:"colored_output"`
	ShowTimestamps bool `koanf:"show_timestamps"`
}

func Load(configPath string) (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(NewDefaultProvider(), nil); err != nil {
		return nil, fmt.Errorf("failed to load defaults: %w", err)
	}

	if configPath != "" {
		configPath = expandPath(configPath)

		if _, err := os.Stat(configPath); err == nil {
			if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
				return nil, fmt.Errorf("failed to load config file: %w", err)
			}
		}
	}

	if err := k.Load(env.Provider("DEEPSEEK_", ".", func(s string) string {
		return s
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load env vars: %w", err)
	}

	// Handle DEEPSEEK_API_KEY environment variable
	if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
		k.Set("deepseek.api_key", apiKey)
		// Also set in legacy api.key for backwards compatibility
		k.Set("api.key", apiKey)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Backwards compatibility: migrate api.key to deepseek.api_key
	if cfg.DeepSeek.APIKey == "" && cfg.API.Key != "" {
		cfg.DeepSeek.APIKey = cfg.API.Key
	}
	if cfg.DeepSeek.BaseURL == "" && cfg.API.BaseURL != "" {
		cfg.DeepSeek.BaseURL = cfg.API.BaseURL
	}
	if cfg.DeepSeek.Timeout == 0 && cfg.API.Timeout > 0 {
		cfg.DeepSeek.Timeout = cfg.API.Timeout
	}

	cfg.Session.HistoryFile = expandPath(cfg.Session.HistoryFile)

	return &cfg, nil
}

func (c *Config) Validate() error {
	// Provider-specific validation
	switch c.Provider {
	case ProviderDeepSeek:
		if c.DeepSeek.APIKey == "" {
			return fmt.Errorf("DeepSeek API key is required (set DEEPSEEK_API_KEY or add to config file)")
		}
	case ProviderOllama:
		// Ollama doesn't require API key, but we could check if Ollama is running
		// For now, just validate that base URL is set (has a default)
		if c.Ollama.BaseURL == "" {
			c.Ollama.BaseURL = "http://localhost:11434"
		}
	default:
		return fmt.Errorf("unknown provider: %s (supported: %s, %s)",
			c.Provider, ProviderDeepSeek, ProviderOllama)
	}

	if c.Model.Name == "" {
		return fmt.Errorf("model name is required")
	}

	if c.Model.MaxTokens <= 0 {
		return fmt.Errorf("max_tokens must be positive")
	}

	if c.Model.Temperature < 0 || c.Model.Temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	if c.Session.MaxHistory <= 0 {
		return fmt.Errorf("max_history must be positive")
	}

	return nil
}

// ProviderConfig contains provider-specific configuration for the API package.
type ProviderConfig struct {
	Type     string
	DeepSeek DeepSeekConfig
	Ollama   OllamaConfig
	Model    ModelSettings
}

// ModelSettings contains model parameters used by all providers.
type ModelSettings struct {
	Name        string
	MaxTokens   int
	Temperature float64
}

// GetProviderConfig returns the provider configuration for the API package.
func (c *Config) GetProviderConfig() *ProviderConfig {
	return &ProviderConfig{
		Type:     c.Provider,
		DeepSeek: c.DeepSeek,
		Ollama:   c.Ollama,
		Model: ModelSettings{
			Name:        c.Model.Name,
			MaxTokens:   c.Model.MaxTokens,
			Temperature: c.Model.Temperature,
		},
	}
}

func expandPath(path string) string {
	if path == "" {
		return path
	}

	if len(path) >= 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}

	return path
}
