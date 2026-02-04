package codeindex

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
)

// IndexedChunk represents a code chunk with its embedding.
type IndexedChunk struct {
	Chunk     CodeChunk `json:"chunk"`
	Embedding []float64 `json:"embedding"`
}

// CodeIndex manages the searchable code index.
type CodeIndex struct {
	Chunks    []IndexedChunk `json:"chunks"`
	ModelName string         `json:"model_name"`
	indexPath string
}

// NewCodeIndex creates a new empty code index.
func NewCodeIndex(modelName string) *CodeIndex {
	return &CodeIndex{
		Chunks:    []IndexedChunk{},
		ModelName: modelName,
	}
}

// LoadIndex loads an existing index from disk.
func LoadIndex(path string) (*CodeIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read index file: %w", err)
	}

	var idx CodeIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("unmarshal index: %w", err)
	}

	idx.indexPath = path
	return &idx, nil
}

// Save saves the index to disk.
func (idx *CodeIndex) Save(path string) error {
	idx.indexPath = path

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create index directory: %w", err)
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal index: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write index file: %w", err)
	}

	return nil
}

// AddChunk adds a chunk with its embedding to the index.
func (idx *CodeIndex) AddChunk(chunk CodeChunk, embedding []float64) {
	idx.Chunks = append(idx.Chunks, IndexedChunk{
		Chunk:     chunk,
		Embedding: embedding,
	})
}

// SearchResult represents a search result with similarity score.
type SearchResult struct {
	Chunk      CodeChunk `json:"chunk"`
	Similarity float64   `json:"similarity"`
}

// Search searches the index for chunks similar to the query.
func (idx *CodeIndex) Search(ctx context.Context, queryEmbedding []float64, topK int) []SearchResult {
	if len(idx.Chunks) == 0 {
		return nil
	}

	// Calculate cosine similarity for each chunk
	similarities := make([]SearchResult, len(idx.Chunks))
	for i, indexed := range idx.Chunks {
		sim := cosineSimilarity(queryEmbedding, indexed.Embedding)
		similarities[i] = SearchResult{
			Chunk:      indexed.Chunk,
			Similarity: sim,
		}
	}

	// Sort by similarity (descending)
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].Similarity > similarities[j].Similarity
	})

	// Return top K results
	if topK > len(similarities) {
		topK = len(similarities)
	}

	return similarities[:topK]
}

// cosineSimilarity calculates the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// Stats returns statistics about the index.
func (idx *CodeIndex) Stats() map[string]interface{} {
	fileMap := make(map[string]int)
	for _, chunk := range idx.Chunks {
		fileMap[chunk.Chunk.FilePath]++
	}

	return map[string]interface{}{
		"total_chunks": len(idx.Chunks),
		"total_files":  len(fileMap),
		"model":        idx.ModelName,
		"index_path":   idx.indexPath,
	}
}

// Clear removes all chunks from the index.
func (idx *CodeIndex) Clear() {
	idx.Chunks = []IndexedChunk{}
}

// IsEmpty returns true if the index has no chunks.
func (idx *CodeIndex) IsEmpty() bool {
	return len(idx.Chunks) == 0
}
