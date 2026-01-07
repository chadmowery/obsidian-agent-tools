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

// ensureCollection creates the collection if it doesn't exist, or recreates it if dimensions mismatch
func (s *QdrantStore) ensureCollection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Determine expected vector size from embedder
	expectedSize := uint64(1536) // Default for OpenAI
	if s.embedder != nil && s.embedder.IsConfigured() {
		testEmb, err := s.embedder.Embed("test")
		if err == nil && len(testEmb) > 0 {
			expectedSize = uint64(len(testEmb))
		}
	}

	// Check if collection exists
	exists, err := s.client.CollectionExists(ctx, s.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		// Check if dimensions match
		info, err := s.client.GetCollectionInfo(ctx, s.collectionName)
		if err != nil {
			return fmt.Errorf("failed to get collection info: %w", err)
		}

		// Use getters to safely access nested proto fields
		// Note: GetVectorsConfig() returns the config, GetParams() returns the single vector params (if configured that way)
		// If it was configured as a map (named vectors), GetParams() might be nil or empty, and Size would be 0.
		// In that case, we'll likely recreate it, which is fine as we want a single unnamed vector configuration.
		currentSize := info.GetConfig().GetParams().GetVectorsConfig().GetParams().GetSize()

		if currentSize != expectedSize {
			fmt.Fprintf(os.Stderr, "⚠ Warning: Collection '%s' has dimension %d, but embedder uses %d.\n",
				s.collectionName, currentSize, expectedSize)
			fmt.Fprintf(os.Stderr, "↺ Recreating collection to match new embedder config...\n")

			if err := s.client.DeleteCollection(ctx, s.collectionName); err != nil {
				return fmt.Errorf("failed to delete mismatched collection: %w", err)
			}
			exists = false
		}
	}

	if exists {
		return nil
	}

	// Create collection
	err = s.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: s.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     expectedSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	return nil
}

// IndexDocument adds or updates a document in the store
// Automatically chunks long documents to fit within embedding context limits
func (s *QdrantStore) IndexDocument(id, title, content string) error {
	if s.embedder == nil || !s.embedder.IsConfigured() {
		return fmt.Errorf("embedder not configured")
	}

	// First, remove any existing chunks for this document
	if err := s.RemoveDocument(id); err != nil {
		// Non-fatal, just log
		fmt.Fprintf(os.Stderr, "Warning: failed to remove existing document: %v\n", err)
	}

	// Chunk the content
	chunks := ChunkText(content, id, ChunkConfig{})
	if len(chunks) == 0 {
		// Nothing to index (empty file)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(chunks)*30)*time.Second)
	defer cancel()

	var points []*qdrant.PointStruct

	for _, chunk := range chunks {
		// Generate embedding for this chunk
		embedding, err := s.embedder.Embed(chunk.Text)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for chunk %d: %w", chunk.Index, err)
		}

		// Create unique ID for this chunk
		chunkID := fmt.Sprintf("%s#chunk%d", id, chunk.Index)
		idHash := sha256.Sum256([]byte(chunkID))
		idUUID := hex.EncodeToString(idHash[:16])

		point := &qdrant.PointStruct{
			Id:      qdrant.NewID(idUUID),
			Vectors: qdrant.NewVectors(embedding...),
			Payload: qdrant.NewValueMap(map[string]any{
				"id":           id,
				"title":        title,
				"content":      chunk.Text,
				"chunk_index":  chunk.Index,
				"total_chunks": chunk.TotalChunks,
				"is_chunked":   chunk.TotalChunks > 1,
			}),
		}

		points = append(points, point)
	}

	// Upsert all chunks
	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collectionName,
		Points:         points,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

// RemoveDocument removes a document and all its chunks from the store
func (s *QdrantStore) RemoveDocument(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Delete by filter (all points with matching parent ID)
	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Filter{
				Filter: &qdrant.Filter{
					Must: []*qdrant.Condition{
						{
							ConditionOneOf: &qdrant.Condition_Field{
								Field: &qdrant.FieldCondition{
									Key: "id",
									Match: &qdrant.Match{
										MatchValue: &qdrant.Match_Keyword{
											Keyword: id,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	return nil
}

// SemanticSearch finds documents similar to the query
// Automatically aggregates chunks from the same document
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

	// Query Qdrant with higher limit to account for chunks
	searchLimit := uint64(limit * 3) // Get more results to aggregate chunks
	searchResult, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collectionName,
		Query:          qdrant.NewQuery(queryEmb...),
		Limit:          qdrant.PtrOf(searchLimit),
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query Qdrant: %w", err)
	}

	// Aggregate chunks by document ID
	docMap := make(map[string]*SearchResult)

	for _, point := range searchResult {
		payload := point.GetPayload()

		id := extractStringFromValue(payload["id"])
		title := extractStringFromValue(payload["title"])
		content := extractStringFromValue(payload["content"])
		score := point.GetScore()

		// Check if we've seen this document before
		if existing, ok := docMap[id]; ok {
			// Keep the highest similarity score
			if score > existing.Similarity {
				existing.Similarity = score
			}
			// Append chunk content
			existing.Document.Content += "\n\n" + content
		} else {
			// New document
			docMap[id] = &SearchResult{
				Document: Document{
					ID:      id,
					Title:   title,
					Content: content,
				},
				Similarity: score,
			}
		}
	}

	// Convert map to sorted slice
	var results []SearchResult
	for _, result := range docMap {
		results = append(results, *result)
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
	if len(results) > limit {
		results = results[:limit]
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
