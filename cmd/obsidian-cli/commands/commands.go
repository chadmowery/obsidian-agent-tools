package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"obsidian-agent/internal/gardener"
	"obsidian-agent/internal/llm"
	"obsidian-agent/internal/vault"
	"obsidian-agent/internal/vectorstore"
	"obsidian-agent/internal/watcher"
)

type Dependencies struct {
	VaultPath  string
	JsonOutput bool
}

// RunRead implements US-001: Read a file
func RunRead(deps *Dependencies, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: read <filename>")
	}
	filename := args[0]

	reader := vault.NewReader(deps.VaultPath)
	content, err := reader.ReadNote(filename)
	if err != nil {
		return err
	}

	if deps.JsonOutput {
		printJson(map[string]string{"content": content})
	} else {
		fmt.Println(content)
	}
	return nil
}

// RunCreate implements US-001: Create a file
func RunCreate(deps *Dependencies, args []string) error {
	// Simplified implementation for skeleton
	if len(args) < 1 {
		return fmt.Errorf("usage: create <path> (content via stdin or flag not impl yet in skeleton)")
	}
	// path := args[0]
	// For now just error as "not fully implemented" or do a basic write if easy
	// We will focus on search for now as the user requested US-002 primarily
	return fmt.Errorf("create not yet fully implemented in recovery phase")
}

// RunSearch implements US-002: Simple text search
func RunSearch(deps *Dependencies, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: search <query>")
	}
	query := strings.Join(args, " ")

	// Simple recursive glob/scan
	// For efficiency, we might want to use `internal/vault` if it has search capabilities,
	// otherwise just walk the directory.

	var matches []string
	err := filepath.Walk(deps.VaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // ignore errors
		}
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if strings.Contains(strings.ToLower(string(content)), strings.ToLower(query)) {
			rel, _ := filepath.Rel(deps.VaultPath, path)
			matches = append(matches, rel)
		}
		return nil
	})

	if err != nil {
		return err
	}

	if deps.JsonOutput {
		printJson(matches)
	} else {
		for _, m := range matches {
			fmt.Println(m)
		}
	}
	return nil
}

// RunSearchSemantic implements US-002: Vector search
func RunSearchSemantic(deps *Dependencies, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: search-semantic <query>")
	}
	query := strings.Join(args, " ")

	// Initialize components
	emb := vectorstore.NewEmbedderAuto()

	// This assumes Qdrant is running or we need to start it?
	// The PRD says "Integration with existing internal/vectorstore"
	// We'll assume the standard Qdrant store
	config := vectorstore.QdrantConfig{
		Host: os.Getenv("QDRANT_HOST"),
		Port: getEnvInt("QDRANT_PORT", 6334),
	}
	store, err := vectorstore.NewQdrantStore(config, emb)
	if err != nil {
		return fmt.Errorf("failed to connect to vector store: %w", err)
	}

	results, err := store.SemanticSearch(query, 5) // Default limit 5
	if err != nil {
		return err
	}

	if deps.JsonOutput {
		printJson(results)
	} else {
		for _, r := range results {
			fmt.Printf("- [%.2f] %s\n", r.Similarity, r.Document.Title)
		}
	}
	return nil
}

// RunAsk implements US-002: RAG
func RunAsk(deps *Dependencies, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: ask <question>")
	}
	question := strings.Join(args, " ")

	// 1. Search
	emb := vectorstore.NewEmbedderAuto()
	config := vectorstore.QdrantConfig{
		Host: os.Getenv("QDRANT_HOST"),
		Port: getEnvInt("QDRANT_PORT", 6334),
	}
	store, err := vectorstore.NewQdrantStore(config, emb)
	if err != nil {
		return fmt.Errorf("vector store error: %w", err)
	}

	docs, err := store.SemanticSearch(question, 3)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	// 2. Construct Context
	var contextBuilder strings.Builder
	for _, doc := range docs {
		contextBuilder.WriteString(fmt.Sprintf("---\nFile: %s\nContent:\n%s\n\n", doc.Document.Title, doc.Document.Content))
	}

	// 3. Call LLM
	// We need an LLM client. checking internal/llm
	client := llm.NewOpenAIClient(os.Getenv("OPENAI_API_KEY"))
	answer, err := client.ChatCompletion(
		"You are a helpful assistant answering based on the provided notes.",
		fmt.Sprintf("Context:\n%s\n\nQuestion: %s", contextBuilder.String(), question),
	)
	if err != nil {
		return fmt.Errorf("llm error: %w", err)
	}

	if deps.JsonOutput {
		printJson(map[string]string{"answer": answer, "context": contextBuilder.String()})
	} else {
		fmt.Println(answer)
	}

	return nil
}

func printJson(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func getEnvInt(key string, defaultVal int) int {
	// simplified
	return defaultVal
}

// RunOrphans implements US-003: List orphan notes
func RunOrphans(deps *Dependencies, args []string) error {
	finder := gardener.NewOrphanFinder(deps.VaultPath)
	orphans, err := finder.FindOrphans()
	if err != nil {
		return err
	}

	if deps.JsonOutput {
		printJson(orphans)
	} else {
		for _, note := range orphans {
			fmt.Println(note)
		}
	}
	return nil
}

// RunStats implements US-003: Vault statistics
func RunStats(deps *Dependencies, args []string) error {
	finder := gardener.NewOrphanFinder(deps.VaultPath)
	stats, err := finder.GetLinkStats()
	if err != nil {
		return err
	}

	if deps.JsonOutput {
		printJson(stats)
	} else {
		fmt.Printf("Total Notes: %d\n", stats["total_notes"])
		fmt.Printf("Orphans:     %d\n", stats["orphans"])
		fmt.Printf("Dead Ends:   %d\n", stats["dead_ends"])
		fmt.Printf("Well Linked: %d\n", stats["well_linked"])
	}
	return nil
}

// RunTags implements US-003: List all tags
func RunTags(deps *Dependencies, args []string) error {
	reader := vault.NewReader(deps.VaultPath)
	tags, err := reader.ListTags()
	if err != nil {
		return err
	}

	if deps.JsonOutput {
		printJson(tags)
	} else {
		for _, tag := range tags {
			fmt.Println(tag)
		}
	}
	return nil
}

// RunLink implements US-003: Create a wikilink
func RunLink(deps *Dependencies, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: link <source> <target>")
	}
	source := args[0]
	target := args[1]

	writer := vault.NewWriter(deps.VaultPath)
	if err := writer.LinkNotes(source, target); err != nil {
		return err
	}

	if deps.JsonOutput {
		printJson(map[string]string{"status": "linked", "source": source, "target": target})
	} else {
		fmt.Printf("Linked '%s' to '%s'\n", source, target)
	}
	return nil
}

// RunWatch implements US-004: File Watcher
func RunWatch(deps *Dependencies, args []string) error {
	// 1. Initialize Vector Store
	emb := vectorstore.NewEmbedderAuto()
	config := vectorstore.QdrantConfig{
		Host: os.Getenv("QDRANT_HOST"),
		Port: getEnvInt("QDRANT_PORT", 6334),
	}
	store, err := vectorstore.NewQdrantStore(config, emb)
	if err != nil {
		return fmt.Errorf("failed to connect to vector store: %w", err)
	}
	// Verify connection
	if count := store.DocumentCount(); count >= 0 {
		fmt.Printf("✓ Connected to vector store (Documents: %d)\n", count)
	}

	// 2. Initialize Watcher
	w, err := watcher.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to initialize watcher: %w", err)
	}
	defer w.Close()

	// 3. Set Callback
	reader := vault.NewReader(deps.VaultPath)
	w.SetCallback(func(path string, op watcher.FileOp) {
		relPath, err := filepath.Rel(deps.VaultPath, path)
		if err != nil {
			fmt.Printf("⚠️  Failed to get relative path: %v\n", err)
			return
		}

		switch op {
		case watcher.OpCreate:
			fmt.Printf("📝 New note detected: %s\n", relPath)
			indexNote(store, reader, relPath)

		case watcher.OpModify:
			fmt.Printf("✏️  Modified note detected: %s\n", relPath)
			indexNote(store, reader, relPath)

		case watcher.OpDelete:
			fmt.Printf("🗑️  Deleted note detected: %s\n", relPath)
			if err := store.RemoveDocument(relPath); err != nil {
				fmt.Printf("⚠️  Failed to remove from index: %v\n", err)
			} else {
				fmt.Printf("✓ Removed from index: %s\n", relPath)
			}
		}
	})

	// 4. Start Watching
	if err := w.Start(deps.VaultPath); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	// 5. Block forever
	select {}
}

// indexNote indexes a single note into the vector store
func indexNote(vecStore interface {
	IndexDocument(id, title, content string) error
}, reader *vault.Reader, path string) {
	content, err := reader.ReadNote(path)
	if err != nil {
		fmt.Printf("⚠️  Failed to read note: %v\n", err)
		return
	}

	// Extract title from first heading or filename
	title := filepath.Base(path)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			title = strings.TrimPrefix(line, "# ")
			break
		}
	}

	if err := vecStore.IndexDocument(path, title, content); err != nil {
		fmt.Printf("⚠️  Failed to index: %v\n", err)
	} else {
		fmt.Printf("✓ Indexed: %s\n", path)
	}
}

// RunIndex implements Bulk Index Command
func RunIndex(deps *Dependencies, args []string) error {
	// 1. Initialize Vector Store
	emb := vectorstore.NewEmbedderAuto()
	config := vectorstore.QdrantConfig{
		Host: os.Getenv("QDRANT_HOST"),
		Port: getEnvInt("QDRANT_PORT", 6334),
	}
	store, err := vectorstore.NewQdrantStore(config, emb)
	if err != nil {
		return fmt.Errorf("failed to connect to vector store: %w", err)
	}
	// Verify connection
	if count := store.DocumentCount(); count >= 0 {
		fmt.Printf("✓ Connected to vector store (Documents: %d)\n", count)
	}

	reader := vault.NewReader(deps.VaultPath)
	count := 0

	fmt.Printf("📂 Scanning vault: %s\n", deps.VaultPath)

	err = filepath.Walk(deps.VaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden files/dirs
		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			relPath, _ := filepath.Rel(deps.VaultPath, path)
			fmt.Printf("Indexing: %s\n", relPath)
			indexNote(store, reader, relPath)
			count++
		}
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Printf("✨ Bulk indexing complete. Indexed %d notes.\n", count)
	return nil
}
