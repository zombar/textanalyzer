package analyzer

import (
	"log"
	"regexp"
	"strings"
)

// ParagraphScore represents the quality score for a paragraph
type ParagraphScore struct {
	Text             string
	Score            float64
	WordCount        int
	LinkDensity      float64
	StopwordRatio    float64
	NamedEntityCount int
	CapitalizedRatio float64
	AvgWordLength    float64
	HasImageMarkers  bool
	IsBoilerplate    bool
	Reasons          []string
}

// cleanTextOffline performs sophisticated offline text cleaning using heuristics
// This provides a clean article text that can be used as a template for AI enhancement
func (a *Analyzer) cleanTextOffline(text string) string {
	log.Println("Starting offline text cleaning with advanced heuristics...")

	// Split into paragraphs
	paragraphs := splitIntoParagraphs(text)
	if len(paragraphs) == 0 {
		log.Println("No paragraphs found, returning original text")
		return text
	}

	log.Printf("Analyzing %d paragraphs...", len(paragraphs))

	// Score each paragraph
	scores := make([]ParagraphScore, 0, len(paragraphs))
	for _, para := range paragraphs {
		score := a.scoreParagraph(para)
		scores = append(scores, score)
	}

	// Calculate threshold - keep paragraphs above median score
	threshold := calculateDynamicThreshold(scores)
	log.Printf("Paragraph quality threshold: %.2f", threshold)

	// Filter paragraphs and reconstruct clean text
	cleanParagraphs := make([]string, 0, len(paragraphs))
	kept := 0
	removed := 0

	for i, score := range scores {
		if score.Score >= threshold && !score.IsBoilerplate {
			cleanParagraphs = append(cleanParagraphs, score.Text)
			kept++
		} else {
			removed++
			if len(score.Reasons) > 0 {
				log.Printf("Removed paragraph %d (score=%.2f): %s", i+1, score.Score, strings.Join(score.Reasons, ", "))
			}
		}
	}

	log.Printf("Offline cleaning complete: kept %d paragraphs, removed %d", kept, removed)

	cleanText := strings.Join(cleanParagraphs, "\n\n")
	return cleanText
}

// scoreParagraph scores a paragraph based on multiple quality factors
func (a *Analyzer) scoreParagraph(para string) ParagraphScore {
	score := ParagraphScore{
		Text:    para,
		Score:   0.5, // Start neutral
		Reasons: []string{},
	}

	// Quick reject: empty or too short
	trimmed := strings.TrimSpace(para)
	if len(trimmed) < 20 {
		score.Score = 0.0
		score.Reasons = append(score.Reasons, "too_short")
		return score
	}

	words := strings.Fields(para)
	score.WordCount = len(words)

	// Factor 1: Word count (sweet spot is 20-200 words per paragraph)
	if score.WordCount < 10 {
		score.Score -= 0.3
		score.Reasons = append(score.Reasons, "very_few_words")
	} else if score.WordCount >= 20 && score.WordCount <= 200 {
		score.Score += 0.2
		score.Reasons = append(score.Reasons, "good_length")
	} else if score.WordCount > 300 {
		score.Score -= 0.1
		score.Reasons = append(score.Reasons, "very_long")
	}

	// Factor 2: Link density (URLs and link-like patterns)
	linkCount := strings.Count(strings.ToLower(para), "http://") +
		strings.Count(strings.ToLower(para), "https://") +
		strings.Count(strings.ToLower(para), "www.") +
		strings.Count(para, "→") + // Common navigation arrow
		strings.Count(para, "»")   // Common navigation marker

	score.LinkDensity = float64(linkCount) / float64(score.WordCount)
	if score.LinkDensity > 0.1 {
		score.Score -= 0.4
		score.Reasons = append(score.Reasons, "high_link_density")
	}

	// Factor 3: Stopword ratio (natural text has 40-60% stopwords)
	stopwordCount := 0
	for _, word := range words {
		if a.stopWords[strings.ToLower(word)] {
			stopwordCount++
		}
	}
	score.StopwordRatio = float64(stopwordCount) / float64(score.WordCount)
	if score.StopwordRatio >= 0.35 && score.StopwordRatio <= 0.65 {
		score.Score += 0.15
		score.Reasons = append(score.Reasons, "natural_stopword_ratio")
	} else if score.StopwordRatio < 0.25 {
		score.Score -= 0.2
		score.Reasons = append(score.Reasons, "low_stopwords")
	}

	// Factor 4: Named entities (article content has more proper nouns)
	entities := extractNamedEntities(para)
	score.NamedEntityCount = len(entities)
	if score.NamedEntityCount >= 2 {
		score.Score += 0.1
		score.Reasons = append(score.Reasons, "has_named_entities")
	}

	// Factor 5: Average word length (articles have balanced word length)
	totalLength := 0
	for _, word := range words {
		totalLength += len(word)
	}
	score.AvgWordLength = float64(totalLength) / float64(score.WordCount)
	if score.AvgWordLength >= 4.0 && score.AvgWordLength <= 6.0 {
		score.Score += 0.1
		score.Reasons = append(score.Reasons, "balanced_word_length")
	}

	// Factor 6: Image markers (captions, credits, attributions)
	imageMarkers := []string{
		"image source:", "photo by:", "credit:", "getty images",
		"photograph:", "photographer:", "©", "copyright",
		"image caption:", "picture:", "courtesy of",
		"[image:", "[photo:", "source:", "via:",
	}
	lowerPara := strings.ToLower(para)
	for _, marker := range imageMarkers {
		if strings.Contains(lowerPara, marker) {
			score.HasImageMarkers = true
			score.Score -= 0.4
			score.Reasons = append(score.Reasons, "image_attribution")
			break
		}
	}

	// Factor 7: Boilerplate detection (navigation, ads, disclaimers)
	boilerplatePatterns := []string{
		"click here", "read more", "subscribe", "sign up", "newsletter",
		"share this", "follow us", "connect with us", "related articles",
		"you may also like", "recommended for you", "advertisement",
		"sponsored content", "cookie policy", "privacy policy",
		"terms of service", "all rights reserved", "view comments",
		"post comment", "log in to", "register now", "free trial",
		"buy now", "shop now", "add to cart", "learn more about",
		"trending now", "popular posts", "recent posts", "categories:",
		"tags:", "filed under:", "posted in:", "previous article",
		"next article", "back to top", "skip to content",
	}
	for _, pattern := range boilerplatePatterns {
		if strings.Contains(lowerPara, pattern) {
			score.IsBoilerplate = true
			score.Score -= 0.5
			score.Reasons = append(score.Reasons, "boilerplate_pattern")
			break
		}
	}

	// Factor 8: Capitalization ratio (headers/navigation often all caps)
	upperCount := 0
	lowerCount := 0
	for _, r := range para {
		if r >= 'A' && r <= 'Z' {
			upperCount++
		} else if r >= 'a' && r <= 'z' {
			lowerCount++
		}
	}
	if upperCount+lowerCount > 0 {
		score.CapitalizedRatio = float64(upperCount) / float64(upperCount+lowerCount)
		if score.CapitalizedRatio > 0.5 {
			score.Score -= 0.3
			score.Reasons = append(score.Reasons, "excessive_caps")
		}
	}

	// Factor 9: Punctuation overload (spam indicators)
	punctCount := strings.Count(para, "!") + strings.Count(para, "?") +
		strings.Count(para, "*") + strings.Count(para, "#")
	if punctCount > score.WordCount/5 {
		score.Score -= 0.2
		score.Reasons = append(score.Reasons, "excessive_punctuation")
	}

	// Factor 10: List-like structure (disconnected bullet points)
	if strings.HasPrefix(trimmed, "•") || strings.HasPrefix(trimmed, "-") ||
		strings.HasPrefix(trimmed, "*") || regexp.MustCompile(`^\d+\.`).MatchString(trimmed) {
		// It's a list item - only bad if very short
		if score.WordCount < 15 {
			score.Score -= 0.2
			score.Reasons = append(score.Reasons, "short_list_item")
		}
	}

	// Factor 11: Social media patterns
	if strings.Contains(lowerPara, "tweet") || strings.Contains(lowerPara, "facebook") ||
		strings.Contains(lowerPara, "instagram") || strings.Contains(lowerPara, "linkedin") {
		// Only penalize if it's clearly a sharing prompt, not content about social media
		if strings.Contains(lowerPara, "share on") || strings.Contains(lowerPara, "follow on") {
			score.Score -= 0.3
			score.Reasons = append(score.Reasons, "social_media_prompt")
		}
	}

	// Factor 12: Date/timestamp patterns (often navigation)
	datePattern := regexp.MustCompile(`(?i)posted on|published on|updated on|last modified|^\w+\s+\d{1,2},\s+\d{4}`)
	if datePattern.MatchString(para) && score.WordCount < 20 {
		score.Score -= 0.2
		score.Reasons = append(score.Reasons, "metadata_line")
	}

	// Factor 13: Author bylines (not main content)
	authorPattern := regexp.MustCompile(`(?i)^by\s+[A-Z][a-z]+|^written by|^author:`)
	if authorPattern.MatchString(trimmed) && score.WordCount < 15 {
		score.Score -= 0.2
		score.Reasons = append(score.Reasons, "author_byline")
	}

	// Ensure score is within bounds
	if score.Score < 0.0 {
		score.Score = 0.0
	}
	if score.Score > 1.0 {
		score.Score = 1.0
	}

	return score
}

// splitIntoParagraphs splits text into paragraphs intelligently
func splitIntoParagraphs(text string) []string {
	// Split by double newlines (standard paragraph separator)
	paragraphs := strings.Split(text, "\n\n")

	// Also split by single newlines if paragraphs are too long
	result := make([]string, 0, len(paragraphs))
	for _, para := range paragraphs {
		trimmed := strings.TrimSpace(para)
		if trimmed == "" {
			continue
		}

		// If paragraph is very long, try splitting by single newline
		if len(trimmed) > 1000 {
			subParas := strings.Split(para, "\n")
			for _, subPara := range subParas {
				subTrimmed := strings.TrimSpace(subPara)
				if subTrimmed != "" {
					result = append(result, subTrimmed)
				}
			}
		} else {
			result = append(result, trimmed)
		}
	}

	return result
}

// calculateDynamicThreshold calculates a threshold based on score distribution
func calculateDynamicThreshold(scores []ParagraphScore) float64 {
	if len(scores) == 0 {
		return 0.5
	}

	// Calculate median score
	sortedScores := make([]float64, len(scores))
	for i, s := range scores {
		sortedScores[i] = s.Score
	}

	// Simple bubble sort for median calculation
	for i := 0; i < len(sortedScores); i++ {
		for j := i + 1; j < len(sortedScores); j++ {
			if sortedScores[i] > sortedScores[j] {
				sortedScores[i], sortedScores[j] = sortedScores[j], sortedScores[i]
			}
		}
	}

	median := sortedScores[len(sortedScores)/2]

	// Use median as base, but ensure we don't set threshold too high
	threshold := median
	if threshold > 0.6 {
		threshold = 0.6
	}
	if threshold < 0.3 {
		threshold = 0.3
	}

	return threshold
}
