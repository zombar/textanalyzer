package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
)

const (
	DefaultModel   = "gpt-oss:20b"
	DefaultTimeout = 360 * time.Second
)

// Client wraps the Ollama API client
type Client struct {
	client  *api.Client
	model   string
	timeout time.Duration
}

// New creates a new Ollama client
func New(ollamaURL, model string) (*Client, error) {
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	if model == "" {
		model = DefaultModel
	}

	// Parse the base URL
	baseURL, err := url.Parse(ollamaURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama URL: %w", err)
	}

	// Create client with the provided URL
	client := api.NewClient(baseURL, http.DefaultClient)

	return &Client{
		client:  client,
		model:   model,
		timeout: DefaultTimeout,
	}, nil
}

// GenerateResponse generates a response from the LLM
func (c *Client) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	log.Printf("Ollama: Sending request to model %s (timeout: %v)", c.model, c.timeout)

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &api.GenerateRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: new(bool), // false
	}

	var response strings.Builder
	err := c.client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		response.WriteString(resp.Response)
		return nil
	})

	if err != nil {
		log.Printf("Ollama: Generation failed: %v", err)
		return "", fmt.Errorf("generation failed: %w", err)
	}

	result := strings.TrimSpace(response.String())
	log.Printf("Ollama: Response received (%d chars)", len(result))
	return result, nil
}

// GenerateSynopsis creates a 3-4 sentence synopsis of the text
func (c *Client) GenerateSynopsis(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Analyze the following text and provide a concise synopsis that captures the main points and key ideas.

Requirements:
- Write EXACTLY 2 or 3 short sentences summarizing the content
- Keep each sentence under 15 words
- Use simple, clear language
- Avoid complex or compound sentences
- Do NOT use numbering or bullet points
- Do NOT provide meta-commentary (e.g., "the text has...", "this article discusses...")
- Write the synopsis as if describing the content to someone

Text:
%s

Synopsis:`, text)

	return c.GenerateResponse(ctx, prompt)
}

// CleanText removes artifacts and non-relevant content from the text
func (c *Client) CleanText(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Your task is to clean the following text by removing artifacts, formatting issues, advertisements, navigation elements, and other non-relevant content.

IMPORTANT INSTRUCTIONS:
- If the text is already clean and well-formatted, return it EXACTLY as provided
- If there are issues to clean, return ONLY the cleaned article content
- Do NOT add any commentary, explanations, or meta-analysis
- Do NOT say things like "the text is clean" or "no changes needed"
- Simply return the text (cleaned or as-is)

Text to process:
%s

Output the text:`, text)

	return c.GenerateResponse(ctx, prompt)
}

// EditorialAnalysis provides analysis of bias, motivation, and editorial slant
func (c *Client) EditorialAnalysis(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Analyze the following text and provide an unbiased assessment of the nature and purpose of this text (informational, persuasive, entertainment, etc.), possible motivations behind the writing, any editorial slant or bias (left/right, commercial, academic, etc.), and the overall tone and approach.

Requirements:
- Write EXACTLY 2 short sentences
- Keep each sentence under 15 words
- Use simple, clear language
- Avoid complex or compound sentences
- Be objective and analytical
- Do NOT use numbering or bullet points

Text:
%s

Analysis:`, text)

	return c.GenerateResponse(ctx, prompt)
}

// GenerateTags generates up to 5 relevant tags for the text
func (c *Client) GenerateTags(ctx context.Context, text string, metadata map[string]interface{}) ([]string, error) {
	// Include some context from metadata if available
	sentiment := ""
	if s, ok := metadata["sentiment"].(string); ok {
		sentiment = s
	}

	prompt := fmt.Sprintf(`Analyze the following text and generate up to 10 relevant tags that categorize and describe the content.

Tag formatting rules:
- Prefer single-word tags whenever possible
- Multi-word tags should use hyphens only (no spaces or underscores)
- Names of people, places, and things make excellent tags
- All tags should be lowercase
- Examples: "technology", "climate-change", "new-york", "machine-learning", "einstein"

Consider: topic, domain, sentiment (%s), content type, key themes, named entities (people, places, organizations).

Return ONLY a JSON array of strings, nothing else.

Text:
%s

Tags (JSON array):`, sentiment, text)

	response, err := c.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var tags []string

	// Try to find JSON array in response
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start >= 0 && end > start {
		jsonStr := response[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &tags); err != nil {
			return nil, fmt.Errorf("failed to parse tags JSON: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	// Normalize tags
	for i, tag := range tags {
		tags[i] = normalizeTag(tag)
	}

	// Limit to 10 tags
	if len(tags) > 10 {
		tags = tags[:10]
	}

	return tags, nil
}

// normalizeTag normalizes a tag according to the tagging rules:
// - Converts to lowercase
// - Replaces spaces and underscores with hyphens
// - Removes multiple consecutive hyphens
// - Trims leading/trailing hyphens and whitespace
func normalizeTag(tag string) string {
	// Convert to lowercase
	tag = strings.ToLower(tag)

	// Replace spaces and underscores with hyphens
	tag = strings.ReplaceAll(tag, " ", "-")
	tag = strings.ReplaceAll(tag, "_", "-")

	// Remove multiple consecutive hyphens
	for strings.Contains(tag, "--") {
		tag = strings.ReplaceAll(tag, "--", "-")
	}

	// Trim leading/trailing hyphens and whitespace
	tag = strings.Trim(tag, "- \t\n\r")

	return tag
}

// ExtractReferences extracts and validates references from text
func (c *Client) ExtractReferences(ctx context.Context, text string) ([]Reference, error) {
	prompt := fmt.Sprintf(`Analyze the following text and extract factual claims, statistics, quotes, and assertions that would benefit from verification or citation.

For each reference, identify:
- The exact text of the claim/statistic/quote
- Type (statistic, quote, claim, or citation)
- Brief context (surrounding text)
- Confidence level (high, medium, low)

Return ONLY a JSON array of objects with fields: text, type, context, confidence. Limit to the 10 most significant references.

Text:
%s

References (JSON array):`, text)

	response, err := c.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var references []Reference

	// Try to find JSON array in response
	start := strings.Index(response, "[")
	end := strings.LastIndex(response, "]")
	if start >= 0 && end > start {
		jsonStr := response[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &references); err != nil {
			return nil, fmt.Errorf("failed to parse references JSON: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no JSON array found in response")
	}

	return references, nil
}

// Reference represents a factual claim or citation
type Reference struct {
	Text       string `json:"text"`
	Type       string `json:"type"`
	Context    string `json:"context"`
	Confidence string `json:"confidence"`
}

// AIDetectionResult represents AI-generated content detection
type AIDetectionResult struct {
	Likelihood string   `json:"likelihood"`
	Confidence string   `json:"confidence"`
	Reasoning  string   `json:"reasoning"`
	Indicators []string `json:"indicators"`
	HumanScore float64  `json:"human_score"`
}

// DetectAIContent analyzes whether the text was likely written by AI
func (c *Client) DetectAIContent(ctx context.Context, text string) (*AIDetectionResult, error) {
	prompt := fmt.Sprintf(`Analyze the following text to determine if it was written by an AI or a human. Consider factors such as:

1. Writing patterns (repetitive structures, overly formal tone, perfect grammar)
2. Vocabulary choices (overuse of certain words, lack of colloquialisms)
3. Content structure (formulaic organization, lack of personal anecdotes)
4. Stylistic markers (balanced arguments, hedging language, transitions)
5. Creativity and authenticity (unique insights vs. generic statements)
6. Errors and imperfections (natural human mistakes vs. AI consistency)

Provide your assessment as a JSON object with:
- likelihood: "very_likely" | "likely" | "possible" | "unlikely" | "very_unlikely" (AI-generated)
- confidence: "high" | "medium" | "low"
- reasoning: 2-3 sentences explaining your assessment
- indicators: array of specific markers you found (e.g., "repetitive sentence structure", "lack of personal voice", "perfect grammar")
- human_score: 0-100 where 0 = definitely AI, 100 = definitely human

Text to analyze:
%s

Return ONLY the JSON object, nothing else:`, text)

	response, err := c.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var result AIDetectionResult

	// Try to find JSON object in response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		jsonStr := response[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return nil, fmt.Errorf("failed to parse AI detection JSON: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	return &result, nil
}

// TextQualityScoreResult represents the quality score for text content
type TextQualityScoreResult struct {
	Score             float64  `json:"score"`
	Reason            string   `json:"reason"`
	Categories        []string `json:"categories"`
	QualityIndicators []string `json:"quality_indicators"`
	ProblemsDetected  []string `json:"problems_detected"`
}

// ScoreTextQuality analyzes and scores the quality of text content
func (c *Client) ScoreTextQuality(ctx context.Context, text string) (*TextQualityScoreResult, error) {
	prompt := fmt.Sprintf(`You are a content quality assessment assistant. Analyze the following text and determine its quality for information and knowledge purposes.

Evaluate the text and assign a quality score from 0.0 to 1.0 where:
- 1.0 = Excellent quality (well-written, informative, coherent, valuable)
- 0.7-0.9 = Good quality (useful content with minor issues)
- 0.4-0.6 = Moderate quality (some value but significant issues)
- 0.0-0.3 = Low quality (spam, incoherent, useless)

REJECT (score 0.0-0.3) the following types of content:
- Spam, advertisements, or promotional content
- Incoherent or nonsensical text
- Extremely short or trivial content (< 50 meaningful characters)
- Content that is mostly punctuation, symbols, or gibberish
- Duplicate or repetitive content
- Content that is purely links or navigation
- Offensive, hateful, or harmful content

MODERATE (score 0.4-0.6) content with:
- Poor grammar or structure but some useful information
- Incomplete thoughts or fragmented content
- Mixed quality (good and bad sections)
- Excessive formatting issues

ACCEPT (score 0.7-1.0) content that is:
- Well-written and coherent
- Informative and valuable
- Properly structured
- Original and thoughtful
- Educational or enlightening

Provide your assessment in JSON format:
{
  "score": 0.0-1.0,
  "reason": "Brief explanation of the score",
  "categories": ["category1", "category2"],
  "quality_indicators": ["indicator1", "indicator2"],
  "problems_detected": ["problem1", "problem2"]
}

Categories should include applicable labels: "informative", "educational", "well_written", "coherent", "spam", "low_quality", "incoherent", "promotional", etc.

Quality indicators list positive aspects: "clear_structure", "good_grammar", "valuable_insights", "well_researched", etc.

Problems detected list issues found: "poor_grammar", "incoherent", "too_short", "spam_like", "repetitive", etc.

Text to analyze:
%s

Return ONLY the JSON object, nothing else:`, text)

	response, err := c.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var result TextQualityScoreResult

	// Try to find JSON object in response
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		jsonStr := response[start : end+1]
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			return nil, fmt.Errorf("failed to parse quality score JSON: %w", err)
		}
	} else {
		return nil, fmt.Errorf("no JSON object found in response")
	}

	// Ensure score is within bounds
	if result.Score < 0.0 {
		result.Score = 0.0
	}
	if result.Score > 1.0 {
		result.Score = 1.0
	}

	// Ensure slices are not nil
	if result.Categories == nil {
		result.Categories = []string{}
	}
	if result.QualityIndicators == nil {
		result.QualityIndicators = []string{}
	}
	if result.ProblemsDetected == nil {
		result.ProblemsDetected = []string{}
	}

	return &result, nil
}
