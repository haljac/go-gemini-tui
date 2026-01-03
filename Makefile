VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BINARY_NAME = gemini-tui
DIST_DIR = dist
LDFLAGS = -ldflags "-X main.version=$(VERSION)"

.PHONY: all build clean build-all release help

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/ /'

## build: Build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

## clean: Remove build artifacts
clean:
	rm -rf $(DIST_DIR)
	rm -f $(BINARY_NAME)

## build-all: Build for all supported platforms
build-all: clean
	@mkdir -p $(DIST_DIR)
	@echo "Building $(BINARY_NAME) $(VERSION) for all platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "Done! Binaries in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/

## release: Create a new GitHub release (usage: make release V=v1.0.0)
release:
ifndef V
	$(error VERSION is not set. Usage: make release V=v1.0.0)
endif
	@echo "Creating release $(V)..."
	git tag -a $(V) -m "Release $(V)"
	git push origin $(V)
	$(MAKE) build-all VERSION=$(V)
	gh release create $(V) $(DIST_DIR)/* --title "$(V)" --generate-notes

## install: Install locally
install: build
	@mkdir -p $(HOME)/.local/bin
	cp $(BINARY_NAME) $(HOME)/.local/bin/
	@echo "Installed to $(HOME)/.local/bin/$(BINARY_NAME)"
