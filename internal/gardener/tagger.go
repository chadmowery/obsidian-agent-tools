package gardener

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Tagger provides AI-powered tag suggestions for notes
type Tagger struct {
	apiEndpoint string
	apiKey      string
	model       string
	httpClient  *http.Client
}

// TaggerConfig holds configuration for the tagger
type TaggerConfig struct {
	// APIEndpoint is the URL for the chat/completions API
	// Default: https://api.openai.com/v1/chat/completions
	APIEndpoint string

	// APIKey is the authentication key
	APIKey string

	// Model is the LLM model to use
	// Default: gpt-4o-mini
	Model string
}

// TagSuggestion represents a suggested tag with reasoning
type TagSuggestion struct {
	Tag    string `json:"tag"`
	Reason string `json:"reason"`
}

// NewTagger creates a new tagger with the given configuration
func NewTagger(config TaggerConfig) *Tagger {
	if config.APIEndpoint == "" {
		config.APIEndpoint = "https://api.openai.com/v1/chat/completions"
	}
	if config.Model == "" {
		config.Model = "gpt-4o-mini"
	}
	if config.APIKey == "" {
		config.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	return &Tagger{
		apiEndpoint: config.APIEndpoint,
		apiKey:      config.APIKey,
		model:       config.Model,
		httpClient:  &http.Client{},
	}
}

// chatRequest is the request format for OpenAI-compatible chat APIs
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// SuggestTags analyzes note content and suggests appropriate tags
func (t *Tagger) SuggestTags(content string, existingTags []string) ([]TagSuggestion, error) {
	if t.apiKey == "" {
		return nil, fmt.Errorf("no API key configured for tagger")
	}

	// Build prompt
	existingTagStr := "none"
	if len(existingTags) > 0 {
		existingTagStr = strings.Join(existingTags, ", ")
	}

	systemPrompt := `You are an expert at organizing notes in an Obsidian vault. 
Analyze the given note content and suggest relevant tags.
Return your suggestions as a JSON array of objects with "tag" and "reason" fields.
Tags should be lowercase, use hyphens for multi-word tags (e.g., "machine-learning").
Suggest 3-5 tags that would help categorize and find this note later.
Only return the JSON array, no other text.`

	userPrompt := fmt.Sprintf(`Existing tags in vault: %s

Note content:
---
%s
---

Suggest appropriate tags for this note.`, existingTagStr, truncateContent(content, 2000))

	req := chatRequest{
		Model: t.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", t.apiEndpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("tag suggestion request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	// Parse the JSON response
	responseContent := chatResp.Choices[0].Message.Content
	// Clean up markdown code blocks if present
	responseContent = strings.TrimPrefix(responseContent, "```json")
	responseContent = strings.TrimPrefix(responseContent, "```")
	responseContent = strings.TrimSuffix(responseContent, "```")
	responseContent = strings.TrimSpace(responseContent)

	var suggestions []TagSuggestion
	if err := json.Unmarshal([]byte(responseContent), &suggestions); err != nil {
		return nil, fmt.Errorf("failed to parse tag suggestions: %w", err)
	}

	return suggestions, nil
}

// truncateContent limits content length for API calls
func truncateContent(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "\n...[truncated]"
}

// IsConfigured returns true if the tagger has an API key
func (t *Tagger) IsConfigured() bool {
	return t.apiKey != ""
}
