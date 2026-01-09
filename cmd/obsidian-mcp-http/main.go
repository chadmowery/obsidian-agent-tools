package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"obsidian-agent/internal/mcp/server"
	"obsidian-agent/internal/vault"
	"obsidian-agent/internal/watcher"

	libServer "github.com/mark3labs/mcp-go/server"

	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	godotenv.Load()

	fmt.Println("Obsidian MCP HTTP Server starting...")

	if len(os.Args) < 2 {
		fmt.Println("Usage: obsidian-mcp-http <vault-path>")
		os.Exit(1)
	}

	vaultPath, _ := filepath.Abs(os.Args[1])
	fmt.Printf("Vault path: %s\n", vaultPath)

	// Get port from environment or use default
	port := os.Getenv("MCP_HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	// Start MCP Server
	mcpServer, err := server.NewObsidianServer(vaultPath)
	if err != nil {
		log.Fatal(err)
	}

	// Check if auto-indexing is enabled (default: true)
	enableAutoIndex := os.Getenv("ENABLE_AUTO_INDEX") != "false"

	if enableAutoIndex && mcpServer.GetVectorStore() != nil {
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
					indexNote(mcpServer.GetVectorStore(), reader, relPath)

				case watcher.OpModify:
					log.Printf("✏️  Modified note detected: %s", relPath)
					indexNote(mcpServer.GetVectorStore(), reader, relPath)

				case watcher.OpDelete:
					log.Printf("🗑️  Deleted note detected: %s", relPath)
					if err := mcpServer.GetVectorStore().RemoveDocument(relPath); err != nil {
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

	// Create SSE server
	sseServer := libServer.NewSSEServer(mcpServer.MCPServer,
		libServer.WithSSEEndpoint("http://localhost:"+port+"/mcp/sse"),
		libServer.WithMessageEndpoint("/mcp/message"),
	)

	// Set up router
	mux := http.NewServeMux()

	// Add a simple logging middleware
	logRequest := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("📥 %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			next.ServeHTTP(w, r)
		})
	}

	mux.HandleFunc("/mcp/sse", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📥 %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		if r.Method == http.MethodPost {
			sseServer.MessageHandler().ServeHTTP(w, r)
		} else {
			sseServer.SSEHandler().ServeHTTP(w, r)
		}
	})
	mux.Handle("/mcp/message", logRequest(sseServer.MessageHandler()))

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Set up HTTP server
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		httpServer.Close()
		os.Exit(0)
	}()

	fmt.Printf("🚀 HTTP Server listening on http://localhost:%s\n", port)
	fmt.Println("Endpoints:")
	fmt.Println("  - POST /mcp        - MCP tool calls")
	fmt.Println("  - GET  /mcp/sse    - SSE event stream")
	fmt.Println("  - GET  /health     - Health check")

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
