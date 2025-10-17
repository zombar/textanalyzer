package analyzer

// getStopWords returns common English stop words
func getStopWords() map[string]bool {
	words := []string{
		"a", "about", "above", "after", "again", "against", "all", "am", "an", "and", "any", "are", "aren't",
		"as", "at", "be", "because", "been", "before", "being", "below", "between", "both", "but", "by",
		"can't", "cannot", "could", "couldn't", "did", "didn't", "do", "does", "doesn't", "doing", "don't",
		"down", "during", "each", "few", "for", "from", "further", "had", "hadn't", "has", "hasn't", "have",
		"haven't", "having", "he", "he'd", "he'll", "he's", "her", "here", "here's", "hers", "herself", "him",
		"himself", "his", "how", "how's", "i", "i'd", "i'll", "i'm", "i've", "if", "in", "into", "is", "isn't",
		"it", "it's", "its", "itself", "let's", "me", "more", "most", "mustn't", "my", "myself", "no", "nor",
		"not", "of", "off", "on", "once", "only", "or", "other", "ought", "our", "ours", "ourselves", "out",
		"over", "own", "same", "shan't", "she", "she'd", "she'll", "she's", "should", "shouldn't", "so", "some",
		"such", "than", "that", "that's", "the", "their", "theirs", "them", "themselves", "then", "there",
		"there's", "these", "they", "they'd", "they'll", "they're", "they've", "this", "those", "through", "to",
		"too", "under", "until", "up", "very", "was", "wasn't", "we", "we'd", "we'll", "we're", "we've", "were",
		"weren't", "what", "what's", "when", "when's", "where", "where's", "which", "while", "who", "who's",
		"whom", "why", "why's", "with", "won't", "would", "wouldn't", "you", "you'd", "you'll", "you're",
		"you've", "your", "yours", "yourself", "yourselves",
	}
	
	stopWords := make(map[string]bool)
	for _, word := range words {
		stopWords[word] = true
	}
	return stopWords
}

// getPositiveWords returns common positive sentiment words
func getPositiveWords() map[string]bool {
	words := []string{
		"good", "great", "excellent", "amazing", "wonderful", "fantastic", "best", "love", "loved", "loving",
		"beautiful", "perfect", "awesome", "brilliant", "outstanding", "superb", "exceptional", "incredible",
		"magnificent", "marvelous", "pleasant", "delightful", "enjoyable", "happy", "glad", "pleased",
		"satisfied", "terrific", "fabulous", "splendid", "impressive", "remarkable", "positive", "advantage",
		"benefit", "success", "successful", "win", "winning", "winner", "better", "improvement", "improved",
		"exciting", "excited", "enthusiasm", "enthusiastic", "optimistic", "hopeful", "promising", "favorable",
	}
	
	positiveWords := make(map[string]bool)
	for _, word := range words {
		positiveWords[word] = true
	}
	return positiveWords
}

// getNegativeWords returns common negative sentiment words
func getNegativeWords() map[string]bool {
	words := []string{
		"bad", "terrible", "awful", "horrible", "poor", "worst", "hate", "hated", "hating", "ugly", "disgusting",
		"disappointing", "disappointed", "disappointment", "fail", "failed", "failure", "wrong", "problem",
		"problems", "issue", "issues", "error", "errors", "difficult", "difficulty", "hard", "impossible",
		"negative", "unfortunate", "sad", "unhappy", "angry", "frustrated", "frustrating", "annoying", "annoyed",
		"concern", "concerned", "worried", "worry", "fear", "afraid", "scary", "dangerous", "risk", "threat",
		"damage", "damaged", "harm", "harmful", "worse", "loss", "lost", "losing", "loser", "decline", "declined",
	}
	
	negativeWords := make(map[string]bool)
	for _, word := range words {
		negativeWords[word] = true
	}
	return negativeWords
}
