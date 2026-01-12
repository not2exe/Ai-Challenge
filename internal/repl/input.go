package repl

import (
	"io"
	"strings"

	"github.com/chzyer/readline"
)

func (r *REPL) readInput() (string, error) {
	line, err := r.rl.Readline()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(line), nil
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
