# Lessons Learned

## CLI Conversion
- **Code Reuse**: Reusing logic from `internal/gardener` and `internal/watcher` significantly sped up the implementation of CLI commands (`orphans`, `watch`). It proved that the internal packages were well-structured for reuse.
- **Embedded Flag Parsing**: The `flag` package in Go requires careful ordering (flags before args).
- **Qdrant Integration**: The `vectorstore` package's `NewQdrantStore` signature had to be carefully matched, but once aligned, it provided seamless integration for semantic search in the CLI.
