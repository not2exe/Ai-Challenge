package repl

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/notexe/cli-chat/internal/chat"
	"github.com/notexe/cli-chat/internal/ui"
)

// Styles for question display
var (
	questionTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("81")).
		Bold(true)

	counterStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	selectedResultStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("114")).
		Bold(true)
)

// AskClarifyingQuestions presents questions interactively and collects answers
func (r *REPL) AskClarifyingQuestions(questions []chat.ClarifyQuestion) ([]chat.QuestionAnswer, error) {
	var answers []chat.QuestionAnswer

	// Header
	fmt.Println()
	header := questionTitleStyle.Render("Clarifying Questions")
	subtext := counterStyle.Render(fmt.Sprintf("Please answer %d question(s)", len(questions)))
	fmt.Println(header)
	fmt.Println(subtext)
	fmt.Println()

	// Temporarily close readline to avoid terminal conflicts
	r.rl.Close()

	for i, q := range questions {
		// Show progress
		progress := counterStyle.Render(fmt.Sprintf("[%d/%d]", i+1, len(questions)))
		fmt.Println(progress)

		// Convert options to SelectorOption
		options := make([]ui.SelectorOption, len(q.Options))
		for j, opt := range q.Options {
			options[j] = ui.SelectorOption{Label: opt}
		}

		// Create and run selector
		selector := ui.NewSelector(q.Question, options, false, r.config.UI.ColoredOutput)

		var answer string
		var err error
		if q.AllowCustom {
			// Add "Other" option
			result, needsCustom, runErr := selector.RunWithCustomOption()
			if runErr != nil {
				// Restore readline before returning
				if newRl, rlErr := setupReadline(); rlErr == nil {
					r.rl = newRl
				}
				return nil, runErr
			}
			if needsCustom {
				// Restore readline for custom input
				if newRl, rlErr := setupReadline(); rlErr == nil {
					r.rl = newRl
				}
				answer, err = r.getCustomInput()
				if err != nil {
					return nil, err
				}
				// Close again for next question
				r.rl.Close()
			} else {
				answer = strings.Join(result, ", ")
			}
		} else {
			result, runErr := selector.Run()
			if runErr != nil {
				if newRl, rlErr := setupReadline(); rlErr == nil {
					r.rl = newRl
				}
				return nil, runErr
			}
			answer = strings.Join(result, ", ")
		}

		// Show what was selected
		fmt.Println(selectedResultStyle.Render("→ " + answer))
		fmt.Println()

		answers = append(answers, chat.QuestionAnswer{
			Question: q.Question,
			Answer:   answer,
		})
	}

	// Restore readline
	if newRl, rlErr := setupReadline(); rlErr == nil {
		r.rl = newRl
	}

	return answers, nil
}

// AskUserQuestion presents a single question with options using interactive selector
func (r *REPL) AskUserQuestion(question string, options []string, multiSelect bool) ([]string, error) {
	fmt.Println()

	// Convert to SelectorOption
	selectorOptions := make([]ui.SelectorOption, len(options))
	for i, opt := range options {
		// Split label and description if present
		parts := strings.SplitN(opt, " - ", 2)
		selectorOptions[i] = ui.SelectorOption{Label: parts[0]}
		if len(parts) > 1 {
			selectorOptions[i].Description = parts[1]
		}
	}

	// Temporarily close readline to avoid terminal conflicts
	r.rl.Close()

	// Create selector
	selector := ui.NewSelector(question, selectorOptions, multiSelect, r.config.UI.ColoredOutput)

	// Run with custom option
	result, needsCustom, err := selector.RunWithCustomOption()

	// Recreate readline
	newRl, rlErr := setupReadline()
	if rlErr == nil {
		r.rl = newRl
	}

	if err != nil {
		return nil, err
	}

	if needsCustom {
		custom, err := r.getCustomInput()
		if err != nil {
			return nil, err
		}
		result = []string{custom}
	}

	// Show selection
	fmt.Println(selectedResultStyle.Render("→ " + strings.Join(result, ", ")))
	fmt.Println()

	return result, nil
}

// getCustomInput prompts for custom text input
func (r *REPL) getCustomInput() (string, error) {
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("147"))
	fmt.Print(promptStyle.Render("Your answer: "))

	r.rl.SetPrompt("")
	defer r.rl.SetPrompt("you > ")

	input, err := r.rl.Readline()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(input), nil
}
