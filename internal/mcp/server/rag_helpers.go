package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"obsidian-agent/internal/vault"
	"obsidian-agent/internal/vectorstore"
)

// SearchResultJSON represents a formatted search result
type SearchResultJSON struct {
	Path       string  `json:"path"`
	Title      string  `json:"title"`
	Similarity float32 `json:"similarity"`
	Snippet    string  `json:"snippet,omitempty"`
	Content    string  `json:"content,omitempty"`
}

// SearchResponseJSON represents the complete search response
type SearchResponseJSON struct {
	Results []SearchResultJSON `json:"results"`
	Total   int                `json:"total"`
	Query   string             `json:"query"`
}

// NoteContext holds retrieved note content with metadata
type NoteContext struct {
	Path       string
	Title      string
	Content    string
	Similarity float32
}

// FormatSearchResults formats semantic search results as structured JSON
func FormatSearchResults(results []vectorstore.SearchResult, query string, withContent bool, reader *vault.Reader) (string, error) {
	response := SearchResponseJSON{
		Results: make([]SearchResultJSON, 0, len(results)),
		Total:   len(results),
		Query:   query,
	}

	for _, result := range results {
		resultJSON := SearchResultJSON{
			Path:       result.Document.ID,
			Title:      result.Document.Title,
			Similarity: result.Similarity,
		}

		// Add snippet (first 200 chars of content)
		if len(result.Document.Content) > 0 {
			resultJSON.Snippet = ExtractSnippet(result.Document.Content, 200)
		}

		// Optionally include full content
		if withContent && reader != nil {
			content, err := reader.ReadNote(result.Document.ID)
			if err == nil {
				resultJSON.Content = content
			}
		}

		response.Results = append(response.Results, resultJSON)
	}

	jsonBytes, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal results: %w", err)
	}

	return string(jsonBytes), nil
}

// RetrieveContext retrieves full content for top search results
func RetrieveContext(results []vectorstore.SearchResult, reader *vault.Reader, limit int) []NoteContext {
	contexts := make([]NoteContext, 0, limit)

	for i, result := range results {
		if i >= limit {
			break
		}

		content, err := reader.ReadNote(result.Document.ID)
		if err != nil {
			// Skip notes that can't be read
			continue
		}

		contexts = append(contexts, NoteContext{
			Path:       result.Document.ID,
			Title:      result.Document.Title,
			Content:    content,
			Similarity: result.Similarity,
		})
	}

	return contexts
}

// FormatAnswer formats a RAG answer with citations
func FormatAnswer(question string, contexts []NoteContext, includeSnippets bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Answer to: %s\n\n", question))

	if len(contexts) == 0 {
		sb.WriteString("No relevant information found in the vault.\n")
		return sb.String()
	}

	// Calculate average similarity
	var avgSim float32
	for _, ctx := range contexts {
		avgSim += ctx.Similarity
	}
	avgSim /= float32(len(contexts))

	sb.WriteString("## Relevant Notes Found\n\n")

	for i, ctx := range contexts {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, ctx.Title))
		sb.WriteString(fmt.Sprintf("**Source:** `%s` (similarity: %.3f)\n\n", ctx.Path, ctx.Similarity))

		if includeSnippets {
			snippet := ExtractSnippet(ctx.Content, 300)
			sb.WriteString(fmt.Sprintf("%s...\n\n", snippet))
		}

		sb.WriteString("---\n\n")
	}

	sb.WriteString(fmt.Sprintf("*Retrieved from %d sources with average similarity: %.3f*\n", len(contexts), avgSim))

	return sb.String()
}

// ExtractSnippet extracts a snippet from text, trimming to the nearest word boundary
func ExtractSnippet(text string, maxLength int) string {
	// Remove leading/trailing whitespace
	text = strings.TrimSpace(text)

	// If text is shorter than max length, return as-is
	if len(text) <= maxLength {
		return text
	}

	// Truncate to max length
	snippet := text[:maxLength]

	// Find the last space to avoid cutting mid-word
	lastSpace := strings.LastIndex(snippet, " ")
	if lastSpace > 0 {
		snippet = snippet[:lastSpace]
	}

	return snippet
}
