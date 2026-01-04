package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ObsidianServer struct {
	*server.MCPServer
}

func NewObsidianServer() (*ObsidianServer, error) {
	s := server.NewMCPServer("obsidian-agent", "0.1.0")

	// Register tools
	s.AddTool(mcp.NewTool("ping",
		mcp.WithDescription("Ping the server"),
	), func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("pong"), nil
	})

	return &ObsidianServer{MCPServer: s}, nil
}

func (s *ObsidianServer) Serve() error {
	return server.ServeStdio(s.MCPServer)
}
