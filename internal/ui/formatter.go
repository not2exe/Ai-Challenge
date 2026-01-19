package ui

import (
	"fmt"

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
	colored bool
}

func NewFormatter(colored bool) *Formatter {
	return &Formatter{
		colored: colored,
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
	prefix := "DeepSeek: "
	if f.colored {
		prefix = AssistantStyle.Render("DeepSeek: ")
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

func (f *Formatter) FormatTokenUsage(usage api.Usage) string {
	msg := fmt.Sprintf("(tokens: input=%d, output=%d)",
		usage.InputTokens, usage.OutputTokens)

	if f.colored {
		return TokenStyle.Render(msg)
	}
	return msg
}

func (f *Formatter) FormatWelcome(model string) string {
	lines := []string{
		"",
		"Welcome to CLI Chat with DeepSeek!",
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
		"  /temp <value>        - Set temperature (0-2), or show current if no value",
		"  /format json         - Enable JSON response format",
		"  /format show         - Display current format setting",
		"  /format clear        - Remove format restrictions",
		"  /clarify on          - Enable clarifying questions mode",
		"  /clarify off         - Disable clarifying questions mode",
		"  /clarify show        - Show clarify mode status",
		"  /quit or /exit       - Exit the chat",
		"",
		"Tips:",
		"  - Press Ctrl+C or Ctrl+D to exit",
		"  - Use /clarify on to have AI ask questions before answering",
		"  - Your conversation history is maintained throughout the session",
		"  - Use /format json to get structured responses with tags, steps, URLs, etc.",
		"  - Temperature controls randomness: 0 = focused, 2 = creative",
		"",
	}

	result := ""
	for _, line := range lines {
		result += line + "\n"
	}
	return result
}
