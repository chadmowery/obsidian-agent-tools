package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"obsidian-agent/internal/vault"
)

type ObsidianServer struct {
	*server.MCPServer
	reader *vault.Reader
}

func NewObsidianServer(vaultPath string) (*ObsidianServer, error) {
	s := server.NewMCPServer("obsidian-agent", "0.1.0")
	reader := vault.NewReader(vaultPath)

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

	return &ObsidianServer{MCPServer: s, reader: reader}, nil
}

func (s *ObsidianServer) Serve() error {
	return server.ServeStdio(s.MCPServer)
}
