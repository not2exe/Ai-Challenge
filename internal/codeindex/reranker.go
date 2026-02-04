package codeindex

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// RerankerConfig configures the reranking behavior.
type RerankerConfig struct {
	// MinSimilarity is the minimum similarity threshold (0.0-1.0).
	// Results below this threshold are filtered out.
	// Default: 0.3
	MinSimilarity float64

	// UseLLMRerank enables LLM-based reranking for better relevance.
	// This is slower but more accurate.
	UseLLMRerank bool

	// MaxResultsForLLM limits how many results to send to LLM for reranking.
	// Default: 10
	MaxResultsForLLM int
}

// DefaultRerankerConfig returns the default reranker configuration.
func DefaultRerankerConfig() RerankerConfig {
	return RerankerConfig{
		MinSimilarity:    0.3,
		UseLLMRerank:     false,
		MaxResultsForLLM: 10,
	}
}

// Reranker filters and reranks search results.
type Reranker struct {
	config RerankerConfig
	ollama *OllamaClient
}

// NewReranker creates a new reranker.
func NewReranker(config RerankerConfig, ollama *OllamaClient) *Reranker {
	return &Reranker{
		config: config,
		ollama: ollama,
	}
}

// RerankedResult extends SearchResult with reranking metadata.
type RerankedResult struct {
	SearchResult
	LLMScore    float64 `json:"llm_score,omitempty"`    // Score from LLM reranking (0-1)
	FinalScore  float64 `json:"final_score"`            // Combined final score
	FilteredOut bool    `json:"filtered_out,omitempty"` // True if below threshold
}

// Rerank filters and optionally reranks search results.
func (r *Reranker) Rerank(ctx context.Context, query string, results []SearchResult) ([]RerankedResult, *RerankerStats) {
	stats := &RerankerStats{
		OriginalCount:   len(results),
		MinSimilarity:   r.config.MinSimilarity,
		UsedLLMRerank:   false,
	}

	if len(results) == 0 {
		return nil, stats
	}

	// Step 1: Filter by similarity threshold
	filtered := make([]RerankedResult, 0, len(results))
	for _, res := range results {
		rr := RerankedResult{
			SearchResult: res,
			FinalScore:   res.Similarity,
			FilteredOut:  res.Similarity < r.config.MinSimilarity,
		}
		if !rr.FilteredOut {
			filtered = append(filtered, rr)
		}
	}
	stats.AfterThresholdCount = len(filtered)

	if len(filtered) == 0 {
		return filtered, stats
	}

	// Step 2: LLM reranking (if enabled)
	if r.config.UseLLMRerank && r.ollama != nil {
		// Limit results for LLM to avoid token overflow
		toRerank := filtered
		if len(toRerank) > r.config.MaxResultsForLLM {
			toRerank = toRerank[:r.config.MaxResultsForLLM]
		}

		reranked, err := r.llmRerank(ctx, query, toRerank)
		if err == nil {
			filtered = reranked
			stats.UsedLLMRerank = true
		}
		// If LLM reranking fails, we just use the original filtered results
	}

	// Sort by final score (descending)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].FinalScore > filtered[j].FinalScore
	})

	stats.FinalCount = len(filtered)
	return filtered, stats
}

// RerankerStats provides statistics about the reranking process.
type RerankerStats struct {
	OriginalCount       int     `json:"original_count"`
	AfterThresholdCount int     `json:"after_threshold_count"`
	FinalCount          int     `json:"final_count"`
	MinSimilarity       float64 `json:"min_similarity"`
	UsedLLMRerank       bool    `json:"used_llm_rerank"`
}

// llmRerank uses Ollama to rerank results based on relevance to the query.
func (r *Reranker) llmRerank(ctx context.Context, query string, results []RerankedResult) ([]RerankedResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	// Build prompt for LLM
	prompt := buildRerankPrompt(query, results)

	// Call Ollama for reranking
	response, err := r.ollama.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("llm rerank failed: %w", err)
	}

	// Parse LLM response
	scores, err := parseRerankResponse(response, len(results))
	if err != nil {
		return nil, fmt.Errorf("parse rerank response: %w", err)
	}

	// Apply LLM scores
	for i := range results {
		if i < len(scores) {
			results[i].LLMScore = scores[i]
			// Combine embedding similarity with LLM score
			// Weight: 40% embedding, 60% LLM (LLM understands context better)
			results[i].FinalScore = 0.4*results[i].Similarity + 0.6*results[i].LLMScore
		}
	}

	return results, nil
}

// buildRerankPrompt creates a prompt for LLM reranking.
func buildRerankPrompt(query string, results []RerankedResult) string {
	var sb strings.Builder

	sb.WriteString("You are a code relevance scorer. Given a search query and code snippets, ")
	sb.WriteString("rate each snippet's relevance from 0.0 to 1.0.\n\n")
	sb.WriteString("QUERY: ")
	sb.WriteString(query)
	sb.WriteString("\n\nCODE SNIPPETS:\n")

	for i, res := range results {
		sb.WriteString(fmt.Sprintf("\n--- SNIPPET %d ---\n", i+1))
		sb.WriteString(fmt.Sprintf("File: %s\n", res.Chunk.FilePath))
		// Truncate long snippets
		content := res.Chunk.Content
		if len(content) > 500 {
			content = content[:500] + "..."
		}
		sb.WriteString(content)
		sb.WriteString("\n")
	}

	sb.WriteString("\nRespond with ONLY a JSON array of scores, one for each snippet in order.\n")
	sb.WriteString("Example: [0.9, 0.7, 0.3, 0.8]\n")
	sb.WriteString("Scores should reflect how well each snippet answers or relates to the query.\n")
	sb.WriteString("JSON array:")

	return sb.String()
}

// parseRerankResponse parses the LLM response to extract scores.
func parseRerankResponse(response string, expectedCount int) ([]float64, error) {
	// Try to find JSON array in response
	response = strings.TrimSpace(response)

	// Find array bounds
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	jsonStr := response[start : end+1]

	var scores []float64
	if err := json.Unmarshal([]byte(jsonStr), &scores); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate and normalize scores
	for i := range scores {
		if scores[i] < 0 {
			scores[i] = 0
		}
		if scores[i] > 1 {
			scores[i] = 1
		}
	}

	// Pad with zeros if not enough scores
	for len(scores) < expectedCount {
		scores = append(scores, 0.5) // Default to middle score
	}

	return scores, nil
}

// FilterByThreshold is a simple filter that removes results below the threshold.
func FilterByThreshold(results []SearchResult, minSimilarity float64) []SearchResult {
	filtered := make([]SearchResult, 0, len(results))
	for _, res := range results {
		if res.Similarity >= minSimilarity {
			filtered = append(filtered, res)
		}
	}
	return filtered
}

// FormatRerankedResults formats reranked results with stats.
func FormatRerankedResults(results []RerankedResult, stats *RerankerStats) string {
	if len(results) == 0 {
		msg := fmt.Sprintf("No relevant results found (threshold: %.2f).\n", stats.MinSimilarity)
		if stats.OriginalCount > 0 {
			msg += fmt.Sprintf("Found %d results but all were below relevance threshold.\n", stats.OriginalCount)
			msg += "Try a more specific query or lower the threshold."
		}
		return msg
	}

	var builder strings.Builder

	// Stats header
	builder.WriteString(fmt.Sprintf("Found %d relevant result(s)", len(results)))
	if stats.OriginalCount > len(results) {
		builder.WriteString(fmt.Sprintf(" (filtered %d below %.2f threshold)",
			stats.OriginalCount-stats.AfterThresholdCount, stats.MinSimilarity))
	}
	if stats.UsedLLMRerank {
		builder.WriteString(" [LLM reranked]")
	}
	builder.WriteString(":\n\n")

	for i, result := range results {
		builder.WriteString(fmt.Sprintf("Result %d", i+1))
		if stats.UsedLLMRerank {
			builder.WriteString(fmt.Sprintf(" (similarity: %.3f, llm: %.3f, final: %.3f)",
				result.Similarity, result.LLMScore, result.FinalScore))
		} else {
			builder.WriteString(fmt.Sprintf(" (similarity: %.3f)", result.Similarity))
		}
		builder.WriteString(":\n")
		builder.WriteString(fmt.Sprintf("File: %s (lines %d-%d)\n",
			result.Chunk.FilePath, result.Chunk.Start, result.Chunk.End))
		builder.WriteString("```\n")
		builder.WriteString(result.Chunk.Content)
		builder.WriteString("\n```\n\n")
	}

	return builder.String()
}
