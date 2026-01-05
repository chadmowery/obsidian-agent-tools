package main

import (
	"fmt"
	"log"
	"path/filepath"

	"obsidian-agent/internal/vault"
	"obsidian-agent/internal/vectorstore"
)

func main() {
	// Configuration
	vaultPath, _ := filepath.Abs("obsidian")

	// Initialize embedder and vector store
	vecStore, err := vectorstore.NewVectorStore(vectorstore.StoreConfig{
		StorePath:    filepath.Join(vaultPath, ".obsidian-agent", "vectors.json"),
		Embedder:     nil, // Will auto-select Ollama
		PreferQdrant: true,
	})
	if err != nil {
		log.Fatalf("Failed to create vector store: %v\n", err)
	}

	// Query
	query := "guitar practice recent months what working on"
	limit := 10

	fmt.Printf("🔍 Searching for: %s\n\n", query)

	// Perform semantic search
	results, err := vecStore.SemanticSearch(query, limit)
	if err != nil {
		log.Fatalf("Search failed: %v\n", err)
	}

	fmt.Printf("Found %d results:\n\n", len(results))

	// Initialize vault reader to get full content
	reader := vault.NewReader(vaultPath)

	// Display results
	for i, result := range results {
		fmt.Printf("─────────────────────────────────────────────────────────\n")
		fmt.Printf("[%d] %s (similarity: %.3f)\n", i+1, result.Document.Title, result.Similarity)
		fmt.Printf("Path: %s\n\n", result.Document.ID)

		// Read full note content
		content, err := reader.ReadNote(result.Document.ID)
		if err == nil {
			// Show first 500 characters
			if len(content) > 500 {
				fmt.Printf("%s...\n\n", content[:500])
			} else {
				fmt.Printf("%s\n\n", content)
			}
		}
	}

	fmt.Printf("─────────────────────────────────────────────────────────\n")
}
