# Makefile for h3c_exporter project

# Variables
BINARY_NAME = h3c_exporter
CONFIG_FILE = config.yaml
GO = go
GOFLAGS = -v
LDFLAGS = -w -s  # Strip debug info and symbols
SRC = $(wildcard *.go) $(wildcard */*.go)

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

# Run the program
.PHONY: run
run: build
	./$(BINARY_NAME) -config $(CONFIG_FILE)

# Run tests
.PHONY: test
test:
	$(GO) test $(GOFLAGS) ./...

# Format code
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Run go vet
.PHONY: vet
vet:
	$(GO) vet ./...

# Clean up
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	$(GO) clean

# Install dependencies
.PHONY: deps
deps:
	$(GO) mod tidy
	$(GO) mod download

# Help
# Cross compile for different platforms
.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-linux-amd64 .

.PHONY: build-windows-amd64
build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-windows-amd64.exe .

.PHONY: build-darwin-amd64
build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME)-darwin-amd64 .

	@echo "  build-linux-amd64 - Cross compile for Linux AMD64"
	@echo "  build-windows-amd64 - Cross compile for Windows AMD64"
	@echo "  build-darwin-amd64 - Cross compile for macOS AMD64"
# Add these targets to the help message
.PHONY: help
help:
	@echo "Makefile for h3c_exporter"
	@echo ""
	@echo "Targets:"
	@echo "  all     - Build the project (default)"
	@echo "  build   - Compile the binary"
	@echo "  run     - Build and run with default config ($(CONFIG_FILE))"
	@echo "  test    - Run all tests"
	@echo "  fmt     - Format Go code"
	@echo "  vet     - Run go vet for static analysis"
	@echo "  clean   - Remove binary and clean Go cache"
	@echo "  deps    - Install dependencies"
	@echo "  help    - Show this help message"