# Quick Start Guide

Get up and running with Text Analyzer in under 5 minutes.

## Installation

### Option 1: Build from Source

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

The server will start on `http://localhost:8080`

### Option 2: Using Make

```bash
# Build and run
make build
make run

# Or in one command
make build && ./textanalyzer
```

### Option 3: Docker

```bash
# Build and run with Docker
docker build -t textanalyzer .
docker run -p 8080:8080 textanalyzer

# Or use Docker Compose
docker-compose up
```

## First Analysis

Once the server is running, try analyzing some text:

### Using cURL

```bash
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello world! This is a test."}'
```

### Using Example Files

```bash
# Analyze the simple test example
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d @examples/simple_test.json

# Analyze climate change article
curl -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d @examples/climate_change.json
```

### Using Browser

1. Open your browser to `http://localhost:8080/health` - you should see:
   ```json
   {"status":"ok","time":"..."}
   ```

2. Use a tool like [Postman](https://www.postman.com/) or [Insomnia](https://insomnia.rest/) to send POST requests to `/api/analyze`

## Understanding the Response

A typical response looks like this:

```json
{
  "id": "20250115103000-123456",
  "text": "Your original text...",
  "metadata": {
    "word_count": 25,
    "sentiment": "positive",
    "sentiment_score": 0.35,
    "readability_level": "easy",
    "tags": ["positive", "short", "easy"],
    ...
  },
  "created_at": "2025-01-15T10:30:00Z",
  "updated_at": "2025-01-15T10:30:00Z"
}
```

Key fields to note:
- **id**: Unique identifier for this analysis
- **sentiment**: Overall tone (positive/negative/neutral)
- **readability_level**: How easy the text is to read
- **tags**: Auto-generated categories

## Common Operations

### Retrieve an Analysis

```bash
curl http://localhost:8080/api/analyses/20250115103000-123456
```

### List All Analyses

```bash
# Get first 10 analyses
curl http://localhost:8080/api/analyses

# Pagination
curl "http://localhost:8080/api/analyses?limit=20&offset=20"
```

### Search by Tag

```bash
curl "http://localhost:8080/api/search?tag=positive"
```

### Delete an Analysis

```bash
curl -X DELETE http://localhost:8080/api/analyses/20250115103000-123456
```

## Configuration

### Custom Port

```bash
./textanalyzer -port 3000
```

### Custom Database Location

```bash
./textanalyzer -db /path/to/database.db
```

### Both Options

```bash
./textanalyzer -port 3000 -db /path/to/mydata.db
```

## Testing

### Run Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/analyzer
```

### Using Make

```bash
make test
make test-coverage
make bench
```

## Example Workflow

Here's a complete example workflow:

```bash
# 1. Start the server
./textanalyzer &

# 2. Analyze some text
ANALYSIS_ID=$(curl -s -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d @examples/simple_test.json | jq -r '.id')

echo "Analysis ID: $ANALYSIS_ID"

# 3. Retrieve the analysis
curl http://localhost:8080/api/analyses/$ANALYSIS_ID | jq

# 4. Search by tag
curl "http://localhost:8080/api/search?tag=positive" | jq

# 5. List all analyses
curl http://localhost:8080/api/analyses | jq

# 6. Stop the server
killall textanalyzer
```

## Integrating with Your Application

### JavaScript/TypeScript

```javascript
async function analyzeText(text) {
  const response = await fetch('http://localhost:8080/api/analyze', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text })
  });
  return await response.json();
}

// Usage
const result = await analyzeText('Your text here...');
console.log('Sentiment:', result.metadata.sentiment);
console.log('Word count:', result.metadata.word_count);
```

### Python

```python
import requests

def analyze_text(text):
    response = requests.post(
        'http://localhost:8080/api/analyze',
        json={'text': text}
    )
    return response.json()

# Usage
result = analyze_text('Your text here...')
print(f"Sentiment: {result['metadata']['sentiment']}")
print(f"Word count: {result['metadata']['word_count']}")
```

### Go

```go
import (
    "bytes"
    "encoding/json"
    "net/http"
)

type AnalyzeRequest struct {
    Text string `json:"text"`
}

func analyzeText(text string) (*Analysis, error) {
    reqBody, _ := json.Marshal(AnalyzeRequest{Text: text})
    
    resp, err := http.Post(
        "http://localhost:8080/api/analyze",
        "application/json",
        bytes.NewBuffer(reqBody),
    )
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var analysis Analysis
    json.NewDecoder(resp.Body).Decode(&analysis)
    return &analysis, nil
}
```

## Troubleshooting

### Port Already in Use

```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or use a different port
./textanalyzer -port 8081
```

### Database Locked

```bash
# Stop the server
killall textanalyzer

# Remove lock files
rm textanalyzer.db-shm textanalyzer.db-wal

# Restart server
./textanalyzer
```

### Build Errors (CGO)

```bash
# Install GCC (macOS)
xcode-select --install

# Install GCC (Linux)
sudo apt-get install gcc

# Install GCC (Windows)
# Use MinGW or WSL
```

## Next Steps

- Read the [API Reference](API.md) for detailed endpoint documentation
- Check out the [Development Guide](DEVELOPMENT.md) for contributing
- Explore the [examples/](examples/) directory for more sample texts
- Review the [README](README.md) for comprehensive documentation

## Getting Help

- Check existing issues on GitHub
- Read the documentation in the docs/ directory
- Create a new issue with:
  - What you tried
  - What happened
  - What you expected
  - Your environment (OS, Go version)

## Performance Tips

- For large texts (>100k words), expect longer processing times
- Use pagination when listing many analyses
- Consider running the server behind a reverse proxy (nginx, Apache)
- For production, migrate to PostgreSQL for better performance

## Security Considerations

For production deployments:
- Add authentication (JWT, API keys)
- Enable HTTPS/TLS
- Implement rate limiting
- Use environment variables for sensitive configuration
- Run with least privilege user
- Keep dependencies updated

Enjoy using Text Analyzer! 🚀
