# Product Requirements Document: Personal Obsidian Agent (POA)

## 1. Executive Summary
The Personal Obsidian Agent (POA) is a lightweight, external AI service designed to deeply connect a larger personal agent ecosystem to the user's Obsidian knowledge base. It functions as a **Model Context Protocol (MCP) Server**, allowing other agents (or the user directly) to perform Retrieval Augmented Generation (RAG) and active knowledge gardening on the vault. It is designed to run on low-power hardware (Raspberry Pi Zero) and interacts with the vault via the local filesystem.

## 2. Goals & Personalities
- **Headless & Lightweight**: Must run efficiently on a Raspberry Pi Zero (ARM, low RAM).
- **Universal Connector**: Acts as the "API" to the user's brain (notes) for other agents.
- **Privacy First**: Operates entirely locally.
- **Bi-Directional**: Not just reading notes, but actively organizing, tagging, and creating them.

## 3. Architecture

### 3.1 System Context
- **Host**: Raspberry Pi Zero (or any *nix machine).
- **Data Source**: Local filesystem access to the Obsidian Vault (synced via Google Drive).
- **Interface**: Model Context Protocol (MCP) over Stdio or SSE (Server-Sent Events).
- **Brain**: External LLMs (accessed via API) acting as the tool provider for a smarter client (e.g., Gemini CLI, AMP, Claude Code).

### 3.2 Core Components
1.  **File Watcher**: Low-resource daemon to index changes (inotify).
2.  **Vector Store (Lightweight)**: A file-based embedding index (e.g., LanceDB or simple Faiss flat index serialized to disk) to enable semantic search.
    *   *Constraint*: Embedding generation must be offloaded or use a tiny model (bge-micro) to survive on Pi Zero.
3.  **Markdown Parser**: Robust parser to read/write Frontmatter (YAML) and Content without breaking Obsidian formatting.
4.  **MCP Server Layer**: Exposes tools and resources to the client.

## 4. Feature Requirements

### 4.1 Retrieval (The "Read" Capability)
The agent exposes tools to find information:
- **`search_notes(query: string)`**: Fuzzy text search and Semantic search.
- **`read_note(filename: string)`**: Returns content + metadata.
- **`get_daily_note(date: string)`**: specialized retrieval for daily logging.
- **`list_tags()`**: Returns all tags used in the vault.

### 4.2 Action (The "Write" Capability)
The agent exposes tools to modify the vault:
- **`append_to_daily_note(text: string)`**: Adds a log entry with timestamp.
- **`create_note(path: string, content: string, frontmatter: object)`**: Creates new atomic notes.
- **`update_frontmatter(path: string, key: string, value: any)`**: For gardening (e.g., updating status, adding tags).
- **`link_notes(source: string, target: string)`**: Appends a wikilink `[[Target]]` to the source note.

### 4.3 "Gardener" Capabilities
Background tasks that run periodically (optional, triggered by the core agent):
- **Orphan Finder**: Identify notes with no links.
- **Auto-Tagger**: Suggest tags based on content (requires external LLM call).

## 5. Technical Stack Proposal
- **Language**: **Go (Golang)** or **Rust**.
    *   *Why?* Python is heavy for a Pi Zero if we load ML libraries. pure-binary languages are better for this hardware constraint. Rust has excellent MCP SDKs emerging.
- **Protocol**: [Model Context Protocol (MCP)](https://modelcontextprotocol.io/).
- **Sync**: User responsible for syncing files to the Pi (likely Syncthing).

## 6. Risks & Mitigations
- **Pi Zero Performance**: Vector embedding is CPU intensive.
    *   *Mitigation*: Use an external embedding provider (OpenAI/Cohere API) or a very quantized local model (ONNX).
- **Concurrent Writes**: User editing a note while Agent edits it.
    *   *Mitigation*: Check file modification times. Append-only bias for logs.

## 7. Roadmap
- **Phase 1: MVP**: Read/Write filesystem tools, exposed via MCP stdio. Text-only search (grep-like).
- **Phase 2: Semantics**: Add vector embeddings database.
- **Phase 3: Deep Link**: "Gardening" logic to auto-link concepts.
