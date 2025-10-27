# TextAnalyzer Service

[![Go Report Card](https://goreportcard.com/badge/github.com/zombar/purpletab)](https://goreportcard.com/report/github.com/zombar/purpletab)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zombar/purpletab)](go.mod)

A comprehensive text analysis service built in Go that extracts extensive metadata from text, including sentiment analysis, readability scoring, named entity recognition, and AI-powered content analysis. Features a sophisticated two-stage pipeline with advanced offline cleaning and async queue-based processing.

## Features

### Core Analysis

- Word, sentence, and paragraph statistics
- Sentiment analysis with scoring
- Top words and phrases extraction
- Named entity recognition
- Date, URL, and email extraction
- Flesch Reading Ease readability scoring
- Reference extraction for fact-checking

### Advanced Two-Stage Pipeline

**Stage 1: Offline Content Extraction (Fast, Rule-Based)**
- **13-Factor Heuristic Algorithm** for intelligent content identification
- Paragraph-level quality scoring (0.0 to 1.0)
- Dynamic thresholding based on content distribution
- Automatic removal of:
  - Navigation elements and breadcrumbs
  - Image captions, photo credits, and attributions
  - Boilerplate (CTAs, newsletter signups, social sharing prompts)
  - Advertisements and related article links
  - Comments sections and metadata
- **70-80% noise reduction** before AI processing
- Fallback mechanism when AI is unavailable

**Stage 2: AI Enhancement (Intelligent, Context-Aware)**
- **Template-guided extraction** using offline analysis as reference
- Compares cleaned text template with original HTML
- Automatic English translation of non-English content
- Enhanced artifact removal and text polishing
- Context-aware synopsis and editorial analysis
- Leverages cleaned template for faster, more accurate AI processing

**HTML Storage & Compression**
- Original HTML/raw text preserved with gzip + base64 encoding
- **70-80% size reduction** for efficient storage
- Enables AI to reference original HTML for better extraction
- Lossless round-trip compression/decompression

### AI-Powered Features (Ollama)

- Text synopsis generation (3-4 sentences)
- Text cleaning (removes artifacts)
- Editorial bias and motivation analysis
- AI-generated tags (up to 5 high-quality tags)
- Enhanced reference extraction
- AI content detection with confidence scoring

### API Features

- RESTful API with CORS support
- **Async queue-based processing** with Asynq/Redis
- PostgreSQL storage with automatic migrations
- Tag-based search
- Reference text search
- Pagination support
- Original HTML storage with compression
- OpenTelemetry distributed tracing
- Prometheus metrics
- Connection pooling and instrumented queries

## Requirements

- Go 1.24 or higher
- PostgreSQL 16 or higher
- Redis (for async queue processing)
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
- `OLLAMA_URL` - Ollama API URL
- `OLLAMA_MODEL` - Ollama model name
- `USE_OLLAMA` - Enable/disable Ollama (true/false/1/0/yes/no)
- `DB_HOST` - PostgreSQL host (default: postgres)
- `DB_PORT` - PostgreSQL port (default: 5432)
- `DB_USER` - Database user (default: docutab)
- `DB_PASSWORD` - Database password
- `DB_NAME` - Database name (default: docutab)

Command-line flags take precedence over environment variables.

### Quick Examples

```bash
# Analyze text (simple)
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"text": "Your text to analyze here..."}'

# Analyze with original HTML for enhanced extraction
# (HTML will be compressed and stored for AI context)
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Extracted article text...",
    "original_html": "<html><body><article>Original HTML...</article></body></html>",
    "images": ["https://example.com/image1.jpg"]
  }'

# Note: API returns 202 Accepted (analysis queued)
# Response includes analysis_id and task_id

# Get analysis by ID (once processing is complete)
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
  - **offline_cleaner.go** - 13-factor heuristic algorithm
- **internal/api** - HTTP handlers with async queue processing
- **internal/database** - Data persistence layer with migrations
- **internal/models** - Shared data structures
- **internal/queue** - Asynq task queue client and workers
- **internal/ollama** - AI integration with Ollama

### Two-Stage Analysis Pipeline

The analyzer uses a sophisticated two-stage approach for optimal quality:

**Stage 1: Offline Analysis (Immediate)**

When text is submitted via POST /api/analyze:
1. API returns 202 Accepted with analysis_id and task_id
2. Background worker starts **offline analysis**:
   - Basic statistics (word/sentence/paragraph counts)
   - Sentiment analysis (lexicon-based)
   - Frequency analysis (top words and phrases)
   - Content extraction (entities, dates, URLs, emails)
   - Readability scoring (Flesch Reading Ease)
   - Reference extraction (statistics, quotes, claims)
   - Auto-tagging (sentiment, length, readability, topics)
   - **13-Factor Offline Cleaning** (removes 70-80% of noise)
3. Results saved to database with `cleaned_text` field
4. Stage 2 (AI enrichment) task queued

**13-Factor Offline Cleaning Algorithm:**

Each paragraph is scored (0.0 to 1.0) using:
1. **Word count** (20-200 words = optimal)
2. **Link density** (>10% = navigation/menu)
3. **Stopword ratio** (35-65% = natural prose)
4. **Named entities** (≥2 = article content)
5. **Average word length** (4-6 chars = balanced)
6. **Image markers** (Photo by, Getty Images = caption)
7. **Boilerplate patterns** (Click here, Subscribe = UI)
8. **Capitalization ratio** (>50% = headers)
9. **Punctuation overload** (>20% = spam/lists)
10. **List structure** (short bullets = navigation)
11. **Social media patterns** (Share on = prompt)
12. **Date/timestamp metadata** (Posted on = metadata)
13. **Author bylines** (By [Name] = metadata)

Dynamic threshold calculated from median score (bounded 0.3-0.6).

**Stage 2: AI Enhancement (Async, Queued)**

Executed by background worker when Ollama is available:
1. Decompress original HTML (if provided)
2. Use cleaned text from Stage 1 as **extraction template**
3. Instruct Ollama to:
   - Compare template with original HTML for context
   - Extract clean article text (removing image attributions)
   - Translate to English if needed
   - Generate synopsis, editorial analysis, tags
   - Detect AI-generated content
   - Score content quality
4. Update database with AI-enhanced metadata
5. Preserve both cleaned_text and AI-enhanced results

**Benefits of Two-Stage Approach:**
- Fast initial results (Stage 1 completes in <1 second)
- Works offline when Ollama unavailable
- 70-80% smaller text sent to AI (faster, cheaper)
- Template-guided extraction improves AI accuracy
- Graceful degradation if AI fails

### Database

PostgreSQL database with two tables:

- **analyses** - Stores text, JSON metadata, timestamps
- **tags** - Many-to-many relationship for tag search
- **text_references** - Stores references for fact-checking

Indexes on `created_at`, `tag`, and reference fields for performance. The shared database package (`pkg/database`) provides connection pooling and OpenTelemetry instrumentation.

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
│       └── main.go                      # Application entry point
├── internal/
│   ├── analyzer/
│   │   ├── analyzer.go                  # Text analysis logic
│   │   ├── analyzer_test.go             # Analysis tests
│   │   ├── offline_cleaner.go           # 13-factor heuristic algorithm
│   │   ├── offline_cleaner_test.go      # Offline cleaning tests
│   │   └── lexicon.go                   # Sentiment and stop words
│   ├── api/
│   │   ├── handler.go                   # HTTP handlers (async)
│   │   ├── handler_test.go              # API tests
│   │   └── tracing_test.go              # OpenTelemetry tracing tests
│   ├── database/
│   │   ├── db.go                        # Database connection
│   │   ├── migrations.go                # Schema migrations (v6: original_html)
│   │   ├── queries.go                   # Database queries
│   │   └── queries_test.go              # Database tests
│   ├── queue/
│   │   ├── client.go                    # Asynq queue client
│   │   ├── tasks.go                     # Queue task handlers
│   │   └── tasks_test.go                # Queue tests (compression, payloads)
│   ├── ollama/
│   │   └── client.go                    # AI integration
│   └── models/
│       └── models.go                    # Data structures
├── examples/                            # Example JSON files
├── OFFLINE_CLEANING_ALGORITHM.md        # Algorithm documentation
├── README.md                            # This file
└── API.md                               # API reference
```

## Offline Cleaning Algorithm

The 13-factor heuristic algorithm intelligently identifies article content vs. noise. For complete technical details, see [OFFLINE_CLEANING_ALGORITHM.md](OFFLINE_CLEANING_ALGORITHM.md).

**Key Features:**
- Paragraph-level scoring (13 quality factors)
- Dynamic thresholding based on content distribution
- 70-80% noise reduction typical
- Preserves article order and paragraph structure
- Detailed logging of removal reasons

**Example Output:**
```
Analyzing 45 paragraphs...
Removed paragraph 3 (score=0.15): too_short, high_link_density
Removed paragraph 7 (score=0.22): image_attribution
Removed paragraph 12 (score=0.18): boilerplate_pattern
Paragraph quality threshold: 0.52
Offline cleaning complete: kept 28 paragraphs, removed 17
Offline cleaning: 1247 words → 892 words (28.5% reduction)
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

- **Async queue processing** for non-blocking API responses (202 Accepted)
- **Two-stage pipeline** provides fast initial results (<1 second)
- **70-80% text reduction** before AI processing (faster, cheaper)
- **HTML compression** achieves 70-80% storage savings (gzip + base64)
- API operations use goroutines for parallel processing
- Default timeout: 30 seconds for analysis, 10 seconds for queries
- AI analysis has extended timeout (up to 7 minutes)
- Database indexes on `created_at` and `tag` fields
- Connection pooling recommended for high-traffic scenarios
- Redis used for distributed task queue (Asynq)

## API Documentation

See [API.md](API.md) for complete API reference including:
- Endpoint specifications
- Request/response formats
- Error handling
- Code examples
- Data type definitions

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.
