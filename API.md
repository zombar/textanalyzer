# API Reference

Complete API reference for the Text Analyzer service.

## Base URL

```
http://localhost:8080
```

## Authentication

Currently, the API does not require authentication. For production use, consider implementing JWT tokens or API keys.

---

## Endpoints

### Health Check

Check if the service is running.

**Endpoint:** `GET /health`

**Response:** `200 OK`

```json
{
  "status": "ok",
  "time": "2025-01-15T10:30:00Z"
}
```

---

### Analyze Text

Submit text for comprehensive analysis.

**Endpoint:** `POST /api/analyze`

**Headers:**
```
Content-Type: application/json
```

**Request Body:**

```json
{
  "text": "Your text content here..."
}
```

**Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| text | string | Yes | The text to analyze (1-1000000 characters) |

**Response:** `201 Created`

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
    "top_words": [
      {
        "word": "example",
        "count": 5
      }
    ],
    "top_phrases": [
      {
        "phrase": "text analysis",
        "count": 3
      }
    ],
    "unique_words": 20,
    "key_terms": ["analysis", "metadata"],
    "named_entities": ["John Smith", "New York"],
    "potential_dates": ["2024-01-15", "March 25, 2024"],
    "potential_urls": ["https://example.com"],
    "email_addresses": ["contact@example.com"],
    "readability_score": 65.5,
    "readability_level": "standard",
    "complex_word_count": 5,
    "avg_sentence_length": 8.33,
    "references": [
      {
        "text": "Studies show that 75% of users prefer",
        "type": "statistic",
        "context": "...surrounding context...",
        "confidence": "medium"
      }
    ],
    "tags": ["positive", "medium", "standard"],
    "language": "english",
    "question_count": 2,
    "exclamation_count": 1,
    "capitalized_percent": 12.5
  },
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Error Responses:**

`400 Bad Request` - Invalid request body or empty text
```json
{
  "error": "Text field is required"
}
```

`408 Request Timeout` - Analysis took too long
```json
{
  "error": "Analysis timeout"
}
```

`500 Internal Server Error` - Server error
```json
{
  "error": "Failed to save analysis"
}
```

---

### Get Analysis

Retrieve a specific analysis by ID.

**Endpoint:** `GET /api/analyses/{id}`

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | The unique analysis ID |

**Response:** `200 OK`

```json
{
  "id": "20250115103000-123456",
  "text": "...",
  "metadata": { ... },
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

**Error Responses:**

`404 Not Found` - Analysis doesn't exist
```json
{
  "error": "analysis not found"
}
```

---

### List Analyses

Retrieve all analyses with pagination.

**Endpoint:** `GET /api/analyses`

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | integer | 10 | Number of results to return (1-100) |
| offset | integer | 0 | Number of results to skip |

**Example:** `GET /api/analyses?limit=20&offset=40`

**Response:** `200 OK`

```json
[
  {
    "id": "20250115103000-123456",
    "text": "...",
    "metadata": { ... },
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  },
  {
    "id": "20250115102500-789012",
    "text": "...",
    "metadata": { ... },
    "created_at": "2025-01-15T10:25:00Z",
    "updated_at": "2025-01-15T10:25:00Z"
  }
]
```

**Notes:**
- Results are ordered by `created_at` descending (newest first)
- Returns empty array `[]` if no results

---

### Search by Tag

Find analyses that have a specific tag.

**Endpoint:** `GET /api/search`

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| tag | string | Yes | Tag to search for (case-sensitive) |

**Example:** `GET /api/search?tag=positive`

**Response:** `200 OK`

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

**Error Responses:**

`400 Bad Request` - Missing tag parameter
```json
{
  "error": "Tag parameter is required"
}
```

**Common Tags:**

Auto-generated tags include:
- **Sentiment**: `positive`, `negative`, `neutral`
- **Length**: `short` (<100 words), `medium` (100-500 words), `long` (>500 words)
- **Readability**: `very_easy`, `easy`, `fairly_easy`, `standard`, `fairly_difficult`, `difficult`, `very_difficult`
- **Content Type**: `faq` (many questions), `web-content` (many URLs), `research` (many references)
- **Topics**: Top 3 key terms from the text

---

### Delete Analysis

Delete a specific analysis.

**Endpoint:** `DELETE /api/analyses/{id}`

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | string | The unique analysis ID |

**Response:** `204 No Content`

No response body

**Error Responses:**

`404 Not Found` - Analysis doesn't exist
```json
{
  "error": "analysis not found"
}
```

**Note:** This operation also deletes all associated tags (cascade delete).

---

## Data Types

### Analysis Object

```typescript
{
  id: string,              // Unique identifier
  text: string,            // Original text
  metadata: Metadata,      // Analysis results
  created_at: string,      // ISO 8601 timestamp
  updated_at: string       // ISO 8601 timestamp
}
```

### Metadata Object

```typescript
{
  character_count: number,
  word_count: number,
  sentence_count: number,
  paragraph_count: number,
  average_word_length: number,
  sentiment: "positive" | "negative" | "neutral",
  sentiment_score: number,  // -1.0 to 1.0
  top_words: WordFrequency[],
  top_phrases: PhraseInfo[],
  unique_words: number,
  key_terms: string[],
  named_entities: string[],
  potential_dates: string[],
  potential_urls: string[],
  email_addresses: string[],
  readability_score: number,  // 0-100
  readability_level: string,
  complex_word_count: number,
  avg_sentence_length: number,
  references: Reference[],
  tags: string[],
  language: string,
  question_count: number,
  exclamation_count: number,
  capitalized_percent: number
}
```

### WordFrequency Object

```typescript
{
  word: string,
  count: number
}
```

### PhraseInfo Object

```typescript
{
  phrase: string,
  count: number
}
```

### Reference Object

```typescript
{
  text: string,                              // The claim or statistic
  type: "statistic" | "quote" | "claim" | "citation",
  context: string,                           // Surrounding text
  confidence: "high" | "medium" | "low"      // Confidence level
}
```

---

## Rate Limiting

Currently, no rate limiting is implemented. For production use, consider implementing:
- Per-IP rate limiting
- API key-based quotas
- Throttling for expensive operations

---

## Error Handling

All errors follow this format:

```json
{
  "error": "Human-readable error message"
}
```

### HTTP Status Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created successfully |
| 204 | Success with no content |
| 400 | Bad request (invalid input) |
| 404 | Resource not found |
| 405 | Method not allowed |
| 408 | Request timeout |
| 500 | Internal server error |

---

## CORS

The API supports CORS with the following configuration:
- **Allowed Origins**: All (`*`)
- **Allowed Methods**: GET, POST, PUT, DELETE, OPTIONS
- **Allowed Headers**: All
- **Credentials**: Supported

---

## Examples

### cURL Examples

```bash
# Analyze text
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"text": "This is a test sentence."}'

# Get analysis
curl http://localhost:8080/api/analyses/20250115103000-123456

# List with pagination
curl "http://localhost:8080/api/analyses?limit=5&offset=10"

# Search by tag
curl "http://localhost:8080/api/search?tag=positive"

# Delete
curl -X DELETE http://localhost:8080/api/analyses/20250115103000-123456
```

### JavaScript Examples

```javascript
// Analyze text
async function analyzeText(text) {
  const response = await fetch('http://localhost:8080/api/analyze', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ text })
  });
  return await response.json();
}

// Get analysis
async function getAnalysis(id) {
  const response = await fetch(`http://localhost:8080/api/analyses/${id}`);
  return await response.json();
}

// Search by tag
async function searchByTag(tag) {
  const response = await fetch(
    `http://localhost:8080/api/search?tag=${encodeURIComponent(tag)}`
  );
  return await response.json();
}

// List with pagination
async function listAnalyses(limit = 10, offset = 0) {
  const response = await fetch(
    `http://localhost:8080/api/analyses?limit=${limit}&offset=${offset}`
  );
  return await response.json();
}

// Delete analysis
async function deleteAnalysis(id) {
  const response = await fetch(
    `http://localhost:8080/api/analyses/${id}`,
    { method: 'DELETE' }
  );
  return response.status === 204;
}
```

### Python Examples

```python
import requests

BASE_URL = "http://localhost:8080"

# Analyze text
def analyze_text(text):
    response = requests.post(
        f"{BASE_URL}/api/analyze",
        json={"text": text}
    )
    return response.json()

# Get analysis
def get_analysis(analysis_id):
    response = requests.get(f"{BASE_URL}/api/analyses/{analysis_id}")
    return response.json()

# Search by tag
def search_by_tag(tag):
    response = requests.get(
        f"{BASE_URL}/api/search",
        params={"tag": tag}
    )
    return response.json()

# List analyses
def list_analyses(limit=10, offset=0):
    response = requests.get(
        f"{BASE_URL}/api/analyses",
        params={"limit": limit, "offset": offset}
    )
    return response.json()

# Delete analysis
def delete_analysis(analysis_id):
    response = requests.delete(f"{BASE_URL}/api/analyses/{analysis_id}")
    return response.status_code == 204
```

---

## Best Practices

1. **Batch Processing**: For multiple texts, send requests in parallel using goroutines or async functions
2. **Error Handling**: Always check status codes and handle errors appropriately
3. **Pagination**: Use reasonable limit values (10-50) to avoid large responses
4. **Caching**: Consider caching analysis results on the client side for frequently accessed data
5. **Text Length**: Very long texts (>100k words) may take longer to process
6. **Tag Searches**: Tags are case-sensitive, so use exact matches

---

## Changelog

### Version 1.0.0 (2025-01-15)
- Initial release
- Basic text analysis functionality
- RESTful API with CRUD operations
- Tag-based search
- SQLite database support
