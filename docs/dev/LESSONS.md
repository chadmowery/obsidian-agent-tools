### Packaging & Distribution
- **Go Modules**: When renaming a module path in `go.mod`, ensure all internal imports are updated. Use `grep` to verify no old paths remain.
- **Docker Compose**: Prefer `docker-compose.yml` over custom shell scripts for orchestrating multi-container setups. It's standard, declarative, and easier to manage.
- **Makefile**: Use a `Makefile` to abstract complex commands (`docker-compose up`, `go install`, model pulling) into simple targets (`make setup`).
