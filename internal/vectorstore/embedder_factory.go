package vectorstore

import (
	"fmt"
	"os"
)

// EmbedderInterface defines the interface for embedding backends
type EmbedderInterface interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	IsConfigured() bool
}

// NewEmbedderAuto creates an embedder based on environment configuration
// Priority: Ollama > OpenAI
func NewEmbedderAuto() EmbedderInterface {
	// Try Ollama first (local, no API key needed)
	if os.Getenv("OLLAMA_ENDPOINT") != "" || os.Getenv("USE_OLLAMA") == "true" {
		ollama := NewOllamaEmbedder(OllamaEmbedderConfig{})
		if ollama.IsConfigured() {
			fmt.Fprintf(os.Stderr, "✓ Using Ollama embeddings (local)\n")
			return ollama
		}
		fmt.Fprintf(os.Stderr, "⚠ Ollama not available, trying OpenAI\n")
	}

	// Fall back to OpenAI
	openai := NewEmbedder(EmbedderConfig{})
	if openai.IsConfigured() {
		fmt.Fprintf(os.Stderr, "✓ Using OpenAI embeddings\n")
		return openai
	}

	fmt.Fprintf(os.Stderr, "⚠ No embedder configured\n")
	return openai // Return unconfigured embedder (will error on use)
}
