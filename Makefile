.PHONY: build install release clean

BINARY_NAME=obsidian-cli
CMD_PATH=./cmd/obsidian-cli

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
	rm -rf dist
