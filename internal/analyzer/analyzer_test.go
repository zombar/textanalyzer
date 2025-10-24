package analyzer

import (
	"context"
	"strings"
	"testing"

	"github.com/zombar/textanalyzer/internal/ollama"
)

func TestAnalyze(t *testing.T) {
	a := New()

	text := `Climate change is a pressing global issue. Scientists have documented a 1.1°C increase in global temperatures since 1880. 
	The effects are devastating: rising sea levels, extreme weather events, and loss of biodiversity. 
	According to recent studies, we need to reduce carbon emissions by 45% by 2030 to avoid catastrophic consequences.
	Many experts believe this is achievable with renewable energy adoption.`

	metadata := a.Analyze(text)

	// Test basic statistics
	if metadata.WordCount == 0 {
		t.Error("Word count should not be zero")
	}
	if metadata.CharacterCount == 0 {
		t.Error("Character count should not be zero")
	}
	if metadata.SentenceCount == 0 {
		t.Error("Sentence count should not be zero")
	}

	// Test sentiment
	if metadata.Sentiment == "" {
		t.Error("Sentiment should not be empty")
	}

	// Test word frequency
	if len(metadata.TopWords) == 0 {
		t.Error("Top words should not be empty")
	}

	// Test key terms
	if len(metadata.KeyTerms) == 0 {
		t.Error("Key terms should not be empty")
	}

	// Test tags
	if len(metadata.Tags) == 0 {
		t.Error("Tags should not be empty")
	}
}

func TestExtractWords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"simple text", "Hello world", 2},
		{"with punctuation", "Hello, world! How are you?", 5},
		{"empty string", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words := extractWords(tt.input)
			if len(words) != tt.expected {
				t.Errorf("expected %d words, got %d", tt.expected, len(words))
			}
		})
	}
}

func TestCountSentences(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"single sentence", "Hello world.", 1},
		{"multiple sentences", "Hello. How are you? I'm fine!", 3},
		{"no punctuation", "Hello world", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countSentences(tt.input)
			if count != tt.expected {
				t.Errorf("expected %d sentences, got %d", tt.expected, count)
			}
		})
	}
}

func TestCountParagraphs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"single paragraph", "Hello world", 1},
		{"multiple paragraphs", "Hello\n\nWorld", 2},
		{"empty lines", "Hello\n\n\n\nWorld", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countParagraphs(tt.input)
			if count != tt.expected {
				t.Errorf("expected %d paragraphs, got %d", tt.expected, count)
			}
		})
	}
}

func TestSentimentAnalysis(t *testing.T) {
	_ = New()

	tests := []struct {
		name              string
		input             string
		expectedSentiment string
	}{
		{"positive text", "This is a great and wonderful amazing experience!", "positive"},
		{"negative text", "This is terrible, awful, and horrible bad experience!", "negative"},
		{"neutral text", "The cat sat on the mat.", "neutral"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentiment, _ := analyzeSentiment(tt.input)
			if sentiment != tt.expectedSentiment {
				t.Errorf("expected sentiment %s, got %s", tt.expectedSentiment, sentiment)
			}
		})
	}
}

func TestExtractNamedEntities(t *testing.T) {
	text := "John Smith went to New York City to meet Jane Doe."
	entities := extractNamedEntities(text)

	if len(entities) == 0 {
		t.Error("Should extract named entities")
	}

	found := false
	for _, entity := range entities {
		if strings.Contains(entity, "John") || strings.Contains(entity, "New York") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Should find capitalized names")
	}
}

func TestExtractDates(t *testing.T) {
	text := "The meeting is on 12/25/2024 or December 25, 2024 or 25 Dec 2024."
	dates := extractDates(text)

	if len(dates) == 0 {
		t.Error("Should extract dates")
	}
}

func TestExtractURLs(t *testing.T) {
	text := "Visit https://example.com and http://test.org for more info."
	urls := extractURLs(text)

	if len(urls) != 2 {
		t.Errorf("expected 2 URLs, got %d", len(urls))
	}
}

func TestExtractEmails(t *testing.T) {
	text := "Contact us at info@example.com or support@test.org"
	emails := extractEmails(text)

	if len(emails) != 2 {
		t.Errorf("expected 2 emails, got %d", len(emails))
	}
}

func TestCalculateReadability(t *testing.T) {
	text := "The cat sat on the mat. The dog ran in the park."
	score := calculateReadability(text, 12, 2)

	if score == 0 {
		t.Error("Readability score should not be zero")
	}
}

func TestCountSyllables(t *testing.T) {
	tests := []struct {
		word     string
		expected int
	}{
		{"cat", 1},
		{"hello", 2},
		{"beautiful", 3},
		{"university", 5},
	}

	for _, tt := range tests {
		t.Run(tt.word, func(t *testing.T) {
			count := countSyllablesInWord(tt.word)
			if count != tt.expected {
				t.Errorf("word %s: expected %d syllables, got %d", tt.word, tt.expected, count)
			}
		})
	}
}

func TestGetTopWords(t *testing.T) {
	a := New()
	words := []string{"test", "test", "test", "example", "example", "hello", "world", "the", "a"}

	topWords := a.getTopWords(words, 5)

	if len(topWords) == 0 {
		t.Error("Should return top words")
	}

	// First word should be "test" as it appears most frequently
	if topWords[0].Word != "test" {
		t.Errorf("expected 'test' as top word, got %s", topWords[0].Word)
	}

	if topWords[0].Count != 3 {
		t.Errorf("expected count 3 for 'test', got %d", topWords[0].Count)
	}
}

func TestExtractReferences(t *testing.T) {
	text := `Studies show that 75% of people prefer this method. 
	"This is a notable quote," said the researcher. 
	The population has increased by 2.5 million people.`

	references := extractReferences(text)

	if len(references) == 0 {
		t.Error("Should extract references")
	}

	// Check for statistics
	foundStat := false
	for _, ref := range references {
		if ref.Type == "statistic" {
			foundStat = true
			break
		}
	}

	if !foundStat {
		t.Error("Should find statistics")
	}
}

func TestGenerateTags(t *testing.T) {
	a := New()
	text := "This is a short positive text."
	metadata := a.Analyze(text)

	if len(metadata.Tags) == 0 {
		t.Error("Should generate tags")
	}

	// Check for expected tags
	foundSentiment := false
	for _, tag := range metadata.Tags {
		if tag == "positive" {
			foundSentiment = true
			break
		}
	}

	if !foundSentiment {
		t.Error("Should include sentiment tag")
	}
}

func TestCalculateCapitalizedPercent(t *testing.T) {
	text := "Hello World This Is A Test"
	percent := calculateCapitalizedPercent(text)

	if percent != 100.0 {
		t.Errorf("expected 100%%, got %.2f%%", percent)
	}

	text2 := "hello world this is a test"
	percent2 := calculateCapitalizedPercent(text2)

	if percent2 != 0.0 {
		t.Errorf("expected 0%%, got %.2f%%", percent2)
	}
}

func TestTextQualityScoring(t *testing.T) {
	// This test requires Ollama to be available
	// It will skip if Ollama is not running
	ollamaClient, err := ollama.New("http://localhost:11434", "gpt-oss:20b")
	if err != nil {
		t.Skip("Ollama client creation failed, skipping test")
	}

	a := NewWithOllama(ollamaClient)

	text := `Artificial intelligence is revolutionizing how we approach complex problems in science and technology.
	Machine learning algorithms can now process vast amounts of data to identify patterns that would be impossible
	for humans to detect manually. This technology has applications in healthcare, climate science, and many other fields.`

	ctx := context.Background()
	metadata := a.AnalyzeWithContext(ctx, text)

	// Verify quality score is present (only if Ollama is available and working)
	// Note: This might be nil if Ollama fails
	if metadata.QualityScore != nil {
		t.Logf("✓ Quality score present: score=%.2f, recommended=%v",
			metadata.QualityScore.Score, metadata.QualityScore.IsRecommended)

		// Verify score is within valid range
		if metadata.QualityScore.Score < 0.0 || metadata.QualityScore.Score > 1.0 {
			t.Errorf("Score should be between 0.0 and 1.0, got %.2f", metadata.QualityScore.Score)
		}

		// Verify reason is not empty
		if metadata.QualityScore.Reason == "" {
			t.Error("Expected non-empty reason for score")
		}

		// Verify categories array exists
		if metadata.QualityScore.Categories == nil {
			t.Error("Expected categories array to be initialized")
		}

		// For well-written informative text, we expect a good score
		if metadata.QualityScore.Score >= 0.7 {
			t.Logf("✓ High quality text detected: score=%.2f", metadata.QualityScore.Score)
		}
	} else {
		t.Log("⚠ Quality score not present (Ollama may not be available or failed)")
	}
}

// TestScoreTextQualityFallbackShort tests fallback scoring for short content
func TestScoreTextQualityFallbackShort(t *testing.T) {
	score := scoreTextQualityFallback("Too short", 2, 0)

	if score.Score >= 0.5 {
		t.Errorf("Expected low score for very short content, got %.2f", score.Score)
	}

	if !containsStringSlice(score.ProblemsDetected, "too_few_words") && !containsStringSlice(score.ProblemsDetected, "extremely_short") {
		t.Errorf("Expected problems detected for short content, got: %v", score.ProblemsDetected)
	}

	if score.IsRecommended {
		t.Error("Expected short content to not be recommended")
	}
}

// TestScoreTextQualityFallbackSpam tests fallback scoring for spam content
func TestScoreTextQualityFallbackSpam(t *testing.T) {
	spamText := "Click here! Buy now! Buy now! Limited offer! Act now! Free money! Earn $$$ today!"
	score := scoreTextQualityFallback(spamText, 13, 50)

	if score.Score >= 0.4 {
		t.Errorf("Expected very low score for spam, got %.2f", score.Score)
	}

	if !containsStringSlice(score.ProblemsDetected, "spam_keywords") {
		t.Errorf("Expected spam_keywords in problems detected, got: %v", score.ProblemsDetected)
	}

	if !containsStringSlice(score.Categories, "spam") {
		t.Errorf("Expected 'spam' category, got: %v", score.Categories)
	}

	if score.IsRecommended {
		t.Error("Expected spam content to not be recommended")
	}
}

// TestScoreTextQualityFallbackQuality tests fallback scoring for quality content
func TestScoreTextQualityFallbackQuality(t *testing.T) {
	qualityText := strings.Repeat("This research study demonstrates clear evidence and findings about climate change. The analysis shows important data and results that conclude significant environmental impacts. ", 3)
	wordCount := len(strings.Fields(qualityText))
	score := scoreTextQualityFallback(qualityText, wordCount, 65)

	if score.Score < 0.6 {
		t.Errorf("Expected good score for quality content, got %.2f", score.Score)
	}

	if !containsStringSlice(score.QualityIndicators, "academic_language") {
		t.Errorf("Expected academic_language in quality indicators, got: %v", score.QualityIndicators)
	}

	if !score.IsRecommended {
		t.Error("Expected quality content to be recommended")
	}

	if !strings.Contains(score.Reason, "Rule-based") {
		t.Errorf("Expected reason to mention rule-based assessment, got: %s", score.Reason)
	}
}

// TestScoreTextQualityFallbackExcessiveCaps tests fallback scoring for excessive capitalization
func TestScoreTextQualityFallbackExcessiveCaps(t *testing.T) {
	capsText := "THIS IS ALL CAPS TEXT SHOUTING AT THE READER ALL THE TIME VERY LOUD AND ANNOYING"
	wordCount := len(strings.Fields(capsText))
	score := scoreTextQualityFallback(capsText, wordCount, 50)

	if score.Score >= 0.5 {
		t.Errorf("Expected low score for excessive caps, got %.2f", score.Score)
	}

	if !containsStringSlice(score.ProblemsDetected, "excessive_capitalization") {
		t.Errorf("Expected excessive_capitalization in problems, got: %v", score.ProblemsDetected)
	}
}

// TestScoreTextQualityFallbackGibberish tests fallback scoring for gibberish content
func TestScoreTextQualityFallbackGibberish(t *testing.T) {
	gibberishText := "aaaaa bbbbb ccccc ddddd eeeee fffff ggggg hhhhh iiiii jjjjj kkkkk lllll mmmmm nnnnn"
	wordCount := len(strings.Fields(gibberishText))
	score := scoreTextQualityFallback(gibberishText, wordCount, 50)

	if score.Score >= 0.4 {
		t.Errorf("Expected low score for gibberish, got %.2f", score.Score)
	}

	if !containsStringSlice(score.ProblemsDetected, "excessive_character_repetition") {
		t.Errorf("Expected excessive_character_repetition in problems, got: %v", score.ProblemsDetected)
	}

	if !containsStringSlice(score.Categories, "incoherent") {
		t.Errorf("Expected 'incoherent' category, got: %v", score.Categories)
	}
}

// TestAnalyzeWithFallbackScoring tests that analysis works with fallback scoring when Ollama is down
func TestAnalyzeWithFallbackScoring(t *testing.T) {
	// Create analyzer WITHOUT Ollama client (will use fallback)
	a := New()

	text := `This is a well-written article about important research findings. The study demonstrates clear evidence of significant results.
	Furthermore, the data shows consistent patterns across multiple trials. These findings suggest that the hypothesis is supported by empirical evidence.
	However, additional research may be needed to confirm these results. The implications of this work are far-reaching and could impact future studies.
	In conclusion, this research contributes valuable insights to the field. The methodology was rigorous and the analysis was thorough.`

	metadata := a.Analyze(text)

	// Verify quality score is present (from fallback)
	if metadata.QualityScore == nil {
		t.Fatal("Expected QualityScore to be present from fallback scoring")
	}

	// Score should be decent for quality content
	if metadata.QualityScore.Score < 0.5 {
		t.Errorf("Expected reasonable fallback score for good content, got %.2f", metadata.QualityScore.Score)
	}

	// Reason should indicate rule-based assessment
	if !strings.Contains(metadata.QualityScore.Reason, "Rule-based") {
		t.Errorf("Expected reason to indicate rule-based fallback, got: %s", metadata.QualityScore.Reason)
	}

	// Categories should not be empty
	if len(metadata.QualityScore.Categories) == 0 {
		t.Error("Expected categories from fallback scoring")
	}

	// Verify AIUsed is false for rule-based fallback
	if metadata.QualityScore.AIUsed {
		t.Error("Expected AIUsed to be false for rule-based fallback")
	}

	t.Logf("✓ Fallback quality score: %.2f (recommended: %v, ai_used: %v)",
		metadata.QualityScore.Score, metadata.QualityScore.IsRecommended, metadata.QualityScore.AIUsed)
}

// TestScoreTextQualityDisconnectedHeadlines tests that disconnected news headlines are detected
func TestScoreTextQualityDisconnectedHeadlines(t *testing.T) {
	text := `Gaza doctors struggle to investigate 'signs of torture' on unnamed dead returned by Israel.
	Vance and Rubio criticise Israeli parliament's vote on West Bank annexation.
	New images show Israeli control line deeper into Gaza than expected.
	UN's top court says Israel obliged to allow UN aid into Gaza.
	'Fatal combination' of disease, injuries and famine in Gaza is generational crisis, WHO tells BBC.
	Israel identifies bodies of two hostages returned by Hamas.
	Gaza ceasefire deal going better than expected, Vance says.
	Israel's 'yellow line' in Gaza gives Netanyahu room for manoeuvre.
	British officers sent to Israel to help monitor Gaza ceasefire.
	Hamas ruled Gaza with an iron rod - will it really give up control?`

	a := New()
	metadata := a.Analyze(text)

	if metadata.QualityScore == nil {
		t.Fatal("Expected QualityScore to be present")
	}

	// Should score very poorly due to lack of flow
	if metadata.QualityScore.Score >= 0.35 {
		t.Errorf("Expected low score for disconnected headlines, got %.2f", metadata.QualityScore.Score)
	}

	// Should detect the issues
	if !containsStringSlice(metadata.QualityScore.ProblemsDetected, "disconnected_sentences") &&
		!containsStringSlice(metadata.QualityScore.ProblemsDetected, "no_flow") &&
		!containsStringSlice(metadata.QualityScore.ProblemsDetected, "poor_continuity") {
		t.Errorf("Expected disconnected/flow problems to be detected, got: %v", metadata.QualityScore.ProblemsDetected)
	}

	if !containsStringSlice(metadata.QualityScore.Categories, "incoherent") &&
		!containsStringSlice(metadata.QualityScore.Categories, "list_like") {
		t.Errorf("Expected 'incoherent' or 'list_like' category, got: %v", metadata.QualityScore.Categories)
	}

	t.Logf("✓ Disconnected headlines scored %.2f with problems: %v",
		metadata.QualityScore.Score, metadata.QualityScore.ProblemsDetected)
}

func containsStringSlice(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "lowercase conversion",
			input:    "Machine Learning",
			expected: "machine-learning",
		},
		{
			name:     "underscore to hyphen",
			input:    "climate_change",
			expected: "climate-change",
		},
		{
			name:     "multiple spaces",
			input:    "New  York  City",
			expected: "new-york-city",
		},
		{
			name:     "mixed spaces and underscores",
			input:    "Social_Media Platform",
			expected: "social-media-platform",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  einstein  ",
			expected: "einstein",
		},
		{
			name:     "multiple consecutive hyphens",
			input:    "foo--bar---baz",
			expected: "foo-bar-baz",
		},
		{
			name:     "already normalized",
			input:    "machine-learning",
			expected: "machine-learning",
		},
		{
			name:     "single word uppercase",
			input:    "TECHNOLOGY",
			expected: "technology",
		},
		{
			name:     "readability level",
			input:    "very_easy",
			expected: "very-easy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTag(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func BenchmarkAnalyze(b *testing.B) {
	a := New()
	text := `Climate change is a pressing global issue. Scientists have documented a 1.1°C increase in global temperatures since 1880.
	The effects are devastating: rising sea levels, extreme weather events, and loss of biodiversity.
	According to recent studies, we need to reduce carbon emissions by 45% by 2030 to avoid catastrophic consequences.
	Many experts believe this is achievable with renewable energy adoption.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Analyze(text)
	}
}
