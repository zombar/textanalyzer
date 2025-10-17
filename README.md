# Text Analyzer

A comprehensive web-based text analysis tool built in Go that extracts extensive JSON metadata from text, including sentiment analysis, word counts, key phrases, and references to verify.

## Features

- **Comprehensive Text Analysis**
  - Word, sentence, and paragraph counts
  - Sentiment analysis with scoring
  - Top words and phrases extraction
  - Named entity recognition
  - Date, URL, and email extraction
  - Readability scoring (Flesch Reading Ease)
  - Reference extraction for fact-checking

- **RESTful API**
  - Upload text for analysis
  - Retrieve analysis results
  - Search by tags
  - List all analyses with pagination
  - Delete analyses

- **Performance**
  - Goroutine-based parallel processing
  - Efficient SQLite storage
  - CORS-enabled for web frontend integration

- **Database**
  - SQLite with migration system
  - Designed for easy PostgreSQL migration
  - No ORM dependencies

## Installation

### Prerequisites

- Go 1.21 or higher
- GCC (for SQLite CGO compilation)

### Setup

```bash
# Clone the repository
git clone <repository-url>
cd textanalyzer

# Install dependencies
go mod download

# Build the server
go build -o textanalyzer ./cmd/server

# Run the server
./textanalyzer
```

### Build Options

```bash
# Build with custom flags
go build -o textanalyzer -ldflags="-s -w" ./cmd/server

# Run with custom port and database
./textanalyzer -port 3000 -db /path/to/database.db
```

## Usage

### Starting the Server

```bash
# Default (port 8080, textanalyzer.db)
./textanalyzer

# Custom configuration
./textanalyzer -port 3000 -db mydata.db
```

### API Endpoints

#### Health Check

```bash
GET /health
```

Response:
```json
{
  "status": "ok",
  "time": "2025-01-15T10:30:00Z"
}
```

#### Analyze Text

```bash
POST /api/analyze
Content-Type: application/json

{
  "text": "Your text to analyze here..."
}
```

Response:
```json
{
  "id": "20250115103000-123456",
  "text": "Your text to analyze here...",
  "metadata": {
    "character_count": 150,
    "word_count": 25,
    "sentence_count": 3,
    "paragraph_count": 1,
    "average_word_length": 5.2,
    "sentiment": "positive",
    "sentiment_score": 0.35,
    "top_words": [
      {"word": "analyze", "count": 3},
      {"word": "text", "count": 2}
    ],
    "top_phrases": [
      {"phrase": "text analysis", "count": 2}
    ],
    "unique_words": 20,
    "key_terms": ["analysis", "metadata", "extraction"],
    "named_entities": ["Python", "JavaScript"],
    "potential_dates": ["2024-01-15"],
    "potential_urls": ["https://example.com"],
    "email_addresses": ["info@example.com"],
    "readability_score": 65.5,
    "readability_level": "standard",
    "complex_word_count": 5,
    "avg_sentence_length": 8.33,
    "references": [
      {
        "text": "45% increase",
        "type": "statistic",
        "context": "...surrounding text...",
        "confidence": "medium"
      }
    ],
    "tags": ["positive", "medium", "standard", "analysis"],
    "language": "english",
    "question_count": 1,
    "exclamation_count": 0,
    "capitalized_percent": 8.5
  },
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

#### Get Analysis by ID

```bash
GET /api/analyses/{id}
```

#### List All Analyses

```bash
GET /api/analyses?limit=10&offset=0
```

Query Parameters:
- `limit`: Number of results (default: 10)
- `offset`: Offset for pagination (default: 0)

#### Search by Tag

```bash
GET /api/search?tag=positive
```

#### Delete Analysis

```bash
DELETE /api/analyses/{id}
```

Returns: `204 No Content`

## API Examples

### Using cURL

```bash
# Analyze text from file
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d @examples/climate_change.json

# Get specific analysis
curl http://localhost:8080/api/analyses/20250115103000-123456

# Search by tag
curl "http://localhost:8080/api/search?tag=positive"

# List analyses with pagination
curl "http://localhost:8080/api/analyses?limit=5&offset=0"

# Delete analysis
curl -X DELETE http://localhost:8080/api/analyses/20250115103000-123456
```

### Using JavaScript (Fetch API)

```javascript
// Analyze text
const response = await fetch('http://localhost:8080/api/analyze', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    text: 'Your text here...'
  })
});
const analysis = await response.json();
console.log(analysis);

// Search by tag
const searchResponse = await fetch('http://localhost:8080/api/search?tag=positive');
const results = await searchResponse.json();
```

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test -cover ./...
```

### Run Tests with Verbose Output

```bash
go test -v ./...
```

### Generate Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Specific Package Tests

```bash
go test ./internal/analyzer
go test ./internal/database
go test ./internal/api
```

### Run Benchmarks

```bash
go test -bench=. ./internal/analyzer
```

## Project Structure

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
├── examples/                 # Example text files
│   ├── climate_change.json
│   ├── product_review.json
│   └── technical_article.json
├── go.mod
├── go.sum
└── README.md
```

## Database Schema

### Analyses Table

```sql
CREATE TABLE analyses (
    id TEXT PRIMARY KEY,
    text TEXT NOT NULL,
    metadata TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Tags Table

```sql
CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    analysis_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);
```

## Migrating to PostgreSQL

The codebase is designed for easy PostgreSQL migration. Here's how:

1. **Update Dependencies**

```bash
go get github.com/lib/pq
```

2. **Update `database/db.go`**

```go
import _ "github.com/lib/pq"

func New(connectionString string) (*DB, error) {
    conn, err := sql.Open("postgres", connectionString)
    // ... rest of the code
}
```

3. **Update Migration SQL**

PostgreSQL uses slightly different syntax:
- Change `AUTOINCREMENT` to `SERIAL`
- Change `DATETIME` to `TIMESTAMP`
- Update any SQLite-specific functions

4. **Connection String**

```bash
./textanalyzer -db "postgres://user:password@localhost/textanalyzer?sslmode=disable"
```

## Metadata Fields Reference

| Field | Type | Description |
|-------|------|-------------|
| `character_count` | int | Total characters including spaces |
| `word_count` | int | Total number of words |
| `sentence_count` | int | Number of sentences |
| `paragraph_count` | int | Number of paragraphs |
| `average_word_length` | float64 | Average word length in characters |
| `sentiment` | string | Overall sentiment: positive, negative, neutral |
| `sentiment_score` | float64 | Sentiment score from -1.0 to 1.0 |
| `top_words` | array | Most frequent words with counts |
| `top_phrases` | array | Most frequent phrases (2-3 words) |
| `unique_words` | int | Number of unique words |
| `key_terms` | array | Important terms based on frequency and length |
| `named_entities` | array | Capitalized words/phrases (potential names) |
| `potential_dates` | array | Extracted dates in various formats |
| `potential_urls` | array | Extracted URLs |
| `email_addresses` | array | Extracted email addresses |
| `readability_score` | float64 | Flesch Reading Ease score (0-100) |
| `readability_level` | string | Reading difficulty level |
| `complex_word_count` | int | Words with 3+ syllables |
| `avg_sentence_length` | float64 | Average words per sentence |
| `references` | array | Potential claims/facts to verify |
| `tags` | array | Auto-generated categorization tags |
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

- All API operations use goroutines for parallel processing
- Default timeout is 30 seconds for analysis, 10 seconds for queries
- Database uses indexes on `created_at` and `tag` fields
- Consider connection pooling for high-traffic scenarios
- For large-scale deployments, migrate to PostgreSQL

## CORS Configuration

The API is CORS-enabled with the following defaults:
- **Allowed Origins**: `*` (all origins)
- **Allowed Methods**: GET, POST, PUT, DELETE, OPTIONS
- **Allowed Headers**: All headers
- **Credentials**: Enabled

To restrict CORS, modify `internal/api/handler.go`:

```go
c := cors.New(cors.Options{
    AllowedOrigins: []string{"https://yourdomain.com"},
    AllowedMethods: []string{"GET", "POST", "DELETE"},
    // ... other options
})
```

## Example Use Cases

1. **Content Analysis**: Analyze blog posts, articles, or documents for readability and sentiment
2. **Fact-Checking**: Extract claims and statistics that need verification
3. **SEO Optimization**: Analyze content for key terms and readability
4. **Social Media Monitoring**: Track sentiment across posts
5. **Academic Research**: Extract metadata from research papers
6. **Quality Control**: Ensure content meets readability standards

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Submit a pull request

## License

[Your chosen license]

## Support

For issues, questions, or contributions, please open an issue on GitHub.
