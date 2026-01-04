package vectorstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Embedder generates vector embeddings for text
type Embedder struct {
	apiEndpoint string
	apiKey      string
	model       string
	httpClient  *http.Client
}

// EmbedderConfig holds configuration for the embedder
type EmbedderConfig struct {
	// APIEndpoint is the URL for the embedding API (OpenAI-compatible)
	// Default: https://api.openai.com/v1/embeddings
	APIEndpoint string

	// APIKey is the authentication key
	APIKey string

	// Model is the embedding model to use
	// Default: text-embedding-3-small
	Model string
}

// embeddingRequest is the request format for OpenAI-compatible APIs
type embeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

// embeddingResponse is the response format for OpenAI-compatible APIs
type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// NewEmbedder creates a new embedder with the given configuration
func NewEmbedder(config EmbedderConfig) *Embedder {
	if config.APIEndpoint == "" {
		config.APIEndpoint = "https://api.openai.com/v1/embeddings"
	}
	if config.Model == "" {
		config.Model = "text-embedding-3-small"
	}
	if config.APIKey == "" {
		config.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	return &Embedder{
		apiEndpoint: config.APIEndpoint,
		apiKey:      config.APIKey,
		model:       config.Model,
		httpClient:  &http.Client{},
	}
}

// Embed generates an embedding vector for the given text
func (e *Embedder) Embed(text string) ([]float32, error) {
	if e.apiKey == "" {
		return nil, fmt.Errorf("no API key configured for embedder")
	}

	reqBody := embeddingRequest{
		Input: text,
		Model: e.model,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", e.apiEndpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embResp.Error != nil {
		return nil, fmt.Errorf("embedding API error: %s", embResp.Error.Message)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return embResp.Data[0].Embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
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

// IsConfigured returns true if the embedder has an API key
func (e *Embedder) IsConfigured() bool {
	return e.apiKey != ""
}
