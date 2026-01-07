package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"obsidian-agent/internal/mcp/server"
	"obsidian-agent/internal/vault"
	"obsidian-agent/internal/watcher"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	// First try to load from the directory where the executable is located
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		envPath := filepath.Join(execDir, ".env")
		if err := godotenv.Load(envPath); err != nil {
			// If not found in exec dir, try loading from current working directory
			godotenv.Load()
		}
	} else {
		// Fallback to current working directory
		godotenv.Load()
	}

	// Configure logging to stderr (MCP servers must only write JSON-RPC to stdout)
	log.SetOutput(os.Stderr)

	fmt.Fprintln(os.Stderr, "Obsidian MCP Server starting...")

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: obsidian-mcp <vault-path>")
		os.Exit(1)
	}

	vaultPath, _ := filepath.Abs(os.Args[1])
	fmt.Fprintf(os.Stderr, "Monitoring vault at: %s\n", vaultPath)

	// Start MCP Server
	s, err := server.NewObsidianServer(vaultPath)
	if err != nil {
		log.Fatal(err)
	}

	// Check if auto-indexing is enabled (default: true)
	enableAutoIndex := os.Getenv("ENABLE_AUTO_INDEX") != "false"

	if enableAutoIndex && s.GetVectorStore() != nil {
		// Initialize Watcher
		w, err := watcher.NewWatcher()
		if err != nil {
			log.Printf("Warning: failed to initialize watcher: %v", err)
		} else {
			defer w.Close()

			// Set up callback for file changes
			reader := vault.NewReader(vaultPath)
			w.SetCallback(func(path string, op watcher.FileOp) {
				relPath, err := filepath.Rel(vaultPath, path)
				if err != nil {
					log.Printf("⚠️  Failed to get relative path: %v", err)
					return
				}

				switch op {
				case watcher.OpCreate:
					log.Printf("📝 New note detected: %s", relPath)
					indexNote(s.GetVectorStore(), reader, relPath)

				case watcher.OpModify:
					log.Printf("✏️  Modified note detected: %s", relPath)
					indexNote(s.GetVectorStore(), reader, relPath)

				case watcher.OpDelete:
					log.Printf("🗑️  Deleted note detected: %s", relPath)
					if err := s.GetVectorStore().RemoveDocument(relPath); err != nil {
						log.Printf("⚠️  Failed to remove from index: %v", err)
					} else {
						log.Printf("✓ Removed from index: %s", relPath)
					}
				}
			})

			// Start watching
			if err := w.Start(vaultPath); err != nil {
				log.Printf("Warning: failed to start watcher: %v", err)
			} else {
				log.Println("✓ Auto-indexing enabled")
			}
		}
	} else {
		if !enableAutoIndex {
			log.Println("ℹ️  Auto-indexing disabled (ENABLE_AUTO_INDEX=false)")
		} else {
			log.Println("ℹ️  Auto-indexing unavailable (vector store not initialized)")
		}
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nShutting down gracefully...")
		os.Exit(0)
	}()

	fmt.Fprintln(os.Stderr, "Starting MCP Server over Stdio...")
	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}

// indexNote indexes a single note into the vector store
func indexNote(vecStore interface {
	IndexDocument(id, title, content string) error
}, reader *vault.Reader, path string) {
	content, err := reader.ReadNote(path)
	if err != nil {
		log.Printf("⚠️  Failed to read note: %v", err)
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
		log.Printf("⚠️  Failed to index: %v", err)
	} else {
		log.Printf("✓ Indexed: %s", path)
	}
}
