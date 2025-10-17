# Ollama Integration Guide

This guide explains how to use the Ollama-powered AI features in Text Analyzer.

## Prerequisites

1. Install Ollama from https://ollama.ai
2. Pull the default model:
   ```bash
   ollama pull gpt-oss:20b
   ```

## Starting Ollama

```bash
# Start Ollama server (usually runs on localhost:11434)
ollama serve
```

## Running Text Analyzer with Ollama

### Default Configuration

```bash
# Build and run with Ollama enabled (default)
go build -o textanalyzer ./cmd/server
./textanalyzer
```

### Custom Configuration

```bash
# Disable Ollama
./textanalyzer -use-ollama=false

# Use different model
./textanalyzer -ollama-model=llama2:13b

# Use remote Ollama instance
./textanalyzer -ollama-url=http://remote-server:11434
```

## AI-Powered Features

When Ollama is enabled, the following metadata fields are AI-generated:

### 1. Synopsis
A 3-4 sentence summary of the text that captures main points and key ideas.

**Example:**
```json
{
  "synopsis": "This article discusses the impact of climate change on global agriculture. Rising temperatures and changing precipitation patterns are threatening food security worldwide. Adaptation strategies include drought-resistant crops and improved irrigation systems."
}
```

### 2. Cleaned Text
Text with artifacts, formatting issues, ads, and non-relevant content removed.

**Use case:** Extract clean article content from web pages with navigation elements, ads, etc.

### 3. Editorial Analysis
Unbiased assessment of the text's nature, possible motivations, editorial slant, and bias.

**Example:**
```json
{
  "editorial_analysis": "This piece is informational with a scientific tone, aimed at educating readers about climate impacts. The author presents data-driven arguments with minimal emotional appeal. Slight environmental advocacy bias is present but backed by cited research."
}
```

### 4. AI-Generated Tags
Up to 5 relevant tags that categorize the content, replacing rule-based tag generation.

**Example:**
```json
{
  "tags": ["climate-change", "agriculture", "food-security", "environmental-science", "sustainability"]
}
```

### 5. AI-Extracted References
More accurate extraction and pruning of factual claims, statistics, quotes, and citations.

**Example:**
```json
{
  "references": [
    {
      "text": "Global temperatures have risen 1.1°C since pre-industrial times",
      "type": "statistic",
      "context": "According to the IPCC report...",
      "confidence": "high"
    }
  ]
}
```

### 6. AI Content Detection
Analyzes whether the text was likely written by AI or a human, with detailed reasoning.

**Example:**
```json
{
  "ai_detection": {
    "likelihood": "unlikely",
    "confidence": "medium",
    "reasoning": "The text shows natural imperfections, personal voice, and stylistic variations typical of human writing. While well-structured, it contains informal elements and varied sentence patterns.",
    "indicators": [
      "varied sentence structure",
      "personal voice and opinions",
      "natural flow with minor imperfections",
      "domain-specific terminology used naturally"
    ],
    "human_score": 75.5
  }
}
```

**Fields:**
- `likelihood`: How likely the text is AI-generated (very_likely, likely, possible, unlikely, very_unlikely)
- `confidence`: How confident the assessment is (high, medium, low)
- `reasoning`: 2-3 sentence explanation of the assessment
- `indicators`: Specific markers that influenced the decision
- `human_score`: 0-100 score where 0 = definitely AI, 100 = definitely human

## New API Endpoint

### Search by Reference

Search for analyses containing specific reference text:

```bash
GET /api/search/reference?reference=climate+change

# Example
curl "http://localhost:8080/api/search/reference?reference=1.1°C"
```

**Response:**
```json
[
  {
    "id": "20250117120000-123456",
    "text": "Full article text...",
    "metadata": {
      "synopsis": "...",
      "references": [...]
    }
  }
]
```

## Performance Considerations

- **Initial request**: First analysis with Ollama may take 30-60 seconds as the model loads
- **Subsequent requests**: 5-15 seconds depending on text length and model size
- **Timeout**: Analysis has a 60-second timeout by default
- **Fallback**: If Ollama fails or times out, the system falls back to rule-based analysis

## Choosing a Model

### Recommended Models

| Model | Size | Speed | Quality | Use Case |
|-------|------|-------|---------|----------|
| gpt-oss:20b | 4.7GB | Fast | Good | Default, balanced performance |
| llama2:13b | 7.4GB | Medium | Better | Higher quality analysis |
| mistral:7b | 4.1GB | Fast | Good | Faster, similar to qwen |
| llama2:70b | 39GB | Slow | Best | Maximum quality (requires powerful hardware) |

### Pull a Model

```bash
ollama pull gpt-oss:20b
ollama pull llama2:13b
```

## Troubleshooting

### Ollama Connection Failed

**Error:** `Warning: Failed to initialize Ollama client`

**Solutions:**
1. Check Ollama is running: `curl http://localhost:11434/api/version`
2. Verify the model is pulled: `ollama list`
3. Check firewall settings if using remote Ollama

### Slow Performance

1. Use a smaller model (gpt-oss:20b or mistral:7b)
2. Ensure adequate RAM (8GB minimum, 16GB recommended)
3. Use GPU acceleration if available

### Analysis Timeout

Increase timeout in analyzer by modifying `internal/ollama/client.go`:

```go
const DefaultTimeout = 120 * time.Second // Increase from 60s
```

## Development

### Testing Without Ollama

```bash
# Run with Ollama disabled for faster testing
./textanalyzer -use-ollama=false
```

### Testing with Mock Ollama

Set up a test Ollama instance or use the fallback mode for CI/CD pipelines.

## Example Usage

```bash
# Start Ollama
ollama serve

# In another terminal, start text analyzer
cd textanalyzer
go build -o textanalyzer ./cmd/server
./textanalyzer

# Test with example (note: -m 420 sets 7 minute timeout for AI processing)
curl -m 420 -X POST http://localhost:8080/api/analyze \
  -H "Content-Type: application/json" \
  -d @examples/climate_change.json

# The response will include AI-generated fields:
# - synopsis
# - cleaned_text
# - editorial_analysis
# - ai_detection (NEW!)
# - AI-generated tags
# - AI-extracted references
```

## Database Schema

References are now stored in a separate searchable table:

```sql
CREATE TABLE text_references (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    analysis_id TEXT NOT NULL,
    text TEXT NOT NULL,
    type TEXT NOT NULL,
    context TEXT,
    confidence TEXT,
    FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE
);
```

This enables efficient searching across all references in all analyses.
