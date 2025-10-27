package analyzer

import (
	"testing"
)

/**
 * Regression Test Suite: Tag Merging
 *
 * Purpose: Verify that computed tags and AI tags are merged (not replaced)
 *
 * Previous behavior:
 * - Used AI tags OR computed tags (as fallback)
 * - Lost computed tags when AI tags were generated
 *
 * New behavior:
 * - Merge both AI tags and computed tags
 * - Remove duplicates
 * - Preserve both rule-based and AI-generated categorization
 *
 * These tests verify the tag merging logic works correctly.
 * Since Ollama may not be available in test environment, these tests
 * focus on verifying computed tag generation which is always available.
 */

// Test that computed tags are generated based on text characteristics
func TestTagMerging_ComputedTagsGenerated(t *testing.T) {
	analyzer := New()

	// Test text with known characteristics
	text := "This is a positive article about technology."

	metadata := analyzer.Analyze(text)

	// Verify tags were generated
	if len(metadata.Tags) == 0 {
		t.Fatal("Expected computed tags to be generated, got none")
	}

	// Verify computed tags include expected values:
	// - sentiment tag (positive/neutral/negative)
	// - length tag (short/medium/long)
	// - readability tag

	hasSentimentTag := false
	hasLengthTag := false

	for _, tag := range metadata.Tags {
		if tag == "positive" || tag == "neutral" || tag == "negative" {
			hasSentimentTag = true
		}
		if tag == "short" || tag == "medium" || tag == "long" {
			hasLengthTag = true
		}
	}

	if !hasSentimentTag {
		t.Errorf("Expected sentiment tag (positive/neutral/negative), got: %v", metadata.Tags)
	}
	if !hasLengthTag {
		t.Errorf("Expected length tag (short/medium/long), got: %v", metadata.Tags)
	}

	t.Logf("✓ Computed tags generated: %v", metadata.Tags)
}

// Test that tags include sentiment analysis
func TestTagMerging_SentimentTagIncluded(t *testing.T) {
	analyzer := New()

	tests := []struct {
		name              string
		text              string
		expectedSentiment string
	}{
		{
			name:              "positive text",
			text:              "This is wonderful! Everything is amazing and fantastic.",
			expectedSentiment: "positive",
		},
		{
			name:              "negative text",
			text:              "This is terrible. Everything is awful and horrible.",
			expectedSentiment: "negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := analyzer.Analyze(tt.text)

			hasSentiment := false
			for _, tag := range metadata.Tags {
				if tag == tt.expectedSentiment {
					hasSentiment = true
					break
				}
			}

			if !hasSentiment {
				t.Errorf("Expected '%s' sentiment tag, got: %v", tt.expectedSentiment, metadata.Tags)
			}
		})
	}
}

// Test that length-based tags are generated
func TestTagMerging_LengthTagsGenerated(t *testing.T) {
	analyzer := New()

	tests := []struct {
		name           string
		text           string
		expectedLength string
	}{
		{
			name:           "short text",
			text:           "This is short.",
			expectedLength: "short",
		},
		{
			name: "medium text",
			text: `This is a medium length text with multiple sentences and paragraphs that contains enough words to be classified as medium length content according to the classification system.
We need to have several sentences to reach this classification level properly.
The analyzer will categorize this appropriately based on word count thresholds that have been configured in the system settings.
Medium text should have between one hundred and five hundred words total to meet the criteria.
This text continues to add more words to reach that important threshold.
We are getting closer to the required word count now with each additional sentence that we write here.
The content needs to be substantial enough to demonstrate medium length classification behavior.
Adding more sentences helps us reach the minimum word count required for this test case to pass successfully.
This should now be sufficient to be classified as medium length text content.
The analyzer processes all of these words and determines the appropriate length category.`,
			expectedLength: "medium",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := analyzer.Analyze(tt.text)

			hasLength := false
			for _, tag := range metadata.Tags {
				if tag == tt.expectedLength {
					hasLength = true
					break
				}
			}

			if !hasLength {
				t.Errorf("Expected '%s' length tag, got: %v", tt.expectedLength, metadata.Tags)
			}
		})
	}
}

// Test that no duplicate tags are generated
func TestTagMerging_NoDuplicates(t *testing.T) {
	analyzer := New()

	text := "This is a positive message about technology and innovation."

	metadata := analyzer.Analyze(text)

	// Count occurrences of each tag
	tagCounts := make(map[string]int)
	for _, tag := range metadata.Tags {
		tagCounts[tag]++
	}

	// Verify no duplicates
	for tag, count := range tagCounts {
		if count > 1 {
			t.Errorf("Tag '%s' appears %d times (expected 1). All tags: %v", tag, count, metadata.Tags)
		}
	}

	t.Logf("✓ No duplicate tags: %v", metadata.Tags)
}

// Test that computed tags include readability level
func TestTagMerging_ReadabilityTagIncluded(t *testing.T) {
	analyzer := New()

	text := "This is a simple and easy to understand message."

	metadata := analyzer.Analyze(text)

	// Readability level should be included as a tag
	// Could be: "very-easy", "fairly-easy", "easy", "fairly-difficult", "difficult", "very-difficult"
	hasReadability := false
	readabilityTags := []string{"very-easy", "fairly-easy", "easy", "fairly-difficult", "difficult", "very-difficult"}

	for _, tag := range metadata.Tags {
		for _, readabilityTag := range readabilityTags {
			if tag == readabilityTag {
				hasReadability = true
				break
			}
		}
		if hasReadability {
			break
		}
	}

	if !hasReadability {
		t.Errorf("Expected readability tag, got: %v", metadata.Tags)
	}

	t.Logf("✓ Readability tag included: %v", metadata.Tags)
}
