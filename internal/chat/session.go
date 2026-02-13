package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/config"
)

type Session struct {
	history         *History
	systemPrompt    string
	formatPrompt    string
	toolsPrompt     string // Additional prompt for available tools guidance
	projectPrompt   string // Auto-detected project/git context
	askUserEnabled  bool   // Enable ask_user tool for interactive questions
	clarifyEnabled  bool
	config          *config.ModelConfig
	contextMgr      *ContextManager
	lastInputTokens int  // Tokens from last API request (for tracking)
	autoSummarize   bool // Whether to auto-summarize when threshold reached
}

type SessionData struct {
	Messages     []api.Message `json:"messages"`
	SystemPrompt string        `json:"system_prompt"`
	FormatPrompt string        `json:"format_prompt"`
	Timestamp    time.Time     `json:"timestamp"`
}

func NewSession(cfg *config.ModelConfig, maxHistory int) *Session {
	return &Session{
		history:        NewHistory(maxHistory),
		systemPrompt:   cfg.SystemPrompt,
		config:         cfg,
		contextMgr:     NewContextManager(0.70, 0.40), // Default thresholds
		autoSummarize:  true,
		askUserEnabled: true, // Enable ask_user tool by default
	}
}

// NewSessionWithContext creates a new session with custom context configuration.
func NewSessionWithContext(cfg *config.ModelConfig, maxHistory int, contextCfg *config.ContextConfig) *Session {
	session := &Session{
		history:        NewHistory(maxHistory),
		systemPrompt:   cfg.SystemPrompt,
		config:         cfg,
		autoSummarize:  true,
		askUserEnabled: true, // Enable ask_user tool by default
	}

	if contextCfg != nil {
		session.contextMgr = NewContextManager(contextCfg.SummarizeAt, contextCfg.TargetAfter)
		session.autoSummarize = contextCfg.AutoSummarize
	} else {
		session.contextMgr = NewContextManager(0.70, 0.40)
	}

	// Apply custom context window if specified
	if cfg.ContextWindow > 0 {
		session.contextMgr.SetModelLimit(cfg.Name, cfg.ContextWindow)
	}

	return session
}

func (s *Session) AddUserMessage(content string) {
	s.history.Add(api.Message{
		Role:    "user",
		Content: content,
	})
}

func (s *Session) AddAssistantMessage(content string) {
	s.history.Add(api.Message{
		Role:    "assistant",
		Content: content,
	})
}

func (s *Session) GetMessages() []api.Message {
	return s.history.GetAll()
}

func (s *Session) SetSystemPrompt(prompt string) error {
	if err := ValidateSystemPrompt(prompt); err != nil {
		return err
	}
	s.systemPrompt = prompt
	return nil
}

func (s *Session) GetSystemPrompt() string {
	return s.systemPrompt
}

func (s *Session) SetFormatPrompt(prompt string) error {
	if err := ValidateFormatPrompt(prompt); err != nil {
		return err
	}
	s.formatPrompt = prompt
	return nil
}

func (s *Session) GetFormatPrompt() string {
	return s.formatPrompt
}

func (s *Session) ClearFormatPrompt() {
	s.formatPrompt = ""
}

// SetToolsPrompt sets additional guidance for available tools.
func (s *Session) SetToolsPrompt(prompt string) {
	s.toolsPrompt = prompt
}

// GetToolsPrompt returns the current tools prompt.
func (s *Session) GetToolsPrompt() string {
	return s.toolsPrompt
}

// SetProjectPrompt sets the auto-detected project context prompt.
func (s *Session) SetProjectPrompt(prompt string) {
	s.projectPrompt = prompt
}

func (s *Session) SetClarifyMode(enabled bool) {
	s.clarifyEnabled = enabled
}

func (s *Session) IsClarifyEnabled() bool {
	return s.clarifyEnabled
}

// SetAskUserEnabled enables or disables the ask_user interactive tool.
func (s *Session) SetAskUserEnabled(enabled bool) {
	s.askUserEnabled = enabled
}

// IsAskUserEnabled returns whether the ask_user tool is enabled.
func (s *Session) IsAskUserEnabled() bool {
	return s.askUserEnabled
}

func (s *Session) SetTemperature(temp float64) error {
	if temp < 0 || temp > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}
	s.config.Temperature = temp
	return nil
}

func (s *Session) GetTemperature() float64 {
	return s.config.Temperature
}

func (s *Session) Clear() {
	s.history.Clear()
	s.ClearFormatPrompt()
}

func (s *Session) IsEmpty() bool {
	return s.history.IsEmpty()
}

func (s *Session) MessageCount() int {
	return s.history.Size()
}

func (s *Session) BuildAPIRequest() api.MessageRequest {
	return s.buildAPIRequest(true)
}

func (s *Session) BuildAPIRequestWithoutClarify() api.MessageRequest {
	return s.buildAPIRequest(false)
}

func (s *Session) buildAPIRequest(includeClarify bool) api.MessageRequest {
	var clarifyPrompt string
	if s.clarifyEnabled && includeClarify {
		clarifyPrompt = GetClarifyPrompt()
	}

	// Add ask_user prompt to guide the model on when to use the tool
	var askUserPrompt string
	if s.askUserEnabled {
		askUserPrompt = AskUserToolPrompt
	}

	systemPrompt := BuildSystemPrompt(s.systemPrompt, s.projectPrompt, s.toolsPrompt, s.formatPrompt, clarifyPrompt, askUserPrompt)

	return api.MessageRequest{
		Messages:    s.history.GetAll(),
		System:      systemPrompt,
		Model:       s.config.Name,
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
	}
}

func (s *Session) Save(filepath string) error {
	data := SessionData{
		Messages:     s.history.GetAll(),
		SystemPrompt: s.systemPrompt,
		FormatPrompt: s.formatPrompt,
		Timestamp:    time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(filepath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

func (s *Session) Load(filepath string) error {
	jsonData, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read session file: %w", err)
	}

	var data SessionData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return fmt.Errorf("failed to unmarshal session: %w", err)
	}

	s.history.Clear()
	for _, msg := range data.Messages {
		s.history.Add(msg)
	}
	s.systemPrompt = data.SystemPrompt
	s.formatPrompt = data.FormatPrompt

	return nil
}

// UpdateTokensFromResponse updates the session's token tracking from API response.
func (s *Session) UpdateTokensFromResponse(usage api.Usage) {
	s.lastInputTokens = usage.InputTokens
}

// ResetInputTokens resets the token counter (used after summarization).
func (s *Session) ResetInputTokens() {
	s.lastInputTokens = 0
}

// NeedsSummarization checks if the context needs summarization based on current token usage.
func (s *Session) NeedsSummarization() bool {
	if !s.autoSummarize || s.lastInputTokens == 0 {
		return false
	}

	modelLimit := s.contextMgr.GetModelLimit(s.config.Name)
	return s.contextMgr.ShouldSummarize(s.lastInputTokens, modelLimit)
}

// GetContextStatus returns the current context usage status.
// Returns: used tokens, model limit, percentage used.
func (s *Session) GetContextStatus() (used int, limit int, pct float64) {
	limit = s.contextMgr.GetModelLimit(s.config.Name)
	used = s.lastInputTokens
	pct = s.contextMgr.GetUsagePercent(used, limit)
	return
}

// GetMessagesToSummarize returns the messages that should be summarized.
// keepLast specifies how many recent message pairs to preserve.
func (s *Session) GetMessagesToSummarize(keepLast int) (toSummarize []api.Message, toKeep []api.Message) {
	return CalculateMessagesToSummarize(s.history.GetAll(), keepLast)
}

// ApplySummary replaces old messages with a summary.
func (s *Session) ApplySummary(summary api.Message, keptMessages int) {
	s.history.ReplaceWithSummary(summary, keptMessages)
}

// SetAutoSummarize enables or disables automatic summarization.
func (s *Session) SetAutoSummarize(enabled bool) {
	s.autoSummarize = enabled
}

// IsAutoSummarizeEnabled returns whether automatic summarization is enabled.
func (s *Session) IsAutoSummarizeEnabled() bool {
	return s.autoSummarize
}

// GetModelName returns the current model name.
func (s *Session) GetModelName() string {
	return s.config.Name
}

// GetMaxTokens returns the max tokens setting for API requests.
func (s *Session) GetMaxTokens() int {
	return s.config.MaxTokens
}

// GetContextManager returns the context manager for advanced operations.
func (s *Session) GetContextManager() *ContextManager {
	return s.contextMgr
}

// AddAssistantMessageWithToolCalls adds an assistant message that contains tool call requests.
// This must be called before AddToolResult to maintain proper message ordering.
func (s *Session) AddAssistantMessageWithToolCalls(content string, toolCalls []api.ToolCall) {
	s.history.Add(api.Message{
		Role:      "assistant",
		Content:   content,
		ToolCalls: toolCalls,
	})
}

// AddToolResult adds a tool execution result to the conversation history.
func (s *Session) AddToolResult(toolCallID, toolName, result string) {
	s.history.Add(api.Message{
		Role:       "tool",
		Content:    result,
		ToolCallID: toolCallID,
	})
}

// BuildAPIRequestWithToolResults builds a request that includes pending tool results.
func (s *Session) BuildAPIRequestWithToolResults() api.MessageRequest {
	systemPrompt := BuildSystemPrompt(s.systemPrompt, s.projectPrompt, s.toolsPrompt, s.formatPrompt, "")

	return api.MessageRequest{
		Messages:    s.history.GetAll(),
		System:      systemPrompt,
		Model:       s.config.Name,
		MaxTokens:   s.config.MaxTokens,
		Temperature: s.config.Temperature,
	}
}
