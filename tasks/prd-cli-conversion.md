# PRD: Obsidian Agent CLI Conversion

## Introduction
Convert the existing Obsidian MCP server into a standalone CLI tool (`obsidian-cli`). This tool will provide AI agents with a direct, environment-agnostic interface to interact with an Obsidian vault, replacing the complex MCP architecture.

## Goals
- Provide a simple, standard CLI interface for all vault operations.
- Ensure 100% functional parity with the existing MCP toolset.
- Output JSON-formatted data (`--json`) to simplify parsing for AI agents.
- Remove the legacy MCP server implementation to reduce maintenance burden.

## User Stories

### US-001: Basic Note Operations (Read/Write)
**Description:** As an agent, I want to read, create, and append to notes so I can manage knowledge.

**Acceptance Criteria:**
- [ ] `read <file>` returns file content.
- [ ] `create <path> --content "..."` creates a new note.
- [ ] `daily-append <text>` appends to today's daily note.
- [ ] `daily [date]` returns the daily note content.
- [ ] `frontmatter <path> <key> <value>` updates YAML frontmatter.
- [ ] All commands support `--json` output.
- [ ] Verify functionality with `obsidian-cli` commands.

### US-002: Search and Retrieval
**Description:** As an agent, I want to find relevant notes using both keyword and semantic search.

**Acceptance Criteria:**
- [ ] `search <query>` performs text search (grep-like).
- [ ] `search-semantic <query>` performs vector-based similarity search.
- [ ] `ask <question>` returns a RAG-generated answer.
- [ ] Integration with existing `internal/vectorstore` and `internal/vault`.
- [ ] Verify search results match expected hits.

### US-003: Vault Maintenance (Gardener)
**Description:** As an agent, I want to analyze the vault structure to keep it organized.

**Acceptance Criteria:**
- [ ] `orphans` lists notes with no links.
- [ ] `tags` lists all tags.
- [ ] `stats` returns vault statistics.
- [ ] `link <source> <target>` creates a wikilink between notes.

### US-004: File Watcher (Foreground)
**Description:** As a user/agent, I want to run a watcher process to keep the search index up to date.

**Acceptance Criteria:**
- [ ] `watch` command starts a blocking process.
- [ ] Monitors file system changes (`fsnotify`).
- [ ] Updates vector index on file create/modify/delete.
- [ ] Logs activity to stdout.

### US-005: Legacy Cleanup
**Description:** As a maintainer, I want to remove the obsolete MCP server code.

**Acceptance Criteria:**
- [ ] Delete `cmd/obsidian-mcp`.
- [ ] Delete `internal/mcp`.
- [ ] Ensure no `github.com/mark3labs/mcp-go` dependencies remain in `go.mod`.
- [ ] Build and test `cmd/obsidian-cli` to ensure no regression.

## Functional Requirements
- **FR-1**: CLI must accept `--vault` path or `OBSIDIAN_VAULT_PATH` env var.
- **FR-2**: CLI must support `--json` flag for all outputs.
- **FR-3**: RAG commands (`ask`, `suggest-tags`) must load `OPENAI_API_KEY` from `.env`.
- **FR-4**: `watch` command must run in the foreground until interrupted.

## Non-Goals
- Background daemon management (systemd, etc.).
- New features beyond existing MCP parity.
- Complex TUI (Text User Interface); simple stream output is preferred.

## Technical Considerations
- **Entry Point**: `cmd/obsidian-cli/main.go`.
- **Libraries**: `flag` for parsing, existing `internal/*` packages for logic.
- **Migration**: Refactor `internal` packages if they leak MCP types (e.g., `mcp.CallToolResult`).

## Success Metrics
- 0% dependency on `mcp-go`.
- All 15+ existing tool functions available via CLI.
- Agent successfully performs a "research & write" task using only the CLI.

## Open Questions
- None. Scope is strictly defined as parity conversion.
