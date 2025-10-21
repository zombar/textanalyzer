# TextAnalyzer Service

[![Go Report Card](https://goreportcard.com/badge/github.com/zombar/purpletab)](https://goreportcard.com/report/github.com/zombar/purpletab)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zombar/purpletab)](go.mod)

A comprehensive text analysis service built in Go that extracts extensive metadata from text, including sentiment analysis, readability scoring, named entity recognition, and AI-powered content analysis.

## Features

### Core Analysis

- Word, sentence, and paragraph statistics
- Sentiment analysis with scoring
- Top words and phrases extraction
- Named entity recognition
- Date, URL, and email extraction
- Flesch Reading Ease readability scoring
- Reference extraction for fact-checking

### AI-Powered Features (Ollama)

- Text synopsis generation (3-4 sentences)
- Text cleaning (removes artifacts)
- Editorial bias and motivation analysis
- AI-generated tags (up to 5 high-quality tags)
- Enhanced reference extraction
- AI content detection with confidence scoring

### API Features

- RESTful API with CORS support
- SQLite storage with migrations
- Tag-based search
- Reference text search
- Pagination support
- Designed for PostgreSQL migration

## Requirements

- Go 1.24 or higher
- GCC (for SQLite CGO compilation)
- [Ollama](https://ollama.ai) (optional, for AI features)

### Ollama Setup

For AI-powered analysis (optional but recommended):

```bash
# Install Ollama from https://ollama.ai

# Pull the default model
ollama pull gpt-oss:20b
```

## Installation

```bash
# Clone and navigate to directory
cd textanalyzer

# Install dependencies
go mod download

# Build the server
go build -o textanalyzer ./cmd/server

# Run the server
./textanalyzer
```

## Usage

### Starting the Server

```bash
# Default configuration (port 8080, textanalyzer.db, Ollama enabled)
./textanalyzer

# Custom port and database
./textanalyzer -port 3000 -db mydata.db

# Disable Ollama (rule-based analysis only)
./textanalyzer -use-ollama=false

# Custom Ollama configuration
./textanalyzer -ollama-url=http://localhost:11434 -ollama-model=gpt-oss:20b
```

### Configuration Options

**Command-Line Flags:**
- `-port` - Server port (default: 8080)
- `-db` - Database file path (default: textanalyzer.db)
- `-ollama-url` - Ollama API URL (default: http://localhost:11434)
- `-ollama-model` - Ollama model name (default: gpt-oss:20b)
- `-use-ollama` - Enable/disable Ollama (default: true)

**Environment Variables:**
- `PORT` - Server port
- `DB_PATH` - Database file path
- `OLLAMA_URL` - Ollama API URL
- `OLLAMA_MODEL` - Ollama model name
- `USE_OLLAMA` - Enable/disable Ollama (true/false/1/0/yes/no)

Command-line flags take precedence over environment variables.

### Quick Examples

```bash
# Analyze text
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"text": "Your text to analyze here..."}'

# Get analysis by ID
curl http://localhost:8080/api/analyses/20250115103000-123456

# Search by tag
curl "http://localhost:8080/api/search?tag=positive"

# Search by reference text
curl "http://localhost:8080/api/search/reference?reference=climate"

# List all analyses
curl "http://localhost:8080/api/analyses?limit=10&offset=0"
```

## Output Format

The analyzer returns comprehensive metadata:

```json
{
  "id": "20250115103000-123456",
  "text": "Original text...",
  "metadata": {
    "character_count": 150,
    "word_count": 25,
    "sentence_count": 3,
    "sentiment": "positive",
    "sentiment_score": 0.35,
    "top_words": [{"word": "analyze", "count": 3}],
    "readability_score": 65.5,
    "readability_level": "standard",
    "references": [
      {
        "text": "45% increase",
        "type": "statistic",
        "context": "...surrounding text...",
        "confidence": "medium"
      }
    ],
    "tags": ["positive", "standard", "analysis"],
    "synopsis": "AI-generated summary...",
    "cleaned_text": "Text with artifacts removed...",
    "editorial_analysis": "Assessment of bias...",
    "ai_detection": {
      "likelihood": "unlikely",
      "confidence": "medium",
      "reasoning": "Analysis explanation...",
      "indicators": ["varied sentence structure"],
      "human_score": 75.5
    }
  },
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Note:** AI-specific fields (`synopsis`, `cleaned_text`, `editorial_analysis`, `ai_detection`) are only populated when Ollama is enabled.

## Architecture

### Package Structure

- **cmd/server** - Application entry point
- **internal/analyzer** - Text analysis logic and algorithms
- **internal/api** - HTTP handlers with goroutine-based parallel processing
- **internal/database** - Data persistence layer with migrations
- **internal/models** - Shared data structures

### Analysis Pipeline

The analyzer performs parallel analysis operations:

1. Basic statistics (word/sentence/paragraph counts)
2. Sentiment analysis (lexicon-based)
3. Frequency analysis (top words and phrases)
4. Content extraction (entities, dates, URLs, emails)
5. Readability scoring (Flesch Reading Ease)
6. Reference extraction (statistics, quotes, claims)
7. Auto-tagging (sentiment, length, readability, topics)
8. AI analysis (if Ollama enabled)

### Database

SQLite database with two tables:

- **analyses** - Stores text, JSON metadata, timestamps
- **tags** - Many-to-many relationship for tag search

Indexes on `created_at` and `tag` fields for performance.

## Development

### Make Commands

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

### Running Tests

```bash
# Run all tests
make test

# Generate coverage report
make test-coverage

# Using Go directly
go test ./...
go test -v ./...
go test -cover ./...

# Test specific packages
go test ./internal/analyzer
go test ./internal/database
go test ./internal/api

# Run benchmarks
go test -bench=. ./internal/analyzer
```

### Project Structure

```
textanalyzer/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
├── internal/
│   ├── analyzer/
│   │   ├── analyzer.go       # Text analysis logic
│   │   ├── analyzer_test.go  # Analysis tests
│   │   └── lexicon.go        # Sentiment and stop words
│   ├── api/
│   │   ├── handler.go        # HTTP handlers
│   │   └── handler_test.go   # API tests
│   ├── database/
│   │   ├── db.go             # Database connection
│   │   ├── migrations.go     # Schema migrations
│   │   ├── queries.go        # Database queries
│   │   └── queries_test.go   # Database tests
│   └── models/
│       └── models.go         # Data structures
├── examples/                 # Example JSON files
├── README.md                 # This file
└── API.md                    # API reference
```

## Switching to PostgreSQL

The codebase is designed for easy PostgreSQL migration:

1. Add PostgreSQL driver:
   ```bash
   go get github.com/lib/pq
   ```

2. Update `internal/database/db.go`:
   ```go
   import _ "github.com/lib/pq"

   func New(connectionString string) (*DB, error) {
       conn, err := sql.Open("postgres", connectionString)
       // ... rest remains the same
   }
   ```

3. Update SQL syntax in `migrations.go`:
   - `AUTOINCREMENT` → `SERIAL`
   - `DATETIME` → `TIMESTAMP`

4. Use PostgreSQL connection string:
   ```bash
   ./textanalyzer -db "postgres://user:password@localhost/textanalyzer?sslmode=disable"
   ```

## Metadata Fields Reference

| Field | Type | Description |
|-------|------|-------------|
| `character_count` | int | Total characters including spaces |
| `word_count` | int | Total words |
| `sentence_count` | int | Number of sentences |
| `paragraph_count` | int | Number of paragraphs |
| `average_word_length` | float64 | Average word length |
| `sentiment` | string | positive, negative, or neutral |
| `sentiment_score` | float64 | Score from -1.0 to 1.0 |
| `top_words` | array | Most frequent words with counts |
| `top_phrases` | array | Most frequent 2-3 word phrases |
| `unique_words` | int | Number of unique words |
| `key_terms` | array | Important terms by frequency |
| `named_entities` | array | Capitalized words/phrases |
| `potential_dates` | array | Extracted dates |
| `potential_urls` | array | Extracted URLs |
| `email_addresses` | array | Extracted email addresses |
| `readability_score` | float64 | Flesch Reading Ease (0-100) |
| `readability_level` | string | Reading difficulty level |
| `complex_word_count` | int | Words with 3+ syllables |
| `avg_sentence_length` | float64 | Average words per sentence |
| `references` | array | Claims/facts to verify |
| `tags` | array | Auto-generated tags |
| `language` | string | Detected language |
| `question_count` | int | Number of questions |
| `exclamation_count` | int | Number of exclamations |
| `capitalized_percent` | float64 | Percentage of capitalized words |

## Readability Levels

| Score | Level | Description |
|-------|-------|-------------|
| 90-100 | very_easy | 5th grade |
| 80-89 | easy | 6th grade |
| 70-79 | fairly_easy | 7th grade |
| 60-69 | standard | 8th-9th grade |
| 50-59 | fairly_difficult | 10th-12th grade |
| 30-49 | difficult | College |
| 0-29 | very_difficult | College graduate |

## Performance Considerations

- API operations use goroutines for parallel processing
- Default timeout: 30 seconds for analysis, 10 seconds for queries
- AI analysis has extended timeout (up to 7 minutes)
- Database indexes on `created_at` and `tag` fields
- Connection pooling recommended for high-traffic scenarios

## API Documentation

See [API.md](API.md) for complete API reference including:
- Endpoint specifications
- Request/response formats
- Error handling
- Code examples
- Data type definitions

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.
