# Makefile for kustomize-upsert

BINARY_NAME=kustomize-upsert
VERSION?=v1.0.0
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

.PHONY: build test clean release examples

# Build for current platform
build:
	go build -o $(BINARY_NAME) .

# Run tests
test:
	go test ./...

# Build for all platforms
release:
	@echo "Building release $(VERSION) for multiple platforms..."
	@mkdir -p dist
	@for platform in $(PLATFORMS); do \
		OS=$$(echo $$platform | cut -d'/' -f1); \
		ARCH=$$(echo $$platform | cut -d'/' -f2); \
		echo "Building for $$OS/$$ARCH..."; \
		GOOS=$$OS GOARCH=$$ARCH go build -o dist/$(BINARY_NAME)-$$OS-$$ARCH .; \
	done
	@echo "Release build complete. Binaries in dist/"

# Install locally for development
install: build
	cp $(BINARY_NAME) /usr/local/bin/

# Test examples
examples: build
	@echo "Testing OTEL example..."
	kustomize build --enable-alpha-plugins examples/otel/
	@echo "Testing generic example..."
	kustomize build --enable-alpha-plugins examples/generic/

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf dist/

# Format code
fmt:
	go fmt ./...

# Lint code  
lint:
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy
