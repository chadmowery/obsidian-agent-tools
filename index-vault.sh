#!/bin/bash
set -e

echo "=== Obsidian Vault Indexing Test ==="
echo ""

# Configuration
VAULT_PATH="$(pwd)/obsidian"
export OBSIDIAN_VAULT_PATH="$VAULT_PATH"
export QDRANT_HOST=localhost
export QDRANT_PORT=6334

# Check for OpenAI API key
if [ -z "$OPENAI_API_KEY" ]; then
    echo "❌ Error: OPENAI_API_KEY not set"
    echo "Please set it with: export OPENAI_API_KEY=sk-..."
    exit 1
fi

echo "✓ Configuration:"
echo "  - Vault: $VAULT_PATH"
echo "  - Qdrant: $QDRANT_HOST:$QDRANT_PORT"
echo "  - OpenAI API Key: configured"
echo ""

# Count notes
NOTE_COUNT=$(find "$VAULT_PATH" -name "*.md" -type f | wc -l | tr -d ' ')
echo "📚 Found $NOTE_COUNT markdown notes in vault"
echo ""

# Check Qdrant is running
echo "Checking Qdrant connection..."
if ! curl -s http://localhost:6333/healthz > /dev/null 2>&1; then
    echo "❌ Qdrant is not running"
    echo "Start it with: docker run -d --name obsidian-qdrant -p 6333:6333 -p 6334:6334 qdrant/qdrant"
    exit 1
fi
echo "✓ Qdrant is healthy"
echo ""

# Build the MCP server
echo "Building MCP server..."
go build -o ./obsidian-mcp ./cmd/obsidian-mcp
echo "✓ Build successful"
echo ""

# Create indexing script using MCP JSON-RPC
echo "Creating indexing script..."
cat > /tmp/index-all-notes.sh << 'INDEXSCRIPT'
#!/bin/bash

VAULT_PATH="$1"
MCP_SERVER="$2"

# Find all markdown files
find "$VAULT_PATH" -name "*.md" -type f | while read -r note; do
    # Get relative path
    REL_PATH="${note#$VAULT_PATH/}"
    
    echo "Indexing: $REL_PATH"
    
    # Create MCP request to index the note
    REQUEST=$(cat <<EOF
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "index_note",
    "arguments": {
      "path": "$REL_PATH"
    }
  }
}
EOF
)
    
    # Send request to MCP server (via stdin)
    echo "$REQUEST" | "$MCP_SERVER" "$VAULT_PATH" 2>/dev/null || true
    
    sleep 0.1  # Small delay to avoid overwhelming
done

echo ""
echo "✓ Indexing complete"
INDEXSCRIPT

chmod +x /tmp/index-all-notes.sh

echo "✓ Indexing script created"
echo ""

echo "=== Ready to Index ==="
echo ""
echo "To index all notes, you'll need to use an MCP client."
echo "The MCP server is stdio-based, so we need a client to send requests."
echo ""
echo "Manual indexing steps:"
echo "1. Start the server: ./obsidian-mcp $VAULT_PATH"
echo "2. Use an MCP client (Claude Desktop, Gemini CLI) to call 'index_note' for each file"
echo ""
echo "Alternatively, we can create a simple test that indexes a few sample notes."
echo ""

# Show sample notes
echo "Sample notes in vault:"
find "$VAULT_PATH" -name "*.md" -type f | head -5
echo ""

echo "=== Test Complete ==="
