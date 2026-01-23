package chat

import (
	"fmt"
	"strings"

	"github.com/notexe/cli-chat/internal/api"
)

const summarizationPrompt = `Create a concise summary of the following conversation, preserving:
1. Key decisions and facts
2. Important details and numbers
3. Context needed to continue the conversation
4. Unfinished tasks or pending items

Format: Write as a coherent paragraph, not a list. Aim for ~25% of the original length.
Focus on information that would be needed to continue this conversation naturally.

Conversation to summarize:`

// BuildSummarizationRequest creates an API request for summarizing messages.
func BuildSummarizationRequest(messages []api.Message, modelName string, maxTokens int, temperature float64) api.MessageRequest {
	// Build conversation text from messages
	var conversationBuilder strings.Builder
	for _, msg := range messages {
		var role string
		switch msg.Role {
		case "user":
			role = "User"
		case "assistant":
			role = "Assistant"
		default:
			role = msg.Role
		}
		conversationBuilder.WriteString(fmt.Sprintf("%s: %s\n\n", role, msg.Content))
	}

	// Create the summarization request
	userMessage := fmt.Sprintf("%s\n\n%s", summarizationPrompt, conversationBuilder.String())

	return api.MessageRequest{
		Messages: []api.Message{
			{Role: "user", Content: userMessage},
		},
		System:      "You are a helpful assistant that creates concise conversation summaries. Keep essential context while reducing length.",
		Model:       modelName,
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}
}

// FormatSummaryMessage wraps the summary as a system-style message for history.
func FormatSummaryMessage(summary string) api.Message {
	return api.Message{
		Role:    "assistant",
		Content: fmt.Sprintf("[Previous conversation summary]\n%s\n[End of summary]", summary),
	}
}

// CalculateMessagesToSummarize determines which messages should be summarized.
// It returns the messages to summarize and the messages to keep.
// preferKeepPairs specifies the preferred number of recent message pairs to preserve.
// If there are fewer messages than preferKeepPairs*2, summarizes 60% of oldest messages.
func CalculateMessagesToSummarize(messages []api.Message, preferKeepPairs int) (toSummarize []api.Message, toKeep []api.Message) {
	totalMessages := len(messages)

	// Need at least 2 messages to summarize (keep at least 1)
	if totalMessages < 2 {
		return nil, messages
	}

	// Calculate how many messages to keep
	keepMessages := preferKeepPairs * 2

	// If fewer messages than threshold, summarize 60% of oldest
	if totalMessages < keepMessages {
		summarizeCount := int(float64(totalMessages) * 0.6)
		if summarizeCount < 1 {
			summarizeCount = 1
		}
		keepCount := totalMessages - summarizeCount
		if keepCount < 1 {
			keepCount = 1
			summarizeCount = totalMessages - 1
		}
		return messages[:summarizeCount], messages[summarizeCount:]
	}

	cutPoint := totalMessages - keepMessages
	return messages[:cutPoint], messages[cutPoint:]
}

// EstimateTokenSavings estimates how many tokens will be saved by summarization.
// This is a rough estimate based on the 25% target compression ratio.
func EstimateTokenSavings(originalTokens int) int {
	// If we compress to 25%, we save 75%
	return int(float64(originalTokens) * 0.75)
}
