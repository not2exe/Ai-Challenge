package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/notexe/cli-chat/internal/api"
)

var (
	UserStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	AssistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226"))

	SystemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("201")).
			Italic(true)

	StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)

	TokenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))
)

type Formatter struct {
	colored         bool
	provider        string // display name (e.g., "DeepSeek", "Ollama")
	providerRaw     string // raw name (e.g., "deepseek", "ollama")
}

func NewFormatter(colored bool, provider ...string) *Formatter {
	displayName := "AI"
	rawName := ""
	if len(provider) > 0 && provider[0] != "" {
		rawName = provider[0]
		displayName = formatProviderName(provider[0])
	}
	return &Formatter{
		colored:     colored,
		provider:    displayName,
		providerRaw: rawName,
	}
}

// formatProviderName returns a display-friendly provider name.
func formatProviderName(provider string) string {
	switch provider {
	case "deepseek":
		return "DeepSeek"
	case "ollama":
		return "Ollama"
	default:
		// Capitalize first letter for unknown providers
		if len(provider) > 0 {
			return string(provider[0]-32) + provider[1:]
		}
		return provider
	}
}

func (f *Formatter) FormatUserMessage(msg string) string {
	prefix := "You: "
	if f.colored {
		prefix = UserStyle.Render("You: ")
	}
	return prefix + msg
}

func (f *Formatter) FormatAssistantMessage(msg string) string {
	prefix := f.provider + ": "
	if f.colored {
		prefix = AssistantStyle.Render(f.provider + ": ")
	}
	return prefix + msg
}

func (f *Formatter) FormatError(err error) string {
	prefix := "Error: "
	if f.colored {
		prefix = ErrorStyle.Render("Error: ")
	}
	return prefix + err.Error()
}

func (f *Formatter) FormatInfo(info string) string {
	if f.colored {
		return InfoStyle.Render(info)
	}
	return info
}

func (f *Formatter) FormatSystem(msg string) string {
	if f.colored {
		return SystemStyle.Render(msg)
	}
	return msg
}

func (f *Formatter) FormatStatus(msg string) string {
	if f.colored {
		return StatusStyle.Render(msg)
	}
	return msg
}

// TokenUsageOptions contains optional parameters for token usage display.
type TokenUsageOptions struct {
	Duration time.Duration
	Model    string
}

func (f *Formatter) FormatTokenUsage(usage api.Usage, opts ...TokenUsageOptions) string {
	var duration time.Duration
	var model string

	if len(opts) > 0 {
		duration = opts[0].Duration
		model = opts[0].Model
	}

	// Build the message parts
	parts := []string{
		fmt.Sprintf("tokens: input=%d, output=%d", usage.InputTokens, usage.OutputTokens),
	}

	// Add duration if provided
	if duration > 0 {
		parts = append(parts, fmt.Sprintf("time: %s", formatDuration(duration)))
	}

	// Add cost if applicable (DeepSeek models)
	cost := calculateCost(usage, model, f.providerRaw)
	if cost > 0 {
		parts = append(parts, fmt.Sprintf("cost: $%.6f", cost))
	}

	msg := "(" + joinParts(parts) + ")"

	if f.colored {
		return TokenStyle.Render(msg)
	}
	return msg
}

func joinParts(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += " | "
		}
		result += p
	}
	return result
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// DeepSeek pricing per 1M tokens (USD)
// https://api-docs.deepseek.com/quick_start/pricing
var deepSeekPricing = map[string]struct {
	inputPer1M  float64
	outputPer1M float64
}{
	"deepseek-chat": {
		inputPer1M:  0.14,  // $0.14 per 1M input tokens (cache miss)
		outputPer1M: 0.28,  // $0.28 per 1M output tokens
	},
	"deepseek-reasoner": {
		inputPer1M:  0.55,  // $0.55 per 1M input tokens (cache miss)
		outputPer1M: 2.19,  // $2.19 per 1M output tokens
	},
}

func calculateCost(usage api.Usage, model, provider string) float64 {
	// Ollama is free (local)
	if provider == "ollama" {
		return 0
	}

	// Look up pricing for the model
	pricing, ok := deepSeekPricing[model]
	if !ok {
		// Default to deepseek-chat pricing for unknown models
		pricing = deepSeekPricing["deepseek-chat"]
	}

	inputCost := float64(usage.InputTokens) * pricing.inputPer1M / 1_000_000
	outputCost := float64(usage.OutputTokens) * pricing.outputPer1M / 1_000_000

	return inputCost + outputCost
}

func (f *Formatter) FormatWelcome(model string, provider ...string) string {
	providerName := "DeepSeek"
	if len(provider) > 0 && provider[0] != "" {
		providerName = formatProviderName(provider[0])
	}

	lines := []string{
		"",
		fmt.Sprintf("Welcome to CLI Chat with %s!", providerName),
		fmt.Sprintf("Model: %s", model),
		"Type /help for available commands or start chatting.",
		"",
	}

	if f.colored {
		welcome := lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Bold(true)

		result := ""
		for i, line := range lines {
			if i == 1 {
				result += welcome.Render(line) + "\n"
			} else {
				result += line + "\n"
			}
		}
		return result
	}

	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}

func (f *Formatter) FormatHelp() string {
	lines := []string{
		"",
		"Available commands:",
		"  /help                - Show this help message",
		"  /clear               - Clear conversation history",
		"  /system <prompt>     - Update system prompt",
		"  /show                - Show current system prompt",
		"  /provider            - Show current provider and model",
		"  /temp <value>        - Set temperature (0-2), or show current if no value",
		"  /file <filename>     - Send file contents as prompt",
		"  /format json         - Enable JSON response format",
		"  /format show         - Display current format setting",
		"  /format clear        - Remove format restrictions",
		"  /clarify on          - Enable clarifying questions mode",
		"  /clarify off         - Disable clarifying questions mode",
		"  /clarify show        - Show clarify mode status",
		"  /context             - Show context window usage",
		"  /context on          - Enable auto-summarization",
		"  /context off         - Disable auto-summarization",
		"  /count               - Show message count in conversation",
		"  /quit or /exit       - Exit the chat",
		"",
		"CLI flags:",
		"  --provider <name>    - Use provider (deepseek, ollama)",
		"  --model <name>       - Override model name",
		"  --system-prompt      - Override system prompt",
		"  --no-color           - Disable colored output",
		"",
		"Tips:",
		"  - Press Ctrl+C or Ctrl+D to exit",
		"  - Use /clarify on to have AI ask questions before answering",
		"  - Your conversation history is maintained throughout the session",
		"  - Use /format json to get structured responses with tags, steps, URLs, etc.",
		"  - Temperature controls randomness: 0 = focused, 2 = creative",
		"  - For Ollama: ollama pull <model> to download models locally",
		"  - Context is auto-summarized at 70% to preserve conversation flow",
		"",
	}

	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}
