package repl

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/notexe/cli-chat/internal/api"
	"github.com/notexe/cli-chat/internal/chat"
)

func (r *REPL) displayResponse(response *api.MessageResponse) {
	r.status.Hide()

	displayContent := response.Content

	if r.session.GetFormatPrompt() != "" {
		if chat.HasMarkdownCodeBlocks(response.Content) {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Bold(true)

			fmt.Println()
			fmt.Println(errorStyle.Render("╔════════════════════════════════════════════════════════════════════╗"))
			fmt.Println(errorStyle.Render("║                    ⚠️  MODEL FORMAT ERROR ⚠️                      ║"))
			fmt.Println(errorStyle.Render("╠════════════════════════════════════════════════════════════════════╣"))
			fmt.Println(errorStyle.Render("║ The model responded with markdown code blocks (```)               ║"))
			fmt.Println(errorStyle.Render("║ instead of raw JSON as instructed.                                 ║"))
			fmt.Println(errorStyle.Render("║                                                                    ║"))
			fmt.Println(errorStyle.Render("║ This violates the format instructions.                            ║"))
			fmt.Println(errorStyle.Render("║ The response will be cleaned automatically, but the model         ║"))
			fmt.Println(errorStyle.Render("║ should follow instructions properly.                               ║"))
			fmt.Println(errorStyle.Render("╚════════════════════════════════════════════════════════════════════╝"))
			fmt.Println()
		}
		displayContent = chat.CleanMarkdownCodeBlocks(displayContent)
	}

	fmt.Println(r.formatter.FormatAssistantMessage(displayContent))

	if r.session.GetFormatPrompt() != "" {
		if parsed, err := chat.ParseJSONResponse(response.Content); err == nil {
			fmt.Println(chat.FormatJSONTable(parsed))
		}
	}

	if r.config.UI.ShowTokenCount {
		fmt.Println(r.formatter.FormatTokenUsage(response.Usage))
	}

	fmt.Println()
}

func (r *REPL) displayError(err error) {
	r.status.Hide()
	fmt.Println(r.formatter.FormatError(err))
	fmt.Println()
}

func (r *REPL) displayWelcome() {
	fmt.Print(r.formatter.FormatWelcome(r.config.Model.Name))
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
