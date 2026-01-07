#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Obsidian Agent Setup ===${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to print status
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC}  $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check prerequisites
echo -e "${BLUE}Checking prerequisites...${NC}"

if ! command_exists go; then
    print_error "Go is not installed"
    echo "Please install Go 1.21 or later from https://go.dev/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
print_status "Go $GO_VERSION installed"

if ! command_exists docker; then
    print_error "Docker is not installed"
    echo "Please install Docker from https://docs.docker.com/get-docker/"
    exit 1
fi
print_status "Docker installed"

if ! command_exists jq; then
    print_warning "jq is not installed (optional, used for JSON parsing)"
    echo "Install with: brew install jq (macOS) or apt-get install jq (Linux)"
fi

if ! command_exists curl; then
    print_error "curl is not installed"
    exit 1
fi
print_status "curl installed"

echo ""

# Install Go dependencies
echo -e "${BLUE}Installing Go dependencies...${NC}"
go mod download
print_status "Go dependencies installed"
echo ""

# Build binaries
echo -e "${BLUE}Building binaries...${NC}"

BINARIES=("obsidian-mcp" "obsidian-mcp-http" "bulk-index" "test-search")
for binary in "${BINARIES[@]}"; do
    echo "Building $binary..."
    go build -o "$binary" "./cmd/$binary"
    chmod +x "$binary"
    print_status "$binary built successfully"
done
echo ""

# Setup Docker containers
echo -e "${BLUE}Setting up Docker containers...${NC}"

# Check if Qdrant container exists
if docker ps -a --format '{{.Names}}' | grep -q "^obsidian-qdrant$"; then
    if docker ps --format '{{.Names}}' | grep -q "^obsidian-qdrant$"; then
        print_status "Qdrant container already running"
    else
        echo "Starting existing Qdrant container..."
        docker start obsidian-qdrant
        print_status "Qdrant container started"
    fi
else
    echo "Creating Qdrant container..."
    docker run -d \
        --name obsidian-qdrant \
        -p 6333:6333 \
        -p 6334:6334 \
        -v "obsidian_qdrant_data:/qdrant/storage" \
        qdrant/qdrant
    print_status "Qdrant container created and started"
fi

# Wait for Qdrant to be healthy
echo "Waiting for Qdrant to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:6333/healthz >/dev/null 2>&1; then
        print_status "Qdrant is healthy"
        break
    fi
    if [ $i -eq 30 ]; then
        print_error "Qdrant failed to start"
        exit 1
    fi
    sleep 1
done

# Check if Ollama container exists
if docker ps -a --format '{{.Names}}' | grep -q "^ollama$"; then
    if docker ps --format '{{.Names}}' | grep -q "^ollama$"; then
        print_status "Ollama container already running"
    else
        echo "Starting existing Ollama container..."
        docker start ollama
        print_status "Ollama container started"
    fi
else
    echo "Creating Ollama container..."
    docker run -d \
        --name ollama \
        -p 11434:11434 \
        -v ollama_data:/root/.ollama \
        ollama/ollama
    print_status "Ollama container created and started"
fi

# Wait for Ollama to be ready
echo "Waiting for Ollama to be ready..."
for i in {1..30}; do
    if curl -s http://localhost:11434/api/tags >/dev/null 2>&1; then
        print_status "Ollama is healthy"
        break
    fi
    if [ $i -eq 30 ]; then
        print_error "Ollama failed to start"
        exit 1
    fi
    sleep 1
done

# Pull embedding model
echo "Checking for nomic-embed-text model..."
if docker exec ollama ollama list | grep -q "nomic-embed-text"; then
    print_status "nomic-embed-text model already installed"
else
    echo "Pulling nomic-embed-text model (this may take a few minutes)..."
    docker exec ollama ollama pull nomic-embed-text
    print_status "nomic-embed-text model installed"
fi

echo ""

# Create example .env file
if [ ! -f ".env.example" ]; then
    echo -e "${BLUE}Creating example .env file...${NC}"
    cat > .env.example << 'EOF'
# Obsidian Agent Configuration

# Required: Path to your Obsidian vault
OBSIDIAN_VAULT_PATH=/path/to/your/vault

# Embedding Configuration (choose one)
# Option 1: Local embeddings with Ollama (recommended, privacy-first)
USE_OLLAMA=true
OLLAMA_ENDPOINT=http://localhost:11434
OLLAMA_MODEL=nomic-embed-text

# Option 2: OpenAI embeddings (requires API key)
# OPENAI_API_KEY=sk-...

# Qdrant Configuration
QDRANT_HOST=localhost
QDRANT_PORT=6334
# QDRANT_API_KEY=  # Only needed for Qdrant Cloud
# QDRANT_USE_TLS=false  # Set to true for Qdrant Cloud

# Auto-indexing (file watcher)
ENABLE_AUTO_INDEX=true

# HTTP Server Configuration (for obsidian-mcp-http)
MCP_HTTP_PORT=8080
EOF
    print_status ".env.example created"
else
    print_status ".env.example already exists"
fi

echo ""
echo -e "${GREEN}=== Setup Complete! ===${NC}"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo ""
echo "1. Configure your environment:"
echo "   cp .env.example .env"
echo "   # Edit .env and set OBSIDIAN_VAULT_PATH to your vault location"
echo ""
echo "2. Index your vault (one-time):"
echo "   source .env"
echo "   ./bulk-index \$OBSIDIAN_VAULT_PATH"
echo ""
echo "3. Start the MCP server:"
echo "   # For stdio mode (Claude Desktop, Gemini CLI):"
echo "   ./obsidian-mcp \$OBSIDIAN_VAULT_PATH"
echo ""
echo "   # For HTTP/SSE mode (remote access):"
echo "   ./obsidian-mcp-http \$OBSIDIAN_VAULT_PATH"
echo ""
echo "4. Configure your MCP client (see README.md for examples)"
echo ""
echo -e "${BLUE}Docker Containers Running:${NC}"
docker ps --filter name=obsidian-qdrant --filter name=ollama --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
echo ""
echo -e "${BLUE}Binaries Built:${NC}"
ls -lh obsidian-mcp obsidian-mcp-http bulk-index test-search | awk '{print $9, "(" $5 ")"}'
echo ""
