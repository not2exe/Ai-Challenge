package codeindex

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// IndexDirName is the hidden directory name for storing code index
	IndexDirName = ".codeindex"
	// IndexFileName is the index file name inside the index directory
	IndexFileName = "index.json"
)

// Indexer orchestrates the indexing process.
type Indexer struct {
	ollama      *OllamaClient
	chunkCfg    ChunkConfig
	modelName   string
	index       *CodeIndex
	projectRoot string // Root directory of the indexed project
}

// IndexerConfig defines indexer configuration.
type IndexerConfig struct {
	OllamaURL   string
	ModelName   string
	IndexPath   string // Deprecated: index is now stored in project's .codeindex/
	ChunkConfig ChunkConfig
}

// NewIndexer creates a new code indexer.
func NewIndexer(cfg IndexerConfig) (*Indexer, error) {
	ollama := NewOllamaClient(cfg.OllamaURL, cfg.ModelName)

	return &Indexer{
		ollama:    ollama,
		chunkCfg:  cfg.ChunkConfig,
		modelName: cfg.ModelName,
		index:     NewCodeIndex(cfg.ModelName),
	}, nil
}

// getIndexPath returns the path to the index file for a given project root.
func getIndexPath(projectRoot string) string {
	return filepath.Join(projectRoot, IndexDirName, IndexFileName)
}

// findProjectIndex searches for .codeindex directory starting from dir and going up.
func findProjectIndex(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		indexDir := filepath.Join(dir, IndexDirName)
		if info, err := os.Stat(indexDir); err == nil && info.IsDir() {
			return filepath.Join(indexDir, IndexFileName), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, no index found
			return "", fmt.Errorf("no .codeindex found (run index_directory first)")
		}
		dir = parent
	}
}

// IndexDirectory indexes all code files in a directory recursively.
func (idx *Indexer) IndexDirectory(ctx context.Context, dirPath string, progress func(string)) error {
	// Get absolute path for the project root
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("get absolute path: %w", err)
	}
	idx.projectRoot = absPath

	// Clear existing index
	idx.index = NewCodeIndex(idx.modelName)

	var filesToIndex []string

	// Walk directory and collect files
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-code files
		if info.IsDir() {
			// Skip common non-source directories
			name := info.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" ||
				name == ".idea" || name == "build" || name == "dist" || name == "target" ||
				name == IndexDirName {
				return filepath.SkipDir
			}
			return nil
		}

		if !ShouldIndexFile(path) {
			return nil
		}

		filesToIndex = append(filesToIndex, path)
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk directory: %w", err)
	}

	// Index each file
	for _, filePath := range filesToIndex {
		if progress != nil {
			relPath, _ := filepath.Rel(absPath, filePath)
			progress(fmt.Sprintf("Indexing: %s", relPath))
		}

		if err := idx.IndexFile(ctx, filePath); err != nil {
			return fmt.Errorf("index file %s: %w", filePath, err)
		}
	}

	// Create .codeindex directory in project root
	indexDir := filepath.Join(absPath, IndexDirName)
	if err := os.MkdirAll(indexDir, 0o755); err != nil {
		return fmt.Errorf("create index directory: %w", err)
	}

	// Save index
	indexPath := getIndexPath(absPath)
	if err := idx.index.Save(indexPath); err != nil {
		return fmt.Errorf("save index: %w", err)
	}

	return nil
}

// IndexFile indexes a single file.
func (idx *Indexer) IndexFile(ctx context.Context, filePath string) error {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Clean and chunk the code
	cleanedCode := CleanCode(string(content))
	chunks := ChunkCode(filePath, cleanedCode, idx.chunkCfg)

	// Generate embeddings for each chunk
	for _, chunk := range chunks {
		embedding, err := idx.ollama.GenerateEmbedding(ctx, chunk.Content)
		if err != nil {
			return fmt.Errorf("generate embedding for chunk %d: %w", chunk.Index, err)
		}

		idx.index.AddChunk(chunk, embedding)
	}

	return nil
}

// Search searches the index for code similar to the query.
func (idx *Indexer) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	// Try to load index from current directory if not already loaded
	if idx.index.IsEmpty() {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}

		indexPath, err := findProjectIndex(cwd)
		if err != nil {
			return nil, err
		}

		loadedIndex, err := LoadIndex(indexPath)
		if err != nil {
			return nil, fmt.Errorf("load index: %w", err)
		}
		idx.index = loadedIndex
	}

	// Generate embedding for query
	queryEmbedding, err := idx.ollama.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}

	// Search index
	results := idx.index.Search(ctx, queryEmbedding, topK)
	return results, nil
}

// Stats returns index statistics.
func (idx *Indexer) Stats() map[string]interface{} {
	// Try to load index if empty
	if idx.index.IsEmpty() {
		cwd, err := os.Getwd()
		if err == nil {
			if indexPath, err := findProjectIndex(cwd); err == nil {
				if loadedIndex, err := LoadIndex(indexPath); err == nil {
					idx.index = loadedIndex
				}
			}
		}
	}
	return idx.index.Stats()
}

// CheckHealth verifies that Ollama is available.
func (idx *Indexer) CheckHealth(ctx context.Context) error {
	return idx.ollama.CheckHealth(ctx)
}

// SaveIndex saves the current index to disk.
func (idx *Indexer) SaveIndex() error {
	if idx.projectRoot == "" {
		return fmt.Errorf("no project indexed yet")
	}
	return idx.index.Save(getIndexPath(idx.projectRoot))
}

// LoadIndex reloads the index from disk.
func (idx *Indexer) LoadIndex() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	indexPath, err := findProjectIndex(cwd)
	if err != nil {
		return err
	}

	index, err := LoadIndex(indexPath)
	if err != nil {
		return err
	}
	idx.index = index
	return nil
}

// FormatSearchResults formats search results as a readable string.
func FormatSearchResults(results []SearchResult) string {
	if len(results) == 0 {
		return "No results found."
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Found %d result(s):\n\n", len(results)))

	for i, result := range results {
		builder.WriteString(fmt.Sprintf("Result %d (similarity: %.3f):\n", i+1, result.Similarity))
		builder.WriteString(fmt.Sprintf("File: %s (lines %d-%d)\n",
			result.Chunk.FilePath, result.Chunk.Start, result.Chunk.End))
		builder.WriteString("```\n")
		builder.WriteString(result.Chunk.Content)
		builder.WriteString("\n```\n\n")
	}

	return builder.String()
}
