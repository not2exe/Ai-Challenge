package chat

import (
	"encoding/json"
	"fmt"
	"strings"
)

// AskUserQuestion represents a question to ask the user with options
type AskUserQuestion struct {
	Question    string   `json:"question"`
	Header      string   `json:"header,omitempty"`      // Short label for the question
	Options     []Option `json:"options"`
	MultiSelect bool     `json:"multiSelect,omitempty"` // Allow multiple selections
}

// Option represents a single option in a question
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// AskUserRequest represents a request from AI to ask the user questions
type AskUserRequest struct {
	Questions []AskUserQuestion `json:"questions"`
}

// AskUserResponse contains the user's answers
type AskUserResponse struct {
	Answers map[string]string `json:"answers"` // question -> selected option(s)
}

// AskUserToolPrompt is the system prompt that encourages AI to use the ask_user tool
const AskUserToolPrompt = `
IMPORTANT: You have an "ask_user" tool available. USE IT when:
- The user asks you to ask them a question with choices/options
- You need to clarify the user's preferences before answering
- You want to offer multiple approaches or solutions
- The user explicitly requests an interactive menu or selection

When the user says things like "ask me...", "give me options", "let me choose", or similar -
you MUST use the ask_user tool instead of writing plain text questions.

The ask_user tool will display an interactive menu where the user can select from options.
`

// ParseAskUserRequest detects and parses an ask_user request from AI response
func ParseAskUserRequest(content string) (*AskUserRequest, string, error) {
	// Look for <ask_user>...</ask_user> tags
	startTag := "<ask_user>"
	endTag := "</ask_user>"

	startIdx := strings.Index(content, startTag)
	if startIdx == -1 {
		return nil, content, nil // No ask_user request found
	}

	endIdx := strings.Index(content, endTag)
	if endIdx == -1 {
		return nil, content, nil // Malformed, ignore
	}

	// Extract JSON between tags
	jsonStart := startIdx + len(startTag)
	jsonContent := strings.TrimSpace(content[jsonStart:endIdx])

	// Parse JSON
	var req AskUserRequest
	if err := json.Unmarshal([]byte(jsonContent), &req); err != nil {
		return nil, content, fmt.Errorf("failed to parse ask_user JSON: %w", err)
	}

	// Validate
	if len(req.Questions) == 0 {
		return nil, content, nil
	}

	for i := range req.Questions {
		if len(req.Questions[i].Options) < 2 {
			return nil, content, fmt.Errorf("question %d needs at least 2 options", i+1)
		}
	}

	// Extract text before the ask_user block (if any)
	textBefore := strings.TrimSpace(content[:startIdx])

	return &req, textBefore, nil
}

// FormatAskUserAnswers formats the user's answers for the AI
func FormatAskUserAnswers(questions []AskUserQuestion, answers [][]string) string {
	var sb strings.Builder
	sb.WriteString("User's selections:\n")

	for i, q := range questions {
		if i < len(answers) {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", q.Header, strings.Join(answers[i], ", ")))
		}
	}

	return sb.String()
}

// HasAskUserRequest checks if content contains an ask_user request
func HasAskUserRequest(content string) bool {
	return strings.Contains(content, "<ask_user>")
}
