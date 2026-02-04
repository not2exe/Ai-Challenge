package codeindex

import (
	"strings"
	"unicode"
)

// ChunkConfig defines chunking parameters.
type ChunkConfig struct {
	MaxChunkSize int // Maximum characters per chunk
	Overlap      int // Overlap between chunks in characters
}

// DefaultChunkConfig returns sensible defaults for code chunking.
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		MaxChunkSize: 1000, // ~200-250 tokens for most models
		Overlap:      200,  // 20% overlap to preserve context
	}
}

// CodeChunk represents a chunk of code with metadata.
type CodeChunk struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
	Start    int    `json:"start_line"`
	End      int    `json:"end_line"`
	Index    int    `json:"chunk_index"`
}

// ChunkCode splits code into overlapping chunks.
// It tries to split on natural boundaries (newlines, function boundaries).
func ChunkCode(filePath string, content string, cfg ChunkConfig) []CodeChunk {
	lines := strings.Split(content, "\n")
	chunks := []CodeChunk{}

	currentChunk := ""
	currentStart := 1
	chunkIndex := 0

	for i, line := range lines {
		lineNum := i + 1
		testChunk := currentChunk
		if testChunk != "" {
			testChunk += "\n"
		}
		testChunk += line

		// If adding this line exceeds max size, save current chunk
		if len(testChunk) > cfg.MaxChunkSize && currentChunk != "" {
			chunks = append(chunks, CodeChunk{
				FilePath: filePath,
				Content:  strings.TrimSpace(currentChunk),
				Start:    currentStart,
				End:      lineNum - 1,
				Index:    chunkIndex,
			})
			chunkIndex++

			// Start new chunk with overlap
			overlapLines := getOverlapLines(lines, i, cfg.Overlap)
			currentChunk = strings.Join(overlapLines, "\n")
			if len(overlapLines) > 0 {
				currentStart = lineNum - len(overlapLines) + 1
			} else {
				currentStart = lineNum
			}
		}

		// Add line to current chunk
		if currentChunk != "" {
			currentChunk += "\n"
		}
		currentChunk += line
	}

	// Add final chunk
	if strings.TrimSpace(currentChunk) != "" {
		chunks = append(chunks, CodeChunk{
			FilePath: filePath,
			Content:  strings.TrimSpace(currentChunk),
			Start:    currentStart,
			End:      len(lines),
			Index:    chunkIndex,
		})
	}

	return chunks
}

// getOverlapLines returns the last N characters worth of lines for overlap.
func getOverlapLines(lines []string, currentIndex int, overlapSize int) []string {
	if currentIndex <= 0 || overlapSize <= 0 {
		return nil
	}

	overlap := []string{}
	charCount := 0

	// Work backwards from current position
	for i := currentIndex - 1; i >= 0; i-- {
		line := lines[i]
		if charCount+len(line) > overlapSize {
			break
		}
		overlap = append([]string{line}, overlap...)
		charCount += len(line) + 1 // +1 for newline
	}

	return overlap
}

// ShouldIndexFile determines if a file should be indexed based on extension.
func ShouldIndexFile(filename string) bool {
	codeExtensions := map[string]bool{
		".go":   true,
		".js":   true,
		".ts":   true,
		".jsx":  true,
		".tsx":  true,
		".py":   true,
		".java": true,
		".c":    true,
		".cpp":  true,
		".h":    true,
		".hpp":  true,
		".rs":   true,
		".rb":   true,
		".php":  true,
		".cs":   true,
		".swift": true,
		".kt":   true,
		".scala": true,
		".sh":   true,
		".bash": true,
		".sql":  true,
		".proto": true,
		".thrift": true,
		".graphql": true,
		".yaml": true,
		".yml":  true,
		".json": true,
		".xml":  true,
		".md":   true,
	}

	for ext := range codeExtensions {
		if strings.HasSuffix(strings.ToLower(filename), ext) {
			return true
		}
	}

	return false
}

// CleanCode removes excessive whitespace while preserving code structure.
func CleanCode(code string) string {
	lines := strings.Split(code, "\n")
	cleaned := []string{}

	for _, line := range lines {
		// Trim trailing whitespace but preserve indentation
		trimmed := strings.TrimRightFunc(line, unicode.IsSpace)
		cleaned = append(cleaned, trimmed)
	}

	// Remove excessive blank lines (more than 2 consecutive)
	result := []string{}
	blankCount := 0
	for _, line := range cleaned {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 2 {
				result = append(result, line)
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
