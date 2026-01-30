package repl

import (
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
)

func (r *REPL) readInput() (string, error) {
	// Read first line
	line, err := r.rl.Readline()
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(line)

	// Check for paste mode command
	if trimmed == "/paste" {
		return r.readPasteMode()
	}

	// If it's a command, return immediately
	if strings.HasPrefix(trimmed, "/") {
		return trimmed, nil
	}

	// Check if line is empty or just whitespace
	if trimmed == "" {
		return "", nil
	}

	// Start collecting lines
	var lines []string
	lines = append(lines, line)

	// Enter multi-line mode
	fmt.Print("\033[90m(Press Enter twice to submit, or type 'END' on new line)\033[0m\n")
	r.rl.SetPrompt("... ")

	for {
		nextLine, err := r.rl.Readline()
		if err != nil {
			r.rl.SetPrompt("Type here: ")
			return "", err
		}

		nextTrimmed := strings.TrimSpace(nextLine)

		// Check for END terminator
		if nextTrimmed == "END" || nextTrimmed == "<<<" {
			break
		}

		// If line is empty, submit
		if nextTrimmed == "" {
			break
		}

		lines = append(lines, nextLine)
	}

	r.rl.SetPrompt("Type here: ")
	result := strings.Join(lines, "\n")
	return strings.TrimSpace(result), nil
}

// readPasteMode enters paste mode for multi-line content
func (r *REPL) readPasteMode() (string, error) {
	fmt.Println("\033[92m=== PASTE MODE ===\033[0m")
	fmt.Println("\033[90mPaste your content, then type 'END' on a new line and press Enter\033[0m")
	fmt.Println()

	var lines []string
	r.rl.SetPrompt("")

	for {
		line, err := r.rl.Readline()
		if err != nil {
			r.rl.SetPrompt("Type here: ")
			return "", err
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "END" || trimmed == "<<<" {
			break
		}

		lines = append(lines, line)
	}

	r.rl.SetPrompt("Type here: ")
	result := strings.Join(lines, "\n")
	fmt.Println("\033[92m=== END PASTE MODE ===\033[0m")
	return strings.TrimSpace(result), nil
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

func setupReadline() (*readline.Instance, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:              "Type here: ",
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
