package crawler

import (
	"context"
	"github.com/amosWeiskopf/crawlsmith/internal/models"
)

// Crawler defines the interface for web crawling operations
type Crawler interface {
	// Crawl starts the crawling process
	Crawl() (*models.CrawlResult, error)
	
	// CrawlWithContext starts crawling with a context for cancellation
	CrawlWithContext(ctx context.Context) (*models.CrawlResult, error)
	
	// SetRateLimit sets the requests per second limit
	SetRateLimit(requestsPerSecond int)
	
	// SetMaxDepth sets the maximum crawl depth
	SetMaxDepth(depth int)
	
	// SetUserAgent sets the user agent string
	SetUserAgent(userAgent string)
	
	// AddExcludePattern adds a URL pattern to exclude from crawling
	AddExcludePattern(pattern string)
	
	// EnableJavaScript enables JavaScript rendering (requires headless browser)
	EnableJavaScript(enabled bool)
}

// Options contains configuration for the crawler
type Options struct {
	MaxPerPath       int      // Maximum pages per path pattern
	MaxPathTypes     int      // Maximum number of path types
	MaxDepth         int      // Maximum crawl depth
	RequestsPerSec   int      // Rate limit
	UserAgent        string   // User agent string
	FollowRobotsTxt  bool     // Respect robots.txt
	ExtractContacts  bool     // Extract contact information
	EnableJS         bool     // Enable JavaScript rendering
	Timeout          int      // Request timeout in seconds
	ExcludePatterns  []string // URL patterns to exclude
	IncludeSubdomains bool    // Include subdomains in crawl
}