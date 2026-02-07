.PHONY: build install release clean up down setup init-llm

BINARY_NAME=obsidian-cli
CMD_PATH=./cmd/obsidian-cli
OLLAMA_MODEL=nomic-embed-text

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) $(CMD_PATH)

install:
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	go install $(CMD_PATH)

release:
	@echo "Creating release with goreleaser..."
	goreleaser release --snapshot --clean

clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rn dist

up:
	@echo "Starting services..."
	docker-compose up -d

down:
	@echo "Stopping services..."
	docker-compose down

init-llm:
	@echo "Pulling embedding model $(OLLAMA_MODEL)..."
	docker-compose exec ollama ollama pull $(OLLAMA_MODEL)

setup: build install up
	@echo "Waiting for services to be ready..."
	@sleep 5
	@$(MAKE) init-llm
	@echo "Setup complete! Run '$(BINARY_NAME) --help' to get started."
