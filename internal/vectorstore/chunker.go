package vectorstore

import (
	"strings"
	"unicode"
)

// ChunkConfig holds configuration for text chunking
type ChunkConfig struct {
	// MaxChunkSize is the maximum number of characters per chunk
	// Default: 6000 (safe for 8192 token limit at ~4 chars/token)
	MaxChunkSize int

	// OverlapSize is the number of characters to overlap between chunks
	// Default: 600 (provides context continuity)
	OverlapSize int
}

// Chunk represents a chunk of text from a document
type Chunk struct {
	Text        string // The chunk text
	Index       int    // Chunk index (0-based)
	TotalChunks int    // Total number of chunks for this document
	ParentID    string // ID of the parent document
}

// ChunkText splits text into chunks based on the configuration
func ChunkText(text string, parentID string, config ChunkConfig) []Chunk {
	// Set defaults
	if config.MaxChunkSize == 0 {
		config.MaxChunkSize = 6000
	}
	if config.OverlapSize == 0 {
		config.OverlapSize = 600
	}

	// If text fits in one chunk, return it as-is
	if len(text) <= config.MaxChunkSize {
		return []Chunk{{
			Text:        text,
			Index:       0,
			TotalChunks: 1,
			ParentID:    parentID,
		}}
	}

	var chunks []Chunk
	start := 0

	for start < len(text) {
		// Calculate end position
		end := start + config.MaxChunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at a natural boundary (paragraph, sentence, or word)
		if end < len(text) {
			end = findBreakPoint(text, start, end)
		}

		// Extract chunk
		chunkText := text[start:end]
		chunks = append(chunks, Chunk{
			Text:        chunkText,
			Index:       len(chunks),
			TotalChunks: 0, // Will be set after all chunks are created
			ParentID:    parentID,
		})

		// Move start position with overlap
		start = end - config.OverlapSize
		if start < 0 {
			start = 0
		}

		// Prevent infinite loop if overlap is too large
		if start <= chunks[len(chunks)-1].Index && len(chunks) > 1 {
			start = end
		}
	}

	// Set total chunks for all chunks
	totalChunks := len(chunks)
	for i := range chunks {
		chunks[i].TotalChunks = totalChunks
	}

	return chunks
}

// findBreakPoint finds a natural break point near the target position
func findBreakPoint(text string, start, target int) int {
	// Look back from target for a natural break
	searchStart := target - 200 // Look back up to 200 chars
	if searchStart < start {
		searchStart = start
	}

	// Try to find paragraph break (double newline)
	if idx := strings.LastIndex(text[searchStart:target], "\n\n"); idx != -1 {
		return searchStart + idx + 2
	}

	// Try to find sentence break (period, question mark, exclamation)
	for i := target - 1; i >= searchStart; i-- {
		if text[i] == '.' || text[i] == '?' || text[i] == '!' {
			// Make sure it's followed by whitespace or end of text
			if i+1 >= len(text) || unicode.IsSpace(rune(text[i+1])) {
				return i + 1
			}
		}
	}

	// Try to find newline
	if idx := strings.LastIndex(text[searchStart:target], "\n"); idx != -1 {
		return searchStart + idx + 1
	}

	// Try to find word boundary (space)
	for i := target - 1; i >= searchStart; i-- {
		if unicode.IsSpace(rune(text[i])) {
			return i + 1
		}
	}

	// No natural break found, use target
	return target
}
