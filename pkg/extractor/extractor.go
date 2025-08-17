package extractor

import (
	"regexp"
	"strings"
	"golang.org/x/net/html"
	"github.com/markusmobius/go-trafilatura"
)

// Extractor handles content extraction from HTML
type Extractor struct {
	emailRegex    *regexp.Regexp
	phoneRegex    *regexp.Regexp
	whatsappRegex *regexp.Regexp
}

// New creates a new Extractor instance
func New() *Extractor {
	return &Extractor{
		emailRegex:    regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
		phoneRegex:    regexp.MustCompile(`(?:\+?[1-9]\d{0,2}[\s.-]?)?\(?\d{1,4}\)?[\s.-]?\d{1,4}[\s.-]?\d{1,4}[\s.-]?\d{0,4}`),
		whatsappRegex: regexp.MustCompile(`(?:whatsapp|wa\.me)/?\+?(\d{10,15})`),
	}
}

// ExtractText extracts clean text from HTML using trafilatura
func (e *Extractor) ExtractText(htmlContent string) (string, error) {
	result, err := trafilatura.Extract(strings.NewReader(htmlContent), trafilatura.Options{})
	if err != nil {
		return "", err
	}
	if result == nil {
		return "", nil
	}
	return result.ContentText, nil
}

// ExtractMetadata extracts meta tags from HTML
func (e *Extractor) ExtractMetadata(htmlContent string) (title, description string, err error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", "", err
	}
	
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "title" && n.FirstChild != nil {
				title = n.FirstChild.Data
			} else if n.Data == "meta" {
				var name, content string
				for _, attr := range n.Attr {
					if attr.Key == "name" && attr.Val == "description" {
						name = attr.Val
					}
					if attr.Key == "content" {
						content = attr.Val
					}
				}
				if name == "description" {
					description = content
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	
	extract(doc)
	return title, description, nil
}

// ExtractEmails finds all email addresses in the content
func (e *Extractor) ExtractEmails(content string) []string {
	matches := e.emailRegex.FindAllString(content, -1)
	return uniqueStrings(matches)
}

// ExtractPhones finds all phone numbers in the content
func (e *Extractor) ExtractPhones(content string) []string {
	matches := e.phoneRegex.FindAllString(content, -1)
	cleaned := make([]string, 0, len(matches))
	for _, match := range matches {
		// Clean and validate phone numbers
		cleaned = append(cleaned, cleanPhoneNumber(match))
	}
	return uniqueStrings(cleaned)
}

// ExtractSocialHandles extracts social media handles
func (e *Extractor) ExtractSocialHandles(content string) (twitter, linkedin []string) {
	// Twitter/X handles
	twitterRegex := regexp.MustCompile(`(?:twitter\.com|x\.com)/([a-zA-Z0-9_]+)`)
	twitterMatches := twitterRegex.FindAllStringSubmatch(content, -1)
	for _, match := range twitterMatches {
		if len(match) > 1 {
			twitter = append(twitter, "@"+match[1])
		}
	}
	
	// LinkedIn profiles
	linkedinRegex := regexp.MustCompile(`linkedin\.com/in/([a-zA-Z0-9-]+)`)
	linkedinMatches := linkedinRegex.FindAllStringSubmatch(content, -1)
	for _, match := range linkedinMatches {
		if len(match) > 1 {
			linkedin = append(linkedin, match[1])
		}
	}
	
	return uniqueStrings(twitter), uniqueStrings(linkedin)
}

// ExtractLinks extracts all links from HTML
func (e *Extractor) ExtractLinks(htmlContent string, baseURL string) ([]Link, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}
	
	var links []Link
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var href, text string
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href = attr.Val
				}
			}
			if n.FirstChild != nil {
				text = extractText(n)
			}
			if href != "" {
				links = append(links, Link{
					URL:        resolveURL(baseURL, href),
					AnchorText: strings.TrimSpace(text),
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	
	extract(doc)
	return links, nil
}

// Link represents an extracted hyperlink
type Link struct {
	URL        string
	AnchorText string
}

// Helper functions

func uniqueStrings(strings []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range strings {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func cleanPhoneNumber(phone string) string {
	// Remove common separators but keep the number structure
	cleaned := strings.ReplaceAll(phone, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	return cleaned
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += extractText(c)
	}
	return text
}

func resolveURL(base, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}
	if strings.HasPrefix(href, "/") {
		// Absolute path
		if idx := strings.Index(base, "://"); idx > 0 {
			if idx2 := strings.Index(base[idx+3:], "/"); idx2 > 0 {
				return base[:idx+3+idx2] + href
			}
			return base + href
		}
	}
	// Relative path
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + href
}