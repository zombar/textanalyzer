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

	// EARLY QUALITY CHECK: Run quality scoring BEFORE expensive AI analysis
	// This filters out garbage content before sending to Ollama
	log.Println("Running early quality assessment...")
	earlyQualityScore := scoreTextQualityFallback(text, metadata.WordCount, metadata.ReadabilityScore)

	const QUALITY_THRESHOLD = 0.35 // Skip AI processing for content below this threshold

	if earlyQualityScore.Score < QUALITY_THRESHOLD {
		log.Printf("Content quality too low (%.2f < %.2f), skipping AI analysis. Reason: %s",
			earlyQualityScore.Score, QUALITY_THRESHOLD, earlyQualityScore.Reason)

		// Return minimal metadata with quality score
		metadata.QualityScore = &earlyQualityScore
		metadata.References = extractReferences(text)
		metadata.Tags = generateTags(text, metadata)

		// Language indicators
		metadata.Language = detectLanguage(text)
		metadata.QuestionCount = strings.Count(text, "?")
		metadata.ExclamationCount = strings.Count(text, "!")
		metadata.CapitalizedPercent = calculateCapitalizedPercent(text)

		return metadata
	}

	log.Printf("Content quality sufficient (%.2f >= %.2f), proceeding with AI analysis",
		earlyQualityScore.Score, QUALITY_THRESHOLD)

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

		// Generate computed tags from metadata
		computedTags := generateTags(text, metadata)

		// AI-generated tags
		log.Println("Generating AI tags...")
		metadataMap := map[string]interface{}{
			"sentiment": metadata.Sentiment,
		}
		if aiTags, err := a.ollamaClient.GenerateTags(ctx, text, metadataMap); err == nil {
			// Merge AI tags with computed tags (remove duplicates)
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
			metadata.Tags = mergedTags
			log.Printf("Merged %d computed tags + %d AI tags = %d total tags", len(computedTags), len(aiTags), len(mergedTags))
		} else {
			log.Printf("AI tag generation failed, using computed tags only: %v", err)
			metadata.Tags = computedTags
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

		// Text quality scoring (with fallback to rule-based scoring)
		log.Println("Scoring text quality...")
		if qualityScore, err := a.ollamaClient.ScoreTextQuality(ctx, text); err == nil {
			metadata.QualityScore = &models.TextQualityScore{
				Score:             qualityScore.Score,
				Reason:            qualityScore.Reason,
				Categories:        qualityScore.Categories,
				IsRecommended:     qualityScore.Score >= 0.5, // Default threshold
				QualityIndicators: qualityScore.QualityIndicators,
				ProblemsDetected:  qualityScore.ProblemsDetected,
				AIUsed:            true, // AI-powered scoring
			}
			log.Printf("Text quality scored (AI): score=%.2f, recommended=%v",
				qualityScore.Score, metadata.QualityScore.IsRecommended)
		} else {
			// Fallback to rule-based scoring when Ollama is unavailable
			log.Printf("Ollama scoring failed, using rule-based fallback: %v", err)
			fallbackScore := scoreTextQualityFallback(text, metadata.WordCount, metadata.ReadabilityScore)
			metadata.QualityScore = &fallbackScore
			log.Printf("Text quality scored (fallback): score=%.2f, recommended=%v",
				fallbackScore.Score, fallbackScore.IsRecommended)
		}

	} else {
		log.Println("Ollama client not available, using rule-based analysis")
		// Fallback to rule-based analysis when Ollama is not available
		metadata.References = extractReferences(text)
		metadata.Tags = generateTags(text, metadata)

		// Add rule-based quality scoring
		fallbackScore := scoreTextQualityFallback(text, metadata.WordCount, metadata.ReadabilityScore)
		metadata.QualityScore = &fallbackScore
		log.Printf("Text quality scored (fallback): score=%.2f, recommended=%v",
			fallbackScore.Score, fallbackScore.IsRecommended)
	}

	// Language indicators
	metadata.Language = detectLanguage(text)
	metadata.QuestionCount = strings.Count(text, "?")
	metadata.ExclamationCount = strings.Count(text, "!")
	metadata.CapitalizedPercent = calculateCapitalizedPercent(text)

	return metadata
}

// AnalyzeOffline performs offline text analysis without Ollama (Stage 1)
// This method only uses rule-based heuristics and is fast for initial processing
func (a *Analyzer) AnalyzeOffline(text string) models.Metadata {
	metadata := models.Metadata{}

	// Basic statistics
	metadata.CharacterCount = len(text)
	words := extractWords(text)
	metadata.WordCount = len(words)
	metadata.SentenceCount = countSentences(text)
	metadata.ParagraphCount = countParagraphs(text)
	metadata.AverageWordLength = calculateAverageWordLength(words)

	// Sentiment analysis (rule-based)
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

	// Advanced offline text cleaning using heuristics
	// This extracts article content and removes boilerplate/navigation
	metadata.CleanedText = a.cleanTextOffline(text)
	cleanedWordCount := len(extractWords(metadata.CleanedText))
	log.Printf("Offline cleaning: %d words → %d words (%.1f%% reduction)",
		metadata.WordCount, cleanedWordCount,
		100*(1-float64(cleanedWordCount)/float64(metadata.WordCount)))

	// Rule-based quality scoring
	qualityScore := scoreTextQualityFallback(text, metadata.WordCount, metadata.ReadabilityScore)
	metadata.QualityScore = &qualityScore

	// Rule-based references and tags
	metadata.References = extractReferences(text)
	metadata.Tags = generateTags(text, metadata)

	// Language indicators
	metadata.Language = detectLanguage(text)
	metadata.QuestionCount = strings.Count(text, "?")
	metadata.ExclamationCount = strings.Count(text, "!")
	metadata.CapitalizedPercent = calculateCapitalizedPercent(text)

	log.Printf("Offline analysis completed: %d words, quality=%.2f, language=%s",
		metadata.WordCount, qualityScore.Score, metadata.Language)

	return metadata
}

// ExtractImageMetadata extracts offline metadata from an image URL
// This is a placeholder for basic image processing without AI
func (a *Analyzer) ExtractImageMetadata(imageURL string) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Extract basic information from URL
	metadata["url"] = imageURL

	// Detect image format from URL
	lowerURL := strings.ToLower(imageURL)
	if strings.HasSuffix(lowerURL, ".jpg") || strings.HasSuffix(lowerURL, ".jpeg") {
		metadata["format"] = "jpeg"
	} else if strings.HasSuffix(lowerURL, ".png") {
		metadata["format"] = "png"
	} else if strings.HasSuffix(lowerURL, ".gif") {
		metadata["format"] = "gif"
	} else if strings.HasSuffix(lowerURL, ".webp") {
		metadata["format"] = "webp"
	} else if strings.HasSuffix(lowerURL, ".svg") {
		metadata["format"] = "svg"
	} else {
		metadata["format"] = "unknown"
	}

	// Extract domain
	if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
		parts := strings.Split(imageURL, "/")
		if len(parts) >= 3 {
			metadata["domain"] = parts[2]
		}
	}

	// TODO: When Ollama supports vision models, add AI image analysis here
	metadata["ai_analysis_pending"] = true

	log.Printf("Offline image metadata extracted: url=%s, format=%s",
		imageURL, metadata["format"])

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
	// Use map to deduplicate tags
	tagSet := make(map[string]bool)

	// Sentiment tag
	tagSet[normalizeTag(metadata.Sentiment)] = true

	// Length tags
	if metadata.WordCount < 100 {
		tagSet["short"] = true
	} else if metadata.WordCount < 500 {
		tagSet["medium"] = true
	} else {
		tagSet["long"] = true
	}

	// Readability tags (normalize in case they have underscores)
	tagSet[normalizeTag(metadata.ReadabilityLevel)] = true

	// Content type tags
	if metadata.QuestionCount > 3 {
		tagSet["faq"] = true
	}
	if len(metadata.PotentialURLs) > 2 {
		tagSet["web-content"] = true
	}
	if len(metadata.References) > 5 {
		tagSet["research"] = true
	}

	// Topic tags from key terms (top 3) - normalize them
	for i := 0; i < len(metadata.KeyTerms) && i < 3; i++ {
		tagSet[normalizeTag(metadata.KeyTerms[i])] = true
	}

	// Named entities make good tags (people, places, things)
	// Add up to 5 named entities as tags
	for i := 0; i < len(metadata.NamedEntities) && i < 5; i++ {
		tagSet[normalizeTag(metadata.NamedEntities[i])] = true
	}

	// Convert set to slice
	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}

	return tags
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

// detectListLikeStructure checks if text is just a disconnected list of items
func detectListLikeStructure(text string) (bool, float64) {
	sentences := regexp.MustCompile(`[^.!?]+[.!?]`).FindAllString(text, -1)
	if len(sentences) < 3 {
		return false, 0.0
	}

	// Check for patterns that suggest list-like structure:
	// 1. Many short, disconnected sentences
	// 2. Little vocabulary overlap between consecutive sentences
	// 3. Abrupt topic changes

	shortSentenceCount := 0
	for _, sentence := range sentences {
		words := strings.Fields(sentence)
		if len(words) < 15 {
			shortSentenceCount++
		}
	}

	shortSentenceRatio := float64(shortSentenceCount) / float64(len(sentences))

	// Check vocabulary overlap between consecutive sentences
	lowOverlapCount := 0
	for i := 0; i < len(sentences)-1; i++ {
		words1 := extractWords(sentences[i])
		words2 := extractWords(sentences[i+1])

		// Calculate Jaccard similarity
		set1 := make(map[string]bool)
		for _, w := range words1 {
			if len(w) > 3 { // Only meaningful words
				set1[w] = true
			}
		}

		set2 := make(map[string]bool)
		for _, w := range words2 {
			if len(w) > 3 {
				set2[w] = true
			}
		}

		// Count intersection
		intersection := 0
		for w := range set1 {
			if set2[w] {
				intersection++
			}
		}

		// If very little overlap, it's likely disconnected
		union := len(set1) + len(set2) - intersection
		if union > 0 {
			similarity := float64(intersection) / float64(union)
			if similarity < 0.15 { // Very low overlap threshold
				lowOverlapCount++
			}
		}
	}

	lowOverlapRatio := float64(lowOverlapCount) / float64(len(sentences)-1)

	// If most sentences are short AND have low overlap, it's list-like
	isListLike := shortSentenceRatio > 0.6 && lowOverlapRatio > 0.5

	return isListLike, lowOverlapRatio
}

// calculateTransitionWordScore checks for connective language
func calculateTransitionWordScore(text string) float64 {
	textLower := strings.ToLower(text)

	transitionWords := []string{
		// Addition
		"additionally", "furthermore", "moreover", "also", "besides",
		// Contrast
		"however", "nevertheless", "nonetheless", "although", "despite", "yet", "but",
		// Cause/Effect
		"therefore", "thus", "consequently", "hence", "accordingly", "as a result",
		// Sequence
		"first", "second", "third", "next", "then", "finally", "subsequently",
		// Example
		"for example", "for instance", "specifically", "namely",
		// Emphasis
		"indeed", "in fact", "certainly", "clearly",
	}

	transitionCount := 0
	for _, word := range transitionWords {
		transitionCount += strings.Count(textLower, word)
	}

	// Normalize by sentence count
	sentenceCount := strings.Count(text, ".") + strings.Count(text, "!") + strings.Count(text, "?")
	if sentenceCount == 0 {
		sentenceCount = 1
	}

	// Good writing has ~0.3-0.5 transitions per sentence
	score := float64(transitionCount) / float64(sentenceCount)
	return score
}

// detectCoherenceMarkers looks for pronouns and references that connect content
func detectCoherenceMarkers(text string) int {
	textLower := strings.ToLower(text)

	markers := []string{
		// Pronouns that refer back to previous content
		" it ", " this ", " that ", " these ", " those ", " they ", " them ",
		// Demonstratives
		" such ", " said ",
		// Articles that suggest previous reference
		" the ",
	}

	markerCount := 0
	for _, marker := range markers {
		markerCount += strings.Count(textLower, marker)
	}

	return markerCount
}

// scoreTextQualityFallback provides rule-based text quality scoring when Ollama is unavailable
func scoreTextQualityFallback(text string, wordCount int, readabilityScore float64) models.TextQualityScore {
	score := 0.5 // Start with neutral score
	categories := []string{}
	qualityIndicators := []string{}
	problemsDetected := []string{}
	reasons := []string{}

	textLower := strings.ToLower(text)

	// Check for very short content
	if len(text) < 50 {
		score = 0.1
		categories = append(categories, "too_short", "low_quality")
		problemsDetected = append(problemsDetected, "extremely_short")
		reasons = append(reasons, "Content too short (< 50 characters)")
		return models.TextQualityScore{
			Score:             score,
			Reason:            strings.Join(reasons, "; "),
			Categories:        categories,
			IsRecommended:     false,
			QualityIndicators: qualityIndicators,
			ProblemsDetected:  problemsDetected,
			AIUsed:            false, // Rule-based fallback
		}
	}

	if wordCount < 20 {
		score -= 0.3
		categories = append(categories, "minimal_content")
		problemsDetected = append(problemsDetected, "too_few_words")
		reasons = append(reasons, "Very few words")
	} else if wordCount < 50 {
		score -= 0.1
		reasons = append(reasons, "Short content")
	} else if wordCount > 200 {
		score += 0.2
		categories = append(categories, "informative")
		qualityIndicators = append(qualityIndicators, "substantial_length")
		reasons = append(reasons, "Substantial content")
	}

	// Check for list-like structure (disconnected sentences)
	isListLike, overlapRatio := detectListLikeStructure(text)
	if isListLike {
		score -= 0.4
		categories = append(categories, "incoherent", "list_like", "low_quality")
		problemsDetected = append(problemsDetected, "disconnected_sentences", "no_flow")
		reasons = append(reasons, "Text appears to be disconnected list items without flow")
	} else if overlapRatio > 0.4 {
		// Many disconnected sentences but not quite list-like
		score -= 0.2
		problemsDetected = append(problemsDetected, "poor_continuity")
		reasons = append(reasons, "Weak continuity between sentences")
	}

	// Check for transition words (coherence indicators)
	transitionScore := calculateTransitionWordScore(text)
	if transitionScore >= 0.2 {
		score += 0.1
		qualityIndicators = append(qualityIndicators, "good_transitions")
	} else if transitionScore < 0.05 && wordCount > 100 {
		score -= 0.15
		problemsDetected = append(problemsDetected, "lacks_transitions")
		reasons = append(reasons, "Few transition words, may lack flow")
	}

	// Check for coherence markers (pronouns, references)
	coherenceMarkers := detectCoherenceMarkers(text)
	markerRatio := float64(coherenceMarkers) / float64(wordCount)
	if markerRatio >= 0.05 && markerRatio <= 0.15 {
		// Good use of references
		score += 0.1
		qualityIndicators = append(qualityIndicators, "good_reference_usage")
	} else if markerRatio < 0.02 && wordCount > 100 {
		// Very few references in longer text suggests disconnected content
		score -= 0.1
		problemsDetected = append(problemsDetected, "lacks_coherence_markers")
	}

	// Check for spam indicators
	spamKeywords := []string{"click here", "buy now", "limited offer", "act now", "call now", "free money", "earn $$$"}
	spamCount := 0
	for _, keyword := range spamKeywords {
		spamCount += strings.Count(textLower, keyword)
	}

	if spamCount > 3 {
		score -= 0.4
		categories = append(categories, "spam", "low_quality")
		problemsDetected = append(problemsDetected, "spam_keywords", "promotional")
		reasons = append(reasons, "Multiple spam indicators")
	} else if spamCount > 0 {
		score -= 0.2
		problemsDetected = append(problemsDetected, "some_promotional_language")
	}

	// Check for excessive punctuation
	exclamationCount := strings.Count(text, "!")

	if exclamationCount > wordCount/10 && exclamationCount > 5 {
		score -= 0.2
		problemsDetected = append(problemsDetected, "excessive_exclamations")
		reasons = append(reasons, "Excessive exclamation marks")
	}

	// Check for all caps (shouting)
	upperRatio := 0.0
	upperCount := 0
	lowerCount := 0
	for _, r := range text {
		if unicode.IsUpper(r) {
			upperCount++
		} else if unicode.IsLower(r) {
			lowerCount++
		}
	}
	if upperCount+lowerCount > 0 {
		upperRatio = float64(upperCount) / float64(upperCount+lowerCount)
	}

	if upperRatio > 0.5 {
		score -= 0.3
		problemsDetected = append(problemsDetected, "excessive_capitalization")
		reasons = append(reasons, "Excessive capitalization (shouting)")
	}

	// Check readability
	if readabilityScore > 0 {
		if readabilityScore >= 60 && readabilityScore <= 70 {
			score += 0.1
			qualityIndicators = append(qualityIndicators, "good_readability")
		} else if readabilityScore < 30 || readabilityScore > 80 {
			score -= 0.1
			if readabilityScore < 30 {
				problemsDetected = append(problemsDetected, "difficult_to_read")
			}
		}
	}

	// Check for coherence indicators (sentences, paragraphs)
	sentenceCount := strings.Count(text, ".") + strings.Count(text, "!") + strings.Count(text, "?")
	if sentenceCount == 0 {
		sentenceCount = 1
	}

	avgWordsPerSentence := float64(wordCount) / float64(sentenceCount)
	if avgWordsPerSentence >= 10 && avgWordsPerSentence <= 25 {
		score += 0.1
		qualityIndicators = append(qualityIndicators, "good_sentence_length")
	} else if avgWordsPerSentence < 5 || avgWordsPerSentence > 40 {
		score -= 0.1
		if avgWordsPerSentence < 5 {
			problemsDetected = append(problemsDetected, "choppy_sentences")
		} else {
			problemsDetected = append(problemsDetected, "overly_long_sentences")
		}
	}

	// Check for gibberish (excessive repeated characters)
	repeatedChars := 0
	for i := 0; i < len(text)-2; i++ {
		if text[i] == text[i+1] && text[i] == text[i+2] {
			repeatedChars++
		}
	}

	if repeatedChars > wordCount/5 {
		score -= 0.3
		categories = append(categories, "incoherent", "low_quality")
		problemsDetected = append(problemsDetected, "excessive_character_repetition", "possibly_gibberish")
		reasons = append(reasons, "Excessive repeated characters (gibberish)")
	}

	// Check for educational/informative keywords
	qualityKeywords := []string{"research", "study", "analysis", "demonstrate", "evidence", "conclude", "data", "results", "findings"}
	qualityCount := 0
	for _, keyword := range qualityKeywords {
		if strings.Contains(textLower, keyword) {
			qualityCount++
		}
	}

	if qualityCount >= 3 {
		score += 0.2
		categories = append(categories, "informative", "educational")
		qualityIndicators = append(qualityIndicators, "academic_language")
		reasons = append(reasons, "Contains informative/academic language")
	}

	// Ensure score is within bounds
	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}

	// Determine if recommended
	isRecommended := score >= 0.5

	// Build reason string
	var reason string
	if len(reasons) == 0 {
		reason = "Rule-based assessment (Ollama unavailable)"
	} else {
		reason = "Rule-based: " + strings.Join(reasons, "; ")
	}

	// Ensure categories based on score
	if len(categories) == 0 {
		if score >= 0.7 {
			categories = []string{"informative", "good_quality"}
		} else if score >= 0.5 {
			categories = []string{"acceptable"}
		} else {
			categories = []string{"low_quality"}
		}
	}

	// Ensure slices are not nil
	if qualityIndicators == nil {
		qualityIndicators = []string{}
	}
	if problemsDetected == nil {
		problemsDetected = []string{}
	}

	return models.TextQualityScore{
		Score:             score,
		Reason:            reason,
		Categories:        categories,
		IsRecommended:     isRecommended,
		QualityIndicators: qualityIndicators,
		ProblemsDetected:  problemsDetected,
		AIUsed:            false, // Rule-based fallback
	}
}

// AnalyzeWithHTMLContext performs AI-powered analysis using offline text as a template and original HTML
// This provides enhanced cleaning by instructing the LLM to use the offline text as a reference
// and extract the cleanest version from the original HTML, removing image attributions and translating to English
func (a *Analyzer) AnalyzeWithHTMLContext(ctx context.Context, text, offlineText, originalHTML string) models.Metadata {
	metadata := models.Metadata{}

	// Basic statistics from original text
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

	// Language indicators
	metadata.Language = detectLanguage(text)
	metadata.QuestionCount = strings.Count(text, "?")
	metadata.ExclamationCount = strings.Count(text, "!")
	metadata.CapitalizedPercent = calculateCapitalizedPercent(text)

	// AI-powered analysis with HTML context (if Ollama client is available)
	if a.ollamaClient != nil {
		log.Println("Ollama client available, starting enhanced AI-powered analysis with HTML context")

		// Enhanced text cleaning using offline text as template and original HTML
		log.Println("Performing enhanced text cleaning with HTML context...")
		if cleanedText, err := a.ollamaClient.CleanTextWithHTMLContext(ctx, text, offlineText, originalHTML); err == nil {
			metadata.CleanedText = cleanedText
			log.Printf("Enhanced text cleaning completed: %d characters (original: %d)", len(cleanedText), len(text))
		} else {
			log.Printf("Enhanced text cleaning failed, falling back to standard cleaning: %v", err)
			// Fallback to standard cleaning
			if cleanedText, err := a.ollamaClient.CleanText(ctx, text); err == nil {
				metadata.CleanedText = cleanedText
				log.Printf("Standard text cleaning completed: %d characters", len(cleanedText))
			} else {
				log.Printf("Standard text cleaning also failed: %v", err)
			}
		}

		// Use cleaned text for subsequent AI analysis if available
		analysisText := text
		if metadata.CleanedText != "" {
			analysisText = metadata.CleanedText
		}

		// Generate synopsis
		log.Println("Generating synopsis...")
		if synopsis, err := a.ollamaClient.GenerateSynopsis(ctx, analysisText); err == nil {
			metadata.Synopsis = synopsis
			log.Printf("Synopsis generated: %d characters", len(synopsis))
		} else {
			log.Printf("Synopsis generation failed: %v", err)
		}

		// Editorial analysis
		log.Println("Performing editorial analysis...")
		if editorial, err := a.ollamaClient.EditorialAnalysis(ctx, analysisText); err == nil {
			metadata.EditorialAnalysis = editorial
			log.Printf("Editorial analysis completed: %d characters", len(editorial))
		} else {
			log.Printf("Editorial analysis failed: %v", err)
		}

		// Generate computed tags from metadata
		computedTags := generateTags(text, metadata)

		// AI-generated tags
		log.Println("Generating AI tags...")
		metadataMap := map[string]interface{}{
			"sentiment": metadata.Sentiment,
		}
		if aiTags, err := a.ollamaClient.GenerateTags(ctx, analysisText, metadataMap); err == nil {
			// Merge AI tags with computed tags (remove duplicates)
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
			metadata.Tags = mergedTags
			log.Printf("Merged %d computed tags + %d AI tags = %d total tags", len(computedTags), len(aiTags), len(mergedTags))
		} else {
			log.Printf("AI tag generation failed, using computed tags only: %v", err)
			metadata.Tags = computedTags
		}

		// AI-extracted and pruned references
		log.Println("Extracting references with AI...")
		if refs, err := a.ollamaClient.ExtractReferences(ctx, analysisText); err == nil {
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
		if aiDetection, err := a.ollamaClient.DetectAIContent(ctx, analysisText); err == nil {
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

		// Text quality scoring (with fallback to rule-based scoring)
		log.Println("Scoring text quality...")
		if qualityScore, err := a.ollamaClient.ScoreTextQuality(ctx, analysisText); err == nil {
			metadata.QualityScore = &models.TextQualityScore{
				Score:             qualityScore.Score,
				Reason:            qualityScore.Reason,
				Categories:        qualityScore.Categories,
				IsRecommended:     qualityScore.Score >= 0.5,
				QualityIndicators: qualityScore.QualityIndicators,
				ProblemsDetected:  qualityScore.ProblemsDetected,
				AIUsed:            true,
			}
			log.Printf("Text quality scored (AI): score=%.2f, recommended=%v",
				qualityScore.Score, metadata.QualityScore.IsRecommended)
		} else {
			log.Printf("Ollama scoring failed, using rule-based fallback: %v", err)
			fallbackScore := scoreTextQualityFallback(text, metadata.WordCount, metadata.ReadabilityScore)
			metadata.QualityScore = &fallbackScore
			log.Printf("Text quality scored (fallback): score=%.2f, recommended=%v",
				fallbackScore.Score, fallbackScore.IsRecommended)
		}

	} else {
		log.Println("Ollama client not available, using rule-based analysis")
		// Fallback to rule-based analysis when Ollama is not available
		metadata.References = extractReferences(text)
		metadata.Tags = generateTags(text, metadata)

		// Add rule-based quality scoring
		fallbackScore := scoreTextQualityFallback(text, metadata.WordCount, metadata.ReadabilityScore)
		metadata.QualityScore = &fallbackScore
		log.Printf("Text quality scored (fallback): score=%.2f, recommended=%v",
			fallbackScore.Score, fallbackScore.IsRecommended)
	}

	return metadata
}
