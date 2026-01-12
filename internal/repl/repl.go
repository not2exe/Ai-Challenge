package repl

import (
	"context"
	"fmt"

	"github.com/chzyer/readline"
	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/chat"
	"github.com/notexe/cli-chat/internal/config"
	"github.com/notexe/cli-chat/internal/ui"
)

type REPL struct {
	session   *chat.Session
	client    *api.Client
	config    *config.Config
	rl        *readline.Instance
	formatter *ui.Formatter
	status    *ui.StatusDisplay
}

func NewREPL(session *chat.Session, client *api.Client, cfg *config.Config) (*REPL, error) {
	rl, err := setupReadline()
	if err != nil {
		return nil, fmt.Errorf("failed to setup readline: %w", err)
	}

	formatter := ui.NewFormatter(cfg.UI.ColoredOutput)
	status := ui.NewStatusDisplay(formatter, true)

	return &REPL{
		session:   session,
		client:    client,
		config:    cfg,
		rl:        rl,
		formatter: formatter,
		status:    status,
	}, nil
}

func (r *REPL) Start(ctx context.Context) error {
	defer r.rl.Close()

	r.displayWelcome()

	for {
		input, err := r.readInput()
		if err != nil {
			if isEOF(err) {
				fmt.Println("\nGoodbye!")
				return nil
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		if input == "" {
			continue
		}

		isCommand, command, args := r.parseCommand(input)
		if isCommand {
			if err := r.handleCommand(command, args); err != nil {
				r.displayError(err)
			}

			if command == "/quit" || command == "/exit" {
				return nil
			}

			continue
		}

		if err := r.handleMessage(ctx, input); err != nil {
			r.displayError(err)
		}
	}
}

func (r *REPL) Stop() {
	r.rl.Close()
}

func (r *REPL) handleMessage(ctx context.Context, message string) error {
	r.session.AddUserMessage(message)
	r.status.Show("Waiting for response...")

	req := r.session.BuildAPIRequest()

	response, err := r.client.SendMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	r.session.AddAssistantMessage(response.Content)
	r.displayResponse(response)

	return nil
}

func (r *REPL) handleCommand(command, args string) error {
	switch command {
	case "/help", "/h":
		r.displayHelp()
		return nil

	case "/clear", "/c":
		r.session.Clear()
		r.displaySystem("Conversation history cleared.")
		return nil

	case "/system", "/s":
		if args == "" {
			return fmt.Errorf("usage: /system <prompt>")
		}
		if err := r.session.SetSystemPrompt(args); err != nil {
			return err
		}
		r.displaySystem("System prompt updated.")
		return nil

	case "/show":
		prompt := r.session.GetSystemPrompt()
		if prompt == "" {
			r.displayInfo("No system prompt set (using DeepSeek's default behavior).")
		} else {
			r.displayInfo(fmt.Sprintf("Current system prompt:\n%s", prompt))
		}
		return nil

	case "/quit", "/exit", "/q":
		fmt.Println("\nGoodbye!")
		return nil

	case "/count":
		count := r.session.MessageCount()
		r.displayInfo(fmt.Sprintf("Current conversation has %d messages.", count))
		return nil

	default:
		return fmt.Errorf("unknown command: %s (type /help for available commands)", command)
	}
}

func (r *REPL) SaveHistory() error {
	if !r.config.Session.SaveHistory {
		return nil
	}

	if r.session.IsEmpty() {
		return nil
	}

	return r.session.Save(r.config.Session.HistoryFile)
}
