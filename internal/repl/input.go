package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/chzyer/readline"
)

// Styles for input UI
var (
	pastedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
)

func (r *REPL) readInput() (string, error) {
	line, err := r.rl.Readline()
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(line)

	// If it's a command, return immediately
	if strings.HasPrefix(trimmed, "/") {
		return trimmed, nil
	}

	// Check if empty
	if trimmed == "" {
		return "", nil
	}

	// Check for bracketed paste (embedded newlines)
	if strings.Contains(line, "\n") {
		lineCount := strings.Count(line, "\n") + 1
		r.showPastedIndicator(lineCount)
	}

	return trimmed, nil
}

// showPastedIndicator clears the pasted lines and shows "[Pasted X lines]"
func (r *REPL) showPastedIndicator(lineCount int) {
	// Clear the pasted content and show indicator
	// Move up for each line we need to clear
	for i := 0; i < lineCount; i++ {
		fmt.Print("\033[1A") // Move up
		fmt.Print("\033[K")  // Clear line
	}

	plural := "lines"
	if lineCount == 1 {
		plural = "line"
	}

	indicator := pastedStyle.Render(fmt.Sprintf("[Pasted %d %s]", lineCount, plural))
	fmt.Println(getPrompt() + indicator)
}

func (r *REPL) parseCommand(input string) (bool, string, string) {
	if !strings.HasPrefix(input, "/") {
		return false, "", ""
	}

	parts := strings.SplitN(input, " ", 2)
	command := strings.ToLower(parts[0])

	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	return true, command, args
}

// getPrompt returns the styled prompt string
func getPrompt() string {
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("62"))
	arrowStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("114")).
		Bold(true)
	return promptStyle.Render("you") + arrowStyle.Render(" > ")
}

func setupReadline() (*readline.Instance, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:              getPrompt(),
		HistoryFile:         "",
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})

	return rl, err
}

func filterInput(r rune) (rune, bool) {
	switch r {
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

func isEOF(err error) bool {
	return err == io.EOF || err == readline.ErrInterrupt
}
