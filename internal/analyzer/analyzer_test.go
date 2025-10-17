package analyzer

import (
	"strings"
	"testing"
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
	a := New()
	
	tests := []struct {
		name             string
		input            string
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
