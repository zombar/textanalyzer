# Ollama Integration - Change Log

## Summary

Successfully integrated Ollama LLM for AI-powered text analysis features. The system maintains backward compatibility with optional Ollama support and graceful fallback to rule-based analysis.

## New Features

### 1. AI-Generated Metadata Fields

Added three new metadata fields populated by Ollama:

- **synopsis**: 3-4 sentence summary capturing main points
- **cleaned_text**: Text with artifacts and non-relevant content removed
- **editorial_analysis**: Unbiased assessment of bias, motivation, and editorial slant

### 2. AI-Powered Tag Generation

- Replaced rule-based tag generation with LLM-generated tags
- Limited to 5 high-quality, relevant tags per text
- Falls back to rule-based tags if Ollama unavailable

### 3. AI-Enhanced Reference Extraction

- LLM extracts and prunes references more accurately
- Better identification of statistics, quotes, claims, and citations
- Improved context and confidence scoring
- Falls back to regex-based extraction if Ollama unavailable

### 4. Searchable References

- New database table `text_references` for storing references separately
- New API endpoint: `GET /api/search/reference?reference=<text>`
- Enables searching across all analyses by reference content
- Indexed for performance

## Code Changes

### New Files

1. **internal/ollama/client.go**
   - Ollama API client wrapper
   - Methods for synopsis, text cleaning, editorial analysis
   - Tag generation and reference extraction functions
   - 60-second timeout with context support

2. **OLLAMA_SETUP.md**
   - Comprehensive guide for Ollama setup and usage
   - Model recommendations
   - Performance tuning tips
   - Troubleshooting guide

3. **CHANGELOG_OLLAMA.md** (this file)
   - Summary of all changes

### Modified Files

1. **internal/models/models.go**
   - Added `Synopsis`, `CleanedText`, `EditorialAnalysis` fields to Metadata struct

2. **internal/analyzer/analyzer.go**
   - Added `NewWithOllama()` constructor
   - New `AnalyzeWithContext()` method with context support
   - Integrated AI analysis with fallback logic
   - Modified `Analyze()` to call `AnalyzeWithContext()`

3. **internal/database/migrations.go**
   - Added migration #4: `create_text_references_table`
   - Stores references in separate table for searching

4. **internal/database/db.go**
   - Added `PRAGMA foreign_keys = ON` for cascade delete support

5. **internal/database/queries.go**
   - Modified `SaveAnalysis()` to insert references into `text_references` table
   - Added `GetAnalysesByReference()` for reference-based search

6. **internal/api/handler.go**
   - Added `handleSearchByReference()` handler
   - Registered `/api/search/reference` route

7. **cmd/server/main.go**
   - Added Ollama-related command-line flags:
     - `-use-ollama` (default: true)
     - `-ollama-url` (default: http://localhost:11434)
     - `-ollama-model` (default: qwen2.5:7b)
   - Initialize Ollama client with error handling
   - Graceful fallback to rule-based analysis

8. **go.mod**
   - Added `github.com/ollama/ollama` dependency
   - Upgraded Go version to 1.24.0 (Ollama requirement)

9. **CLAUDE.md**
   - Added Ollama integration section
   - Documented command-line flags
   - Listed AI-powered features

### Test Files

1. **internal/analyzer/analyzer_test.go**
   - Fixed unused variable warning

2. **internal/api/handler_test.go**
   - Updated handler initialization for new structure

## API Changes

### New Endpoint

```
GET /api/search/reference?reference=<text>
```

Searches for analyses containing the specified reference text.

**Example:**
```bash
curl "http://localhost:8080/api/search/reference?reference=climate"
```

### Enhanced Response

All analysis responses now include (when Ollama enabled):

```json
{
  "metadata": {
    "synopsis": "3-4 sentence summary...",
    "cleaned_text": "Text with artifacts removed...",
    "editorial_analysis": "Assessment of bias and motivation...",
    "tags": ["ai-generated", "tags", "..."],
    "references": [
      {
        "text": "Extracted claim or statistic",
        "type": "statistic|quote|claim|citation",
        "context": "Surrounding text",
        "confidence": "high|medium|low"
      }
    ]
  }
}
```

## Database Schema Changes

### New Table: text_references

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

CREATE INDEX idx_text_references_analysis_id ON text_references(analysis_id);
CREATE INDEX idx_text_references_text ON text_references(text);
CREATE INDEX idx_text_references_type ON text_references(type);
```

## Command-Line Usage

### Basic Usage

```bash
# With Ollama (default)
./textanalyzer

# Without Ollama
./textanalyzer -use-ollama=false

# Custom Ollama configuration
./textanalyzer -ollama-url=http://custom:11434 -ollama-model=llama2:13b
```

## Performance Impact

### With Ollama Enabled

- **First request**: 30-60s (model loading)
- **Subsequent requests**: 5-15s (depending on text length)
- **Memory**: +2-8GB (depending on model size)
- **CPU/GPU**: Higher utilization during analysis

### With Ollama Disabled

- **Request time**: <1s (no change from original)
- **Memory**: No change
- **CPU**: No change

## Backward Compatibility

- ✅ Existing API endpoints unchanged
- ✅ All existing tests pass
- ✅ Existing metadata fields preserved
- ✅ Graceful degradation when Ollama unavailable
- ✅ Can disable Ollama via flag for original behavior

## Migration Path

### For Existing Deployments

1. **Optional upgrade** - Ollama integration is opt-in
2. Run migrations to create `text_references` table
3. Install Ollama and pull model (if desired)
4. Enable with `-use-ollama=true` flag
5. Monitor performance and adjust timeout if needed

### Rollback Strategy

Simply disable Ollama:
```bash
./textanalyzer -use-ollama=false
```

All functionality reverts to rule-based analysis.

## Testing

All tests pass:
```bash
$ go test ./...
ok      github.com/zombar/textanalyzer/internal/analyzer        0.205s
ok      github.com/zombar/textanalyzer/internal/api             0.330s
ok      github.com/zombar/textanalyzer/internal/database        0.489s
```

## Dependencies Added

- `github.com/ollama/ollama` v0.12.6
- `github.com/google/uuid` v1.6.0 (transitive)
- `golang.org/x/crypto` v0.36.0 (transitive)
- `golang.org/x/sys` v0.31.0 (transitive)

## Future Enhancements

Possible improvements:
- [ ] Caching of LLM responses for repeated text
- [ ] Batch processing for multiple texts
- [ ] Custom prompt templates for specialized analysis
- [ ] Support for multiple LLM providers (OpenAI, Anthropic, etc.)
- [ ] Fine-tuned models for specific domains
- [ ] Asynchronous analysis with webhooks
- [ ] Confidence scoring for AI-generated content

## Documentation

New/updated documentation files:
- `OLLAMA_SETUP.md` - Setup and usage guide
- `CLAUDE.md` - Updated with Ollama integration section
- `CHANGELOG_OLLAMA.md` - This changelog

## Contributors

Integration completed on 2025-10-17.
