package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"obsidian-agent/cmd/obsidian-cli/commands"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Load Environment
	loadEnv()

	// 2. Parse Global Flags
	vaultPath := flag.String("vault", os.Getenv("OBSIDIAN_VAULT_PATH"), "Path to Obsidian vault")
	jsonOutput := flag.Bool("json", false, "Output results as JSON")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: obsidian-cli [global flags] <command> [arguments]\n")
		fmt.Fprintf(os.Stderr, "\nGlobal Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  search <query>          Simple text search\n")
		fmt.Fprintf(os.Stderr, "  search-semantic <query> Semantic search using vector embeddings\n")
		fmt.Fprintf(os.Stderr, "  ask <question>          Ask a question about your notes (RAG)\n")
		fmt.Fprintf(os.Stderr, "  read <file>             Read a note\n")
		fmt.Fprintf(os.Stderr, "  create <path>           Create a note\n")
		fmt.Fprintf(os.Stderr, "  orphans                 List notes with no links\n")
		fmt.Fprintf(os.Stderr, "  tags                    List all tags\n")
		fmt.Fprintf(os.Stderr, "  stats                   Show vault statistics\n")
		fmt.Fprintf(os.Stderr, "  link <source> <target>  Link two notes\n")
		fmt.Fprintf(os.Stderr, "  watch                   Watch vault for changes and auto-index\n")
	}
	flag.Parse()

	// 3. Validate Global Config
	if *vaultPath == "" {
		// Try current directory as fallback or error
		wd, _ := os.Getwd()
		*vaultPath = wd
		// Optionally check if it looks like a vault?
	}
	absVaultPath, err := filepath.Abs(*vaultPath)
	if err != nil {
		fatal(*jsonOutput, "Invalid vault path: %v", err)
	}

	// 4. Handle Subcommands
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	cmd := args[0]
	cmdArgs := args[1:]

	// Define a dependencies struct to pass around
	deps := &commands.Dependencies{
		VaultPath:  absVaultPath,
		JsonOutput: *jsonOutput,
	}

	var cmdErr error

	switch cmd {
	case "search":
		cmdErr = commands.RunSearch(deps, cmdArgs)
	case "search-semantic":
		cmdErr = commands.RunSearchSemantic(deps, cmdArgs)
	case "ask":
		cmdErr = commands.RunAsk(deps, cmdArgs)
	case "read": // Recovery of US-001
		cmdErr = commands.RunRead(deps, cmdArgs)
	case "create": // Recovery of US-001
		cmdErr = commands.RunCreate(deps, cmdArgs)
	case "orphans":
		cmdErr = commands.RunOrphans(deps, cmdArgs)
	case "tags":
		cmdErr = commands.RunTags(deps, cmdArgs)
	case "stats":
		cmdErr = commands.RunStats(deps, cmdArgs)
	case "link":
		cmdErr = commands.RunLink(deps, cmdArgs)
	case "watch":
		cmdErr = commands.RunWatch(deps, cmdArgs)
	default:
		fatal(*jsonOutput, "Unknown command: %s", cmd)
	}

	if cmdErr != nil {
		fatal(*jsonOutput, "%v", cmdErr)
	}
}

func loadEnv() {
	// Try loading .env from executable dir first, then CWD
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		godotenv.Load(filepath.Join(execDir, ".env"))
	}
	godotenv.Load() // Fallback to CWD
}

func fatal(jsonMode bool, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if jsonMode {
		fmt.Printf(`{"error": "%s"}`+"\n", strings.ReplaceAll(msg, `"`, `\"`))
	} else {
		fmt.Fprintln(os.Stderr, "Error:", msg)
	}
	os.Exit(1)
}
