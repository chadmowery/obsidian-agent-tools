#!/bin/bash
set -e

echo "=== Starting Obsidian MCP Server with Local Embeddings ==="
echo ""

# Configuration
export OBSIDIAN_VAULT_PATH="$(pwd)/obsidian"
export USE_OLLAMA=true
export QDRANT_HOST=localhost
export QDRANT_PORT=6334

echo "✓ Configuration:"
echo "  - Vault: $OBSIDIAN_VAULT_PATH"
echo "  - Embeddings: Ollama (local)"
echo "  - Vector Store: Qdrant ($QDRANT_HOST:$QDRANT_PORT)"
echo ""

# Check containers
echo "Checking containers..."
if ! docker ps | grep -q "obsidian-qdrant"; then
    echo "⚠️  Qdrant container not running, starting..."
    docker start obsidian-qdrant
    sleep 2
fi

if ! docker ps | grep -q "ollama"; then
    echo "⚠️  Ollama container not running, starting..."
    docker start ollama
    sleep 2
fi

echo "✓ Containers running"
echo ""

# Check Qdrant collection
echo "Checking Qdrant collection..."
POINTS=$(curl -s http://localhost:6333/collections/obsidian_notes | jq -r '.result.points_count // 0')
echo "✓ Qdrant collection has $POINTS points indexed"
echo ""

# Start MCP server
echo "Starting MCP server..."
echo "The server will run in stdio mode and wait for MCP requests."
echo ""
echo "To test, use an MCP client (Claude Desktop, Gemini CLI) or send JSON-RPC requests to stdin."
echo ""

exec ./obsidian-mcp "$OBSIDIAN_VAULT_PATH"
