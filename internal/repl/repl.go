package repl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/chzyer/readline"
	"github.com/go-deepseek/deepseek/request"
	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/chat"
	"github.com/notexe/cli-chat/internal/config"
	"github.com/notexe/cli-chat/internal/mcp"
	"github.com/notexe/cli-chat/internal/ui"
)

type REPL struct {
	session     *chat.Session
	provider    api.Provider
	config      *config.Config
	rl          *readline.Instance
	formatter   *ui.Formatter
	status      *ui.StatusDisplay
	mcpManager  *mcp.Manager
	inputReader *inputReader
}

func NewREPL(session *chat.Session, provider api.Provider, cfg *config.Config) (*REPL, error) {
	rl, err := setupReadline()
	if err != nil {
		return nil, fmt.Errorf("failed to setup readline: %w", err)
	}

	formatter := ui.NewFormatter(cfg.UI.ColoredOutput, provider.Name())
	status := ui.NewStatusDisplay(formatter, true)

	return &REPL{
		session:    session,
		provider:   provider,
		config:     cfg,
		rl:         rl,
		formatter:  formatter,
		status:     status,
		mcpManager: nil, // Set via SetMCPManager if MCP is enabled
	}, nil
}

// SetMCPManager sets the MCP manager for tool integration.
func (r *REPL) SetMCPManager(m *mcp.Manager) {
	r.mcpManager = m

	if m == nil {
		return
	}

	// Build tools prompt based on available tools
	var toolsPrompt string

	if m.HasFilesystemTools() {
		toolsPrompt = chat.FileToolsPrompt
	}

	if m.HasCodeIndexTools() {
		if toolsPrompt != "" {
			toolsPrompt += "\n\n"
		}
		toolsPrompt += chat.CodeIndexToolsPrompt
	}

	if toolsPrompt != "" {
		r.session.SetToolsPrompt(toolsPrompt)
	}
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
			if err := r.handleCommand(ctx, command, args); err != nil {
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
	if r.inputReader != nil {
		r.inputReader.stop()
	}
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
	start := time.Now()
	response, err := r.provider.SendMessage(ctx, req)
	duration := time.Since(start)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}

	r.status.Hide()

	// Step 2: Try to parse clarifying questions
	clarifyResp, err := chat.ParseClarifyResponse(response.Content)
	if err != nil {
		// If parsing fails, treat as normal response
		r.session.AddAssistantMessage(response.Content)
		r.displayResponse(response, duration)
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
	// Check if summarization is needed BEFORE sending (based on previous request tokens)
	if r.session.NeedsSummarization() {
		if err := r.performSummarization(ctx); err != nil {
			r.displaySystem("Warning: Failed to compress history: " + err.Error())
		}
	}

	var req api.MessageRequest
	if includeClarify {
		req = r.session.BuildAPIRequest()
	} else {
		req = r.session.BuildAPIRequestWithoutClarify()
	}

	// Add tools
	var tools []request.Tool
	if r.mcpManager != nil {
		tools = r.mcpManager.GetDeepSeekTools()
	}
	// Add ask_user tool if enabled
	if r.session.IsAskUserEnabled() {
		tools = append(tools, mcp.GetAskUserTool())
	}
	req.Tools = tools

	// Show spinner while waiting for response
	r.status.Show("Generating response...")

	start := time.Now()
	response, err := r.provider.SendMessage(ctx, req)
	if err != nil {
		r.status.Hide()
		return fmt.Errorf("API request failed: %w", err)
	}

	// Handle tool calls loop
	for len(response.ToolCalls) > 0 {
		r.status.Hide()

		// Check if any tool call is ask_user (handle it specially)
		askUserCall := findAskUserCall(response.ToolCalls)
		if askUserCall != nil && r.session.IsAskUserEnabled() {
			return r.handleAskUserToolCall(ctx, response, askUserCall, start)
		}

		// First, add the assistant message with tool calls to history
		// This is required by DeepSeek API - tool results must follow a message with tool_calls
		r.session.AddAssistantMessageWithToolCalls(response.Content, response.ToolCalls)

		// Process each tool call
		for _, tc := range response.ToolCalls {
			r.displayToolCall(tc.Name, tc.Arguments)

			// Execute tool via MCP
			result, err := r.mcpManager.CallTool(ctx, tc.Name, tc.Arguments)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}

			r.displayToolResult(tc.Name, result)

			// Truncate large results to prevent context overflow
			// 32K chars ≈ 8K tokens, reasonable limit for tool results
			const maxToolResultSize = 32000
			if len(result) > maxToolResultSize {
				result = result[:maxToolResultSize] + "\n\n[... truncated - result too large]"
			}

			// Add tool result to session
			r.session.AddToolResult(tc.ID, tc.Name, result)
		}

		// Send follow-up request with tool results
		r.status.Show("Processing tool results...")
		req = r.session.BuildAPIRequestWithToolResults()
		var toolsForResults []request.Tool
		if r.mcpManager != nil {
			toolsForResults = r.mcpManager.GetDeepSeekTools()
		}
		if r.session.IsAskUserEnabled() {
			toolsForResults = append(toolsForResults, mcp.GetAskUserTool())
		}
		req.Tools = toolsForResults

		response, err = r.provider.SendMessage(ctx, req)
		if err != nil {
			return fmt.Errorf("API request failed: %w", err)
		}
	}

	duration := time.Since(start)
	r.status.Hide()

	// Check for ask_user request in response
	if r.session.IsAskUserEnabled() && chat.HasAskUserRequest(response.Content) {
		return r.handleAskUserResponse(ctx, response, duration)
	}

	r.session.AddAssistantMessage(response.Content)
	r.displayResponse(response, duration)

	// Update token tracking from response for next iteration
	r.session.UpdateTokensFromResponse(response.Usage)

	return nil
}

// findAskUserCall finds an ask_user tool call in the list
func findAskUserCall(toolCalls []api.ToolCall) *api.ToolCall {
	for i := range toolCalls {
		if toolCalls[i].Name == "ask_user" {
			return &toolCalls[i]
		}
	}
	return nil
}

// handleAskUserToolCall handles the ask_user tool call from AI
func (r *REPL) handleAskUserToolCall(ctx context.Context, response *api.MessageResponse, tc *api.ToolCall, startTime time.Time) error {
	duration := time.Since(startTime)

	// Parse the ask_user arguments
	var askReq chat.AskUserRequest
	if err := json.Unmarshal([]byte(tc.Arguments), &askReq); err != nil {
		r.displayError(fmt.Errorf("failed to parse ask_user: %v", err))
		return nil
	}

	// Display any text content from the response
	if response.Content != "" {
		fmt.Println()
		fmt.Println(r.formatter.FormatAssistantMessage(response.Content))
	}

	// Display token usage for the request
	if r.config.UI.ShowTokenCount {
		fmt.Println(r.formatter.FormatTokenUsage(response.Usage, ui.TokenUsageOptions{
			Duration: duration,
			Model:    r.config.Model.Name,
		}))
	}

	// Collect answers for all questions
	var allAnswers [][]string
	for _, q := range askReq.Questions {
		// Convert options to string slice
		options := make([]string, len(q.Options))
		for i, opt := range q.Options {
			if opt.Description != "" {
				options[i] = opt.Label + " - " + opt.Description
			} else {
				options[i] = opt.Label
			}
		}

		// Ask the question using interactive UI
		answers, err := r.AskUserQuestion(q.Question, options, q.MultiSelect)
		if err != nil {
			return fmt.Errorf("failed to get user answer: %w", err)
		}
		allAnswers = append(allAnswers, answers)
	}

	// Format answers
	answersText := chat.FormatAskUserAnswers(askReq.Questions, allAnswers)

	// Add the conversation flow: assistant asked, user answered
	r.session.AddAssistantMessageWithToolCalls(response.Content, response.ToolCalls)
	r.session.AddToolResult(tc.ID, tc.Name, answersText)

	// Update token tracking
	r.session.UpdateTokensFromResponse(response.Usage)

	// Continue conversation - AI will process the user's answers
	r.status.Show("Processing your selection...")
	return r.sendMessageAndDisplay(ctx, false)
}

// handleAskUserResponse processes an ask_user request from the AI (tag-based fallback)
func (r *REPL) handleAskUserResponse(ctx context.Context, response *api.MessageResponse, duration time.Duration) error {
	// Parse the ask_user request
	askReq, textBefore, err := chat.ParseAskUserRequest(response.Content)
	if err != nil {
		// If parsing fails, treat as normal response
		r.session.AddAssistantMessage(response.Content)
		r.displayResponse(response, duration)
		r.session.UpdateTokensFromResponse(response.Usage)
		return nil
	}

	if askReq == nil {
		// No valid ask_user request found
		r.session.AddAssistantMessage(response.Content)
		r.displayResponse(response, duration)
		r.session.UpdateTokensFromResponse(response.Usage)
		return nil
	}

	// Display any text before the ask_user block
	if textBefore != "" {
		fmt.Println()
		fmt.Println(r.formatter.FormatAssistantMessage(textBefore))
	}

	// Collect answers for all questions
	var allAnswers [][]string
	for _, q := range askReq.Questions {
		// Convert to simple string slice for options
		options := make([]string, len(q.Options))
		for i, opt := range q.Options {
			if opt.Description != "" {
				options[i] = opt.Label + " - " + opt.Description
			} else {
				options[i] = opt.Label
			}
		}

		// Ask the question using the interactive UI
		answers, err := r.AskUserQuestion(q.Question, options, q.MultiSelect)
		if err != nil {
			return fmt.Errorf("failed to get user answer: %w", err)
		}
		allAnswers = append(allAnswers, answers)
	}

	// Format answers and add to conversation
	answersText := chat.FormatAskUserAnswers(askReq.Questions, allAnswers)
	r.session.AddAssistantMessage(response.Content) // Keep the original response with ask_user
	r.session.AddUserMessage(answersText)

	// Update token tracking
	r.session.UpdateTokensFromResponse(response.Usage)

	// Continue conversation with the answers
	r.status.Show("Processing your answers...")
	return r.sendMessageAndDisplay(ctx, false)
}

func (r *REPL) displayToolCall(name, args string) {
	toolStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("215")).
		Bold(true)
	argsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	fmt.Printf("\n%s %s\n", toolStyle.Render("Tool:"), name)
	if args != "" && args != "{}" {
		fmt.Printf("  %s %s\n", argsStyle.Render("Args:"), args)
	}
}

func (r *REPL) displayToolResult(name, result string) {
	resultLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("114"))

	// Truncate long results for display
	display := result
	maxDisplay := 2000 // Show more of tool results
	if len(display) > maxDisplay {
		display = display[:maxDisplay] + fmt.Sprintf("\n... (truncated, %d more chars)", len(result)-maxDisplay)
	}
	fmt.Printf("  %s %s\n", resultLabelStyle.Render("Result:"), display)
}

// performSummarization compresses the conversation history using AI summarization.
func (r *REPL) performSummarization(ctx context.Context) error {
	r.status.Show("Compressing history...")
	defer r.status.Hide()

	// Get messages to summarize (keep last 4 message pairs = 8 messages)
	toSummarize, toKeep := r.session.GetMessagesToSummarize(4)
	if len(toSummarize) == 0 {
		return nil // Nothing to summarize
	}

	// Build summarization request
	req := chat.BuildSummarizationRequest(
		toSummarize,
		r.session.GetModelName(),
		r.session.GetMaxTokens(),
		r.session.GetTemperature(),
	)

	// Send summarization request
	response, err := r.provider.SendMessage(ctx, req)
	if err != nil {
		return fmt.Errorf("summarization API request failed: %w", err)
	}

	// Create summary message and apply it
	summaryMsg := chat.FormatSummaryMessage(response.Content)
	r.session.ApplySummary(summaryMsg, len(toKeep))

	// Reset lastInputTokens — will be updated after next API call
	r.session.ResetInputTokens()

	r.displaySystem(fmt.Sprintf("History compressed. Summarized %d messages.", len(toSummarize)))
	return nil
}

func (r *REPL) handleCommand(ctx context.Context, command, args string) error {
	switch command {
	case "/help", "/h":
		r.displayHelp()
		return nil

	case "/clear", "/c":
		r.session.Clear()
		if err := r.DeleteHistoryFile(); err != nil {
			r.displayError(fmt.Errorf("failed to delete history file: %w", err))
		}
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
			r.displayInfo(fmt.Sprintf("No system prompt set (using %s's default behavior).", r.provider.Name()))
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

	case "/provider", "/p":
		r.displayInfo(fmt.Sprintf("Provider: %s\nModel: %s", r.provider.Name(), r.config.Model.Name))
		return nil

	case "/format", "/f":
		return r.handleFormatCommand(args)

	case "/clarify", "/cl":
		return r.handleClarifyCommand(args)

	case "/temp", "/temperature", "/t":
		return r.handleTempCommand(args)

	case "/file":
		return r.handleFileCommand(ctx, args)

	case "/context", "/ctx":
		return r.handleContextCommand(args)

	case "/mcp":
		return r.handleMCPCommand(args)

	case "/askuser", "/ask":
		return r.handleAskUserCommand(args)

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

func (r *REPL) handleTempCommand(args string) error {
	if args == "" {
		temp := r.session.GetTemperature()
		r.displayInfo(fmt.Sprintf("Current temperature: %.2f (range: 0-2)", temp))
		return nil
	}

	temp, err := strconv.ParseFloat(strings.TrimSpace(args), 64)
	if err != nil {
		return fmt.Errorf("invalid temperature value: %s (use a number between 0 and 2)", args)
	}

	if err := r.session.SetTemperature(temp); err != nil {
		return err
	}

	r.displaySystem(fmt.Sprintf("Temperature set to %.2f", temp))
	return nil
}

func (r *REPL) handleFileCommand(ctx context.Context, args string) error {
	if args == "" {
		return fmt.Errorf("usage: /file <filename>")
	}

	filename := strings.TrimSpace(args)

	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	fileContent := string(content)
	if fileContent == "" {
		return fmt.Errorf("file %s is empty", filename)
	}

	r.displayInfo(fmt.Sprintf("Loaded %d characters from %s", len(fileContent), filename))

	return r.handleMessage(ctx, fileContent)
}

func (r *REPL) handleContextCommand(args string) error {
	subcommand := strings.ToLower(strings.TrimSpace(args))

	switch subcommand {
	case "", "show", "status":
		used, limit, pct := r.session.GetContextStatus()
		threshold := r.session.GetContextManager().GetThresholdTokens(limit)

		autoStatus := "enabled"
		if !r.session.IsAutoSummarizeEnabled() {
			autoStatus = "disabled"
		}

		info := fmt.Sprintf("Context window: %d / %d tokens (%.1f%%)\n", used, limit, pct)
		info += fmt.Sprintf("Summarization threshold: %d tokens (%.0f%%)\n", threshold, r.session.GetContextManager().GetSummarizeAt()*100)
		info += fmt.Sprintf("Auto-summarization: %s", autoStatus)

		r.displayInfo(info)
		return nil

	case "on", "enable":
		r.session.SetAutoSummarize(true)
		r.displaySystem("Auto-summarization ENABLED.")
		return nil

	case "off", "disable":
		r.session.SetAutoSummarize(false)
		r.displaySystem("Auto-summarization DISABLED.")
		return nil

	default:
		return fmt.Errorf("unknown context command: %s (use: show, on, off)", subcommand)
	}
}

func (r *REPL) handleMCPCommand(args string) error {
	if r.mcpManager == nil {
		r.displayInfo("MCP is not enabled. Add MCP servers to config.yaml and set mcp.enabled: true")
		return nil
	}

	subcommand := strings.ToLower(strings.TrimSpace(args))

	switch subcommand {
	case "", "status", "show":
		servers := r.mcpManager.ListServers()
		if len(servers) == 0 {
			r.displayInfo("No MCP servers connected.")
			return nil
		}

		counts := r.mcpManager.ServerToolCount()
		info := fmt.Sprintf("MCP Servers connected: %d\n", len(servers))
		for _, name := range servers {
			info += fmt.Sprintf("  - %s: %d tools\n", name, counts[name])
		}
		r.displayInfo(info)
		return nil

	case "tools", "list":
		tools := r.mcpManager.GetAllTools()
		if len(tools) == 0 {
			r.displayInfo("No MCP tools available.")
			return nil
		}

		info := fmt.Sprintf("Available MCP tools: %d\n", len(tools))
		for _, t := range tools {
			info += fmt.Sprintf("  - %s: %s\n", t.Name, t.Description)
		}
		r.displayInfo(info)
		return nil

	default:
		return fmt.Errorf("unknown mcp command: %s (use: status, tools)", subcommand)
	}
}

func (r *REPL) handleAskUserCommand(args string) error {
	subcommand := strings.ToLower(strings.TrimSpace(args))

	switch subcommand {
	case "", "show", "status":
		if r.session.IsAskUserEnabled() {
			r.displayInfo("Interactive questions: ENABLED\nAI can present menus with options for you to choose from.")
		} else {
			r.displayInfo("Interactive questions: DISABLED\nAI will not present interactive menus.")
		}
		return nil

	case "on", "enable":
		r.session.SetAskUserEnabled(true)
		r.displaySystem("Interactive questions ENABLED. AI can now present menus with options.")
		return nil

	case "off", "disable":
		r.session.SetAskUserEnabled(false)
		r.displaySystem("Interactive questions DISABLED.")
		return nil

	default:
		return fmt.Errorf("unknown askuser command: %s (use: on, off, show)", subcommand)
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

// DeleteHistoryFile removes the history file from disk.
func (r *REPL) DeleteHistoryFile() error {
	if !r.config.Session.SaveHistory {
		return nil
	}

	historyFile := r.config.Session.HistoryFile
	if historyFile == "" {
		return nil
	}

	// Check if file exists before trying to delete
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(historyFile)
}
