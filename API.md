# TextAnalyzer API Reference

REST API documentation for the text analysis service.

## Base URL

```
http://localhost:8080
```

## Endpoints

### Health Check

Check if the service is running.

**Request:**
```http
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "time": "2025-01-15T10:30:00Z"
}
```

---

### Analyze Text

Submit text for comprehensive analysis.

**Request:**
```http
POST /api/analyze
Content-Type: application/json

{
  "text": "Your text content here..."
}
```

**Parameters:**
- `text` (string, required) - Text to analyze (1-1000000 characters)

**Response:**
```json
{
  "id": "20250115103000-123456",
  "text": "Your text content here...",
  "metadata": {
    "character_count": 150,
    "word_count": 25,
    "sentence_count": 3,
    "paragraph_count": 1,
    "average_word_length": 5.2,
    "sentiment": "positive",
    "sentiment_score": 0.35,
    "top_words": [{"word": "example", "count": 5}],
    "top_phrases": [{"phrase": "text analysis", "count": 3}],
    "unique_words": 20,
    "key_terms": ["analysis", "metadata"],
    "named_entities": ["John Smith", "New York"],
    "potential_dates": ["2024-01-15"],
    "potential_urls": ["https://example.com"],
    "email_addresses": ["contact@example.com"],
    "readability_score": 65.5,
    "readability_level": "standard",
    "complex_word_count": 5,
    "avg_sentence_length": 8.33,
    "references": [
      {
        "text": "Studies show that 75% of users",
        "type": "statistic",
        "context": "...surrounding context...",
        "confidence": "medium"
      }
    ],
    "tags": ["positive", "medium", "standard"],
    "language": "english",
    "question_count": 2,
    "exclamation_count": 1,
    "capitalized_percent": 12.5,
    "synopsis": "AI-generated 3-4 sentence summary...",
    "cleaned_text": "Text with artifacts removed...",
    "editorial_analysis": "Assessment of bias and motivation...",
    "ai_detection": {
      "likelihood": "unlikely",
      "confidence": "medium",
      "reasoning": "Natural variations in sentence structure...",
      "indicators": ["varied sentence structure", "personal voice"],
      "human_score": 75.5
    }
  },
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Note:** AI-specific fields (`synopsis`, `cleaned_text`, `editorial_analysis`, `ai_detection`) are only present when Ollama is enabled.

**Error Responses:**

`400 Bad Request`:
```json
{
  "error": "Text field is required"
}
```

`408 Request Timeout`:
```json
{
  "error": "Analysis timeout"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"text": "Your text here..."}'

# With timeout for AI processing
curl -m 420 -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d @examples/climate_change.json
```

---

### Get Analysis

Retrieve a specific analysis by ID.

**Request:**
```http
GET /api/analyses/{id}
```

**Response:**
```json
{
  "id": "20250115103000-123456",
  "text": "...",
  "metadata": { ... },
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Error Response (404):**
```json
{
  "error": "analysis not found"
}
```

**Example:**
```bash
curl http://localhost:8080/api/analyses/20250115103000-123456
```

---

### List Analyses

Retrieve all analyses with pagination.

**Request:**
```http
GET /api/analyses?limit=10&offset=0
```

**Query Parameters:**
- `limit` (integer, optional) - Number of results (default: 10, max: 100)
- `offset` (integer, optional) - Number to skip (default: 0)

**Response:**
```json
[
  {
    "id": "20250115103000-123456",
    "text": "...",
    "metadata": { ... },
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  }
]
```

Results are ordered by `created_at` descending (newest first). Returns empty array if no results.

**Example:**
```bash
curl "http://localhost:8080/api/analyses?limit=5&offset=0"
```

---

### Search by Tag

Find analyses with a specific tag.

**Request:**
```http
GET /api/search?tag=positive
```

**Query Parameters:**
- `tag` (string, required) - Tag to search for (case-sensitive)

**Response:**
```json
[
  {
    "id": "20250115103000-123456",
    "text": "...",
    "metadata": {
      "tags": ["positive", "long", "easy"],
      ...
    },
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  }
]
```

**Error Response (400):**
```json
{
  "error": "Tag parameter is required"
}
```

**Common Auto-Generated Tags:**
- **Sentiment**: positive, negative, neutral
- **Length**: short (<100 words), medium (100-500), long (>500)
- **Readability**: very_easy, easy, fairly_easy, standard, fairly_difficult, difficult, very_difficult
- **Content Type**: faq (many questions), web-content (many URLs), research (many references)
- **Topics**: Top 3 key terms from text

**Example:**
```bash
curl "http://localhost:8080/api/search?tag=positive"
```

---

### Search by Reference

Find analyses containing specific reference text.

**Request:**
```http
GET /api/search/reference?reference=climate+change
```

**Query Parameters:**
- `reference` (string, required) - Reference text to search for

**Response:**
```json
[
  {
    "id": "20250115103000-123456",
    "text": "...",
    "metadata": {
      "references": [
        {
          "text": "climate change affects 75% of regions",
          "type": "statistic",
          "context": "...",
          "confidence": "high"
        }
      ],
      ...
    },
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  }
]
```

**Example:**
```bash
curl "http://localhost:8080/api/search/reference?reference=climate"
```

---

### Delete Analysis

Delete a specific analysis.

**Request:**
```http
DELETE /api/analyses/{id}
```

**Response:**
```
204 No Content
```

**Error Response (404):**
```json
{
  "error": "analysis not found"
}
```

This operation also deletes all associated tags (cascade delete).

**Example:**
```bash
curl -X DELETE http://localhost:8080/api/analyses/20250115103000-123456
```

---

## Data Types

### Analysis

```go
type Analysis struct {
    ID        string    `json:"id"`
    Text      string    `json:"text"`
    Metadata  Metadata  `json:"metadata"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### Metadata

```go
type Metadata struct {
    CharacterCount       int           `json:"character_count"`
    WordCount            int           `json:"word_count"`
    SentenceCount        int           `json:"sentence_count"`
    ParagraphCount       int           `json:"paragraph_count"`
    AverageWordLength    float64       `json:"average_word_length"`
    Sentiment            string        `json:"sentiment"`
    SentimentScore       float64       `json:"sentiment_score"`
    TopWords             []WordCount   `json:"top_words"`
    TopPhrases           []PhraseCount `json:"top_phrases"`
    UniqueWords          int           `json:"unique_words"`
    KeyTerms             []string      `json:"key_terms"`
    NamedEntities        []string      `json:"named_entities"`
    PotentialDates       []string      `json:"potential_dates"`
    PotentialURLs        []string      `json:"potential_urls"`
    EmailAddresses       []string      `json:"email_addresses"`
    ReadabilityScore     float64       `json:"readability_score"`
    ReadabilityLevel     string        `json:"readability_level"`
    ComplexWordCount     int           `json:"complex_word_count"`
    AvgSentenceLength    float64       `json:"avg_sentence_length"`
    References           []Reference   `json:"references"`
    Tags                 []string      `json:"tags"`
    Language             string        `json:"language"`
    QuestionCount        int           `json:"question_count"`
    ExclamationCount     int           `json:"exclamation_count"`
    CapitalizedPercent   float64       `json:"capitalized_percent"`
    Synopsis             string        `json:"synopsis,omitempty"`
    CleanedText          string        `json:"cleaned_text,omitempty"`
    EditorialAnalysis    string        `json:"editorial_analysis,omitempty"`
    AIDetection          *AIDetection  `json:"ai_detection,omitempty"`
}
```

### Reference

```go
type Reference struct {
    Text       string `json:"text"`
    Type       string `json:"type"`        // "statistic", "quote", "claim"
    Context    string `json:"context"`
    Confidence string `json:"confidence"`  // "high", "medium", "low"
}
```

### AIDetection

```go
type AIDetection struct {
    Likelihood  string   `json:"likelihood"`  // "likely", "unlikely", "uncertain"
    Confidence  string   `json:"confidence"`  // "high", "medium", "low"
    Reasoning   string   `json:"reasoning"`
    Indicators  []string `json:"indicators"`
    HumanScore  float64  `json:"human_score"` // 0-100
}
```

---

## Error Responses

All errors return JSON with an `error` field:

```json
{
  "error": "descriptive error message"
}
```

**HTTP Status Codes:**
- `200 OK` - Success
- `201 Created` - Analysis created
- `204 No Content` - Successful deletion
- `400 Bad Request` - Invalid request
- `404 Not Found` - Resource not found
- `408 Request Timeout` - Analysis timeout
- `500 Internal Server Error` - Server error

---

## Integration Examples

### JavaScript/TypeScript

```typescript
// Analyze text
async function analyzeText(text: string): Promise<Analysis> {
  const response = await fetch('http://localhost:8080/api/analyze', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text })
  });
  return response.json();
}

// Get analysis by ID
async function getAnalysis(id: string): Promise<Analysis> {
  const response = await fetch(`http://localhost:8080/api/analyses/${id}`);
  return response.json();
}

// Search by tag
async function searchByTag(tag: string): Promise<Analysis[]> {
  const response = await fetch(`http://localhost:8080/api/search?tag=${encodeURIComponent(tag)}`);
  return response.json();
}

// List analyses
async function listAnalyses(limit = 10, offset = 0): Promise<Analysis[]> {
  const response = await fetch(
    `http://localhost:8080/api/analyses?limit=${limit}&offset=${offset}`
  );
  return response.json();
}

// Delete analysis
async function deleteAnalysis(id: string): Promise<void> {
  await fetch(`http://localhost:8080/api/analyses/${id}`, {
    method: 'DELETE'
  });
}
```

### Python

```python
import requests

# Analyze text
def analyze_text(text: str) -> dict:
    response = requests.post(
        'http://localhost:8080/api/analyze',
        json={'text': text},
        timeout=420  # Extended timeout for AI processing
    )
    return response.json()

# Get analysis by ID
def get_analysis(id: str) -> dict:
    response = requests.get(f'http://localhost:8080/api/analyses/{id}')
    return response.json()

# Search by tag
def search_by_tag(tag: str) -> list[dict]:
    response = requests.get(
        'http://localhost:8080/api/search',
        params={'tag': tag}
    )
    return response.json()

# Search by reference
def search_by_reference(reference: str) -> list[dict]:
    response = requests.get(
        'http://localhost:8080/api/search/reference',
        params={'reference': reference}
    )
    return response.json()

# List analyses
def list_analyses(limit: int = 10, offset: int = 0) -> list[dict]:
    response = requests.get(
        'http://localhost:8080/api/analyses',
        params={'limit': limit, 'offset': offset}
    )
    return response.json()

# Delete analysis
def delete_analysis(id: str) -> None:
    requests.delete(f'http://localhost:8080/api/analyses/{id}')
```

### cURL

```bash
# Health check
curl http://localhost:8080/health

# Analyze text
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"text": "Your text here..."}'

# Analyze from file (with extended timeout)
curl -m 420 -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d @examples/climate_change.json

# Get analysis
curl http://localhost:8080/api/analyses/20250115103000-123456

# Search by tag
curl "http://localhost:8080/api/search?tag=positive"

# Search by reference
curl "http://localhost:8080/api/search/reference?reference=climate+change"

# List analyses with pagination
curl "http://localhost:8080/api/analyses?limit=5&offset=0"

# Delete analysis
curl -X DELETE http://localhost:8080/api/analyses/20250115103000-123456
```

---

## Configuration

### Command-Line Flags

```bash
./textanalyzer [flags]
```

- `-port` - Server port (default: 8080)
- `-db` - Database file path (default: textanalyzer.db)
- `-ollama-url` - Ollama API URL (default: http://localhost:11434)
- `-ollama-model` - Ollama model (default: gpt-oss:20b)
- `-use-ollama` - Enable/disable Ollama (default: true)

### Environment Variables

```bash
export PORT=8080
export DB_PATH=textanalyzer.db
export OLLAMA_URL=http://localhost:11434
export OLLAMA_MODEL=gpt-oss:20b
export USE_OLLAMA=true
```

Command-line flags take precedence over environment variables.

---

## Performance

### Timeouts

- Standard analysis: 30 seconds
- AI analysis: up to 7 minutes
- Query operations: 10 seconds

### Database

- Indexes on `created_at` and `tag` fields
- Tag search uses indexed lookups
- Reference search uses LIKE queries

### CORS

CORS is enabled for all origins by default. Modify `internal/api/handler.go` to restrict origins:

```go
c := cors.New(cors.Options{
    AllowedOrigins: []string{"https://yourdomain.com"},
    AllowedMethods: []string{"GET", "POST", "DELETE"},
})
```
