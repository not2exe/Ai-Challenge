package repl

import (
	"context"
	"fmt"
	"strings"

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
	// Phase 1: Add user message
	r.session.AddUserMessage(message)

	// Check if clarify mode is enabled
	if r.session.IsClarifyEnabled() {
		return r.handleMessageWithClarify(ctx, message)
	}

	// Normal flow: direct response
	return r.sendMessageAndDisplay(ctx, true)
}

func (r *REPL) handleMessageWithClarify(ctx context.Context, originalMessage string) error {
	// Step 1: Request clarifying questions from AI
	r.status.Show("Analyzing question...")

	req := r.session.BuildAPIRequest()
	response, err := r.client.SendMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	r.status.Hide()

	// Step 2: Try to parse clarifying questions
	clarifyResp, err := chat.ParseClarifyResponse(response.Content)
	if err != nil {
		// If parsing fails, treat as normal response
		r.session.AddAssistantMessage(response.Content)
		r.displayResponse(response)
		return nil
	}

	// Step 3: Display intro message if provided
	if clarifyResp.Message != "" {
		fmt.Println()
		fmt.Println(r.formatter.FormatAssistantMessage(clarifyResp.Message))
	}

	// Step 4: Ask questions interactively
	answers, err := r.AskClarifyingQuestions(clarifyResp.Questions)
	if err != nil {
		return fmt.Errorf("failed to collect answers: %w", err)
	}

	// Step 5: Format answers and add to history
	answersText := chat.FormatQuestionAnswers(answers)
	r.session.AddAssistantMessage("Asked clarifying questions")
	r.session.AddUserMessage(answersText)

	// Step 6: Get final response with clarifications (without asking more questions)
	r.status.Show("Generating response with clarifications...")
	return r.sendMessageAndDisplay(ctx, false)
}

func (r *REPL) sendMessageAndDisplay(ctx context.Context, includeClarify bool) error {
	var req api.MessageRequest
	if includeClarify {
		req = r.session.BuildAPIRequest()
	} else {
		req = r.session.BuildAPIRequestWithoutClarify()
	}

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

	case "/format", "/f":
		return r.handleFormatCommand(args)

	case "/clarify", "/cl":
		return r.handleClarifyCommand(args)

	default:
		return fmt.Errorf("unknown command: %s (type /help for available commands)", command)
	}
}

func (r *REPL) handleFormatCommand(args string) error {
	if args == "" {
		return fmt.Errorf("usage: /format <json|show|clear>")
	}

	parts := strings.Fields(args)
	subcommand := strings.ToLower(parts[0])

	switch subcommand {
	case "json":
		template, err := chat.GetFormatTemplate("json")
		if err != nil {
			return err
		}

		if err := r.session.SetFormatPrompt(template.Prompt); err != nil {
			return err
		}

		r.displaySystem("JSON format template applied. Responses will be in structured JSON format.")
		return nil

	case "show":
		current := r.session.GetFormatPrompt()
		if current == "" {
			r.displayInfo("No format template set (using default behavior).")
		} else {
			r.displayInfo("Current format: JSON")
		}
		return nil

	case "clear", "off":
		r.session.ClearFormatPrompt()
		r.displaySystem("Format template cleared.")
		return nil

	default:
		return fmt.Errorf("unknown format: %s (available: json)", subcommand)
	}
}

func (r *REPL) handleClarifyCommand(args string) error {
	if args == "" {
		return fmt.Errorf("usage: /clarify <on|off|show>")
	}

	subcommand := strings.ToLower(strings.TrimSpace(args))

	switch subcommand {
	case "on", "enable":
		r.session.SetClarifyMode(true)
		r.displaySystem("Clarifying questions mode ENABLED. AI will ask questions before answering.")
		return nil

	case "off", "disable":
		r.session.SetClarifyMode(false)
		r.displaySystem("Clarifying questions mode DISABLED. AI will answer directly.")
		return nil

	case "show", "status":
		if r.session.IsClarifyEnabled() {
			r.displayInfo("Clarifying questions mode: ENABLED ✓")
		} else {
			r.displayInfo("Clarifying questions mode: DISABLED")
		}
		return nil

	default:
		return fmt.Errorf("unknown clarify command: %s (use: on, off, show)", subcommand)
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
