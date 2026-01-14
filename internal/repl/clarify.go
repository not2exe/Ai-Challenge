package repl

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/notexe/cli-chat/internal/chat"
)

var (
	questionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("117")).
		Bold(true)

	optionStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("228"))

	headerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")).
		Bold(true).
		Underline(true)
)

// AskClarifyingQuestions presents questions interactively and collects answers
func (r *REPL) AskClarifyingQuestions(questions []chat.ClarifyQuestion) ([]chat.QuestionAnswer, error) {
	var answers []chat.QuestionAnswer

	fmt.Println("\n" + headerStyle.Render("📋 Clarifying Questions"))
	fmt.Println(r.formatter.FormatInfo(
		fmt.Sprintf("Please answer %d question(s) to help provide a better response:", len(questions)),
	))
	fmt.Println()

	for i, q := range questions {
		answer, err := r.askSingleQuestion(i+1, len(questions), q)
		if err != nil {
			return nil, err
		}

		answers = append(answers, chat.QuestionAnswer{
			Question: q.Question,
			Answer:   answer,
		})
	}

	return answers, nil
}

func (r *REPL) askSingleQuestion(num, total int, q chat.ClarifyQuestion) (string, error) {
	// Display question header
	fmt.Println(headerStyle.Render(fmt.Sprintf("Question %d/%d", num, total)))
	fmt.Println(questionStyle.Render(q.Question))
	fmt.Println()

	// Display options
	for i, opt := range q.Options {
		fmt.Printf("  %s\n", optionStyle.Render(fmt.Sprintf("[%d] %s", i+1, opt)))
	}

	if q.AllowCustom {
		fmt.Printf("  %s\n", optionStyle.Render("[0] Type custom answer"))
	}
	fmt.Println()

	// Get answer with custom prompt
	customPromptText := fmt.Sprintf("Answer (1-%d%s): ", len(q.Options), func() string {
		if q.AllowCustom {
			return " or 0 for custom"
		}
		return ""
	}())

	r.rl.SetPrompt(customPromptText)
	defer r.rl.SetPrompt("Type here: ")

	for {
		input, err := r.readInput()
		if err != nil {
			return "", err
		}

		// Try to parse as number
		choice, parseErr := strconv.Atoi(input)

		// If it's a valid option number
		if parseErr == nil {
			if choice > 0 && choice <= len(q.Options) {
				return q.Options[choice-1], nil
			}

			// Custom answer requested
			if choice == 0 && q.AllowCustom {
				return r.getCustomAnswer()
			}
		}

		// If allow_custom and not a number, treat as custom answer
		if q.AllowCustom && input != "" {
			return input, nil
		}

		// Invalid input
		fmt.Println(r.formatter.FormatError(
			fmt.Errorf("Invalid choice. Please select 1-%d%s", len(q.Options), func() string {
				if q.AllowCustom {
					return " or type custom answer"
				}
				return ""
			}()),
		))
	}
}

func (r *REPL) getCustomAnswer() (string, error) {
	fmt.Println(r.formatter.FormatInfo("Enter your custom answer:"))

	r.rl.SetPrompt("Custom answer: ")
	defer r.rl.SetPrompt("Type here: ")

	return r.readInput()
}
