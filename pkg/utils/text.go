package utils

import (
	"regexp"
	"strings"
	"unicode"
)

// Common stop words for text processing
var stopWords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
	"be": true, "by": true, "for": true, "from": true, "has": true, "he": true,
	"in": true, "is": true, "it": true, "its": true, "of": true, "on": true,
	"that": true, "the": true, "to": true, "was": true, "will": true, "with": true,
	"the": true, "this": true, "but": true, "they": true, "have": true, "had": true,
	"were": true, "been": true, "their": true, "she": true, "which": true, "do": true,
	"or": true, "if": true, "not": true, "what": true, "there": true, "can": true,
	"out": true, "up": true, "one": true, "about": true, "more": true, "so": true,
	"said": true, "when": true, "some": true, "into": true, "them": true, "then": true,
	"two": true, "how": true, "her": true, "than": true, "first": true, "way": true,
	"even": true, "back": true, "any": true, "over": true, "where": true, "just": true,
}

// CleanText removes extra whitespace and normalizes text
func CleanText(text string) string {
	// Remove extra whitespace
	space := regexp.MustCompile(`\s+`)
	text = space.ReplaceAllString(text, " ")
	
	// Trim leading and trailing whitespace
	text = strings.TrimSpace(text)
	
	return text
}

// RemoveStopWords filters out common stop words from text
func RemoveStopWords(text string) string {
	words := strings.Fields(strings.ToLower(text))
	filtered := make([]string, 0, len(words))
	
	for _, word := range words {
		// Remove punctuation from word edges
		word = strings.Trim(word, ".,!?;:'\"")
		if !stopWords[word] && len(word) > 0 {
			filtered = append(filtered, word)
		}
	}
	
	return strings.Join(filtered, " ")
}

// ExtractKeywords extracts important keywords from text
func ExtractKeywords(text string, limit int) []string {
	// Remove stop words
	cleaned := RemoveStopWords(text)
	
	// Count word frequency
	wordCount := make(map[string]int)
	words := strings.Fields(strings.ToLower(cleaned))
	
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:'\"")
		if len(word) > 2 { // Skip very short words
			wordCount[word]++
		}
	}
	
	// Sort by frequency
	type kv struct {
		Key   string
		Value int
	}
	
	var sorted []kv
	for k, v := range wordCount {
		sorted = append(sorted, kv{k, v})
	}
	
	// Sort by count
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Value > sorted[i].Value {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	// Extract top keywords
	keywords := make([]string, 0, limit)
	for i := 0; i < limit && i < len(sorted); i++ {
		keywords = append(keywords, sorted[i].Key)
	}
	
	return keywords
}

// TruncateText truncates text to a maximum length, preserving word boundaries
func TruncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	
	truncated := text[:maxLength]
	lastSpace := strings.LastIndex(truncated, " ")
	
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}
	
	return truncated + "..."
}

// NormalizeURL normalizes a URL for consistent comparison
func NormalizeURL(url string) string {
	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")
	
	// Remove fragment
	if idx := strings.Index(url, "#"); idx > 0 {
		url = url[:idx]
	}
	
	// Convert to lowercase for domain part
	if idx := strings.Index(url, "://"); idx > 0 {
		protocol := url[:idx+3]
		rest := url[idx+3:]
		
		if slashIdx := strings.Index(rest, "/"); slashIdx > 0 {
			domain := strings.ToLower(rest[:slashIdx])
			path := rest[slashIdx:]
			url = protocol + domain + path
		} else {
			url = protocol + strings.ToLower(rest)
		}
	}
	
	return url
}

// IsValidURL checks if a string is a valid URL
func IsValidURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9\-._~:/?#[\]@!$&'()*+,;=]+$`)
	return urlRegex.MatchString(url)
}

// SanitizeFilename removes invalid characters from a filename
func SanitizeFilename(filename string) string {
	// Replace invalid characters
	invalid := regexp.MustCompile(`[<>:"/\\|?*]`)
	filename = invalid.ReplaceAllString(filename, "_")
	
	// Remove control characters
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, filename)
	
	// Limit length
	if len(cleaned) > 255 {
		cleaned = cleaned[:255]
	}
	
	return cleaned
}

// GetDomainFromURL extracts the domain from a URL
func GetDomainFromURL(url string) string {
	// Remove protocol
	if idx := strings.Index(url, "://"); idx > 0 {
		url = url[idx+3:]
	}
	
	// Remove path
	if idx := strings.Index(url, "/"); idx > 0 {
		url = url[:idx]
	}
	
	// Remove port
	if idx := strings.Index(url, ":"); idx > 0 {
		url = url[:idx]
	}
	
	return strings.ToLower(url)
}

// CalculateReadingTime estimates reading time in minutes
func CalculateReadingTime(text string) int {
	wordsPerMinute := 200
	wordCount := len(strings.Fields(text))
	minutes := wordCount / wordsPerMinute
	
	if minutes < 1 {
		return 1
	}
	
	return minutes
}