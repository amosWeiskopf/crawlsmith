package crawler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCrawler(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		maxPerPath  int
		maxPathTypes int
		wantErr     bool
	}{
		{
			name:        "valid URL",
			url:         "https://example.com",
			maxPerPath:  10,
			maxPathTypes: 20,
			wantErr:     false,
		},
		{
			name:        "invalid URL",
			url:         "not-a-url",
			maxPerPath:  10,
			maxPathTypes: 20,
			wantErr:     true,
		},
		{
			name:        "empty URL",
			url:         "",
			maxPerPath:  10,
			maxPathTypes: 20,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := New(tt.url, tt.maxPerPath, tt.maxPathTypes)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, c)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, c)
			}
		})
	}
}

func TestCrawlSinglePage(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Test Page</title>
				<meta name="description" content="Test description">
			</head>
			<body>
				<h1>Test Content</h1>
				<p>This is test content.</p>
				<a href="/page2">Link to page 2</a>
				<a href="mailto:test@example.com">Email</a>
				<p>Call us: +1-234-567-8900</p>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	c, err := New(server.URL, 10, 10)
	require.NoError(t, err)

	result, err := c.Crawl()
	require.NoError(t, err)

	assert.Equal(t, 1, result.TotalPages)
	assert.Len(t, result.Pages, 1)
	
	page := result.Pages[0]
	assert.Equal(t, server.URL+"/", page.URL)
	assert.Equal(t, "Test Page", page.MetaTitle)
	assert.Equal(t, "Test description", page.MetaDescription)
	assert.Contains(t, page.Text, "Test Content")
	assert.Contains(t, page.Emails, "test@example.com")
	assert.Contains(t, page.Phones, "+1-234-567-8900")
}

func TestCrawlMultiplePages(t *testing.T) {
	pageCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++
		w.Header().Set("Content-Type", "text/html")
		
		switch r.URL.Path {
		case "/":
			w.Write([]byte(`
				<html><body>
				<a href="/page1">Page 1</a>
				<a href="/page2">Page 2</a>
				</body></html>
			`))
		case "/page1":
			w.Write([]byte(`<html><body>Page 1</body></html>`))
		case "/page2":
			w.Write([]byte(`<html><body>Page 2</body></html>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c, err := New(server.URL, 10, 10)
	require.NoError(t, err)

	result, err := c.Crawl()
	require.NoError(t, err)

	assert.Equal(t, 3, result.TotalPages)
	assert.Equal(t, 3, pageCount)
}

func TestRespectRobotsTxt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			w.Write([]byte(`
User-agent: *
Disallow: /private/
Allow: /public/
			`))
		case "/":
			w.Write([]byte(`
				<html><body>
				<a href="/public/page">Public</a>
				<a href="/private/page">Private</a>
				</body></html>
			`))
		case "/public/page":
			w.Write([]byte(`<html><body>Public page</body></html>`))
		case "/private/page":
			w.Write([]byte(`<html><body>Private page</body></html>`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	c, err := New(server.URL, 10, 10)
	require.NoError(t, err)

	result, err := c.Crawl()
	require.NoError(t, err)

	// Should crawl root and public page, but not private
	urls := make([]string, 0, len(result.Pages))
	for _, p := range result.Pages {
		urls = append(urls, p.URL)
	}
	
	assert.Contains(t, urls, server.URL+"/")
	assert.Contains(t, urls, server.URL+"/public/page")
	assert.NotContains(t, urls, server.URL+"/private/page")
}

func TestRateLimiting(t *testing.T) {
	requestTimes := []time.Time{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestTimes = append(requestTimes, time.Now())
		w.Write([]byte(`<html><body>Page</body></html>`))
	}))
	defer server.Close()

	c, err := New(server.URL, 10, 10)
	require.NoError(t, err)

	// Set aggressive rate limit for testing
	c.SetRateLimit(2) // 2 requests per second

	// Crawl should respect rate limit
	result, err := c.Crawl()
	require.NoError(t, err)

	if len(requestTimes) > 1 {
		// Check that requests are properly spaced
		for i := 1; i < len(requestTimes); i++ {
			gap := requestTimes[i].Sub(requestTimes[i-1])
			// Should be at least 400ms between requests (with some tolerance)
			assert.Greater(t, gap.Milliseconds(), int64(400))
		}
	}
}

func TestExtractContacts(t *testing.T) {
	html := `
		<html><body>
			<p>Email: contact@example.com</p>
			<p>Phone: +1-234-567-8900</p>
			<p>WhatsApp: +44 20 7946 0958</p>
			<a href="https://twitter.com/testuser">@testuser</a>
			<a href="https://linkedin.com/in/johndoe">LinkedIn Profile</a>
		</body></html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer server.Close()

	c, err := New(server.URL, 10, 10)
	require.NoError(t, err)

	result, err := c.Crawl()
	require.NoError(t, err)

	page := result.Pages[0]
	assert.Contains(t, page.Emails, "contact@example.com")
	assert.Contains(t, page.Phones, "+1-234-567-8900")
	assert.Contains(t, page.WhatsApps, "+44 20 7946 0958")
	assert.Contains(t, page.XHandles, "@testuser")
	assert.Contains(t, page.LinkedIns, "johndoe")
}

func TestSubdomainDiscovery(t *testing.T) {
	html := `
		<html><body>
			<a href="https://subdomain.example.com">Subdomain</a>
			<a href="https://another.example.com">Another</a>
			<a href="https://example.com">Main</a>
			<a href="https://external.com">External</a>
		</body></html>
	`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(html))
	}))
	defer server.Close()

	// Note: This test would need mock DNS resolution for subdomain discovery
	// For now, we just test that the crawler doesn't crash
	c, err := New(server.URL, 10, 10)
	require.NoError(t, err)

	result, err := c.Crawl()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func BenchmarkCrawl(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>Test</title></head>
			<body>
				<p>Content</p>
				<a href="/page1">Link 1</a>
				<a href="/page2">Link 2</a>
			</body>
			</html>
		`))
	}))
	defer server.Close()

	c, _ := New(server.URL, 10, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Crawl()
	}
}