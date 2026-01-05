package vectorstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OllamaEmbedder generates embeddings using Ollama
// Implements EmbedderInterface
type OllamaEmbedder struct {
	endpoint   string
	model      string
	httpClient *http.Client
}

// OllamaEmbedderConfig holds configuration for Ollama embedder
type OllamaEmbedderConfig struct {
	// Endpoint is the Ollama API URL
	// Default: http://localhost:11434
	Endpoint string

	// Model is the embedding model to use
	// Default: nomic-embed-text
	Model string
}

// ollamaEmbedRequest is the request format for Ollama embeddings API
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// ollamaEmbedResponse is the response format for Ollama embeddings API
type ollamaEmbedResponse struct {
	Embedding []float32 `json:"embedding"`
}

// NewOllamaEmbedder creates a new Ollama embedder
func NewOllamaEmbedder(config OllamaEmbedderConfig) *OllamaEmbedder {
	if config.Endpoint == "" {
		config.Endpoint = getEnvOrDefault("OLLAMA_ENDPOINT", "http://localhost:11434")
	}
	if config.Model == "" {
		config.Model = getEnvOrDefault("OLLAMA_MODEL", "nomic-embed-text")
	}

	return &OllamaEmbedder{
		endpoint:   config.Endpoint,
		model:      config.Model,
		httpClient: &http.Client{},
	}
}

// Embed generates an embedding vector for the given text
func (e *OllamaEmbedder) Embed(text string) ([]float32, error) {
	reqBody := ollamaEmbedRequest{
		Model:  e.model,
		Prompt: text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := e.endpoint + "/api/embeddings"
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var embResp ollamaEmbedResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(embResp.Embedding) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return embResp.Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *OllamaEmbedder) EmbedBatch(texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := e.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text %d: %w", i, err)
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}

// IsConfigured returns true if the embedder is configured
func (e *OllamaEmbedder) IsConfigured() bool {
	// Ollama doesn't require API keys, just check if endpoint is reachable
	resp, err := http.Get(e.endpoint)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
