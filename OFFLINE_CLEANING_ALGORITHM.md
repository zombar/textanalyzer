# Advanced Offline Text Cleaning Algorithm

## Overview

The offline text cleaning algorithm performs sophisticated content extraction using **13 heuristic factors** to identify and filter article content from noise (navigation, ads, boilerplate).

## Architecture

### Stage 1: Paragraph Splitting
Text is split into paragraphs using intelligent rules:
- Primary split: double newlines (`\n\n`)
- Secondary split: long paragraphs (>1000 chars) split by single newlines
- Whitespace trimming and empty paragraph removal

### Stage 2: Paragraph Scoring
Each paragraph is scored on a scale of 0.0 to 1.0 using 13 quality factors:

## Scoring Factors

### 1. **Word Count** (Weight: 0.2 to -0.3)
- **Sweet spot**: 20-200 words (+0.2)
- **Too short**: <10 words (-0.3)
- **Too long**: >300 words (-0.1)
- **Rationale**: Article paragraphs have substance; navigation/ads are short

### 2. **Link Density** (Weight: -0.4)
- **Calculation**: (URL count + arrow count) / word count
- **Threshold**: >10% link density = BAD (-0.4)
- **Rationale**: Navigation and menus have high link density

### 3. **Stopword Ratio** (Weight: +0.15 to -0.2)
- **Optimal range**: 35-65% stopwords (+0.15)
- **Too low**: <25% stopwords (-0.2)
- **Rationale**: Natural article text has balanced stopword usage; keyword stuffing/navigation doesn't

### 4. **Named Entities** (Weight: +0.1)
- **Threshold**: ≥2 named entities = GOOD (+0.1)
- **Rationale**: Article content mentions people, places, organizations

### 5. **Average Word Length** (Weight: +0.1)
- **Optimal range**: 4.0-6.0 characters (+0.1)
- **Rationale**: Balanced word length indicates natural prose

### 6. **Image Markers** (Weight: -0.4)
- **Patterns detected**:
  - "Image source:", "Photo by:", "Credit:"
  - "Getty Images", "Photographer:", "©"
  - "[Image:", "Courtesy of:", "Via:"
- **Rationale**: These are captions/attributions, not article content

### 7. **Boilerplate Patterns** (Weight: -0.5)
- **Navigation**: "Click here", "Read more", "Related articles"
- **Engagement**: "Subscribe", "Sign up", "Newsletter", "Share this"
- **Commerce**: "Buy now", "Add to cart", "Free trial"
- **Metadata**: "Filed under:", "Tags:", "Categories:"
- **Comments**: "View comments", "Post comment"
- **Rationale**: These are UI elements, not content

### 8. **Capitalization Ratio** (Weight: -0.3)
- **Threshold**: >50% uppercase = BAD (-0.3)
- **Rationale**: Headers and navigation often use ALL CAPS

### 9. **Punctuation Overload** (Weight: -0.2)
- **Threshold**: Punctuation count > 20% of words = BAD (-0.2)
- **Rationale**: Excessive punctuation indicates spam or lists

### 10. **List-Like Structure** (Weight: -0.2)
- **Patterns**: Starts with "•", "-", "*", or "1."
- **Condition**: Only penalize if <15 words
- **Rationale**: Short bullet points are navigation; longer list items may be content

### 11. **Social Media Patterns** (Weight: -0.3)
- **Patterns**: "Share on Facebook", "Follow on Twitter"
- **Nuance**: Doesn't penalize content *about* social media
- **Rationale**: Sharing prompts are not article content

### 12. **Date/Timestamp Metadata** (Weight: -0.2)
- **Patterns**: "Posted on", "Published on", "Last modified"
- **Condition**: Only penalize if <20 words
- **Rationale**: Metadata lines are not content

### 13. **Author Bylines** (Weight: -0.2)
- **Patterns**: "By [Name]", "Written by", "Author:"
- **Condition**: Only penalize if <15 words
- **Rationale**: Bylines are metadata, not article content

## Stage 3: Dynamic Thresholding

Instead of a fixed threshold, we calculate a **dynamic threshold** based on the score distribution:

```
1. Calculate median score across all paragraphs
2. Use median as threshold (bounded: 0.3 ≤ threshold ≤ 0.6)
3. Keep paragraphs with score ≥ threshold
4. Reject all paragraphs marked as boilerplate (regardless of score)
```

**Rationale**: Different articles have different noise levels; adaptive threshold handles variety

## Stage 4: Reconstruction

- Filtered paragraphs are joined with double newlines
- Paragraph order is preserved
- Result is stored in `metadata.CleanedText`

## Benefits

### 1. **Better AI Template**
The cleaned text provides a much cleaner template for the AI enhancement stage, making it easier for the LLM to:
- Identify article boundaries in the HTML
- Understand content structure
- Remove remaining artifacts

### 2. **Reduced AI Processing**
By removing obvious noise offline:
- Smaller text sent to Ollama
- Faster AI processing
- Lower token costs
- Better AI focus on actual content

### 3. **Consistent Quality**
- Works when Ollama is unavailable
- Provides deterministic baseline cleaning
- Fast execution (no API calls)

### 4. **Visibility**
Logging shows:
- Number of paragraphs analyzed
- Number kept vs. removed
- Reasons for removal (for debugging)
- Percentage reduction in word count

## Example Output

```
Analyzing 45 paragraphs...
Removed paragraph 3 (score=0.15): too_short, high_link_density
Removed paragraph 7 (score=0.22): image_attribution
Removed paragraph 12 (score=0.18): boilerplate_pattern
Removed paragraph 15 (score=0.31): social_media_prompt
Paragraph quality threshold: 0.52
Offline cleaning complete: kept 28 paragraphs, removed 17
Offline cleaning: 1247 words → 892 words (28.5% reduction)
```

## Future Enhancements

Potential improvements to consider:

1. **Machine Learning**: Train a simple classifier on labeled data
2. **Sentence-level Scoring**: Even finer granularity
3. **HTML Structure Awareness**: Use HTML tags if available for better detection
4. **Domain-Specific Rules**: Different rules for news vs. blogs vs. academic papers
5. **Multi-language Support**: Language-specific stopword lists
6. **Coherence Analysis**: Check if consecutive paragraphs flow naturally

## Integration

The offline cleaner is automatically invoked in `AnalyzeOffline()`:

```go
metadata.CleanedText = a.cleanTextOffline(text)
```

The cleaned text is then used as a template in Stage 2 AI enrichment:

```go
w.queueClient.EnqueueEnrichText(ctx, analysisID, text, offlineText, originalHTML)
```

Where `offlineText = metadata.CleanedText` if available, otherwise falls back to original text.
