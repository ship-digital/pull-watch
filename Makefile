.PHONY: build install run test clean release-dry-run help update-usage

# Get the version from git tag, fallback to dev
VERSION ?= $(shell git describe --tags 2>/dev/null || echo "dev")
BINARY_NAME=pull-watch
INSTALL_PATH=/usr/local/bin

# Build flags
LDFLAGS=-ldflags "-s -w -X main.version=${VERSION}"

help: ## Show this help
	@echo "Available targets:"
	@grep -h "##" $(MAKEFILE_LIST) | grep -v grep | sed -e 's/\(.*\):.*##\(.*\)/\1:\2/' | column -t -s ":"

build: ## Build the binary locally
	go build ${LDFLAGS} -o bin/${BINARY_NAME} ./cmd/pull-watch

install: build ## Install the binary to /usr/local/bin (might require sudo)
	install -m 755 bin/${BINARY_NAME} ${INSTALL_PATH}/${BINARY_NAME}

uninstall: ## Remove the installed binary (might require sudo)
	rm -f ${INSTALL_PATH}/${BINARY_NAME}

run: ## Run the application (example: make run ARGS="-- echo hello")
	go run ${LDFLAGS} ./cmd/pull-watch ${ARGS}

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf dist/

release-dry-run: ## Test the release process without publishing
	goreleaser release --snapshot --clean

# Build for all platforms without publishing
build-all: ## Build for all platforms (binaries will be in dist/)
	goreleaser build --snapshot --clean

update-usage: ## Update README.md usage section with current help output
	@chmod +x scripts/update_usage.sh
	@./scripts/update_usage.sh

# Default target
.DEFAULT_GOAL := help