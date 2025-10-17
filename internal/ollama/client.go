package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
		ollamaURL = "http://honker:11434"
		os.Setenv("OLLAMA_HOST", ollamaURL)
	}
	if model == "" {
		model = DefaultModel
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

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
	prompt := fmt.Sprintf(`Analyze the following text and provide a concise 3-4 sentence synopsis that captures the main points and key ideas.

Text:
%s

Synopsis:`, text)

	return c.GenerateResponse(ctx, prompt)
}

// CleanText removes artifacts and non-relevant content from the text
func (c *Client) CleanText(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Clean the following text by removing artifacts, formatting issues, advertisements, navigation elements, and other non-relevant content. Return only the clean article content without any explanations.

Text:
%s

Cleaned text:`, text)

	return c.GenerateResponse(ctx, prompt)
}

// EditorialAnalysis provides analysis of bias, motivation, and editorial slant
func (c *Client) EditorialAnalysis(ctx context.Context, text string) (string, error) {
	prompt := fmt.Sprintf(`Analyze the following text and provide an unbiased assessment of:
1. The nature and purpose of this text (informational, persuasive, entertainment, etc.)
2. Possible motivations behind the writing
3. Any editorial slant or bias (left/right, commercial, academic, etc.)
4. Overall tone and approach

Be objective and analytical in your assessment. Provide 2-3 sentences.

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

	prompt := fmt.Sprintf(`Analyze the following text and generate up to 5 relevant tags that categorize and describe the content. Tags should be single words or short phrases (2-3 words max), lowercase, and hyphenated if multi-word.

Consider: topic, domain, sentiment (%s), content type, and key themes.

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

	// Limit to 5 tags
	if len(tags) > 5 {
		tags = tags[:5]
	}

	return tags, nil
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
