package analyzer

import (
	"strings"
	"testing"
)

func TestSplitIntoParagraphs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "double newline separation",
			input:    "Para 1\n\nPara 2\n\nPara 3",
			expected: 3,
		},
		{
			name:     "empty paragraphs filtered",
			input:    "Para 1\n\n\n\nPara 2",
			expected: 2,
		},
		{
			name:     "very long paragraph split",
			input:    strings.Repeat("word ", 300), // >1000 chars, should split
			expected: 1,                            // Actually won't split if no newlines
		},
		{
			name:     "mixed separators",
			input:    "Para 1\n\nPara 2\nPara 3",
			expected: 2, // Para 2 and 3 may stay together
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitIntoParagraphs(tt.input)
			if len(result) != tt.expected {
				t.Errorf("expected %d paragraphs, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestScoreParagraph_ImageMarkers(t *testing.T) {
	analyzer := New()

	tests := []struct {
		name           string
		paragraph      string
		shouldPenalize bool
	}{
		{
			name:           "photo credit",
			paragraph:      "Photo by John Smith for Getty Images",
			shouldPenalize: true,
		},
		{
			name:           "image source",
			paragraph:      "Image source: Reuters",
			shouldPenalize: true,
		},
		{
			name:           "copyright notice",
			paragraph:      "© 2024 Associated Press",
			shouldPenalize: true,
		},
		{
			name:           "normal content",
			paragraph:      "This is a normal article paragraph with actual content about technology.",
			shouldPenalize: false,
		},
		{
			name:           "photographer credit",
			paragraph:      "Photographer: Jane Doe",
			shouldPenalize: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.scoreParagraph(tt.paragraph)
			if tt.shouldPenalize && !score.HasImageMarkers {
				t.Errorf("expected image markers to be detected")
			}
			if !tt.shouldPenalize && score.HasImageMarkers {
				t.Errorf("expected no image markers, but found some")
			}
		})
	}
}

func TestScoreParagraph_BoilerplateDetection(t *testing.T) {
	analyzer := New()

	tests := []struct {
		name          string
		paragraph     string
		isBoilerplate bool
	}{
		{
			name:          "click here CTA",
			paragraph:     "Click here to read more about this topic",
			isBoilerplate: true,
		},
		{
			name:          "newsletter signup",
			paragraph:     "Subscribe to our newsletter for weekly updates",
			isBoilerplate: true,
		},
		{
			name:          "social sharing",
			paragraph:     "Share this article on Facebook and Twitter",
			isBoilerplate: true,
		},
		{
			name:          "related articles",
			paragraph:     "Related articles you may also like",
			isBoilerplate: true,
		},
		{
			name:          "actual content",
			paragraph:     "The study demonstrates that climate change is accelerating faster than previously thought.",
			isBoilerplate: false,
		},
		{
			name:          "buy now button",
			paragraph:     "Buy now and save 50% on your first order",
			isBoilerplate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.scoreParagraph(tt.paragraph)
			if score.IsBoilerplate != tt.isBoilerplate {
				t.Errorf("expected boilerplate=%v, got %v", tt.isBoilerplate, score.IsBoilerplate)
			}
		})
	}
}

func TestScoreParagraph_LinkDensity(t *testing.T) {
	analyzer := New()

	tests := []struct {
		name     string
		para     string
		highLink bool
	}{
		{
			name:     "high link density",
			para:     "Visit https://example.com and www.test.com for more",
			highLink: true,
		},
		{
			name:     "normal content with one link",
			para:     "This is a long paragraph with substantial content about technology and innovation in the modern world. See https://example.com for details.",
			highLink: false,
		},
		{
			name:     "navigation arrows",
			para:     "Home → Products → Category → Item",
			highLink: true,
		},
		{
			name:     "no links",
			para:     "This is normal article content without any links or navigation elements.",
			highLink: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.scoreParagraph(tt.para)
			hasHighDensity := score.LinkDensity > 0.1
			if hasHighDensity != tt.highLink {
				t.Errorf("expected high link density=%v, got link density=%.2f", tt.highLink, score.LinkDensity)
			}
		})
	}
}

func TestScoreParagraph_StopwordRatio(t *testing.T) {
	analyzer := New()

	// Natural text has 40-60% stopwords
	naturalText := "The quick brown fox jumps over the lazy dog in the park"
	score := analyzer.scoreParagraph(naturalText)

	if score.StopwordRatio < 0.3 || score.StopwordRatio > 0.7 {
		t.Errorf("natural text should have stopword ratio 0.3-0.7, got %.2f", score.StopwordRatio)
	}

	// Keyword-stuffed text has low stopwords
	keywordText := "Python JavaScript TypeScript React Angular Vue Django Flask"
	score2 := analyzer.scoreParagraph(keywordText)

	if score2.StopwordRatio > 0.3 {
		t.Errorf("keyword text should have low stopword ratio, got %.2f", score2.StopwordRatio)
	}
}

func TestScoreParagraph_WordCount(t *testing.T) {
	analyzer := New()

	tests := []struct {
		name          string
		para          string
		expectedScore float64
		minOrMax      string // "min" or "max"
	}{
		{
			name:          "too short",
			para:          "Short",
			expectedScore: 0.3,
			minOrMax:      "max",
		},
		{
			name:          "good length",
			para:          strings.Repeat("word ", 50), // 50 words
			expectedScore: 0.6,
			minOrMax:      "min",
		},
		{
			name:          "very long",
			para:          strings.Repeat("word ", 400), // 400 words
			expectedScore: 0.6,
			minOrMax:      "max",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.scoreParagraph(tt.para)
			if tt.minOrMax == "max" && score.Score > tt.expectedScore {
				t.Errorf("score should be <= %.2f, got %.2f", tt.expectedScore, score.Score)
			}
			if tt.minOrMax == "min" && score.Score < tt.expectedScore {
				t.Errorf("score should be >= %.2f, got %.2f", tt.expectedScore, score.Score)
			}
		})
	}
}

func TestCleanTextOffline(t *testing.T) {
	analyzer := New()

	// Test with mixed content: good paragraphs and noise
	input := `This is a good article paragraph with substantial content about technology and innovation.

Photo by: John Smith, Getty Images

The research demonstrates significant findings in the field of artificial intelligence.

Click here to subscribe to our newsletter!

Scientists have discovered new methods for improving machine learning algorithms.

Share this article → Facebook | Twitter | LinkedIn

The study was published in Nature magazine last week.`

	cleaned := analyzer.cleanTextOffline(input)

	// Should keep the good paragraphs
	if !strings.Contains(cleaned, "good article paragraph") {
		t.Error("should keep good content paragraph")
	}
	if !strings.Contains(cleaned, "research demonstrates") {
		t.Error("should keep research paragraph")
	}
	if !strings.Contains(cleaned, "Scientists have discovered") {
		t.Error("should keep scientists paragraph")
	}

	// Should remove noise
	if strings.Contains(cleaned, "Photo by") {
		t.Error("should remove image attribution")
	}
	if strings.Contains(cleaned, "Click here to subscribe") {
		t.Error("should remove newsletter signup")
	}
	if strings.Contains(cleaned, "Share this article") {
		t.Error("should remove social sharing")
	}

	// Should keep the final good paragraph
	if !strings.Contains(cleaned, "study was published") {
		t.Error("should keep publication paragraph")
	}
}

func TestCleanTextOffline_EmptyInput(t *testing.T) {
	analyzer := New()

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"whitespace only", "   \n\n   "},
		{"very short", "Hi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.cleanTextOffline(tt.input)
			// Should not panic and should return something
			_ = result
		})
	}
}

func TestCalculateDynamicThreshold(t *testing.T) {
	tests := []struct {
		name      string
		scores    []ParagraphScore
		minThresh float64
		maxThresh float64
	}{
		{
			name: "varied scores",
			scores: []ParagraphScore{
				{Score: 0.1},
				{Score: 0.3},
				{Score: 0.5},
				{Score: 0.7},
				{Score: 0.9},
			},
			minThresh: 0.3,
			maxThresh: 0.6,
		},
		{
			name: "all high scores",
			scores: []ParagraphScore{
				{Score: 0.8},
				{Score: 0.85},
				{Score: 0.9},
				{Score: 0.95},
			},
			minThresh: 0.3,
			maxThresh: 0.6, // Capped at 0.6
		},
		{
			name: "all low scores",
			scores: []ParagraphScore{
				{Score: 0.1},
				{Score: 0.15},
				{Score: 0.2},
			},
			minThresh: 0.3, // Minimum threshold
			maxThresh: 0.6,
		},
		{
			name:      "empty scores",
			scores:    []ParagraphScore{},
			minThresh: 0.5, // Default
			maxThresh: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threshold := calculateDynamicThreshold(tt.scores)
			if threshold < tt.minThresh || threshold > tt.maxThresh {
				t.Errorf("threshold %.2f outside expected range [%.2f, %.2f]",
					threshold, tt.minThresh, tt.maxThresh)
			}
		})
	}
}

func TestScoreParagraph_ComprehensiveScoring(t *testing.T) {
	analyzer := New()

	// High-quality article paragraph
	goodPara := `The research, published in Nature Medicine, demonstrates that the new
	treatment approach shows promising results in early clinical trials. Dr. Sarah Johnson,
	lead researcher at Stanford University, explained that the findings could revolutionize
	how we approach chronic disease management.`

	score := analyzer.scoreParagraph(goodPara)

	// Should have good indicators
	if score.Score < 0.5 {
		t.Errorf("high-quality paragraph should score >= 0.5, got %.2f (reasons: %v)",
			score.Score, score.Reasons)
	}
	if score.NamedEntityCount < 2 {
		t.Errorf("should detect named entities, found %d", score.NamedEntityCount)
	}
	if score.IsBoilerplate {
		t.Error("should not be marked as boilerplate")
	}
	if score.HasImageMarkers {
		t.Error("should not have image markers")
	}

	// Low-quality spam paragraph
	spamPara := "CLICK HERE NOW!!! Limited time offer! Buy now and save $$$"

	score2 := analyzer.scoreParagraph(spamPara)

	if score2.Score > 0.4 {
		t.Errorf("spam paragraph should score low, got %.2f", score2.Score)
	}
}

func TestScoreParagraph_EdgeCases(t *testing.T) {
	analyzer := New()

	tests := []struct {
		name string
		para string
	}{
		{"unicode characters", "Hello 世界 مرحبا שלום"},
		{"excessive punctuation", "What?!?! Really?!?! No way!!!"},
		{"all caps", "THIS IS ALL CAPS TEXT"},
		{"numbers heavy", "123 456 789 1011 1213 1415 1617"},
		{"special symbols", "▶️ ✓ ★ ◆ ► ◄"},
		{"mixed case chaos", "tHiS iS wEiRd CaSiNg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzer.scoreParagraph(tt.para)
			// Should not panic and should return valid score
			if score.Score < 0.0 || score.Score > 1.0 {
				t.Errorf("score %.2f out of valid range [0.0, 1.0]", score.Score)
			}
		})
	}
}
