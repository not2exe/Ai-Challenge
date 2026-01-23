package chat

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ClarifyQuestion struct {
	Question    string   `json:"question"`
	Options     []string `json:"options"`
	Importance  int      `json:"importance"`
	AllowCustom bool     `json:"allow_custom"`
}

type ClarifyResponse struct {
	Questions []ClarifyQuestion `json:"questions"`
	Message   string            `json:"message"`
}

type QuestionAnswer struct {
	Question string
	Answer   string
}

const clarifySystemPrompt = "IMPORTANT: Before answering the user's question, first ask clarifying questions to better understand their requirements.\n\n" +
	"Return your clarifying questions in raw JSON format (no markdown code blocks) with this structure:\n" +
	"{\n" +
	"  \"message\": \"Brief intro message explaining why you're asking questions\",\n" +
	"  \"questions\": [\n" +
	"    {\n" +
	"      \"question\": \"What is your question?\",\n" +
	"      \"options\": [\"Option 1\", \"Option 2\", \"Option 3\"],\n" +
	"      \"importance\": 10,\n" +
	"      \"allow_custom\": true\n" +
	"    }\n" +
	"  ]\n" +
	"}\n\n" +
	"Guidelines:\n" +
	"- Ask up to 20 questions maximum\n" +
	"- Sort questions by importance (1-10, where 10 is most important)\n" +
	"- Provide 2-4 clear options for each question\n" +
	"- Set allow_custom to true if user might have a different answer\n" +
	"- Focus on questions that will significantly improve your answer quality\n" +
	"- Avoid asking questions if the user query is already clear and simple\n" +
	"- Only ask questions that matter for giving a better response\n\n" +
	"Return raw JSON only, no markdown formatting."

func GetClarifyPrompt() string {
	return clarifySystemPrompt
}

func ParseClarifyResponse(content string) (*ClarifyResponse, error) {
	cleaned := CleanMarkdownCodeBlocks(content)

	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")

	if start == -1 || end == -1 || start >= end {
		return nil, fmt.Errorf("no valid JSON found in clarify response")
	}

	jsonContent := cleaned[start : end+1]

	var parsed ClarifyResponse
	if err := json.Unmarshal([]byte(jsonContent), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse clarify JSON: %w", err)
	}

	sortQuestionsByImportance(parsed.Questions)

	if len(parsed.Questions) > 20 {
		parsed.Questions = parsed.Questions[:20]
	}

	return &parsed, nil
}

func sortQuestionsByImportance(questions []ClarifyQuestion) {
	for i := 0; i < len(questions)-1; i++ {
		for j := i + 1; j < len(questions); j++ {
			if questions[j].Importance > questions[i].Importance {
				questions[i], questions[j] = questions[j], questions[i]
			}
		}
	}
}

func FormatQuestionAnswers(answers []QuestionAnswer) string {
	var result strings.Builder
	result.WriteString("Clarifying information provided:\n\n")

	for i, qa := range answers {
		result.WriteString(fmt.Sprintf("%d. Q: %s\n", i+1, qa.Question))
		result.WriteString(fmt.Sprintf("   A: %s\n\n", qa.Answer))
	}

	return result.String()
}
