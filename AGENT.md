# AGENT.md - Kustomize Upsert Transformer

## Build/Test/Validation Commands
- `make build` - Build the binary for current platform
- `make test` - Run Go tests
- `make release` - Build for all platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)
- `make examples` - Test example configurations with kustomize
- `make install` - Install binary locally for development
- `go fmt ./...` - Format Go code
- `golangci-lint run` - Lint Go code

## Architecture & Structure
- **Generic Kustomize Transformer**: Solves "append if exists, else create" for any array field
- **Pure Go Implementation**: Type-safe, no external dependencies
- **Kustomize Plugin**: Uses kustomize's transformer framework
- **Multi-platform**: Builds for Linux and macOS on AMD64 and ARM64

## Code Style & Conventions
- Follow standard Go conventions
- Use structured error handling with wrapped errors
- Validate input configurations thoroughly
- Support JSONPath-style field targeting
- Maintain backward compatibility in configuration schema

## Distribution & Release
- Follow same patterns as `kustomize-lint`
- Multi-platform releases via CI/CD
- Semantic versioning (v1.0.0, v1.1.0, etc.)
- Internal distribution through standard tooling channels

## Configuration Schema
- Use `groq.io/v1alpha1` API group for transformer configuration
- Support multiple operations per transformer instance
- Resource targeting via apiVersion, kind, name, namespace selectors
- JSONPath support for nested field access

## Testing Strategy
- Unit tests for core transformation logic
- Integration tests with actual kustomize builds
- Example configurations that demonstrate various use cases
- Platform-specific build verification
