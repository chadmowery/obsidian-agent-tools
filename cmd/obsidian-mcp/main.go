package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"obsidian-agent/internal/mcp/server"
	"obsidian-agent/internal/watcher"
)

func main() {
	fmt.Println("Obsidian MCP Server starting...")

	if len(os.Args) < 2 {
		fmt.Println("Usage: obsidian-mcp <vault-path>")
		os.Exit(1)
	}

	vaultPath, _ := filepath.Abs(os.Args[1])
	fmt.Printf("Monitoring vault at: %s\n", vaultPath)

	// Initialize Watcher
	w, err := watcher.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	w.Start(vaultPath)

	// Start MCP Server
	s, err := server.NewObsidianServer()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Starting MCP Server over Stdio...")
	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}
