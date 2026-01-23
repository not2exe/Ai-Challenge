package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/chat"
	"github.com/notexe/cli-chat/internal/config"
	"github.com/notexe/cli-chat/internal/repl"
)

func main() {
	configPath := flag.String("config", config.GetDefaultConfigPath(), "Path to configuration file")
	provider := flag.String("provider", "", "Provider to use (deepseek, ollama)")
	modelName := flag.String("model", "", "Model name (overrides config)")
	systemPrompt := flag.String("system-prompt", "", "System prompt (overrides config)")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Apply CLI flag overrides
	if *provider != "" {
		cfg.Provider = *provider
	}
	if *modelName != "" {
		cfg.Model.Name = *modelName
	}
	if *systemPrompt != "" {
		cfg.Model.SystemPrompt = *systemPrompt
	}
	if *noColor {
		cfg.UI.ColoredOutput = false
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		if cfg.Provider == config.ProviderDeepSeek {
			fmt.Fprintf(os.Stderr, "Tip: Set DEEPSEEK_API_KEY environment variable or add it to config file\n")
		}
		os.Exit(1)
	}

	providerInstance, err := api.NewProvider(cfg.GetProviderConfig())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating provider: %v\n", err)
		os.Exit(1)
	}
	defer providerInstance.Close()

	session := chat.NewSessionWithContext(&cfg.Model, cfg.Session.MaxHistory, &cfg.Context)

	// Load history from file if enabled
	if cfg.Session.SaveHistory {
		if err := session.Load(cfg.Session.HistoryFile); err != nil {
			// Not an error if file doesn't exist yet
			if !errors.Is(err, os.ErrNotExist) && !os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "Warning: Failed to load history: %v\n", err)
			}
		} else {
			fmt.Printf("Loaded %d messages from history\n", session.MessageCount())
		}
	}

	replInstance, err := repl.NewREPL(session, providerInstance, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating REPL: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nInterrupted. Saving session...")
		cancel()

		if err := replInstance.SaveHistory(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to save history: %v\n", err)
		}

		os.Exit(0)
	}()

	if err := replInstance.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := replInstance.SaveHistory(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save history: %v\n", err)
	}
}
