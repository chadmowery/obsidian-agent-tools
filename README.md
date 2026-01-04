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

### 🔍 Semantic Search
- **`semantic_search`** - Find notes by meaning using embeddings
- **`index_note`** - Add a note to the vector store

### 🌱 Gardener Tools
- **`find_orphans`** - Find notes with no links
- **`vault_stats`** - Get link structure statistics
- **`suggest_tags`** - AI-powered tag suggestions

## Installation

### Prerequisites
- Go 1.21 or later
- (Optional) OpenAI API key for semantic search and AI features

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/obsidian-agent.git
cd obsidian-agent

# Build the binary
go build -o obsidian-mcp ./cmd/obsidian-mcp

# (Optional) Build for Raspberry Pi Zero
GOOS=linux GOARCH=arm GOARM=6 go build -o obsidian-mcp-arm ./cmd/obsidian-mcp
```

### Raspberry Pi Zero Setup

1. **Copy the binary to your Pi:**
   ```bash
   scp obsidian-mcp-arm pi@your-pi:/home/pi/
   ```

2. **Make it executable:**
   ```bash
   ssh pi@your-pi 'chmod +x /home/pi/obsidian-mcp-arm'
   ```

3. **Set up your Obsidian vault:**
   - Sync your vault to the Pi using Syncthing, rsync, or your preferred method
   - Note the path to your vault (e.g., `/home/pi/obsidian`)

4. **Create a systemd service (optional):**
   ```bash
   sudo nano /etc/systemd/system/obsidian-mcp.service
   ```
   
   ```ini
   [Unit]
   Description=Obsidian MCP Server
   After=network.target

   [Service]
   Type=simple
   User=pi
   Environment="OBSIDIAN_VAULT_PATH=/home/pi/obsidian"
   Environment="OPENAI_API_KEY=your-key-here"
   ExecStart=/home/pi/obsidian-mcp-arm
   Restart=on-failure

   [Install]
   WantedBy=multi-user.target
   ```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OBSIDIAN_VAULT_PATH` | Path to your Obsidian vault | Required |
| `OPENAI_API_KEY` | OpenAI API key for semantic search and AI features | Optional |

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
      "command": "/path/to/obsidian-mcp",
      "env": {
        "OBSIDIAN_VAULT_PATH": "/path/to/your/vault",
        "OPENAI_API_KEY": "sk-..."
      }
    }
  }
}
```

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
┌─────────────────┐     MCP/Stdio      ┌──────────────────┐
│   AI Client     │◄──────────────────►│  Obsidian Agent  │
│ (Gemini, Claude)│                    │   (MCP Server)   │
└─────────────────┘                    └────────┬─────────┘
                                                │
                                                ▼
                                       ┌──────────────────┐
                                       │  Obsidian Vault  │
                                       │   (Filesystem)   │
                                       └──────────────────┘
```

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
