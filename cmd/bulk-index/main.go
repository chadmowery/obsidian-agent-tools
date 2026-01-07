package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"obsidian-agent/internal/vault"
	"obsidian-agent/internal/vectorstore"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	godotenv.Load()

	if len(os.Args) < 2 {
		fmt.Println("Usage: bulk-index <vault-path>")
		os.Exit(1)
	}

	vaultPath, _ := filepath.Abs(os.Args[1])
	fmt.Printf("📚 Indexing vault: %s\n\n", vaultPath)

	// Initialize vector store with auto embedder selection
	vecStore, err := vectorstore.NewVectorStore(vectorstore.StoreConfig{
		StorePath:    filepath.Join(vaultPath, ".obsidian-agent", "vectors.json"),
		Embedder:     nil, // Will auto-select Ollama or OpenAI
		PreferQdrant: true,
	})
	if err != nil {
		log.Fatalf("❌ Failed to create vector store: %v\n", err)
	}

	reader := vault.NewReader(vaultPath)

	// Find all markdown files
	var notes []string
	err = filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			relPath, _ := filepath.Rel(vaultPath, path)
			notes = append(notes, relPath)
		}
		return nil
	})

	if err != nil {
		log.Fatalf("❌ Failed to scan vault: %v\n", err)
	}

	fmt.Printf("Found %d markdown notes\n\n", len(notes))

	// Index each note
	indexed := 0
	skipped := 0
	failed := 0

	for i, notePath := range notes {
		fmt.Printf("[%d/%d] Indexing: %s\n", i+1, len(notes), notePath)

		// Read note content
		content, err := reader.ReadNote(notePath)
		if err != nil {
			fmt.Printf("  ⚠️  Failed to read: %v\n", err)
			failed++
			continue
		}

		// Extract title from first heading or filename
		title := filepath.Base(notePath)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "# ") {
				title = strings.TrimPrefix(line, "# ")
				break
			}
		}

		// Index the note
		err = vecStore.IndexDocument(notePath, title, content)
		if err != nil {
			fmt.Printf("  ❌ Failed to index: %v\n", err)
			failed++
			continue
		}

		indexed++
		if indexed%10 == 0 {
			fmt.Printf("  ✓ Progress: %d indexed\n", indexed)
		}
	}

	fmt.Printf("\n=== Indexing Complete ===\n")
	fmt.Printf("✓ Indexed: %d\n", indexed)
	if skipped > 0 {
		fmt.Printf("⊘ Skipped: %d\n", skipped)
	}
	if failed > 0 {
		fmt.Printf("❌ Failed: %d\n", failed)
	}
	fmt.Printf("\nTotal documents in store: %d\n", vecStore.DocumentCount())
}
