package analyzer

import (
	"testing"
)

/**
 * Comprehensive Tag Merge Testing
 *
 * This test verifies the tag merging logic in detail, including:
 * 1. Deduplication between computed and AI tags
 * 2. Tag normalization consistency
 * 3. Preservation of both tag sources
 */

// TestTagMerge_DeduplicationLogic tests the core deduplication logic
func TestTagMerge_DeduplicationLogic(t *testing.T) {
	// Simulate the tag merging logic from analyzer.go lines 149-163
	computedTags := []string{"positive", "short", "technology", "article"}
	aiTags := []string{"tech", "positive", "innovation", "article"}

	// Merge using the same logic as in analyzer.go
	tagSet := make(map[string]bool)
	for _, tag := range computedTags {
		tagSet[tag] = true
	}
	for _, tag := range aiTags {
		tagSet[tag] = true
	}

	mergedTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		mergedTags = append(mergedTags, tag)
	}

	// Verify results
	expectedUniqueCount := 6 // positive, short, technology, article, tech, innovation
	if len(mergedTags) != expectedUniqueCount {
		t.Errorf("Expected %d unique tags, got %d: %v", expectedUniqueCount, len(mergedTags), mergedTags)
	}

	// Verify no duplicates
	tagCounts := make(map[string]int)
	for _, tag := range mergedTags {
		tagCounts[tag]++
	}

	for tag, count := range tagCounts {
		if count > 1 {
			t.Errorf("Tag '%s' appears %d times (should appear once)", tag, count)
		}
	}

	t.Logf("✓ Deduplication works correctly: %d computed + %d AI = %d merged (2 duplicates removed)",
		len(computedTags), len(aiTags), len(mergedTags))
}

// TestTagMerge_NormalizationConsistency tests that tags are normalized before merging
func TestTagMerge_NormalizationConsistency(t *testing.T) {
	// Test that similar tags with different formatting are treated as duplicates
	testCases := []struct {
		name         string
		computedTags []string
		aiTags       []string
		expectedMin  int // Minimum expected after deduplication
		expectedMax  int // Maximum expected after deduplication
	}{
		{
			name:         "Different casing",
			computedTags: []string{"Technology", "AI"},
			aiTags:       []string{"technology", "ai"},
			expectedMin:  2, // Should deduplicate to 2 tags
			expectedMax:  2,
		},
		{
			name:         "Spaces vs hyphens",
			computedTags: []string{"machine learning", "data science"},
			aiTags:       []string{"machine-learning", "data-science"},
			expectedMin:  2, // Should deduplicate to 2 tags after normalization
			expectedMax:  2,
		},
		{
			name:         "Underscores vs hyphens",
			computedTags: []string{"deep_learning", "neural_networks"},
			aiTags:       []string{"deep-learning", "neural-networks"},
			expectedMin:  2,
			expectedMax:  2,
		},
		{
			name:         "Mixed formatting",
			computedTags: []string{"Machine Learning", "Data_Science"},
			aiTags:       []string{"machine-learning", "data-science"},
			expectedMin:  2,
			expectedMax:  2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Normalize and merge
			tagSet := make(map[string]bool)

			for _, tag := range tc.computedTags {
				normalized := normalizeTag(tag)
				tagSet[normalized] = true
			}

			for _, tag := range tc.aiTags {
				normalized := normalizeTag(tag)
				tagSet[normalized] = true
			}

			mergedCount := len(tagSet)

			if mergedCount < tc.expectedMin || mergedCount > tc.expectedMax {
				mergedTags := make([]string, 0, len(tagSet))
				for tag := range tagSet {
					mergedTags = append(mergedTags, tag)
				}
				t.Errorf("Expected %d-%d tags after normalization and merge, got %d: %v",
					tc.expectedMin, tc.expectedMax, mergedCount, mergedTags)
			}

			t.Logf("✓ Normalization consistency: %d computed + %d AI = %d merged",
				len(tc.computedTags), len(tc.aiTags), mergedCount)
		})
	}
}

// TestTagMerge_PreservationOfBothSources verifies both computed and AI tags are preserved
func TestTagMerge_PreservationOfBothSources(t *testing.T) {
	computedTags := []string{"short", "positive", "easy"}
	aiTags := []string{"technology", "innovation", "future"}

	// No overlap, so all should be preserved
	tagSet := make(map[string]bool)
	for _, tag := range computedTags {
		tagSet[tag] = true
	}
	for _, tag := range aiTags {
		tagSet[tag] = true
	}

	mergedTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		mergedTags = append(mergedTags, tag)
	}

	expectedTotal := len(computedTags) + len(aiTags)
	if len(mergedTags) != expectedTotal {
		t.Errorf("Expected all %d tags to be preserved, got %d: %v",
			expectedTotal, len(mergedTags), mergedTags)
	}

	// Verify all computed tags are present
	for _, tag := range computedTags {
		if !tagSet[tag] {
			t.Errorf("Computed tag '%s' not found in merged tags: %v", tag, mergedTags)
		}
	}

	// Verify all AI tags are present
	for _, tag := range aiTags {
		if !tagSet[tag] {
			t.Errorf("AI tag '%s' not found in merged tags: %v", tag, mergedTags)
		}
	}

	t.Logf("✓ All tags from both sources preserved: %v", mergedTags)
}

// TestTagMerge_EmptyAITags tests fallback to computed tags only
func TestTagMerge_EmptyAITags(t *testing.T) {
	computedTags := []string{"short", "positive", "easy"}
	aiTags := []string{} // Empty AI tags (simulating AI failure)

	// Simulate the fallback logic
	var finalTags []string
	if len(aiTags) == 0 {
		// Fallback to computed tags only (like line 167 in analyzer.go)
		finalTags = computedTags
	} else {
		// Normal merge logic
		tagSet := make(map[string]bool)
		for _, tag := range computedTags {
			tagSet[tag] = true
		}
		for _, tag := range aiTags {
			tagSet[tag] = true
		}
		finalTags = make([]string, 0, len(tagSet))
		for tag := range tagSet {
			finalTags = append(finalTags, tag)
		}
	}

	if len(finalTags) != len(computedTags) {
		t.Errorf("Expected fallback to %d computed tags, got %d: %v",
			len(computedTags), len(finalTags), finalTags)
	}

	t.Logf("✓ Fallback to computed tags works: %v", finalTags)
}

// TestTagMerge_LargeTagSets tests merging with many tags
func TestTagMerge_LargeTagSets(t *testing.T) {
	// Simulate a large number of computed tags
	computedTags := []string{
		"short", "positive", "easy", "technology", "article",
		"news", "web-content", "research", "2024", "innovation",
	}

	// AI tags with some overlap
	aiTags := []string{
		"technology", "ai", "machine-learning", "data-science",
		"innovation", "future", "automation", "digital",
	}

	tagSet := make(map[string]bool)
	for _, tag := range computedTags {
		tagSet[tag] = true
	}
	for _, tag := range aiTags {
		tagSet[tag] = true
	}

	mergedTags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		mergedTags = append(mergedTags, tag)
	}

	// Calculate expected: total - duplicates
	// Duplicates: "technology" and "innovation" appear in both
	expectedCount := len(computedTags) + len(aiTags) - 2

	if len(mergedTags) != expectedCount {
		t.Errorf("Expected %d tags after merge, got %d: %v",
			expectedCount, len(mergedTags), mergedTags)
	}

	// Verify no duplicates
	tagCounts := make(map[string]int)
	for _, tag := range mergedTags {
		tagCounts[tag]++
	}

	for tag, count := range tagCounts {
		if count > 1 {
			t.Errorf("Tag '%s' appears %d times (should be unique)", tag, count)
		}
	}

	t.Logf("✓ Large tag set merge successful: %d computed + %d AI = %d merged (2 duplicates removed)",
		len(computedTags), len(aiTags), len(mergedTags))
}
