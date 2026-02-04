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

// CodeIndexToolsPrompt provides guidance for AI to use semantic code search effectively.
// This should be appended to the system prompt when MCP code index tools are available.
const CodeIndexToolsPrompt = `You have semantic code search tools available:
- search_code: Search code by meaning with relevance filtering.
  Parameters:
  - query (required): Natural language description of what to find
  - top_k (optional): Number of results (default: 5)
  - min_similarity (optional): Threshold 0.0-1.0 (default: 0.3). Lower = more results, higher = stricter
  - use_rerank (optional): Enable LLM reranking for better accuracy (slower, needs qwen2.5:1.5b)
- index_directory: Index a directory. Creates .codeindex/ in project root.
- index_stats: Check index status and location.

Index storage: PROJECT_ROOT/.codeindex/index.json (auto-discovered when searching)

IMPORTANT - When to use search_code:
- When user asks "how does X work" or "where is X implemented" - USE search_code FIRST
- When user asks about architecture, patterns, or code structure - USE search_code
- For questions about the codebase - ALWAYS search before answering

Search tips:
- Use specific queries: "JWT token validation" > "authentication"
- If results seem irrelevant, try min_similarity=0.4 or higher
- For complex queries, use use_rerank=true for better relevance
- If too few results, lower min_similarity to 0.2

DO NOT answer questions about the codebase from memory - always search first.`

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
