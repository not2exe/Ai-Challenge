package repl

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/chzyer/readline"
)

// Styles for input UI
var (
	pastedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Italic(true)
)

// pasteTimeout - lines arriving within this time are considered part of a paste
const pasteTimeout = 50 * time.Millisecond

// inputResult holds a line read from readline
type inputResult struct {
	line string
	err  error
}

// inputReader manages async reading from readline
type inputReader struct {
	rl       *readline.Instance
	lineCh   chan inputResult
	running  bool
}

// newInputReader creates a new input reader
func newInputReader(rl *readline.Instance) *inputReader {
	return &inputReader{
		rl:     rl,
		lineCh: make(chan inputResult, 10), // Buffer for paste detection
	}
}

// start begins reading input in background
func (ir *inputReader) start() {
	if ir.running {
		return
	}
	ir.running = true
	go ir.readLoop()
}

// readLoop continuously reads lines and sends them to the channel
func (ir *inputReader) readLoop() {
	for ir.running {
		line, err := ir.rl.Readline()
		ir.lineCh <- inputResult{line, err}
		if err != nil {
			return
		}
	}
}

// stop stops the input reader
func (ir *inputReader) stop() {
	ir.running = false
}

// readWithPasteDetection reads input with paste detection
func (ir *inputReader) readWithPasteDetection() (string, bool, error) {
	// Wait for first line (blocking)
	result := <-ir.lineCh
	if result.err != nil {
		return "", false, result.err
	}

	line := result.line
	trimmed := strings.TrimSpace(line)

	// Check for embedded newlines (bracketed paste)
	if strings.Contains(line, "\n") {
		return strings.TrimSpace(line), true, nil
	}

	// Collect any additional lines that arrive quickly (paste detection)
	lines := []string{line}

	for {
		select {
		case nextResult := <-ir.lineCh:
			if nextResult.err != nil {
				// Error - return what we have
				if len(lines) == 1 {
					return trimmed, false, nil
				}
				return strings.TrimSpace(strings.Join(lines, "\n")), true, nil
			}
			lines = append(lines, nextResult.line)
			// Continue collecting

		case <-time.After(pasteTimeout):
			// No more lines in time window
			if len(lines) == 1 {
				return trimmed, false, nil
			}
			return strings.TrimSpace(strings.Join(lines, "\n")), true, nil
		}
	}
}

func (r *REPL) readInput() (string, error) {
	// Initialize input reader if needed
	if r.inputReader == nil {
		r.inputReader = newInputReader(r.rl)
		r.inputReader.start()
	}

	// Read with paste detection
	content, wasPaste, err := r.inputReader.readWithPasteDetection()
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(content)

	// If it's a command, return immediately
	if strings.HasPrefix(trimmed, "/") {
		return trimmed, nil
	}

	// Check if empty
	if trimmed == "" {
		return "", nil
	}

	// Show paste indicator if it was a paste
	if wasPaste {
		lineCount := strings.Count(content, "\n") + 1
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
