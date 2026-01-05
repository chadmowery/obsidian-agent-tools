package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"obsidian-agent/internal/gardener"
	"obsidian-agent/internal/vault"
	"obsidian-agent/internal/vectorstore"
)

type ObsidianServer struct {
	*server.MCPServer
	reader      *vault.Reader
	writer      *vault.Writer
	vectorStore vectorstore.VectorStore
}

func NewObsidianServer(vaultPath string) (*ObsidianServer, error) {
	s := server.NewMCPServer("obsidian-agent", "0.1.0")
	reader := vault.NewReader(vaultPath)
	writer := vault.NewWriter(vaultPath)

	// Initialize vector store with factory
	embedder := vectorstore.NewEmbedder(vectorstore.EmbedderConfig{})
	storePath := filepath.Join(vaultPath, ".obsidian-agent", "vectors.json")

	vecStore, err := vectorstore.NewVectorStore(vectorstore.StoreConfig{
		StorePath:    storePath,
		Embedder:     embedder,
		PreferQdrant: true, // Try Qdrant first, fallback to JSON
	})
	if err != nil {
		// Non-fatal: just log and continue without vector store
		fmt.Fprintf(os.Stderr, "Warning: failed to initialize vector store: %v\n", err)
		vecStore = nil
	}

	// Register tools
	s.AddTool(mcp.NewTool("ping",
		mcp.WithDescription("Ping the server"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("pong"), nil
	})

	// search_notes tool
	s.AddTool(mcp.NewTool("search_notes",
		mcp.WithDescription("Search for notes containing a query string"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		query, ok := arguments["query"].(string)
		if !ok {
			return mcp.NewToolResultError("query parameter required"), nil
		}

		results, err := reader.SearchNotes(query)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Found %d notes: %v", len(results), results)), nil
	})

	// read_note tool
	s.AddTool(mcp.NewTool("read_note",
		mcp.WithDescription("Read the contents of a specific note"),
		mcp.WithString("filename", mcp.Required(), mcp.Description("Note filename (with or without .md extension)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		filename, ok := arguments["filename"].(string)
		if !ok {
			return mcp.NewToolResultError("filename parameter required"), nil
		}

		content, err := reader.ReadNote(filename)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("read failed: %v", err)), nil
		}

		return mcp.NewToolResultText(content), nil
	})

	// get_daily_note tool
	s.AddTool(mcp.NewTool("get_daily_note",
		mcp.WithDescription("Get the daily note for a specific date"),
		mcp.WithString("date", mcp.Description("Date in YYYY-MM-DD format, or 'today' (default: today)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		date := "today"
		if d, ok := arguments["date"].(string); ok {
			date = d
		}

		content, err := reader.GetDailyNote(date)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get daily note: %v", err)), nil
		}

		return mcp.NewToolResultText(content), nil
	})

	// list_tags tool
	s.AddTool(mcp.NewTool("list_tags",
		mcp.WithDescription("List all unique tags used in the vault"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tags, err := reader.ListTags()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list tags: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Found %d tags: %v", len(tags), tags)), nil
	})

	// ============ ACTION TOOLS ============

	// append_to_daily_note tool
	s.AddTool(mcp.NewTool("append_to_daily_note",
		mcp.WithDescription("Append a timestamped entry to today's daily note. Creates the note if it doesn't exist."),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to append to the daily note")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		text, ok := arguments["text"].(string)
		if !ok {
			return mcp.NewToolResultError("text parameter required"), nil
		}

		if err := writer.AppendToDailyNote(text); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to append to daily note: %v", err)), nil
		}

		return mcp.NewToolResultText("Successfully appended to daily note"), nil
	})

	// create_note tool
	s.AddTool(mcp.NewTool("create_note",
		mcp.WithDescription("Create a new note with optional frontmatter"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path for the new note (relative to vault root)")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Markdown content for the note")),
		mcp.WithObject("frontmatter", mcp.Description("Optional YAML frontmatter as key-value pairs")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		path, ok := arguments["path"].(string)
		if !ok {
			return mcp.NewToolResultError("path parameter required"), nil
		}
		content, ok := arguments["content"].(string)
		if !ok {
			return mcp.NewToolResultError("content parameter required"), nil
		}

		var fm vault.Frontmatter
		if fmArg, ok := arguments["frontmatter"].(map[string]interface{}); ok {
			fm = vault.Frontmatter(fmArg)
		}

		if err := writer.CreateNote(path, content, fm); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create note: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully created note: %s", path)), nil
	})

	// update_frontmatter tool
	s.AddTool(mcp.NewTool("update_frontmatter",
		mcp.WithDescription("Update a specific key in a note's YAML frontmatter"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the note (relative to vault root)")),
		mcp.WithString("key", mcp.Required(), mcp.Description("Frontmatter key to update")),
		mcp.WithString("value", mcp.Required(), mcp.Description("New value for the key")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		path, ok := arguments["path"].(string)
		if !ok {
			return mcp.NewToolResultError("path parameter required"), nil
		}
		key, ok := arguments["key"].(string)
		if !ok {
			return mcp.NewToolResultError("key parameter required"), nil
		}
		value, ok := arguments["value"].(string)
		if !ok {
			return mcp.NewToolResultError("value parameter required"), nil
		}

		if err := writer.UpdateFrontmatter(path, key, value); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update frontmatter: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully updated %s in %s", key, path)), nil
	})

	// link_notes tool
	s.AddTool(mcp.NewTool("link_notes",
		mcp.WithDescription("Add a wikilink from one note to another"),
		mcp.WithString("source", mcp.Required(), mcp.Description("Source note path")),
		mcp.WithString("target", mcp.Required(), mcp.Description("Target note to link to")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		arguments := request.GetArguments()
		source, ok := arguments["source"].(string)
		if !ok {
			return mcp.NewToolResultError("source parameter required"), nil
		}
		target, ok := arguments["target"].(string)
		if !ok {
			return mcp.NewToolResultError("target parameter required"), nil
		}

		if err := writer.LinkNotes(source, target); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to link notes: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully linked %s to %s", source, target)), nil
	})

	// ============ SEMANTIC SEARCH TOOLS ============

	// semantic_search tool
	s.AddTool(mcp.NewTool("semantic_search",
		mcp.WithDescription("Search notes using semantic similarity (requires OPENAI_API_KEY)"),
		mcp.WithString("query", mcp.Required(), mcp.Description("Natural language search query")),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results (default: 5)")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if vecStore == nil {
			return mcp.NewToolResultError("vector store not initialized"), nil
		}

		arguments := request.GetArguments()
		query, ok := arguments["query"].(string)
		if !ok {
			return mcp.NewToolResultError("query parameter required"), nil
		}

		limit := 5
		if l, ok := arguments["limit"].(float64); ok {
			limit = int(l)
		}

		results, err := vecStore.SemanticSearch(query, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("semantic search failed: %v", err)), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d semantically similar notes:\n", len(results)))
		for i, r := range results {
			sb.WriteString(fmt.Sprintf("%d. %s (score: %.3f)\n", i+1, r.Document.ID, r.Similarity))
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// index_note tool (for manually indexing notes into vector store)
	s.AddTool(mcp.NewTool("index_note",
		mcp.WithDescription("Index a note into the vector store for semantic search"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the note to index")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if vecStore == nil {
			return mcp.NewToolResultError("vector store not initialized"), nil
		}

		arguments := request.GetArguments()
		path, ok := arguments["path"].(string)
		if !ok {
			return mcp.NewToolResultError("path parameter required"), nil
		}

		content, err := reader.ReadNote(path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to read note: %v", err)), nil
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
			return mcp.NewToolResultError(fmt.Sprintf("failed to index note: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully indexed: %s", path)), nil
	})

	// ============ GARDENER TOOLS ============

	orphanFinder := gardener.NewOrphanFinder(vaultPath)
	tagger := gardener.NewTagger(gardener.TaggerConfig{})

	// find_orphans tool
	s.AddTool(mcp.NewTool("find_orphans",
		mcp.WithDescription("Find notes with no incoming or outgoing links (orphan notes)"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orphans, err := orphanFinder.FindOrphans()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to find orphans: %v", err)), nil
		}

		if len(orphans) == 0 {
			return mcp.NewToolResultText("No orphan notes found! Your vault is well-connected."), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d orphan notes:\n", len(orphans)))
		for _, orphan := range orphans {
			sb.WriteString(fmt.Sprintf("- %s\n", orphan))
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	// vault_stats tool
	s.AddTool(mcp.NewTool("vault_stats",
		mcp.WithDescription("Get statistics about the vault's link structure"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		stats, err := orphanFinder.GetLinkStats()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
		}

		var sb strings.Builder
		sb.WriteString("Vault Link Statistics:\n")
		sb.WriteString(fmt.Sprintf("- Total notes: %d\n", stats["total_notes"]))
		sb.WriteString(fmt.Sprintf("- Well-linked: %d\n", stats["well_linked"]))
		sb.WriteString(fmt.Sprintf("- Orphans: %d\n", stats["orphans"]))
		sb.WriteString(fmt.Sprintf("- Dead ends: %d\n", stats["dead_ends"]))

		return mcp.NewToolResultText(sb.String()), nil
	})

	// suggest_tags tool
	s.AddTool(mcp.NewTool("suggest_tags",
		mcp.WithDescription("Use AI to suggest tags for a note (requires OPENAI_API_KEY)"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the note")),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !tagger.IsConfigured() {
			return mcp.NewToolResultError("tagger requires OPENAI_API_KEY"), nil
		}

		arguments := request.GetArguments()
		path, ok := arguments["path"].(string)
		if !ok {
			return mcp.NewToolResultError("path parameter required"), nil
		}

		content, err := reader.ReadNote(path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to read note: %v", err)), nil
		}

		// Get existing tags from vault for context
		existingTags, _ := reader.ListTags()

		suggestions, err := tagger.SuggestTags(content, existingTags)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to suggest tags: %v", err)), nil
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Suggested tags for %s:\n", path))
		for _, s := range suggestions {
			sb.WriteString(fmt.Sprintf("- #%s: %s\n", s.Tag, s.Reason))
		}

		return mcp.NewToolResultText(sb.String()), nil
	})

	return &ObsidianServer{MCPServer: s, reader: reader, writer: writer, vectorStore: vecStore}, nil
}

func (s *ObsidianServer) Serve() error {
	return server.ServeStdio(s.MCPServer)
}
