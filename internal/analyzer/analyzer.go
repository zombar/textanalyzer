package analyzer

import (
	"context"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/zombar/textanalyzer/internal/models"
	"github.com/zombar/textanalyzer/internal/ollama"
)

// Analyzer performs text analysis
type Analyzer struct {
	stopWords    map[string]bool
	ollamaClient *ollama.Client
}

// New creates a new Analyzer
func New() *Analyzer {
	return &Analyzer{
		stopWords: getStopWords(),
	}
}

// NewWithOllama creates a new Analyzer with Ollama integration
func NewWithOllama(ollamaClient *ollama.Client) *Analyzer {
	return &Analyzer{
		stopWords:    getStopWords(),
		ollamaClient: ollamaClient,
	}
}

// Analyze performs comprehensive text analysis
func (a *Analyzer) Analyze(text string) models.Metadata {
	return a.AnalyzeWithContext(context.Background(), text)
}

// AnalyzeWithContext performs comprehensive text analysis with context support
func (a *Analyzer) AnalyzeWithContext(ctx context.Context, text string) models.Metadata {
	metadata := models.Metadata{}

	// Basic statistics
	metadata.CharacterCount = len(text)
	words := extractWords(text)
	metadata.WordCount = len(words)
	metadata.SentenceCount = countSentences(text)
	metadata.ParagraphCount = countParagraphs(text)
	metadata.AverageWordLength = calculateAverageWordLength(words)

	// Sentiment analysis
	metadata.Sentiment, metadata.SentimentScore = analyzeSentiment(text)

	// Word frequency analysis
	metadata.TopWords = a.getTopWords(words, 20)
	metadata.UniqueWords = countUniqueWords(words)

	// Phrase analysis
	metadata.TopPhrases = a.getTopPhrases(text, 10)

	// Content extraction
	metadata.KeyTerms = a.extractKeyTerms(words, 15)
	metadata.NamedEntities = extractNamedEntities(text)
	metadata.PotentialDates = extractDates(text)
	metadata.PotentialURLs = extractURLs(text)
	metadata.EmailAddresses = extractEmails(text)

	// Readability
	metadata.ReadabilityScore = calculateReadability(text, metadata.WordCount, metadata.SentenceCount)
	metadata.ReadabilityLevel = getReadabilityLevel(metadata.ReadabilityScore)
	metadata.ComplexWordCount = countComplexWords(words)
	if metadata.SentenceCount > 0 {
		metadata.AvgSentenceLength = float64(metadata.WordCount) / float64(metadata.SentenceCount)
	}

	// AI-powered analysis (if Ollama client is available)
	if a.ollamaClient != nil {
		log.Println("Ollama client available, starting AI-powered analysis")

		// Generate synopsis
		log.Println("Generating synopsis...")
		if synopsis, err := a.ollamaClient.GenerateSynopsis(ctx, text); err == nil {
			metadata.Synopsis = synopsis
			log.Printf("Synopsis generated: %d characters", len(synopsis))
		} else {
			log.Printf("Synopsis generation failed: %v", err)
		}

		// Clean text
		log.Println("Cleaning text...")
		if cleanedText, err := a.ollamaClient.CleanText(ctx, text); err == nil {
			metadata.CleanedText = cleanedText
			log.Printf("Text cleaned: %d characters", len(cleanedText))
		} else {
			log.Printf("Text cleaning failed: %v", err)
		}

		// Editorial analysis
		log.Println("Performing editorial analysis...")
		if editorial, err := a.ollamaClient.EditorialAnalysis(ctx, text); err == nil {
			metadata.EditorialAnalysis = editorial
			log.Printf("Editorial analysis completed: %d characters", len(editorial))
		} else {
			log.Printf("Editorial analysis failed: %v", err)
		}

		// AI-generated tags
		log.Println("Generating AI tags...")
		metadataMap := map[string]interface{}{
			"sentiment": metadata.Sentiment,
		}
		if tags, err := a.ollamaClient.GenerateTags(ctx, text, metadataMap); err == nil {
			metadata.Tags = tags
			log.Printf("Generated %d AI tags: %v", len(tags), tags)
		} else {
			log.Printf("AI tag generation failed, falling back to rule-based: %v", err)
			metadata.Tags = generateTags(text, metadata)
		}

		// AI-extracted and pruned references
		log.Println("Extracting references with AI...")
		if refs, err := a.ollamaClient.ExtractReferences(ctx, text); err == nil {
			// Convert ollama.Reference to models.Reference
			metadata.References = make([]models.Reference, len(refs))
			for i, ref := range refs {
				metadata.References[i] = models.Reference{
					Text:       ref.Text,
					Type:       ref.Type,
					Context:    ref.Context,
					Confidence: ref.Confidence,
				}
			}
			log.Printf("Extracted %d AI references", len(refs))
		} else {
			log.Printf("AI reference extraction failed, falling back to rule-based: %v", err)
			metadata.References = extractReferences(text)
		}

		// AI content detection
		log.Println("Detecting AI-generated content...")
		if aiDetection, err := a.ollamaClient.DetectAIContent(ctx, text); err == nil {
			metadata.AIDetection = models.AIDetectionResult{
				Likelihood: aiDetection.Likelihood,
				Confidence: aiDetection.Confidence,
				Reasoning:  aiDetection.Reasoning,
				Indicators: aiDetection.Indicators,
				HumanScore: aiDetection.HumanScore,
			}
			log.Printf("AI detection completed: likelihood=%s, human_score=%.1f",
				aiDetection.Likelihood, aiDetection.HumanScore)
		} else {
			log.Printf("AI detection failed: %v", err)
		}

	} else {
		log.Println("Ollama client not available, using rule-based analysis")
		// Fallback to rule-based analysis when Ollama is not available
		metadata.References = extractReferences(text)
		metadata.Tags = generateTags(text, metadata)
	}

	// Language indicators
	metadata.Language = detectLanguage(text)
	metadata.QuestionCount = strings.Count(text, "?")
	metadata.ExclamationCount = strings.Count(text, "!")
	metadata.CapitalizedPercent = calculateCapitalizedPercent(text)

	return metadata
}

// extractWords extracts all words from text
func extractWords(text string) []string {
	text = strings.ToLower(text)
	reg := regexp.MustCompile(`[^\w\s]`)
	text = reg.ReplaceAllString(text, " ")
	words := strings.Fields(text)
	return words
}

// countSentences counts the number of sentences
func countSentences(text string) int {
	reg := regexp.MustCompile(`[.!?]+`)
	matches := reg.FindAllString(text, -1)
	if len(matches) == 0 {
		return 1
	}
	return len(matches)
}

// countParagraphs counts the number of paragraphs
func countParagraphs(text string) int {
	paragraphs := strings.Split(text, "\n\n")
	count := 0
	for _, p := range paragraphs {
		if strings.TrimSpace(p) != "" {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

// calculateAverageWordLength calculates average word length
func calculateAverageWordLength(words []string) float64 {
	if len(words) == 0 {
		return 0
	}
	total := 0
	for _, word := range words {
		total += len(word)
	}
	return float64(total) / float64(len(words))
}

// countUniqueWords counts unique words
func countUniqueWords(words []string) int {
	unique := make(map[string]bool)
	for _, word := range words {
		unique[word] = true
	}
	return len(unique)
}

// getTopWords returns the most frequent words
func (a *Analyzer) getTopWords(words []string, limit int) []models.WordFrequency {
	freq := make(map[string]int)
	for _, word := range words {
		if len(word) > 2 && !a.stopWords[word] {
			freq[word]++
		}
	}

	type wordCount struct {
		word  string
		count int
	}
	var counts []wordCount
	for word, count := range freq {
		counts = append(counts, wordCount{word, count})
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	result := []models.WordFrequency{}
	for i := 0; i < len(counts) && i < limit; i++ {
		result = append(result, models.WordFrequency{
			Word:  counts[i].word,
			Count: counts[i].count,
		})
	}

	return result
}

// getTopPhrases extracts common phrases
func (a *Analyzer) getTopPhrases(text string, limit int) []models.PhraseInfo {
	text = strings.ToLower(text)
	words := strings.Fields(text)

	phrases := make(map[string]int)

	// Extract 2-word phrases
	for i := 0; i < len(words)-1; i++ {
		word1 := cleanWord(words[i])
		word2 := cleanWord(words[i+1])
		if len(word1) > 2 && len(word2) > 2 && !a.stopWords[word1] && !a.stopWords[word2] {
			phrase := word1 + " " + word2
			phrases[phrase]++
		}
	}

	// Extract 3-word phrases
	for i := 0; i < len(words)-2; i++ {
		word1 := cleanWord(words[i])
		word2 := cleanWord(words[i+1])
		word3 := cleanWord(words[i+2])
		if len(word1) > 2 && len(word2) > 2 && len(word3) > 2 {
			phrase := word1 + " " + word2 + " " + word3
			phrases[phrase]++
		}
	}

	type phraseCount struct {
		phrase string
		count  int
	}
	var counts []phraseCount
	for phrase, count := range phrases {
		if count > 1 {
			counts = append(counts, phraseCount{phrase, count})
		}
	}

	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	result := []models.PhraseInfo{}
	for i := 0; i < len(counts) && i < limit; i++ {
		result = append(result, models.PhraseInfo{
			Phrase: counts[i].phrase,
			Count:  counts[i].count,
		})
	}

	return result
}

// cleanWord removes punctuation from a word
func cleanWord(word string) string {
	reg := regexp.MustCompile(`[^\w]`)
	return reg.ReplaceAllString(word, "")
}

// extractKeyTerms extracts key terms from text
func (a *Analyzer) extractKeyTerms(words []string, limit int) []string {
	freq := make(map[string]int)
	for _, word := range words {
		if len(word) > 4 && !a.stopWords[word] {
			freq[word]++
		}
	}

	type termScore struct {
		term  string
		score int
	}
	var scores []termScore
	for term, count := range freq {
		score := count * len(term)
		scores = append(scores, termScore{term, score})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	result := []string{}
	for i := 0; i < len(scores) && i < limit; i++ {
		result = append(result, scores[i].term)
	}

	return result
}

// extractNamedEntities extracts potential named entities (capitalized words/phrases)
func extractNamedEntities(text string) []string {
	reg := regexp.MustCompile(`\b[A-Z][a-z]+(?:\s+[A-Z][a-z]+)*\b`)
	matches := reg.FindAllString(text, -1)

	unique := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 2 {
			unique[match] = true
		}
	}

	result := []string{}
	for entity := range unique {
		result = append(result, entity)
	}

	sort.Strings(result)
	return result
}

// extractDates extracts potential dates
func extractDates(text string) []string {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\b\d{1,2}[/-]\d{1,2}[/-]\d{2,4}\b`),
		regexp.MustCompile(`\b(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{1,2},?\s+\d{4}\b`),
		regexp.MustCompile(`\b\d{1,2}\s+(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\s+\d{4}\b`),
		regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}\b`),
	}

	unique := make(map[string]bool)
	for _, pattern := range patterns {
		matches := pattern.FindAllString(text, -1)
		for _, match := range matches {
			unique[match] = true
		}
	}

	result := []string{}
	for date := range unique {
		result = append(result, date)
	}

	sort.Strings(result)
	return result
}

// extractURLs extracts URLs from text
func extractURLs(text string) []string {
	reg := regexp.MustCompile(`https?://[^\s]+`)
	matches := reg.FindAllString(text, -1)

	unique := make(map[string]bool)
	for _, match := range matches {
		unique[match] = true
	}

	result := []string{}
	for url := range unique {
		result = append(result, url)
	}

	sort.Strings(result)
	return result
}

// extractEmails extracts email addresses from text
func extractEmails(text string) []string {
	reg := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)
	matches := reg.FindAllString(text, -1)

	unique := make(map[string]bool)
	for _, match := range matches {
		unique[match] = true
	}

	result := []string{}
	for email := range unique {
		result = append(result, email)
	}

	sort.Strings(result)
	return result
}

// calculateReadability calculates the Flesch Reading Ease score
func calculateReadability(text string, wordCount, sentenceCount int) float64 {
	if wordCount == 0 || sentenceCount == 0 {
		return 0
	}

	syllableCount := countSyllables(text)

	avgWordsPerSentence := float64(wordCount) / float64(sentenceCount)
	avgSyllablesPerWord := float64(syllableCount) / float64(wordCount)

	score := 206.835 - 1.015*avgWordsPerSentence - 84.6*avgSyllablesPerWord

	return math.Round(score*100) / 100
}

// countSyllables counts syllables in text (simplified)
func countSyllables(text string) int {
	words := extractWords(text)
	count := 0
	for _, word := range words {
		count += countSyllablesInWord(word)
	}
	return count
}

// countSyllablesInWord counts syllables in a single word
func countSyllablesInWord(word string) int {
	word = strings.ToLower(word)
	if len(word) == 0 {
		return 0
	}

	count := 0
	vowels := "aeiouy"
	prevWasVowel := false

	for _, char := range word {
		isVowel := strings.ContainsRune(vowels, char)
		if isVowel && !prevWasVowel {
			count++
		}
		prevWasVowel = isVowel
	}

	// Adjust for silent e
	if strings.HasSuffix(word, "e") && count > 1 {
		count--
	}

	if count == 0 {
		count = 1
	}

	return count
}

// getReadabilityLevel returns the readability level based on score
func getReadabilityLevel(score float64) string {
	switch {
	case score >= 90:
		return "very_easy"
	case score >= 80:
		return "easy"
	case score >= 70:
		return "fairly_easy"
	case score >= 60:
		return "standard"
	case score >= 50:
		return "fairly_difficult"
	case score >= 30:
		return "difficult"
	default:
		return "very_difficult"
	}
}

// countComplexWords counts words with 3+ syllables
func countComplexWords(words []string) int {
	count := 0
	for _, word := range words {
		if countSyllablesInWord(word) >= 3 {
			count++
		}
	}
	return count
}

// analyzeSentiment performs basic sentiment analysis
func analyzeSentiment(text string) (string, float64) {
	text = strings.ToLower(text)

	positiveWords := getPositiveWords()
	negativeWords := getNegativeWords()

	words := extractWords(text)
	positiveCount := 0
	negativeCount := 0

	for _, word := range words {
		if positiveWords[word] {
			positiveCount++
		}
		if negativeWords[word] {
			negativeCount++
		}
	}

	total := positiveCount + negativeCount
	if total == 0 {
		return "neutral", 0.0
	}

	score := (float64(positiveCount) - float64(negativeCount)) / float64(len(words))
	score = math.Max(-1.0, math.Min(1.0, score*10))

	sentiment := "neutral"
	if score > 0.1 {
		sentiment = "positive"
	} else if score < -0.1 {
		sentiment = "negative"
	}

	return sentiment, math.Round(score*100) / 100
}

// extractReferences extracts potential references that need verification
func extractReferences(text string) []models.Reference {
	references := []models.Reference{}

	// Extract statistics (numbers with units or percentages)
	statReg := regexp.MustCompile(`\b\d+(?:\.\d+)?%|\b\d+(?:,\d{3})*(?:\.\d+)?\s+(?:million|billion|thousand|percent|dollars?|years?|months?|days?)\b`)
	statMatches := statReg.FindAllString(text, -1)
	for _, match := range statMatches {
		context := extractContext(text, match, 50)
		references = append(references, models.Reference{
			Text:       match,
			Type:       "statistic",
			Context:    context,
			Confidence: "medium",
		})
	}

	// Extract quotes
	quoteReg := regexp.MustCompile(`"[^"]{20,}"`)
	quoteMatches := quoteReg.FindAllString(text, -1)
	for _, match := range quoteMatches {
		references = append(references, models.Reference{
			Text:       match,
			Type:       "quote",
			Context:    "",
			Confidence: "high",
		})
	}

	// Extract claims (sentences with "is", "are", "was", "were")
	sentences := regexp.MustCompile(`[^.!?]+[.!?]`).FindAllString(text, -1)
	claimWords := []string{"is", "are", "was", "were", "has", "have", "shows", "demonstrates", "proves"}
	for _, sentence := range sentences {
		lower := strings.ToLower(sentence)
		for _, word := range claimWords {
			if strings.Contains(lower, " "+word+" ") && len(sentence) > 30 && len(sentence) < 200 {
				references = append(references, models.Reference{
					Text:       strings.TrimSpace(sentence),
					Type:       "claim",
					Context:    "",
					Confidence: "low",
				})
				break
			}
		}
	}

	return references
}

// extractContext extracts text around a match
func extractContext(text, match string, contextLength int) string {
	index := strings.Index(text, match)
	if index == -1 {
		return ""
	}

	start := index - contextLength
	if start < 0 {
		start = 0
	}

	end := index + len(match) + contextLength
	if end > len(text) {
		end = len(text)
	}

	return strings.TrimSpace(text[start:end])
}

// generateTags generates tags based on content
func generateTags(text string, metadata models.Metadata) []string {
	tags := []string{}

	// Sentiment tag
	tags = append(tags, metadata.Sentiment)

	// Length tags
	if metadata.WordCount < 100 {
		tags = append(tags, "short")
	} else if metadata.WordCount < 500 {
		tags = append(tags, "medium")
	} else {
		tags = append(tags, "long")
	}

	// Readability tags
	tags = append(tags, metadata.ReadabilityLevel)

	// Content type tags
	if metadata.QuestionCount > 3 {
		tags = append(tags, "faq")
	}
	if len(metadata.PotentialURLs) > 2 {
		tags = append(tags, "web-content")
	}
	if len(metadata.References) > 5 {
		tags = append(tags, "research")
	}

	// Topic tags from key terms (top 3)
	for i := 0; i < len(metadata.KeyTerms) && i < 3; i++ {
		tags = append(tags, metadata.KeyTerms[i])
	}

	return tags
}

// detectLanguage provides basic language detection
func detectLanguage(text string) string {
	// Simple heuristic - this would be more sophisticated in production
	if len(text) < 10 {
		return "unknown"
	}
	return "english"
}

// calculateCapitalizedPercent calculates percentage of capitalized words
func calculateCapitalizedPercent(text string) float64 {
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0
	}

	capitalizedCount := 0
	for _, word := range words {
		if len(word) > 0 && unicode.IsUpper(rune(word[0])) {
			capitalizedCount++
		}
	}

	return math.Round((float64(capitalizedCount)/float64(len(words)))*10000) / 100
}
