# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Text Analyzer is a Go-based web service that performs comprehensive text analysis, extracting 20+ metadata fields including sentiment analysis, readability scores, named entity recognition, and reference extraction. The service uses a RESTful API with SQLite storage (designed for easy PostgreSQL migration).

## Build and Development Commands

### Essential Commands

```bash
# Build the server
go build -o textanalyzer ./cmd/server

# Run the server (default: port 8080, textanalyzer.db)
./textanalyzer

# Run with custom configuration
./textanalyzer -port 3000 -db mydata.db

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/analyzer
go test ./internal/database
go test ./internal/api

# Run benchmarks
go test -bench=. ./internal/analyzer

# Format code
go fmt ./...

# Vet code
go vet ./...
```

### Using Make

```bash
make build          # Build binary
make test           # Run all tests
make test-coverage  # Generate coverage report
make run            # Start server
make clean          # Clean artifacts
make fmt            # Format code
make lint           # Run linter
make check          # Run fmt, lint, and test
```

## Architecture

### Package Structure

The codebase follows Go's standard project layout with clear separation of concerns:

- **cmd/server/main.go**: Entry point with graceful shutdown, database initialization, and HTTP server setup
- **internal/analyzer**: Text analysis engine (core business logic)
- **internal/api**: HTTP handlers with CORS support and goroutine-based parallel processing
- **internal/database**: Data persistence layer with migration system
- **internal/models**: Shared data structures

### Key Architectural Patterns

**1. Goroutine-Based Parallel Processing**

All API handlers use goroutines with timeout handling:
```go
resultChan := make(chan Result)
errorChan := make(chan error)

go func() {
    result, err := processWork()
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
case <-time.After(30 * time.Second):
    respondError(w, "Timeout", http.StatusRequestTimeout)
}
```

**2. Database Transaction Pattern**

All database operations that modify multiple tables use transactions with deferred rollback:
```go
tx, err := db.conn.Begin()
if err != nil {
    return fmt.Errorf("failed to begin transaction: %w", err)
}
defer tx.Rollback() // Safe to call even after successful commit

// ... multiple operations ...

return tx.Commit()
```

**3. Migration System**

Database schema changes are managed through a version-based migration system in `internal/database/migrations.go`. Migrations run automatically on server startup.

**4. No ORM Philosophy**

The codebase deliberately avoids ORMs, using direct SQL with parameterized queries for:
- Full control over SQL execution
- Easy optimization and debugging
- Simplified PostgreSQL migration path
- No hidden N+1 query problems

### Analysis Pipeline

The analyzer (`internal/analyzer/analyzer.go`) performs parallel analysis operations:

1. **Basic Statistics**: Word/sentence/paragraph counts
2. **Sentiment Analysis**: Using lexicon-based approach with positive/negative word dictionaries
3. **Frequency Analysis**: Top words (excluding stop words) and top phrases (2-3 word combinations)
4. **Content Extraction**: Named entities (capitalized words), dates, URLs, emails
5. **Readability Scoring**: Flesch Reading Ease calculation with syllable counting
6. **Reference Extraction**: Statistics, quotes, and claims that may need fact-checking
7. **Auto-Tagging**: Sentiment, length, readability level, and topic-based tags

### Stop Words and Lexicon

Common stop words and sentiment lexicons are defined in `internal/analyzer/lexicon.go`. When modifying sentiment analysis:
- Add words to `getPositiveWords()` or `getNegativeWords()`
- Maintain alphabetical ordering for readability
- Consider word frequency in typical English text

## API Endpoints

- `GET /health` - Health check
- `POST /api/analyze` - Submit text for analysis (timeout: 30s)
- `GET /api/analyses` - List all analyses with pagination (limit/offset)
- `GET /api/analyses/{id}` - Retrieve specific analysis
- `DELETE /api/analyses/{id}` - Delete analysis (cascade deletes tags)
- `GET /api/search?tag=X` - Search by tag (timeout: 10s)

CORS is enabled for all origins by default (configured in `internal/api/handler.go`).

## Database

### Schema

Two main tables:
- **analyses**: Stores text, JSON metadata blob, timestamps
- **tags**: Many-to-many relationship with analyses (cascade delete)

Indexes on:
- `created_at` for time-based queries
- `tag` for search operations

### PostgreSQL Migration

To migrate from SQLite to PostgreSQL:

1. Add dependency: `go get github.com/lib/pq`
2. Update `internal/database/db.go`: Change `sql.Open("sqlite3", ...)` to `sql.Open("postgres", connectionString)`
3. Update migrations in `migrations.go`:
   - `AUTOINCREMENT` → `SERIAL`
   - `DATETIME` → `TIMESTAMP`
4. Update connection string format

The SQL queries are already PostgreSQL-compatible (using `?` placeholders that can be converted).

## Testing

### Test Organization

- Unit tests: `*_test.go` files alongside implementation
- Table-driven tests for multiple scenarios
- Test helpers in each package for setup/teardown
- Benchmark tests in analyzer package

### Test Database Cleanup

Database tests use temporary databases with cleanup functions:
```go
func setupTestDB(t *testing.T) (*DB, func()) {
    db, err := New("test_" + t.Name() + ".db")
    if err != nil {
        t.Fatalf("setup failed: %v", err)
    }

    cleanup := func() {
        db.Close()
        os.Remove("test_" + t.Name() + ".db")
    }

    return db, cleanup
}
```

Always defer cleanup to prevent test database accumulation.

## Adding Features

### Adding New Analysis Metrics

1. Add field to `Metadata` struct in `internal/models/models.go`
2. Implement analysis function in `internal/analyzer/analyzer.go`
3. Call from `Analyze()` method
4. Add tests in `internal/analyzer/analyzer_test.go`
5. Update API.md documentation

### Adding New Endpoints

1. Add handler function in `internal/api/handler.go`
2. Register route in `setupRoutes()`
3. Follow goroutine + timeout pattern for consistency
4. Add tests in `internal/api/handler_test.go`
5. Update API.md documentation

### Adding Database Migrations

1. Add migration to `migrations` slice in `internal/database/migrations.go`
2. Increment version number
3. Provide descriptive name
4. Test with clean database: `rm textanalyzer.db && go run ./cmd/server`
5. Never modify existing migrations (immutable after release)

## Module Name

The module is named `github.com/zombar/textanalyzer`. When adding new internal packages, import as:
```go
import "github.com/zombar/textanalyzer/internal/packagename"
```

## Dependencies

External dependencies:
- `github.com/mattn/go-sqlite3` - SQLite driver (requires CGO)
- `github.com/rs/cors` - CORS middleware
- `github.com/ollama/ollama/api` - Ollama LLM client for AI-powered analysis

Standard library is preferred for all other functionality.

## Ollama Integration

The analyzer supports optional Ollama integration for AI-powered features:

### Command Line Flags

```bash
# Enable Ollama (default: enabled)
./textanalyzer -use-ollama=true

# Disable Ollama (fall back to rule-based analysis)
./textanalyzer -use-ollama=false

# Custom Ollama URL (default: http://localhost:11434)
./textanalyzer -ollama-url=http://custom-host:11434

# Custom model (default: gpt-oss:20b)
./textanalyzer -ollama-model=llama2:13b
```

### AI-Powered Features

When Ollama is enabled, the following features use LLM analysis:

1. **Synopsis Generation** - 3-4 sentence summary of the text
2. **Text Cleaning** - Removes artifacts and non-relevant content
3. **Editorial Analysis** - Unbiased assessment of bias, motivation, and slant
4. **Tag Generation** - AI-generated tags (up to 5) instead of rule-based tags
5. **Reference Extraction** - AI-extracted and pruned references with better accuracy
6. **AI Content Detection** - Determines if text was written by AI with confidence scoring and detailed reasoning

The system gracefully falls back to rule-based analysis if Ollama is unavailable or disabled. AI detection is only available when Ollama is enabled.

## Performance Considerations

- Analysis typically processes 100-500 words/ms
- Goroutines handle concurrent requests without blocking
- Database is indexed on common query patterns
- Readability scoring uses simplified syllable counting (fast but approximate)
- For production scale (millions of analyses), migrate to PostgreSQL with connection pooling
