# Crush Configuration for xpath-go

## Build Commands
- `go build ./...` - Build all packages
- `go build -o xpath-go` - Build main executable

## Test Commands
- `go test ./...` - Run all tests
- `go test -v <package_path>` - Run verbose tests for specific package
- `go test -run <TestName> <package_path>` - Run a single specific test

## Lint/Format Commands
- `gofmt -w .` - Format all Go files
- `go vet ./...` - Vet all packages
- `golangci-lint run` - Run linter (if installed)

## Code Style Guidelines

### Imports
- Group standard library imports first 
- Then third-party imports
- Then project imports
- Sort alphabetically within groups

### Naming Conventions  
- Function/variable names: camelCase
- Struct/interface names: PascalCase
- Exported entities start with capital letter
- Private entities start with lowercase letter

### Error Handling
- Always check and handle errors from functions
- Use fmt.Errorf() for wrapping errors with context
- Return errors early

### Types
- Prefer explicit types over implicit
- Use type definitions for clarity when needed
- Leverage interfaces for abstraction

### Formatting
- Use gofmt formatting as standard
- Max line length ~100 characters
- Indentation: tabs for Go standard (gofmt handles this)

