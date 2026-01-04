package vectorstore

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
)

// Document represents an indexed document in the vector store
type Document struct {
	ID        string    `json:"id"`        // Relative path to the file
	Title     string    `json:"title"`     // Note title
	Content   string    `json:"content"`   // Full text content
	Embedding []float32 `json:"embedding"` // Vector embedding
}

// Store is a lightweight file-based vector store
type Store struct {
	path      string              // Path to the store file
	documents map[string]Document // In-memory document index
	embedder  *Embedder
	mu        sync.RWMutex
}

// NewStore creates a new vector store at the given path
func NewStore(storePath string, embedder *Embedder) (*Store, error) {
	s := &Store{
		path:      storePath,
		documents: make(map[string]Document),
		embedder:  embedder,
	}

	// Try to load existing store
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load store: %w", err)
	}

	return s, nil
}

// load reads the store from disk
func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}

	var docs []Document
	if err := json.Unmarshal(data, &docs); err != nil {
		return fmt.Errorf("failed to parse store: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, doc := range docs {
		s.documents[doc.ID] = doc
	}

	return nil
}

// save writes the store to disk
func (s *Store) save() error {
	s.mu.RLock()
	docs := make([]Document, 0, len(s.documents))
	for _, doc := range s.documents {
		docs = append(docs, doc)
	}
	s.mu.RUnlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return fmt.Errorf("failed to create store directory: %w", err)
	}

	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal store: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write store: %w", err)
	}

	return nil
}

// IndexDocument adds or updates a document in the store
func (s *Store) IndexDocument(id, title, content string) error {
	if s.embedder == nil || !s.embedder.IsConfigured() {
		// Store without embedding if no embedder configured
		s.mu.Lock()
		s.documents[id] = Document{
			ID:      id,
			Title:   title,
			Content: content,
		}
		s.mu.Unlock()
		return s.save()
	}

	// Generate embedding
	embedding, err := s.embedder.Embed(content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	s.mu.Lock()
	s.documents[id] = Document{
		ID:        id,
		Title:     title,
		Content:   content,
		Embedding: embedding,
	}
	s.mu.Unlock()

	return s.save()
}

// RemoveDocument removes a document from the store
func (s *Store) RemoveDocument(id string) error {
	s.mu.Lock()
	delete(s.documents, id)
	s.mu.Unlock()

	return s.save()
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Document   Document
	Similarity float32
}

// SemanticSearch finds documents similar to the query
func (s *Store) SemanticSearch(query string, limit int) ([]SearchResult, error) {
	if s.embedder == nil || !s.embedder.IsConfigured() {
		return nil, fmt.Errorf("semantic search requires configured embedder")
	}

	// Generate query embedding
	queryEmb, err := s.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Calculate similarities
	var results []SearchResult
	for _, doc := range s.documents {
		if len(doc.Embedding) == 0 {
			continue
		}
		sim := cosineSimilarity(queryEmb, doc.Embedding)
		results = append(results, SearchResult{
			Document:   doc,
			Similarity: sim,
		})
	}

	// Sort by similarity (descending)
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit results
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return float32(dotProduct / (math.Sqrt(normA) * math.Sqrt(normB)))
}

// DocumentCount returns the number of indexed documents
func (s *Store) DocumentCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

// HasDocument checks if a document exists in the store
func (s *Store) HasDocument(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.documents[id]
	return exists
}
