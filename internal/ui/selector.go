package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// SelectorOption represents a single option in the selector
type SelectorOption struct {
	Label       string
	Description string
}

// Selector provides an interactive arrow-key navigable menu
type Selector struct {
	question    string
	options     []SelectorOption
	selected    int
	multiSelect bool
	selections  map[int]bool
	colored     bool

	cursorStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	optionStyle   lipgloss.Style
	dimStyle      lipgloss.Style
	questionStyle lipgloss.Style
	hintStyle     lipgloss.Style
}

// NewSelector creates a new interactive selector
func NewSelector(question string, options []SelectorOption, multiSelect bool, colored bool) *Selector {
	return &Selector{
		question:    question,
		options:     options,
		selected:    0,
		multiSelect: multiSelect,
		selections:  make(map[int]bool),
		colored:     colored,

		cursorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true),
		selectedStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("114")).Bold(true),
		optionStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		dimStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		questionStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true),
		hintStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true),
	}
}

// Run displays the selector and returns the selected option(s)
func (s *Selector) Run() ([]string, error) {
	fd := int(os.Stdin.Fd())

	if !term.IsTerminal(fd) {
		return s.runSimple()
	}

	// Save and set raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return s.runSimple()
	}

	// Cleanup function
	cleanup := func() {
		term.Restore(fd, oldState)
		fmt.Print("\033[?25h") // Show cursor
	}
	defer cleanup()

	// Hide cursor
	fmt.Print("\033[?25l")

	// Calculate total lines for clearing
	totalLines := len(s.options) + 3

	// Initial render
	s.printMenu()

	reader := bufio.NewReader(os.Stdin)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}

		action := ""

		switch b {
		case 13, 10: // Enter
			action = "select"
		case 3: // Ctrl+C
			// Clear and exit
			s.clearMenu(totalLines)
			return nil, fmt.Errorf("cancelled")
		case 'j': // vim down
			s.moveDown()
		case 'k': // vim up
			s.moveUp()
		case ' ': // Space
			if s.multiSelect {
				s.toggleSelection()
			} else {
				action = "select"
			}
		case 27: // Escape sequence
			b2, _ := reader.ReadByte()
			if b2 == '[' {
				b3, _ := reader.ReadByte()
				switch b3 {
				case 'A': // Up
					s.moveUp()
				case 'B': // Down
					s.moveDown()
				}
			}
		default:
			if b >= '1' && b <= '9' {
				idx := int(b - '1')
				if idx < len(s.options) {
					s.selected = idx
					if !s.multiSelect {
						action = "select"
					} else {
						s.toggleSelection()
					}
				}
			}
		}

		if action == "select" {
			s.clearMenu(totalLines)
			return s.getSelected(), nil
		}

		// Redraw
		s.clearMenu(totalLines)
		s.printMenu()
	}
}

func (s *Selector) printMenu() {
	var sb strings.Builder

	// Question
	if s.colored {
		sb.WriteString(s.questionStyle.Render(s.question))
	} else {
		sb.WriteString(s.question)
	}
	sb.WriteString("\r\n")

	// Hint
	hint := "[j/k or arrows] move  [enter] select"
	if s.multiSelect {
		hint = "[j/k or arrows] move  [space] toggle  [enter] confirm"
	}
	if s.colored {
		sb.WriteString(s.hintStyle.Render(hint))
	} else {
		sb.WriteString(hint)
	}
	sb.WriteString("\r\n\r\n")

	// Options
	for i, opt := range s.options {
		cursor := "  "
		if i == s.selected {
			cursor = "> "
		}

		checkbox := ""
		if s.multiSelect {
			if s.selections[i] {
				checkbox = "[x] "
			} else {
				checkbox = "[ ] "
			}
		}

		label := opt.Label
		if opt.Description != "" {
			label += " - " + opt.Description
		}

		if s.colored {
			if i == s.selected {
				sb.WriteString(s.cursorStyle.Render(cursor))
				sb.WriteString(checkbox)
				sb.WriteString(s.selectedStyle.Render(label))
			} else {
				sb.WriteString(s.dimStyle.Render(cursor))
				sb.WriteString(checkbox)
				sb.WriteString(s.optionStyle.Render(label))
			}
		} else {
			sb.WriteString(cursor + checkbox + label)
		}
		sb.WriteString("\r\n")
	}

	fmt.Print(sb.String())
	os.Stdout.Sync()
}

func (s *Selector) clearMenu(lines int) {
	// Move cursor up and clear each line
	for i := 0; i < lines; i++ {
		fmt.Print("\033[A\033[2K\r")
	}
	os.Stdout.Sync()
}

func (s *Selector) runSimple() ([]string, error) {
	fmt.Println(s.question)
	for i, opt := range s.options {
		label := opt.Label
		if opt.Description != "" {
			label += " - " + opt.Description
		}
		fmt.Printf("  [%d] %s\n", i+1, label)
	}
	fmt.Print("Enter number: ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if len(input) >= 1 && input[0] >= '1' && input[0] <= '9' {
		idx := int(input[0] - '1')
		if idx < len(s.options) {
			return []string{s.options[idx].Label}, nil
		}
	}

	return []string{s.options[0].Label}, nil
}

func (s *Selector) moveUp() {
	if s.selected > 0 {
		s.selected--
	} else {
		s.selected = len(s.options) - 1
	}
}

func (s *Selector) moveDown() {
	if s.selected < len(s.options)-1 {
		s.selected++
	} else {
		s.selected = 0
	}
}

func (s *Selector) toggleSelection() {
	s.selections[s.selected] = !s.selections[s.selected]
}

func (s *Selector) getSelected() []string {
	if s.multiSelect {
		var result []string
		for i, opt := range s.options {
			if s.selections[i] {
				result = append(result, opt.Label)
			}
		}
		if len(result) == 0 {
			return []string{s.options[s.selected].Label}
		}
		return result
	}
	return []string{s.options[s.selected].Label}
}

// RunWithCustomOption adds an "Other" option for custom input
func (s *Selector) RunWithCustomOption() ([]string, bool, error) {
	s.options = append(s.options, SelectorOption{
		Label:       "Other",
		Description: "Type custom answer",
	})

	result, err := s.Run()
	if err != nil {
		return nil, false, err
	}

	for _, r := range result {
		if r == "Other" {
			return nil, true, nil
		}
	}

	return result, false, nil
}
