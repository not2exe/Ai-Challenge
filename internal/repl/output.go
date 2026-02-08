package repl

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/chat"
	"github.com/notexe/cli-chat/internal/ui"
)

func (r *REPL) displayResponse(response *api.MessageResponse, duration time.Duration) {
	r.displayResponseWithUsage(response, duration, response.Usage, 1)
}

func (r *REPL) displayResponseWithUsage(response *api.MessageResponse, duration time.Duration, cumulativeUsage api.Usage, apiCallCount int) {
	r.status.Hide()

	// Apply terminal formatting (markdown/LaTeX cleanup)
	displayContent := chat.FormatForTerminal(response.Content)

	if r.session.GetFormatPrompt() != "" {
		if chat.HasMarkdownCodeBlocks(response.Content) {
			// Modern styled error box
			borderStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("203"))
			titleStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("203")).
				Bold(true)
			textStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

			fmt.Println()
			fmt.Println(borderStyle.Render("╭──────────────────────────────────────────╮"))
			fmt.Println(borderStyle.Render("│ ") + titleStyle.Render("Format Warning") + borderStyle.Render("                          │"))
			fmt.Println(borderStyle.Render("├──────────────────────────────────────────┤"))
			fmt.Println(borderStyle.Render("│ ") + textStyle.Render("Model used markdown code blocks") + borderStyle.Render("        │"))
			fmt.Println(borderStyle.Render("│ ") + textStyle.Render("Auto-cleaning response...") + borderStyle.Render("              │"))
			fmt.Println(borderStyle.Render("╰──────────────────────────────────────────╯"))
			fmt.Println()
		}
		displayContent = chat.CleanMarkdownCodeBlocks(displayContent)
	}

	fmt.Println()
	fmt.Println(r.formatter.FormatAssistantMessage(displayContent))

	if r.session.GetFormatPrompt() != "" {
		if parsed, err := chat.ParseJSONResponse(response.Content); err == nil {
			fmt.Println(chat.FormatJSONTable(parsed))
		}
	}

	if r.config.UI.ShowTokenCount {
		fmt.Println(r.formatter.FormatTokenUsage(cumulativeUsage, ui.TokenUsageOptions{
			Duration:     duration,
			Model:        r.config.Model.Name,
			APICallCount: apiCallCount,
		}))
	}

	fmt.Println()
	os.Stdout.Sync() // Flush to ensure output displays immediately
}

func (r *REPL) displayError(err error) {
	r.status.Hide()
	fmt.Println(r.formatter.FormatError(err))
	fmt.Println()
}

func (r *REPL) displayWelcome() {
	fmt.Print(r.formatter.FormatWelcome(r.config.Model.Name, r.provider.Name()))
}

func (r *REPL) displayHelp() {
	fmt.Print(r.formatter.FormatHelp())
}

func (r *REPL) displayInfo(msg string) {
	fmt.Println(r.formatter.FormatInfo(msg))
	fmt.Println()
}

func (r *REPL) displaySystem(msg string) {
	fmt.Println(r.formatter.FormatSystem(msg))
	fmt.Println()
}
