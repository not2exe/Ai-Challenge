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

type Config struct {
	API     APIConfig     `koanf:"api"`
	Model   ModelConfig   `koanf:"model"`
	Session SessionConfig `koanf:"session"`
	UI      UIConfig      `koanf:"ui"`
}

type APIConfig struct {
	Key     string `koanf:"key"`
	BaseURL string `koanf:"base_url"`
	Timeout int    `koanf:"timeout"`
}

type ModelConfig struct {
	Name         string  `koanf:"name"`
	MaxTokens    int     `koanf:"max_tokens"`
	Temperature  float64 `koanf:"temperature"`
	SystemPrompt string  `koanf:"system_prompt"`
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

	if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
		k.Set("api.key", apiKey)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cfg.Session.HistoryFile = expandPath(cfg.Session.HistoryFile)

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.API.Key == "" {
		return fmt.Errorf("API key is required (set DEEPSEEK_API_KEY or add to config file)")
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

func expandPath(path string) string {
	if path == "" {
		return path
	}

	if path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}

	return path
}
