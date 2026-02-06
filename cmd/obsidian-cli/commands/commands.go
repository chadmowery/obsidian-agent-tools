package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"obsidian-agent/internal/llm"
	"obsidian-agent/internal/vault"
	"obsidian-agent/internal/vectorstore"
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
