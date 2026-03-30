# AGENTS.md - Go Practice Repository

This repository contains multiple Go demo modules for learning and practice purposes.

## Repository Structure

```
go-practice/
├── demo/              # Basic Go concepts (CLI, structs, slices)
├── demo2/             # HTTP web services
├── demo3/             # File I/O and logging
├── demo-test/         # Unit testing basics
├── demo-test2/        # Advanced testing, HTTP handlers, concurrency
├── demo4-oop/         # Generics, interfaces, OOP patterns
├── microsvc-demo/     # HTTP server with graceful shutdown
└── upload-demo/       # File upload handling
```

## Build/Lint/Test Commands

### Running All Tests
```bash
cd <module-dir> && go test ./...
```

### Running a Single Test
```bash
# Run specific test by name (supports regex patterns)
cd demo-test2 && go test -run "TestGetOne" -v

# Run a specific test file
cd demo-test2 && go test -run "TestSlow" -v

# Run tests with race detector
cd demo-test2 && go test -race ./...

# Run tests with coverage
cd demo-test2 && go test -coverprofile=cover.out ./...
```

### Building Modules
```bash
# Build a specific module
cd demo && go build -o demo.exe .

# Build all modules
for dir in demo demo2 demo3 demo-test demo-test2 demo4-oop microsvc-demo upload-demo; do
  cd $dir && go build . && cd ..
done
```

### Linting and Formatting
```bash
# Format code (required before committing)
gofmt -w .

# Run go vet (static analysis)
go vet ./...

# Download dependencies
go mod tidy
```

### Running Modules
```bash
# Run demo module
cd demo && go run main.go

# Run web server
cd demo2 && go run main.go

# Run microservice demo
cd microsvc-demo && go run main.go
```

## Code Style Guidelines

### Formatting
- Use `gofmt` for automatic formatting (4-space indentation, tab width 8)
- Group imports into standard library, third-party, and local packages
- Use blank lines to separate logical groups of imports
- Keep lines under 120 characters when practical

### Naming Conventions
- **Packages**: lowercase, short, concise (e.g., `payment`, not `paymentpackage`)
- **Types**: PascalCase (e.g., `CreditCard`, `BankAccount`, `User`)
- **Interfaces**: PascalCase, often with "-er" suffix (e.g., `Reader`, `Writer`)
- **Variables**: camelCase for exported, camelCase or _ for unexported
- **Constants**: PascalCase for exported, camelCase for unexported
- **Acronyms**: Keep original casing (e.g., `ID`, not `Id`; `URL`, not `Url`)

### Types and Interfaces
- Use concrete types when possible; only use interfaces when needed for testing or polymorphism
- Define interfaces where they are consumed, not where they are implemented
- Prefer small, focused interfaces (aim for 1-3 methods)
- Use generics for type-safe collections and algorithms (Go 1.18+)

### Error Handling
- Return errors as last return value: `func() (T, error)`
- Use `fmt.Errorf` with `%w` for wrapping errors: `fmt.Errorf("operation: %w", err)`
- Handle errors explicitly with `if err != nil`
- Use `errors.Is` and `errors.As` for error type checking
- Never ignore errors silently (no `_ =` pattern in production code)
- Log errors with context before returning: `log.Printf("failed: %v", err)`

### Concurrency
- Use `sync.RWMutex` for read-heavy workloads (see `demo-test2/user.go:34`)
- Always use `defer mu.Unlock()` after `mu.Lock()` or `mu.RLock()`
- Use `sync.WaitGroup` for managing goroutine groups when needed
- Mark parallelizable tests with `t.Parallel()`

### HTTP Handlers
- Set Content-Type header before writing response body
- Return appropriate HTTP status codes (200, 201, 400, 404, 405, 500)
- Use `json.NewDecoder(r.Body)` and `json.Marshal` for JSON handling
- Validate input and return errors early

### Testing Conventions
- Test file naming: `<package>_test.go` (internal) or `<package>_test.go` (external)
- Test function naming: `Test<FunctionName>_<Scenario>`
- Use `t.Fatal`/`t.Fatalf` for setup failures, `t.Error`/`t.Errorf` for assertion failures
- Use table-driven tests for multiple test cases
- Test exported functions from external test files; test unexported from internal

### Code Organization
- One package per directory
- Keep `main.go` minimal; move logic to separate files
- Group related functions in domain-specific files (e.g., `user.go`, `service.go`)
- Use `_test.go` suffix for test files only

### Generics (Go 1.18+)
- Use type constraints for generic functions: `type Float interface { float32 | float64 }`
- Apply `*CreditCard[T]` and `*BankAccount[T]` patterns for generic types
- Use factory functions (e.g., `NewCreditCard[T Float](...)`) for construction

### Common Patterns
- Constructor-like functions: `New<TypeName>(...) *TypeName`
- Builder patterns optional for complex initialization
- Mutex patterns: `var m sync.RWMutex` with `defer` unlocking
