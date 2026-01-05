# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Development Workflow

### Initial Setup

```bash
# Run the setup script
./setup.sh

# Configure your test vault
cp .env.example .env
# Edit .env and set OBSIDIAN_VAULT_PATH to a test vault
```

### Building

```bash
# Build all binaries
go build -o obsidian-mcp ./cmd/obsidian-mcp
go build -o obsidian-mcp-http ./cmd/obsidian-mcp-http
go build -o bulk-index ./cmd/bulk-index
go build -o test-search ./cmd/test-search

# Or build for Raspberry Pi
GOOS=linux GOARCH=arm GOARM=6 go build -o obsidian-mcp-arm ./cmd/obsidian-mcp
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Test specific package
go test ./internal/vectorstore/...

# Format code
go fmt ./...
```

### Docker Container Management

```bash
# Check container status
docker ps --filter name=obsidian-qdrant --filter name=ollama

# Start containers
docker start obsidian-qdrant ollama

# Stop containers
docker stop obsidian-qdrant ollama

# View logs
docker logs -f obsidian-qdrant
docker logs -f ollama

# Restart containers
docker restart obsidian-qdrant ollama

# Remove containers (data will be preserved in volumes)
docker rm -f obsidian-qdrant ollama
```

### Manual Testing

```bash
# Index a test vault
source .env
./bulk-index $OBSIDIAN_VAULT_PATH

# Start MCP server (stdio mode)
./obsidian-mcp $OBSIDIAN_VAULT_PATH

# Start HTTP server
./obsidian-mcp-http $OBSIDIAN_VAULT_PATH

# Test semantic search
./test-search $OBSIDIAN_VAULT_PATH "your search query"
```

### Debugging

```bash
# Enable verbose logging
export LOG_LEVEL=debug

# Check Qdrant health
curl http://localhost:6333/healthz

# Check Qdrant collections
curl http://localhost:6333/collections

# Test Ollama
curl http://localhost:11434/api/tags

# Test embedding generation
curl http://localhost:11434/api/embeddings -d '{
  "model": "nomic-embed-text",
  "prompt": "test"
}'
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

