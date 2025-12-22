# Billy Idle - Agent Guidelines

## Build & Test Commands
- `make build` - Build binary for current platform  
- `make test` - Run all tests with coverage (implies lint)
- `make lint` - Format code and run golangci-lint with --fix
- `go test ./... -v` - Run tests without coverage
- `go test -run TestSpecificFunction ./pkg/tracker` - Run single test

## Code Style Guidelines
- **Formatting**: Use `goimports` and `gofmt` (handled by make lint)
- **Imports**: Group standard library, third-party, and internal imports separately
- **Naming**: CamelCase for exported, camelCase for unexported. Use descriptive names.
- **Types**: Use explicit types in function signatures, prefer concrete types over interfaces
- **Error Handling**: Always check errors, use `fmt.Errorf` with wrapping, avoid `panic`
- **Globals**: Minimize global variables (see .golangci.yml exclusions)
- **gRPC**: Follow standard patterns, embed `UnimplementedXxxServer` for forward compatibility
- **Testing**: Use table-driven tests, mock external dependencies, aim for high coverage
- **Cobra**: Follow standard command patterns, use flags for configuration, handle context properly

## Linting Configuration
Uses golangci-lint with strict settings. Key rules enforced:
- No globals (except in cmd/ package)
- Function length < 100 lines, < 50 statements  
- No naked returns, proper error handling
- Magic numbers detected (mnd linter)