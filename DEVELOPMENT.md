# Development Guide

Guide for developers working on the Text Analyzer project.

## Development Setup

### Prerequisites

- Go 1.21 or higher
- GCC (for CGO compilation)
- Git
- Your favorite IDE (VS Code, GoLand, etc.)

### Initial Setup

```bash
# Clone the repository
git clone <repository-url>
cd textanalyzer

# Install dependencies
go mod download

# Verify everything compiles
go build ./...

# Run tests
go test ./...
```

### IDE Configuration

#### VS Code

Recommended extensions:
- Go (golang.go)
- Go Test Explorer
- REST Client

`.vscode/settings.json`:
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.testOnSave": true,
  "go.coverOnSave": true
}
```

#### GoLand

Configure:
- Go Modules: Enabled
- Gofmt on save: Enabled
- Run tests automatically

---

## Project Structure

```
textanalyzer/
├── cmd/                      # Application entry points
│   └── server/
│       └── main.go          # Server startup
├── internal/                # Private application code
│   ├── analyzer/            # Text analysis engine
│   │   ├── analyzer.go      # Core analysis logic
│   │   ├── analyzer_test.go # Tests
│   │   └── lexicon.go       # Word lists
│   ├── api/                 # HTTP API layer
│   │   ├── handler.go       # Request handlers
│   │   └── handler_test.go  # API tests
│   ├── database/            # Data persistence
│   │   ├── db.go           # Connection management
│   │   ├── migrations.go   # Schema migrations
│   │   ├── queries.go      # Database operations
│   │   └── queries_test.go # Database tests
│   └── models/              # Data structures
│       └── models.go        # Shared models
├── examples/                # Example texts
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
├── README.md               # Main documentation
└── API.md                  # API reference
```

---

## Code Organization Principles

### Package Guidelines

1. **internal/**: Private packages not importable by other projects
2. **cmd/**: Application entry points
3. Each package has a clear, single responsibility
4. Test files alongside implementation files

### Naming Conventions

- **Files**: lowercase with underscores (`analyzer_test.go`)
- **Packages**: single word, lowercase (`analyzer`, not `text_analyzer`)
- **Interfaces**: noun or adjective (`Analyzer`, `Readable`)
- **Functions**: camelCase, exported start with uppercase (`AnalyzeText`)
- **Constants**: CamelCase or SCREAMING_SNAKE_CASE based on scope

### Code Style

Follow standard Go conventions:
```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Vet code
go vet ./...
```

---

## Testing Strategy

### Test Organization

Each package has its own test file:
- Unit tests: `*_test.go` files
- Integration tests: `*_integration_test.go` files
- Benchmarks: Included in `*_test.go` files

### Writing Tests

```go
func TestFunctionName(t *testing.T) {
    // Arrange
    input := "test data"
    expected := "expected result"
    
    // Act
    result := FunctionName(input)
    
    // Assert
    if result != expected {
        t.Errorf("expected %s, got %s", expected, result)
    }
}
```

### Table-Driven Tests

```go
func TestMultipleCases(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected int
    }{
        {"empty string", "", 0},
        {"single word", "hello", 1},
        {"multiple words", "hello world", 2},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CountWords(tt.input)
            if result != tt.expected {
                t.Errorf("expected %d, got %d", tt.expected, result)
            }
        })
    }
}
```

### Test Helpers

```go
func setupTestDB(t *testing.T) (*DB, func()) {
    db, err := New("test.db")
    if err != nil {
        t.Fatalf("setup failed: %v", err)
    }
    
    cleanup := func() {
        db.Close()
        os.Remove("test.db")
    }
    
    return db, cleanup
}

func TestWithDatabase(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    // Use db...
}
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/analyzer

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Run specific test
go test -run TestAnalyze ./internal/analyzer

# Benchmarks
go test -bench=. ./internal/analyzer

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## Database Development

### Adding Migrations

1. Add migration to `internal/database/migrations.go`:

```go
{
    Version: 4,
    Name:    "add_user_column",
    SQL: `
        ALTER TABLE analyses ADD COLUMN user_id TEXT;
        CREATE INDEX idx_analyses_user_id ON analyses(user_id);
    `,
}
```

2. Test migration:

```bash
go test ./internal/database -v
```

3. The migration runs automatically on server start

### Database Best Practices

- Always use parameterized queries (prevent SQL injection)
- Use transactions for multi-statement operations
- Add indexes for frequently queried columns
- Keep migrations immutable (never modify existing migrations)
- Test migrations with realistic data volumes

### Query Examples

```go
// Good: Parameterized query
rows, err := db.Query("SELECT * FROM analyses WHERE id = ?", id)

// Bad: String concatenation (SQL injection risk!)
rows, err := db.Query("SELECT * FROM analyses WHERE id = '" + id + "'")

// Transaction example
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback()

// Multiple operations...
_, err = tx.Exec("INSERT INTO...")
if err != nil {
    return err
}

return tx.Commit()
```

---

## API Development

### Adding New Endpoints

1. Define handler function in `internal/api/handler.go`:

```go
func (h *Handler) handleNewFeature(w http.ResponseWriter, r *http.Request) {
    // Validate method
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Parse request
    var req struct {
        Field string `json:"field"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        respondError(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    // Process in goroutine
    resultChan := make(chan Result)
    errorChan := make(chan error)
    
    go func() {
        result, err := h.processFeature(req.Field)
        if err != nil {
            errorChan <- err
            return
        }
        resultChan <- result
    }()
    
    select {
    case result := <-resultChan:
        respondJSON(w, result, http.StatusOK)
    case err := <-errorChan:
        respondError(w, err.Error(), http.StatusInternalServerError)
    case <-time.After(10 * time.Second):
        respondError(w, "Timeout", http.StatusRequestTimeout)
    }
}
```

2. Register route in `setupRoutes()`:

```go
func (h *Handler) setupRoutes() {
    // ... existing routes
    h.mux.HandleFunc("/api/newfeature", h.handleNewFeature)
}
```

3. Add tests in `internal/api/handler_test.go`:

```go
func TestNewFeatureEndpoint(t *testing.T) {
    handler, _, cleanup := setupTestHandler(t)
    defer cleanup()
    
    reqBody := map[string]string{"field": "value"}
    body, _ := json.Marshal(reqBody)
    
    req := httptest.NewRequest(http.MethodPost, "/api/newfeature", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    w := httptest.NewRecorder()
    
    handler.mux.ServeHTTP(w, req)
    
    if w.Code != http.StatusOK {
        t.Errorf("expected 200, got %d", w.Code)
    }
}
```

---

## Analyzer Development

### Adding Analysis Features

1. Add field to `Metadata` struct in `internal/models/models.go`:

```go
type Metadata struct {
    // ... existing fields
    NewFeature int `json:"new_feature"`
}
```

2. Implement analysis in `internal/analyzer/analyzer.go`:

```go
func (a *Analyzer) Analyze(text string) models.Metadata {
    metadata := models.Metadata{}
    
    // ... existing analysis
    metadata.NewFeature = a.analyzeNewFeature(text)
    
    return metadata
}

func (a *Analyzer) analyzeNewFeature(text string) int {
    // Implementation
    return 0
}
```

3. Add tests:

```go
func TestAnalyzeNewFeature(t *testing.T) {
    a := New()
    text := "test text"
    
    result := a.analyzeNewFeature(text)
    
    if result == 0 {
        t.Error("expected non-zero result")
    }
}
```

---

## Performance Optimization

### Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./internal/analyzer
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=. ./internal/analyzer
go tool pprof mem.prof

# Web interface
go tool pprof -http=:8081 cpu.prof
```

### Optimization Guidelines

1. **Measure first**: Always profile before optimizing
2. **Goroutines**: Use for I/O-bound operations, not necessarily CPU-bound
3. **String operations**: Use `strings.Builder` for concatenation
4. **Regular expressions**: Compile once, use many times
5. **Database**: Use prepared statements, batch operations when possible

### Example Optimization

```go
// Slow: Creating many strings
func buildString(words []string) string {
    result := ""
    for _, word := range words {
        result += word + " "
    }
    return result
}

// Fast: Using strings.Builder
func buildStringFast(words []string) string {
    var builder strings.Builder
    for _, word := range words {
        builder.WriteString(word)
        builder.WriteString(" ")
    }
    return builder.String()
}
```

---

## Debugging

### Logging

```go
import "log"

// Add debug logging
log.Printf("Debug: variable value = %v", variable)

// Error logging
log.Printf("Error occurred: %v", err)
```

### Delve Debugger

```bash
# Install
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug tests
dlv test ./internal/analyzer

# Debug running server
dlv exec ./textanalyzer
```

---

## Common Tasks

### Adding a New Sentiment Word

Edit `internal/analyzer/lexicon.go`:

```go
func getPositiveWords() map[string]bool {
    words := []string{
        // ... existing words
        "newword",
    }
    // ...
}
```

### Changing Database Schema

Add new migration to `internal/database/migrations.go`:

```go
{
    Version: X, // Next version number
    Name:    "descriptive_name",
    SQL:     `YOUR SQL HERE`,
}
```

### Updating API Response Format

1. Update `internal/models/models.go`
2. Update documentation in `API.md`
3. Update tests
4. Consider API versioning for breaking changes

---

## CI/CD

### GitHub Actions Example

`.github/workflows/test.yml`:

```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - run: go test -v -cover ./...
```

---

## Troubleshooting

### CGO Errors

```bash
# Install GCC (Linux)
sudo apt-get install gcc

# Install GCC (macOS)
xcode-select --install
```

### Module Issues

```bash
# Clean module cache
go clean -modcache

# Verify dependencies
go mod verify

# Tidy dependencies
go mod tidy
```

### Test Database Conflicts

```bash
# Clean up test databases
rm test_*.db

# Or in tests, ensure cleanup functions run:
defer cleanup()
```

---

## Release Process

1. Update version in code
2. Run full test suite: `go test ./...`
3. Build for multiple platforms:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o textanalyzer-linux-amd64 ./cmd/server

# macOS
GOOS=darwin GOARCH=amd64 go build -o textanalyzer-darwin-amd64 ./cmd/server

# Windows
GOOS=windows GOARCH=amd64 go build -o textanalyzer-windows-amd64.exe ./cmd/server
```

4. Tag release: `git tag v1.0.0`
5. Push tag: `git push origin v1.0.0`
6. Create GitHub release with binaries

---

## Resources

- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Testing Best Practices](https://github.com/golang/go/wiki/TestComments)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
