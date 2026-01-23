package chat

import (
	"fmt"
	"strings"
)

func ValidateSystemPrompt(prompt string) error {
	if prompt == "" {
		return nil
	}

	if len(prompt) > 10000 {
		return fmt.Errorf("system prompt too long (max 10000 characters)")
	}

	return nil
}

func ValidateFormatPrompt(prompt string) error {
	if prompt == "" {
		return nil
	}

	if len(prompt) > 10000 {
		return fmt.Errorf("format prompt too long (max 10000 characters)")
	}

	return nil
}

func BuildSystemPrompt(base string, additions ...string) string {
	if base == "" && len(additions) == 0 {
		return ""
	}

	parts := []string{base}
	for _, addition := range additions {
		if addition != "" {
			parts = append(parts, addition)
		}
	}

	return strings.Join(parts, "\n\n")
}
