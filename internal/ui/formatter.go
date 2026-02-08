package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/notexe/cli-chat/internal/api"
)

var (
	// Modern color palette
	UserStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).  // Bright cyan
			Bold(true)

	AssistantStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114"))  // Soft green

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")).  // Coral red
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("222"))  // Warm yellow

	SystemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("183")).  // Soft purple
			Italic(true)

	StatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).  // Medium gray
			Italic(true)

	TokenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))  // Dim gray

	ToolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("215")).  // Orange
			Bold(true)

	// Box styles for modern UI
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).  // Soft blue border
			Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).  // Green
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("222")).  // Yellow
			Bold(true)

	AccentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("147"))  // Light purple
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

func (f *Formatter) FormatToolLabel(label string) string {
	if f.colored {
		return ToolStyle.Render(label)
	}
	return label
}

// TokenUsageOptions contains optional parameters for token usage display.
type TokenUsageOptions struct {
	Duration     time.Duration
	Model        string
	APICallCount int // Number of API calls made (for multi-step tool calls)
}

func (f *Formatter) FormatTokenUsage(usage api.Usage, opts ...TokenUsageOptions) string {
	var duration time.Duration
	var model string
	var apiCallCount int

	if len(opts) > 0 {
		duration = opts[0].Duration
		model = opts[0].Model
		apiCallCount = opts[0].APICallCount
	}

	// Build the message parts
	parts := []string{
		fmt.Sprintf("tokens: input=%d, output=%d", usage.InputTokens, usage.OutputTokens),
	}

	// Add API call count if more than 1
	if apiCallCount > 1 {
		parts = append(parts, fmt.Sprintf("api_calls: %d", apiCallCount))
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

	if f.colored {
		// Modern styled welcome
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)

		subtitleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

		labelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

		valueStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114"))

		borderStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("62"))

		// Build welcome box
		topBorder := borderStyle.Render("╭─────────────────────────────────────────╮")
		bottomBorder := borderStyle.Render("╰─────────────────────────────────────────╯")
		sideBorder := borderStyle.Render("│")

		title := titleStyle.Render(fmt.Sprintf("CLI Chat • %s", providerName))
		modelLine := labelStyle.Render("Model: ") + valueStyle.Render(model)
		helpLine := subtitleStyle.Render("Type /help for commands")

		// Pad lines to fit box
		padLine := func(content string, width int) string {
			contentLen := lipgloss.Width(content)
			if contentLen < width {
				return content + strings.Repeat(" ", width-contentLen)
			}
			return content
		}

		boxWidth := 39
		lines := []string{
			"",
			topBorder,
			sideBorder + " " + padLine(title, boxWidth) + " " + sideBorder,
			sideBorder + " " + padLine(modelLine, boxWidth) + " " + sideBorder,
			sideBorder + " " + padLine("", boxWidth) + " " + sideBorder,
			sideBorder + " " + padLine(helpLine, boxWidth) + " " + sideBorder,
			bottomBorder,
			"",
		}

		return strings.Join(lines, "\n")
	}

	// Plain text fallback
	lines := []string{
		"",
		fmt.Sprintf("CLI Chat • %s", providerName),
		fmt.Sprintf("Model: %s", model),
		"Type /help for commands",
		"",
	}

	return strings.Join(lines, "\n")
}

func (f *Formatter) FormatHelp() string {
	if f.colored {
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)

		cmdStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114"))

		descStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

		sectionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("147")).
			Bold(true)

		dimStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

		formatCmd := func(cmd, desc string) string {
			return "  " + cmdStyle.Render(cmd) + " " + descStyle.Render(desc)
		}

		lines := []string{
			"",
			headerStyle.Render("Commands"),
			"",
			sectionStyle.Render("General"),
			formatCmd("/help", "Show this help"),
			formatCmd("/clear", "Clear conversation"),
			formatCmd("/quit", "Exit chat"),
			"",
			sectionStyle.Render("Configuration"),
			formatCmd("/system <prompt>", "Set system prompt"),
			formatCmd("/show", "Show system prompt"),
			formatCmd("/provider", "Show provider info"),
			formatCmd("/temp <0-2>", "Set temperature"),
			"",
			sectionStyle.Render("Input"),
			formatCmd("/file <path>", "Send file content"),
			"",
			sectionStyle.Render("Features"),
			formatCmd("/clarify on|off", "Toggle clarifying questions"),
			formatCmd("/askuser on|off", "Toggle interactive menus"),
			formatCmd("/format json|clear", "Response format"),
			formatCmd("/context", "Context window status"),
			formatCmd("/mcp tools", "List MCP tools"),
			"",
			headerStyle.Render("Tips"),
			dimStyle.Render("  Ctrl+C or Ctrl+D to exit"),
			dimStyle.Render("  /clarify on for interactive clarification"),
			dimStyle.Render("  /format json for structured responses"),
			"",
		}

		return strings.Join(lines, "\n")
	}

	// Plain text fallback
	lines := []string{
		"",
		"Commands:",
		"  /help                - Show help",
		"  /clear               - Clear history",
		"  /system <prompt>     - Set system prompt",
		"  /show                - Show system prompt",
		"  /provider            - Show provider",
		"  /temp <value>        - Set temperature",
		"  /file <filename>     - Send file",
		"  /clarify on|off      - Toggle clarification",
		"  /format json|clear   - Response format",
		"  /context             - Context status",
		"  /mcp tools           - MCP tools",
		"  /quit                - Exit",
		"",
	}

	return strings.Join(lines, "\n")
}

// FormatPrompt returns a styled input prompt
func (f *Formatter) FormatPrompt() string {
	if f.colored {
		promptStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("62"))
		arrowStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true)
		return promptStyle.Render("you") + arrowStyle.Render(" > ")
	}
	return "you > "
}

// FormatContinuePrompt returns a styled continuation prompt
func (f *Formatter) FormatContinuePrompt() string {
	if f.colored {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("... ")
	}
	return "... "
}

// FormatPasteInfo returns styled paste mode info
func (f *Formatter) FormatPasteInfo(lineCount int) string {
	if f.colored {
		countStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Bold(true)
		textStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

		plural := "lines"
		if lineCount == 1 {
			plural = "line"
		}
		return textStyle.Render("Pasted ") + countStyle.Render(fmt.Sprintf("%d", lineCount)) + textStyle.Render(fmt.Sprintf(" %s", plural))
	}
	plural := "lines"
	if lineCount == 1 {
		plural = "line"
	}
	return fmt.Sprintf("Pasted %d %s", lineCount, plural)
}

// FormatBox wraps content in a styled box
func (f *Formatter) FormatBox(title, content string) string {
	if f.colored {
		titleStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)

		borderStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

		header := titleStyle.Render(title)
		box := borderStyle.Render(content)

		return header + "\n" + box
	}
	return title + "\n" + content
}
