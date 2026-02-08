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
- semantic_search: Search code by meaning with relevance filtering.
  Parameters:
  - query (required): Natural language description of what to find
  - top_k (optional): Number of results (default: 3)
  - min_similarity (optional): Threshold 0.0-1.0 (default: 0.3). Lower = more results, higher = stricter
  - use_rerank (optional): Enable LLM reranking for better accuracy (slower, needs qwen2.5:1.5b)
  - compact (optional): Return only file paths without code (saves tokens)
  - max_content_length (optional): Truncate snippets (default: 500)
- index_directory: Index a directory. Creates .codeindex/ in project root.
- index_stats: Check index status and location.
- check_health: Check if Ollama and the embedding model are available.

Index storage: PROJECT_ROOT/.codeindex/index.json (auto-discovered when searching)

AUTOMATIC INDEX MANAGEMENT - CRITICAL:
When the user asks ANY question about code, architecture, implementation, or the project:
1. First call index_stats to check if an index exists.
2. If the index EXISTS (has chunks > 0) — use semantic_search to answer the question.
3. If the index DOES NOT EXIST or is empty — automatically call index_directory on the project root to create it, then use semantic_search.
4. Do NOT ask the user whether to create the index. Just do it silently and inform them: "Индекс не найден, создаю..." or similar brief message.
5. If index_directory fails, fall back to filesystem tools (if available) or inform the user.

You must NEVER answer questions about the codebase from memory. Always use the index.
The user should NOT need to explicitly ask you to "use the index" or "create an index" — you handle this automatically.

IMPORTANT - When to use semantic_search:
- When user asks "how does X work" or "where is X implemented" - USE semantic_search FIRST
- When user asks about architecture, patterns, or code structure - USE semantic_search
- For questions about the codebase - ALWAYS search before answering
- For ANY code-related question — check index and search automatically

Search tips:
- Use specific queries: "JWT token validation" > "authentication"
- If results seem irrelevant, try min_similarity=0.4 or higher
- For complex queries, use use_rerank=true for better relevance
- If too few results, lower min_similarity to 0.2

CITATION REQUIREMENTS - MANDATORY:
Search results include citation IDs [1], [2], etc. and a SOURCES block with file paths.
When answering based on search results, you MUST:
1. Reference sources using [N] format inline (e.g., "The authentication is handled in AuthService [1]")
2. Include a "Sources:" section at the END of your response listing all referenced files
3. Format: "Sources:\n[1] path/to/file.go:10-25\n[2] another/file.kt:100-150"

Example response:
"The user authentication uses JWT tokens [1]. The token is validated in the middleware [2].

Sources:
[1] internal/auth/jwt.go:45-67
[2] internal/middleware/auth.go:12-30"

CRITICAL - DO NOT READ FILES AFTER SEARCH:
- semantic_search returns code snippets from the index - this is SUFFICIENT
- DO NOT use read_text_file, read_file, or any file reading tool after semantic_search
- The index already contains the relevant code - use it directly
- If you need more context, use semantic_search with a different query instead
- Reading files wastes tokens and time - the index is your source of truth

DO NOT answer questions about the codebase from memory - always search first.
ALWAYS cite your sources when using information from search results.
NEVER read files after searching - use only the indexed results.`

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
