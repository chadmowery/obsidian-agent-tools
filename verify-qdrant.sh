#!/bin/bash
set -e

echo "=== Obsidian Agent + Qdrant Verification ==="
echo ""

# Set environment variables
export OBSIDIAN_VAULT_PATH=/tmp/obsidian-test-vault
export QDRANT_HOST=localhost
export QDRANT_PORT=6334
export OPENAI_API_KEY="${OPENAI_API_KEY:-}"

echo "✓ Environment configured:"
echo "  - Vault: $OBSIDIAN_VAULT_PATH"
echo "  - Qdrant: $QDRANT_HOST:$QDRANT_PORT"
echo "  - OpenAI API Key: ${OPENAI_API_KEY:+configured}"
echo ""

# Check if we have an API key
if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  Warning: OPENAI_API_KEY not set"
    echo "   Semantic search features will not work"
    echo "   Set it with: export OPENAI_API_KEY=sk-..."
    echo ""
fi

# Build the server
echo "Building MCP server..."
go build -o /tmp/obsidian-mcp ./cmd/obsidian-mcp
echo "✓ Build successful"
echo ""

# Start the server in background
echo "Starting MCP server with Qdrant..."
/tmp/obsidian-mcp "$OBSIDIAN_VAULT_PATH" > /tmp/mcp-server.log 2>&1 &
SERVER_PID=$!
echo "✓ Server started (PID: $SERVER_PID)"
echo ""

# Give it a moment to start
sleep 3

# Check if server is still running
if ! kill -0 $SERVER_PID 2>/dev/null; then
    echo "❌ Server failed to start. Log:"
    cat /tmp/mcp-server.log
    exit 1
fi

echo "✓ Server is running"
echo ""
echo "Server startup log:"
head -20 /tmp/mcp-server.log
echo ""

# Check Qdrant collection was created
echo "Checking Qdrant collections..."
COLLECTIONS=$(curl -s http://localhost:6333/collections | grep -o '"obsidian_notes"' || echo "")
if [ -n "$COLLECTIONS" ]; then
    echo "✓ Qdrant collection 'obsidian_notes' created successfully"
else
    echo "⚠️  Collection not yet created (this is OK if no indexing happened)"
fi
echo ""

# Cleanup
echo "Stopping server..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true
echo "✓ Server stopped"
echo ""

echo "=== Verification Complete ==="
echo ""
echo "Next steps to test manually:"
echo "1. Set OPENAI_API_KEY if you haven't already"
echo "2. Run: OBSIDIAN_VAULT_PATH=/tmp/obsidian-test-vault QDRANT_HOST=localhost ./obsidian-mcp"
echo "3. Use an MCP client to test the tools"
echo ""
echo "Test vault location: /tmp/obsidian-test-vault"
echo "Notes created:"
ls -1 /tmp/obsidian-test-vault/
