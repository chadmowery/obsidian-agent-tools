# Obsidian Agent

A lightweight Model Context Protocol (MCP) server that connects your AI agents to your Obsidian vault. Designed to run on low-power hardware like a Raspberry Pi Zero.

## Features

### 📖 Retrieval Tools (Read)
- **`search_notes`** - Fuzzy text search across all notes
- **`read_note`** - Read the contents of a specific note
- **`get_daily_note`** - Get today's (or a specific date's) daily note
- **`list_tags`** - List all unique tags in the vault

### ✏️ Action Tools (Write)
- **`append_to_daily_note`** - Add a timestamped entry to today's daily note
- **`create_note`** - Create a new note with optional frontmatter
- **`update_frontmatter`** - Update a note's YAML frontmatter
- **`link_notes`** - Add a wikilink from one note to another

### 🔍 Semantic Search & RAG
- **`semantic_search`** - Find notes by meaning using local embeddings (Ollama) or OpenAI
- **`ask_vault`** - Natural language Q&A over your vault with RAG
- **`index_vault`** - Bulk index all notes for semantic search
- **`index_note`** - Add a single note to the vector store
- **Auto-Indexing** - Automatically index notes as they're created/modified (file watcher)
- **Chunking** - Handles long documents by splitting into chunks (100% indexing success)

### 🌱 Gardener Tools
- **`find_orphans`** - Find notes with no links
- **`vault_stats`** - Get link structure statistics
- **`suggest_tags`** - AI-powered tag suggestions

### 🌐 Transport Options
- **Stdio** - Standard MCP transport for local clients
- **HTTP/SSE** - Remote access via HTTP server with Server-Sent Events

## Quick Start

### Automated Setup (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/obsidian-agent.git
cd obsidian-agent

# Run setup script (installs dependencies, builds binaries, sets up Docker)
./setup.sh

# Configure your vault path
cp .env.example .env
# Edit .env and set OBSIDIAN_VAULT_PATH=/path/to/your/vault

# Index your vault
source .env
./bulk-index $OBSIDIAN_VAULT_PATH

# Start the server
./obsidian-mcp $OBSIDIAN_VAULT_PATH
```

The setup script will:
- ✅ Check prerequisites (Go, Docker, curl)
- ✅ Install Go dependencies
- ✅ Build all binaries
- ✅ Set up Qdrant and Ollama containers
- ✅ Pull the `nomic-embed-text` embedding model
- ✅ Create example configuration

### Manual Setup

If you prefer to set up manually:

```bash
# Install dependencies
go mod download

# Build binaries
go build -o obsidian-mcp ./cmd/obsidian-mcp
go build -o obsidian-mcp-http ./cmd/obsidian-mcp-http
go build -o bulk-index ./cmd/bulk-index

# Start Qdrant
docker run -d --name obsidian-qdrant \
  -p 6333:6333 -p 6334:6334 \
  -v obsidian_qdrant_data:/qdrant/storage \
  qdrant/qdrant

# Start Ollama
docker run -d --name ollama \
  -p 11434:11434 \
  -v ollama_data:/root/.ollama \
  ollama/ollama

# Pull embedding model
docker exec ollama ollama pull nomic-embed-text
```


### Raspberry Pi Zero Setup

1. **Build for ARM:**
   ```bash
   GOOS=linux GOARCH=arm GOARM=6 go build -o obsidian-mcp-arm ./cmd/obsidian-mcp
   ```

2. **Copy to your Pi:**
   ```bash
   scp obsidian-mcp-arm pi@your-pi:/home/pi/
   ssh pi@your-pi 'chmod +x /home/pi/obsidian-mcp-arm'
   ```

3. **Set up Docker containers on Pi:**
   ```bash
   # SSH into your Pi and run the same Docker commands from Manual Setup
   ssh pi@your-pi
   # Follow the Docker setup steps for Qdrant (using volume) and Ollama
   ```

4. **Create a systemd service (optional):**
   ```bash
   sudo nano /etc/systemd/system/obsidian-mcp.service
   ```
   
   ```ini
   [Unit]
   Description=Obsidian MCP Server
   After=network.target docker.service
   Requires=docker.service

   [Service]
   Type=simple
   User=pi
   Environment="OBSIDIAN_VAULT_PATH=/home/pi/obsidian"
   Environment="USE_OLLAMA=true"
   Environment="QDRANT_HOST=localhost"
   ExecStart=/home/pi/obsidian-mcp-arm
   Restart=on-failure

   [Install]
   WantedBy=multi-user.target
   ```
   
   ```bash
   sudo systemctl enable obsidian-mcp
   sudo systemctl start obsidian-mcp
   ```


## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OBSIDIAN_VAULT_PATH` | Path to your Obsidian vault | Required |
| `OPENAI_API_KEY` | OpenAI API key for semantic search and AI features | Optional |
| `USE_OLLAMA` | Use Ollama for local embeddings instead of OpenAI | `false` |
| `OLLAMA_ENDPOINT` | Ollama API endpoint | `http://localhost:11434` |
| `OLLAMA_MODEL` | Ollama embedding model | `nomic-embed-text` |
| `QDRANT_HOST` | Qdrant server host | `localhost` |
| `QDRANT_PORT` | Qdrant gRPC port | `6334` |
| `QDRANT_API_KEY` | Qdrant API key (for Qdrant Cloud) | Optional |
| `QDRANT_USE_TLS` | Enable TLS for Qdrant connection | `false` |
| `ENABLE_AUTO_INDEX` | Enable automatic file watcher indexing | `true` |
| `MCP_HTTP_PORT` | HTTP server port (for obsidian-mcp-http) | `8080` |

### Vector Storage

The agent supports two vector storage backends for semantic search:

#### Qdrant (Recommended)

Qdrant provides persistent, scalable vector storage that survives server restarts.

**Local Docker Setup:**
```bash
# Start Qdrant locally
docker run -p 6333:6333 -p 6334:6334 \
  -v obsidian_qdrant_data:/qdrant/storage \
  qdrant/qdrant

# Set environment variables
export QDRANT_HOST=localhost
export QDRANT_PORT=6334
```

**Qdrant Cloud Setup:**
```bash
# Get your cluster URL and API key from https://cloud.qdrant.io
export QDRANT_HOST=xyz-example.eu-central.aws.cloud.qdrant.io
export QDRANT_PORT=6334
export QDRANT_API_KEY=your-api-key-here
export QDRANT_USE_TLS=true
```

#### JSON Store (Fallback)

If Qdrant is not configured or unavailable, the agent automatically falls back to a JSON-based vector store. This is suitable for development but doesn't persist across restarts as efficiently.

## MCP Client Configuration

### Gemini CLI / Google AI Studio

Add to your MCP configuration:

```json
{
  "mcpServers": {
    "obsidian": {
      "command": "/path/to/obsidian-mcp",
      "env": {
        "OBSIDIAN_VAULT_PATH": "/path/to/your/vault"
      }
    }
  }
}
```

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "obsidian": {
      "command": "/absolute/path/to/obsidian-mcp",
      "args": ["/absolute/path/to/your/vault"],
      "env": {
        "USE_OLLAMA": "true"
      }
    }
  }
}
```

> [!NOTE]
> Ensure you use absolute paths for both the command and the vault argument. The server will automatically look for the `.env` file in the same directory as the executable.

### Remote Connection (SSH)

For running on a remote Pi, use SSH tunneling or SSE transport:

```json
{
  "mcpServers": {
    "obsidian": {
      "command": "ssh",
      "args": ["pi@your-pi", "/home/pi/obsidian-mcp-arm"],
      "env": {
        "OBSIDIAN_VAULT_PATH": "/home/pi/obsidian"
      }
    }
  }
}
```

## Usage Examples

### Daily Note Logging
Ask your AI: *"Log that I completed the project review"*

The agent will call `append_to_daily_note` with:
```
- **10:30** Log that I completed the project review
```

### Creating Atomic Notes
Ask your AI: *"Create a note about quantum computing basics"*

The agent will call `create_note` with appropriate content and frontmatter.

### Finding Orphan Notes
Ask your AI: *"Find any orphan notes in my vault"*

The agent will call `find_orphans` and list notes with no links.

### Semantic Search
Ask your AI: *"Find notes related to machine learning projects"*

The agent will call `semantic_search` to find semantically similar notes.

## Architecture

```
┌─────────────────┐     MCP/Stdio      ┌──────────────────────┐
│   AI Client     │◄──────────────────►│  Obsidian Agent      │
│ (Gemini, Claude)│      or HTTP/SSE   │   (MCP Server)       │
└─────────────────┘                    └──────────┬───────────┘
                                                  │
                                    ┌─────────────┼─────────────┐
                                    │             │             │
                           ┌────────▼──────┐ ┌───▼────────┐ ┌──▼──────────┐
                           │  Ollama       │ │  Qdrant    │ │  Obsidian   │
                           │  Embeddings   │ │  Vectors   │ │  Vault      │
                           │  (Local)      │ │  (Docker)  │ │  (Files)    │
                           └───────────────┘ └────────────┘ └─────────────┘
```

**Components:**
- **MCP Server**: Exposes 15 tools via stdio or HTTP/SSE transport
- **Ollama**: Local embedding generation (768-dim vectors, privacy-first)
- **Qdrant**: Persistent vector storage for semantic search
- **Vault**: Direct filesystem access to Obsidian markdown files

## Troubleshooting

### Docker Containers Not Running

```bash
# Check container status
docker ps -a --filter name=obsidian-qdrant --filter name=ollama

# Start containers if stopped
docker start obsidian-qdrant ollama

# View container logs
docker logs obsidian-qdrant
docker logs ollama
```

### Qdrant Connection Issues

```bash
# Verify Qdrant is healthy
curl http://localhost:6333/healthz

# Check collections
curl http://localhost:6333/collections

# Restart Qdrant
docker restart obsidian-qdrant
```

### Ollama Model Issues

```bash
# List installed models
docker exec ollama ollama list

# Pull model if missing
docker exec ollama ollama pull nomic-embed-text

# Test embedding generation
curl http://localhost:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "prompt": "test"
}'
```

### Indexing Failures

If notes fail to index:
- **Long notes**: The chunking system handles notes up to ~50,000 characters
- **Empty files**: Empty markdown files are skipped automatically
- **Special characters**: All Unicode characters are supported

Check indexing logs for specific errors:
```bash
./bulk-index /path/to/vault 2>&1 | tee indexing.log
```

### MCP Client Connection Issues

**Claude Desktop:**
- Check `~/Library/Application Support/Claude/logs/mcp*.log`
- Verify binary path is absolute in config
- Ensure environment variables are set correctly

**Gemini CLI:**
- Run with `--verbose` flag to see MCP communication
- Check that stdio transport is working: `echo '{"jsonrpc":"2.0","id":1,"method":"initialize"}' | ./obsidian-mcp /path/to/vault`

## Development

```bash
# Run tests
go test ./...

# Build and run locally
go run ./cmd/obsidian-mcp

# Format code
go fmt ./...
```

## License

MIT License - See [LICENSE](LICENSE) for details.

## Contributing

Contributions welcome! Please open an issue or submit a PR.
