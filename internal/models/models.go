package models

import "time"

// Analysis represents a text analysis with its metadata
type Analysis struct {
	ID           string    `json:"id"`
	Text         string    `json:"text"`
	OriginalHTML string    `json:"original_html,omitempty"` // Compressed + base64 encoded original HTML/raw text
	Metadata     Metadata  `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Metadata contains all extracted information from text analysis
type Metadata struct {
	// Basic statistics
	CharacterCount    int     `json:"character_count"`
	WordCount         int     `json:"word_count"`
	SentenceCount     int     `json:"sentence_count"`
	ParagraphCount    int     `json:"paragraph_count"`
	AverageWordLength float64 `json:"average_word_length"`

	// Sentiment analysis
	Sentiment      string  `json:"sentiment"`       // positive, negative, neutral
	SentimentScore float64 `json:"sentiment_score"` // -1.0 to 1.0

	// Important words and phrases
	TopWords    []WordFrequency `json:"top_words"`
	TopPhrases  []PhraseInfo    `json:"top_phrases"`
	UniqueWords int             `json:"unique_words"`

	// Content analysis
	KeyTerms       []string `json:"key_terms"`
	NamedEntities  []string `json:"named_entities"`
	PotentialDates []string `json:"potential_dates"`
	PotentialURLs  []string `json:"potential_urls"`
	EmailAddresses []string `json:"email_addresses"`

	// Readability
	ReadabilityScore  float64 `json:"readability_score"`
	ReadabilityLevel  string  `json:"readability_level"`
	ComplexWordCount  int     `json:"complex_word_count"`
	AvgSentenceLength float64 `json:"avg_sentence_length"`

	// References to verify
	References []Reference `json:"references"`

	// Tags for categorization
	Tags []string `json:"tags"`

	// Language indicators
	Language           string  `json:"language"`
	QuestionCount      int     `json:"question_count"`
	ExclamationCount   int     `json:"exclamation_count"`
	CapitalizedPercent float64 `json:"capitalized_percent"`

	// AI-generated content
	Synopsis               string            `json:"synopsis"`                  // 3-4 sentence summary
	CleanedText            string            `json:"cleaned_text"`              // AI-cleaned text with artifacts removed
	HeuristicCleanedText   string            `json:"heuristic_cleaned_text"`    // Rule-based/heuristic cleaned text
	EditorialAnalysis      string            `json:"editorial_analysis"`        // Bias, motivation, and slant analysis
	AIDetection            AIDetectionResult `json:"ai_detection"`              // AI-generated content detection

	// Quality scoring
	QualityScore *TextQualityScore `json:"quality_score,omitempty"` // Text quality assessment
}

// WordFrequency represents a word and its frequency
type WordFrequency struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

// PhraseInfo represents a phrase and its information
type PhraseInfo struct {
	Phrase string `json:"phrase"`
	Count  int    `json:"count"`
}

// Reference represents a claim or fact that should be verified
type Reference struct {
	Text       string `json:"text"`
	Type       string `json:"type"` // claim, statistic, quote, citation
	Context    string `json:"context"`
	Confidence string `json:"confidence"` // high, medium, low
}

// AIDetectionResult represents the analysis of whether content was AI-generated
type AIDetectionResult struct {
	Likelihood string   `json:"likelihood"`  // very_likely, likely, possible, unlikely, very_unlikely
	Confidence string   `json:"confidence"`  // high, medium, low
	Reasoning  string   `json:"reasoning"`   // Explanation of the assessment
	Indicators []string `json:"indicators"`  // Specific indicators found
	HumanScore float64  `json:"human_score"` // 0-100, higher means more likely human-written
}

// TextQualityScore represents quality assessment for text content
type TextQualityScore struct {
	Score               float64  `json:"score"`                // 0.0 to 1.0, higher is better quality
	Reason              string   `json:"reason"`               // Explanation for the score
	Categories          []string `json:"categories"`           // Content categories (e.g., "informative", "spam", "low_quality")
	IsRecommended       bool     `json:"is_recommended"`       // Whether the text is recommended
	QualityIndicators   []string `json:"quality_indicators"`   // Positive quality indicators
	ProblemsDetected    []string `json:"problems_detected"`    // Issues found in the text
	AIUsed              bool     `json:"ai_used"`              // Whether AI (Ollama) was used for scoring (true) or rule-based fallback (false)
}
