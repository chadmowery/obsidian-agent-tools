package vectorstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/qdrant/go-client/qdrant"
)

// QdrantStore is a Qdrant-backed vector store
type QdrantStore struct {
	client         *qdrant.Client
	collectionName string
	embedder       EmbedderInterface
}

// QdrantConfig holds configuration for Qdrant connection
type QdrantConfig struct {
	Host           string
	Port           int
	APIKey         string
	UseTLS         bool
	CollectionName string
}

// NewQdrantStore creates a new Qdrant-backed vector store
func NewQdrantStore(config QdrantConfig, embedder EmbedderInterface) (*QdrantStore, error) {
	// Set defaults
	if config.Host == "" {
		config.Host = getEnvOrDefault("QDRANT_HOST", "localhost")
	}
	if config.Port == 0 {
		config.Port = 6334 // gRPC port
	}
	if config.CollectionName == "" {
		config.CollectionName = "obsidian_notes"
	}
	if config.APIKey == "" {
		config.APIKey = os.Getenv("QDRANT_API_KEY")
	}

	// Create Qdrant client
	clientConfig := &qdrant.Config{
		Host: config.Host,
		Port: config.Port,
	}

	if config.APIKey != "" {
		clientConfig.APIKey = config.APIKey
	}

	if config.UseTLS {
		clientConfig.UseTLS = true
	}

	client, err := qdrant.NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	store := &QdrantStore{
		client:         client,
		collectionName: config.CollectionName,
		embedder:       embedder,
	}

	// Initialize collection
	if err := store.ensureCollection(); err != nil {
		return nil, fmt.Errorf("failed to initialize collection: %w", err)
	}

	return store, nil
}

// ensureCollection creates the collection if it doesn't exist
func (s *QdrantStore) ensureCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if collection exists
	exists, err := s.client.CollectionExists(ctx, s.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		return nil
	}

	// Determine vector size from embedder by generating a test embedding
	vectorSize := uint64(1536) // Default for OpenAI
	if s.embedder != nil && s.embedder.IsConfigured() {
		testEmb, err := s.embedder.Embed("test")
		if err == nil && len(testEmb) > 0 {
			vectorSize = uint64(len(testEmb))
		}
	}

	// Create collection
	err = s.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: s.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// IndexDocument adds or updates a document in the store
func (s *QdrantStore) IndexDocument(id, title, content string) error {
	if s.embedder == nil || !s.embedder.IsConfigured() {
		return fmt.Errorf("embedder not configured")
	}

	// Generate embedding
	embedding, err := s.embedder.Embed(content)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create point with payload
	// Use hash of ID as UUID since file paths can contain invalid characters
	idHash := sha256.Sum256([]byte(id))
	idUUID := hex.EncodeToString(idHash[:16]) // Use first 16 bytes as UUID

	point := &qdrant.PointStruct{
		Id:      qdrant.NewID(idUUID),
		Vectors: qdrant.NewVectors(embedding...),
		Payload: qdrant.NewValueMap(map[string]any{
			"id":      id,
			"title":   title,
			"content": content,
		}),
	}

	// Upsert point
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName,
		Points:         []*qdrant.PointStruct{point},
	})

	if err != nil {
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	return nil
}

// RemoveDocument removes a document from the store
func (s *QdrantStore) RemoveDocument(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use hash of ID as UUID
	idHash := sha256.Sum256([]byte(id))
	idUUID := hex.EncodeToString(idHash[:16])

	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						qdrant.NewID(idUUID),
					},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete point: %w", err)
	}

	return nil
}

// SemanticSearch finds documents similar to the query
func (s *QdrantStore) SemanticSearch(query string, limit int) ([]SearchResult, error) {
	if s.embedder == nil || !s.embedder.IsConfigured() {
		return nil, fmt.Errorf("semantic search requires configured embedder")
	}

	// Generate query embedding
	queryEmb, err := s.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Query Qdrant
	searchResult, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collectionName,
		Query:          qdrant.NewQuery(queryEmb...),
		Limit:          qdrant.PtrOf(uint64(limit)),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query Qdrant: %w", err)
	}

	// Convert results
	var results []SearchResult
	for _, point := range searchResult {
		payload := point.GetPayload()

		id := extractStringFromValue(payload["id"])
		title := extractStringFromValue(payload["title"])
		content := extractStringFromValue(payload["content"])

		results = append(results, SearchResult{
			Document: Document{
				ID:      id,
				Title:   title,
				Content: content,
			},
			Similarity: point.GetScore(),
		})
	}

	return results, nil
}

// DocumentCount returns the number of indexed documents
func (s *QdrantStore) DocumentCount() int {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := s.client.GetCollectionInfo(ctx, s.collectionName)
	if err != nil {
		return 0
	}

	return int(info.GetPointsCount())
}

// HasDocument checks if a document exists in the store
func (s *QdrantStore) HasDocument(id string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use hash of ID as UUID
	idHash := sha256.Sum256([]byte(id))
	idUUID := hex.EncodeToString(idHash[:16])

	points, err := s.client.Get(ctx, &qdrant.GetPoints{
		CollectionName: s.collectionName,
		Ids: []*qdrant.PointId{
			qdrant.NewID(idUUID),
		},
	})

	if err != nil {
		return false
	}

	return len(points) > 0
}

// Close closes the Qdrant client connection
func (s *QdrantStore) Close() error {
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

// extractStringFromValue extracts a string from a Qdrant Value
func extractStringFromValue(v *qdrant.Value) string {
	if v == nil {
		return ""
	}
	kind := v.GetKind()
	if kind == nil {
		return ""
	}

	// Type switch on the Kind interface
	switch k := kind.(type) {
	case *qdrant.Value_StringValue:
		return k.StringValue
	default:
		return ""
	}
}
