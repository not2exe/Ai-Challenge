package chat

import (
	"fmt"
	"strings"
)

// FileToolsPrompt provides guidance for AI to use filesystem tools effectively.
// This should be appended to the system prompt when MCP filesystem tools are available.
const FileToolsPrompt = `You have filesystem tools available for reading and exploring code:
- read_text_file: Read specific files. Use head/tail params for large files.
- list_directory: List contents of ONE directory (not recursive). Preferred for exploration.
- search_files: Find files by glob pattern (e.g., "*.go", "src/**/*.kt")
- directory_tree: Get recursive tree. CAUTION: Only use on small, specific directories!

IMPORTANT - Avoid context overflow:
- NEVER use directory_tree on project root or large directories (.git, build, node_modules)
- Start with list_directory on the specific path the user mentions
- Use search_files to find specific file types instead of browsing everything
- When user gives a path like "iosApp/iosApp", go directly there - don't scan the whole project

Example - if user says "build the iOS app at iosApp/iosApp":
1. list_directory on "iosApp/iosApp" to see the structure
2. Read specific config files (*.xcodeproj, *.swift files you need)
3. Take action based on what you find

Be targeted and efficient with file operations.`

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
