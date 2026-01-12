package repl

import (
	"fmt"

	"github.com/notexe/cli-chat/internal/api"
)

func (r *REPL) displayResponse(response *api.MessageResponse) {
	r.status.Hide()

	fmt.Println(r.formatter.FormatAssistantMessage(response.Content))

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
