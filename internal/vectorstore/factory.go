package vectorstore

import (
	"fmt"
	"os"
)

// VectorStore defines the interface for vector storage backends
type VectorStore interface {
	IndexDocument(id, title, content string) error
	RemoveDocument(id string) error
	SemanticSearch(query string, limit int) ([]SearchResult, error)
	DocumentCount() int
	HasDocument(id string) bool
}

// StoreConfig holds configuration for creating a vector store
type StoreConfig struct {
	// StorePath is the path for JSON-based store (fallback)
	StorePath string

	// Embedder for generating embeddings (optional, will auto-select if nil)
	Embedder EmbedderInterface

	// Qdrant configuration
	QdrantHost           string
	QdrantPort           int
	QdrantAPIKey         string
	QdrantUseTLS         bool
	QdrantCollectionName string

	// PreferQdrant determines whether to try Qdrant first
	PreferQdrant bool
}

// NewVectorStore creates a vector store based on configuration
// It will try Qdrant first if PreferQdrant is true or if Qdrant env vars are set
// Falls back to JSON store if Qdrant is unavailable
func NewVectorStore(config StoreConfig) (VectorStore, error) {
	// Auto-select embedder if not provided
	if config.Embedder == nil {
		config.Embedder = NewEmbedderAuto()
	}

	// Check if Qdrant should be attempted
	shouldTryQdrant := config.PreferQdrant ||
		os.Getenv("QDRANT_HOST") != "" ||
		config.QdrantHost != ""

	if shouldTryQdrant {
		qdrantConfig := QdrantConfig{
			Host:           config.QdrantHost,
			Port:           config.QdrantPort,
			APIKey:         config.QdrantAPIKey,
			UseTLS:         config.QdrantUseTLS,
			CollectionName: config.QdrantCollectionName,
		}

		store, err := NewQdrantStore(qdrantConfig, config.Embedder)
		if err == nil {
			fmt.Fprintf(os.Stderr, "✓ Using Qdrant vector store\n")
			return store, nil
		}

		// Log Qdrant failure but continue to fallback
		fmt.Fprintf(os.Stderr, "⚠ Qdrant unavailable (%v), falling back to JSON store\n", err)
	}

	// Fallback to JSON store
	store, err := NewStore(config.StorePath, config.Embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Using JSON vector store\n")
	return store, nil
}
