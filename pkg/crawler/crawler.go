package crawler

import (
    "bytes"
    "container/list"
    "context"
    "fmt"
    "github.com/markusmobius/go-trafilatura"
    "github.com/temoto/robotstxt"
    "golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
    "golang.org/x/time/rate"
    "io"
    "log"
    "math/rand"
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "os"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
    "sync"
    "sync/atomic"
    "time"
)

var userAgents = []string{
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0",
}

func getRandomUserAgent() string {
    return userAgents[rand.Intn(len(userAgents))]
}

// Import models from internal package
// Page and Link types are defined in internal/models

type LinkQueueEntry struct {
    URL   string
    Depth int
}

type Crawler struct {
    domain       string
    client       *http.Client
    visited      sync.Map
    pathCounts   map[string]int
    pathPages    map[string][]Page
    pathDelays   map[string]time.Time
    mu           sync.Mutex
    wg           sync.WaitGroup
    sem          chan struct{}
    ctx          context.Context
    cancel       context.CancelFunc
    maxPerPath   int
    maxPathTypes int
    totalCrawled int32
    active       int32
    limiter      *rate.Limiter
    logger       *log.Logger
    proxyURLs    []string
    linkQueue    *list.List
    queueMu      sync.Mutex
    queueCond    *sync.Cond
}


func NewCrawler(startURL string, maxPerPath, maxPathTypes int) (*Crawler, error) {
    u, err := url.Parse(startURL)
    if err != nil {
        return nil, fmt.Errorf("invalid URL: %w", err)
    }

    // Extract the effective top-level domain plus one (eTLD+1)
    rootDomain, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
    if err != nil {
        return nil, fmt.Errorf("failed to extract root domain: %w", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)

    jar, _ := cookiejar.New(nil)
    transport := &http.Transport{
        MaxIdleConns:        50,
        MaxIdleConnsPerHost: 50,
        IdleConnTimeout:     30 * time.Second,
    }

    crawler := &Crawler{
        domain:       rootDomain,
        client:       &http.Client{Transport: transport, Timeout: 15 * time.Second, Jar: jar},
        pathCounts:   make(map[string]int),
        pathPages:    make(map[string][]Page),
        pathDelays:   make(map[string]time.Time),
        sem:          make(chan struct{}, 50),
        ctx:          ctx,
        cancel:       cancel,
        maxPerPath:   maxPerPath,
        maxPathTypes: maxPathTypes,
        limiter:      rate.NewLimiter(rate.Every(time.Second), 10),
        logger:       log.New(os.Stdout, "", 0),
        proxyURLs:    []string{},
        linkQueue:    list.New(),
    }
    crawler.queueCond = sync.NewCond(&crawler.queueMu)
    return crawler, nil
}

func (c *Crawler) isAllowedByRobots(pageURL string) bool {
    robotsURL := fmt.Sprintf("http://%s/robots.txt", c.domain)
    resp, err := c.client.Get(robotsURL)
    if err != nil || resp.StatusCode != http.StatusOK {
        return true
    }
    defer resp.Body.Close()

    robots, err := robotstxt.FromResponse(resp)
    if err != nil {
        return true
    }

    return robots.TestAgent(pageURL, "MyCrawler")
}

func (c *Crawler) Crawl(startURL string) {
    go c.trackProgress()
    go c.processQueue()
    c.wg.Add(1)
    c.sem <- struct{}{}
    go c.crawlPage(startURL, 0)

    go func() {
        c.wg.Wait()
        c.cancel()
    }()

    <-c.ctx.Done()
    c.logger.Println("\nTimeout reached or all pages crawled")
}

func (c *Crawler) processQueue() {
    for {
        select {
        case <-c.ctx.Done():
            return
        default:
            c.queueMu.Lock()
            for c.linkQueue.Len() == 0 {
                c.queueCond.Wait()
                if c.ctx.Err() != nil {
                    c.queueMu.Unlock()
                    return
                }
            }
            elem := c.linkQueue.Front()
            entry := elem.Value.(LinkQueueEntry)
            c.linkQueue.Remove(elem)
            c.queueMu.Unlock()

            select {
            case c.sem <- struct{}{}:
                c.wg.Add(1)
                atomic.AddInt32(&c.active, 1)
                go c.crawlPage(entry.URL, entry.Depth)
            case <-c.ctx.Done():
                return
            default:
                c.queueMu.Lock()
                c.linkQueue.PushBack(entry)
                c.queueMu.Unlock()
                time.Sleep(100 * time.Millisecond)
            }
        }
    }
}

func (c *Crawler) crawlPage(pageURL string, depth int) {
    defer func() {
        <-c.sem
        c.wg.Done()
        atomic.AddInt32(&c.active, -1)
        c.queueCond.Signal()
    }()

    time.Sleep(time.Duration(50+rand.Intn(200)) * time.Millisecond)

    select {
    case <-c.ctx.Done():
        c.logger.Printf("Context canceled for %s\n", pageURL)
        return
    default:
    }

    if !c.isAllowedByRobots(pageURL) {
        c.logger.Printf("Skipped %s (disallowed by robots.txt)\n", pageURL)
        return
    }

    if _, loaded := c.visited.LoadOrStore(pageURL, true); loaded {
        c.logger.Printf("Skipped %s (already visited)\n", pageURL)
        return
    }

    if err := c.limiter.Wait(c.ctx); err != nil {
        c.logger.Printf("Rate limiter error for %s: %v\n", pageURL, err)
        return
    }

    ctx, cancel := context.WithTimeout(c.ctx, 20*time.Second)
    defer cancel()
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
    if err != nil {
        c.logger.Printf("Request error for %s: %v\n", pageURL, err)
        return
    }
    req.Header.Set("User-Agent", getRandomUserAgent())
    req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
    req.Header.Set("Accept-Language", "en-US,en;q=0.5")
    req.Header.Set("Connection", "keep-alive")

    for retries := 0; retries < 3; retries++ {
        resp, err := c.client.Do(req)
        if err == nil && resp.StatusCode == http.StatusOK {
            defer resp.Body.Close()
            body, err := io.ReadAll(resp.Body)
            if err != nil {
                c.logger.Printf("Body read error for %s: %v\n", pageURL, err)
                return
            }
            if strings.Contains(string(body), "cf-browser-verification") || strings.Contains(string(body), "Access denied") {
                c.logger.Printf("Anti-bot protection detected for %s\n", pageURL)
                return
            }
            if !isWebpageMIME(resp.Header.Get("Content-Type")) {
                c.logger.Printf("Non-webpage MIME for %s: %s\n", pageURL, resp.Header.Get("Content-Type"))
                return
            }
            etag := resp.Header.Get("ETag")
            if etag == "" {
                etag = "N/A"
            }
            text, links, title, desc, emails, phones, whatsapps, xHandles, linkedins := extractTextLinksAndMetadata(body, pageURL, c.domain)
            pathType := getPathType(pageURL)
            normalized := normalizeText(text)

            c.mu.Lock()
            if c.pathCounts[pathType] >= c.maxPerPath || (len(c.pathCounts) >= c.maxPathTypes && c.pathCounts[pathType] == 0) {
                c.mu.Unlock()
                c.logger.Printf("Skipped %s (path limit reached: %s)\n", pageURL, pathType)
                return
            }
            lastCrawl, exists := c.pathDelays[pathType]
            if exists && time.Since(lastCrawl) < 500*time.Millisecond {
                time.Sleep(500*time.Millisecond - time.Since(lastCrawl))
            }
            c.pathDelays[pathType] = time.Now()
            if normalized != "" {
				c.pathPages[pathType] = append(c.pathPages[pathType], Page{
					URL:             pageURL,
					Text:            normalized,
					Links:           links,
					MetaTitle:       title,
					MetaDescription: desc,
					ETag:            etag,
					Emails: emails,
					Phones: phones,
					WhatsApps: whatsapps,
					XHandles: xHandles,
					LinkedIns: linkedins,
				})
                c.pathCounts[pathType]++
                atomic.AddInt32(&c.totalCrawled, 1)
                c.logger.Printf("Crawled %s (depth: %d, path: %s)\n", pageURL, depth, pathType)
            }
            c.mu.Unlock()

            for _, link := range links {
                absLink := resolveURL(pageURL, link.ToURL)
                if !isWebpageURL(absLink) {
                    c.logger.Printf("Skipped link %s (non-webpage URL)\n", absLink)
                    continue
                }

                u, err := url.Parse(absLink)
                if err != nil || u.Hostname() == "" {
                    c.logger.Printf("Skipped link %s (invalid hostname)\n", absLink)
                    continue
                }

                linkedDomain, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
                if err != nil || linkedDomain != c.domain {
                    // External domain, skip crawling
                    continue
                }

                select {
                case c.sem <- struct{}{}:
                    c.wg.Add(1)
                    atomic.AddInt32(&c.active, 1)
                    go c.crawlPage(absLink, depth+1)
                default:
                    c.queueMu.Lock()
                    c.linkQueue.PushBack(LinkQueueEntry{URL: absLink, Depth: depth + 1})
                    c.queueMu.Unlock()
                    c.queueCond.Signal()
                    c.logger.Printf("Queued link %s (semaphore full)\n", absLink)
                }
            }
            break
        }
        if err != nil {
            c.logger.Printf("Fetch error for %s (retry %d): %v\n", pageURL, retries+1, err)
        } else {
            c.logger.Printf("Non-OK status for %s (retry %d): %d\n", pageURL, retries+1, resp.StatusCode)
            resp.Body.Close()
        }
        time.Sleep(time.Duration(100*(1<<retries)) * time.Millisecond)
        if retries == 2 {
            c.logger.Printf("Giving up on %s after 3 retries\n", pageURL)
            return
        }
    }
}

func isWebpageURL(pageURL string) bool {
    lowercaseURL := strings.ToLower(pageURL)
    nonWebExts := []string{".jpg", ".jpeg", ".png", ".gif", ".pdf", ".zip", ".mp4", ".mp3", ".css", ".js"}
    for _, ext := range nonWebExts {
        if strings.HasSuffix(lowercaseURL, ext) {
            return false
        }
    }
    return !strings.Contains(pageURL, "#")
}

func isWebpageMIME(contentType string) bool {
    mimeType := strings.Split(strings.ToLower(contentType), ";")[0]
    webpageMIMEs := []string{"text/html", "application/xhtml+xml", "application/xhtml", "text/xml", "application/xml"}
    for _, mime := range webpageMIMEs {
        if mime == mimeType {
            return true
        }
    }
    return false
}

func extractTextLinksAndMetadata(body []byte, baseURL, domain string) (string, []Link, string, string, []string, []string, []string, []string, []string) {
    var text, title, desc string
    result, err := trafilatura.Extract(bytes.NewReader(body), trafilatura.Options{})
    if err == nil && result != nil && result.ContentText != "" {
        text = strings.ReplaceAll(result.ContentText, "\n", ";")
    } else {
        text = fallbackTextExtraction(body)
    }

    doc, err := html.Parse(bytes.NewReader(body))
    if err != nil {
        return text, nil, "x", "x", nil, nil, nil, nil, nil
    }
    var links []Link
    seen := make(map[string]bool)
    var foundTitle bool

    var f func(*html.Node)
    f = func(n *html.Node) {
        if n.Type == html.ElementNode {
            switch n.Data {
            case "a":
                var href, anchorText string
                for _, attr := range n.Attr {
                    if attr.Key == "href" {
                        href = attr.Val
                    }
                }
                if href != "" {
                    var extractText func(*html.Node) string
                    extractText = func(n *html.Node) string {
                        if n.Type == html.TextNode {
                            return strings.TrimSpace(n.Data)
                        }
                        if n.Type == html.ElementNode && n.Data == "img" {
                            for _, attr := range n.Attr {
                                if attr.Key == "src" {
                                    return resolveURL(baseURL, attr.Val)
                                }
                            }
                        }
                        var text strings.Builder
                        for c := n.FirstChild; c != nil; c = c.NextSibling {
                            text.WriteString(extractText(c))
                        }
                        return text.String()
                    }
                    anchorText = strings.TrimSpace(extractText(n))
                    if anchorText == "" {
                        anchorText = "N/A"
                    }
                    if !seen[href] {
                        seen[href] = true
                        links = append(links, Link{ToURL: href, AnchorText: anchorText})
                    }
                }
            case "title":
                if !foundTitle && n.FirstChild != nil {
                    title = strings.TrimSpace(n.FirstChild.Data)
                    foundTitle = true
                }
            case "meta":
                var isDesc bool
                var content string
                for _, attr := range n.Attr {
                    if strings.ToLower(attr.Key) == "name" && strings.ToLower(attr.Val) == "description" {
                        isDesc = true
                    }
                    if strings.ToLower(attr.Key) == "content" {
                        content = strings.TrimSpace(attr.Val)
                    }
                }
                if isDesc && content != "" {
                    desc = content
                }
            }
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            f(c)
        }
    }
    f(doc)

    if title == "" {
        title = "x"
    }
    if desc == "" {
        desc = "x"
    }
    // Extract mailto links
	rawBody := string(body)
	combinedText := text + ";" + rawBody

	// Deduplication maps
	seenEmail := make(map[string]bool)
	seenPhone := make(map[string]bool)
	seenWhatsapp := make(map[string]bool)
	seenXHandle := make(map[string]bool)
	seenLinkedIn := make(map[string]bool)

	// Result slices
	var emails, phones, whatsapps, xHandles, linkedins []string

	// Regexes
	emailRegex := regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	phoneRegex := regexp.MustCompile(`(?i)(\+?\d[\d\-\s]{7,}\d)`)
	whatsappRegex := regexp.MustCompile(`(?i)(https?://)?(wa\.me|api\.whatsapp\.com)/[^\s"'<>\)]+`)
	twitterRegex := regexp.MustCompile(`(?i)(https?://)?(www\.)?(x\.com|twitter\.com)/[a-zA-Z0-9_]{1,15}|@[a-zA-Z0-9_]{1,15}`)
	linkedinRegex := regexp.MustCompile(`(?i)(https?://)?(www\.)?linkedin\.com/in/[a-zA-Z0-9\-_%]+`)

	// Extraction
	for _, match := range emailRegex.FindAllString(combinedText, -1) {
		if !seenEmail[match] {
			seenEmail[match] = true
			emails = append(emails, match)
		}
	}
	for _, match := range phoneRegex.FindAllString(combinedText, -1) {
		if !seenPhone[match] {
			seenPhone[match] = true
			phones = append(phones, match)
		}
	}
	for _, match := range whatsappRegex.FindAllString(combinedText, -1) {
		if !seenWhatsapp[match] {
			seenWhatsapp[match] = true
			whatsapps = append(whatsapps, match)
		}
	}
	for _, match := range twitterRegex.FindAllString(combinedText, -1) {
		if !seenXHandle[match] {
			seenXHandle[match] = true
			xHandles = append(xHandles, match)
		}
	}
	for _, match := range linkedinRegex.FindAllString(combinedText, -1) {
		if !seenLinkedIn[match] {
			seenLinkedIn[match] = true
			linkedins = append(linkedins, match)
		}
	}

	return text, links, title, desc, emails, phones, whatsapps, xHandles, linkedins
}

func fallbackTextExtraction(body []byte) string {
    doc, err := html.Parse(bytes.NewReader(body))
    if err != nil {
        return ""
    }
    var b strings.Builder
    var f func(*html.Node)
    f = func(n *html.Node) {
        if n.Type == html.TextNode {
            b.WriteString(n.Data + ";")
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            f(c)
        }
    }
    f(doc)
    return b.String()
}

func getPathType(rawURL string) string {
    u, _ := url.Parse(rawURL)
    u.RawQuery = ""
    segments := strings.Split(strings.Trim(u.Path, "/"), "/")
    if len(segments) > 0 && segments[0] != "" {
        return "/" + segments[0]
    }
    return "/"
}

func resolveURL(base, ref string) string {
    baseURL, err := url.Parse(base)
    if err != nil {
        return ref
    }
    refURL, err := url.Parse(ref)
    if err != nil {
        return ref
    }
    return baseURL.ResolveReference(refURL).String()
}

func normalizeText(s string) string {
    re := regexp.MustCompile(`[^\w\s.,!?-]`)
    return strings.ToLower(strings.TrimSpace(re.ReplaceAllString(s, "")))
}

func (c *Crawler) trackProgress() {
    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-c.ctx.Done():
            return
        case <-ticker.C:
            c.mu.Lock()
            c.logger.Printf("\rCrawled: %d pages | %d path types | Active: %d", atomic.LoadInt32(&c.totalCrawled), len(c.pathCounts), atomic.LoadInt32(&c.active))
            c.mu.Unlock()
        }
    }
}

func (c *Crawler) SaveResults() error {
    if err := os.MkdirAll(".", 0755); err != nil {
        return fmt.Errorf("mkdir failed: %w", err)
    }

    f1, err := os.Create(filepath.Join(".", "urls_with_text.txt"))
    if err != nil {
        return fmt.Errorf("create urls_with_text.txt failed: %w", err)
    }
    defer f1.Close()

    f2, err := os.Create(filepath.Join(".", "all_texts.txt"))
    if err != nil {
        return fmt.Errorf("create all_texts.txt failed: %w", err)
    }
    defer f2.Close()

    f3, err := os.Create(filepath.Join(".", "origin_metadata.tsv"))
    if err != nil {
        return fmt.Errorf("create origin_metadata.tsv failed: %w", err)
    }
    defer f3.Close()

    f4, err := os.Create(filepath.Join(".", "internal_links_map.tsv"))
    if err != nil {
        return fmt.Errorf("create internal_links_map.tsv failed: %w", err)
    }
    defer f4.Close()

    f5, err := os.Create(filepath.Join(".", "external_links_map.tsv"))
    if err != nil {
        return fmt.Errorf("create external_links_map.tsv failed: %w", err)
    }
    defer f5.Close()

    f6, err := os.Create(filepath.Join(".", "internal_links_map_summary.tsv"))
    if err != nil {
        return fmt.Errorf("create internal_links_map_summary.tsv failed: %w", err)
    }
    defer f6.Close()

    f7, err := os.Create(filepath.Join(".", "external_links_map_summary.tsv"))
    if err != nil {
        return fmt.Errorf("create external_links_map_summary.tsv failed: %w", err)
    }
    defer f7.Close()

    f8, err := os.Create(filepath.Join(".", "external_top_linked_domains.tsv"))
    if err != nil {
        return fmt.Errorf("create external_top_linked_domains.tsv failed: %w", err)
    }
    defer f8.Close()

    var allText strings.Builder
    var rows, internalLinks, externalLinks []string
    internalPairs := make(map[string]map[string]bool)
    externalPairs := make(map[string]map[string]bool)
    domainCounts := make(map[string]map[string]bool)

    c.mu.Lock()
    for _, pages := range c.pathPages {
        for _, p := range pages {
			emailList := strings.Join(p.Emails, " ")
			rows = append(rows, fmt.Sprintf("%s\t%s\t%s", p.URL, p.Text, emailList))
            allText.WriteString(p.Text + ";")
            fmt.Fprintf(f3, "%s\t%s\t%s\t%s\n", p.URL, p.MetaTitle, p.MetaDescription, p.ETag)
            for _, link := range p.Links {
                absLink := resolveURL(p.URL, link.ToURL)
                if strings.Contains(absLink, "#") {
                    continue
                }
                u, err := url.Parse(absLink)
                anchorText := strings.ReplaceAll(link.AnchorText, "\t", " ")
                if err == nil {
                    pairKey := p.URL + "\t" + absLink
					linkedDomain, err := publicsuffix.EffectiveTLDPlusOne(u.Hostname())
					if err == nil && linkedDomain == c.domain {
                        if !strings.Contains(p.URL, "#") {
                            internalLinks = append(internalLinks, fmt.Sprintf("%s\t%s\t%s", p.URL, absLink, anchorText))
                            if _, exists := internalPairs[absLink]; !exists {
                                internalPairs[absLink] = make(map[string]bool)
                            }
                            internalPairs[absLink][p.URL] = true
                        }
                    } else {
                        externalLinks = append(externalLinks, fmt.Sprintf("%s\t%s\t%s", p.URL, absLink, anchorText))
                        if _, exists := externalPairs[absLink]; !exists {
                            externalPairs[absLink] = make(map[string]bool)
                        }
                        externalPairs[absLink][p.URL] = true
                        domain := u.Hostname()
                        if _, exists := domainCounts[domain]; !exists {
                            domainCounts[domain] = make(map[string]bool)
                        }
                        domainCounts[domain][pairKey] = true
                    }
                }
            }
        }
    }
    c.mu.Unlock()

    sort.Strings(rows)
    for _, row := range rows {
        fmt.Fprintln(f1, row)
    }
    fmt.Fprint(f2, normalizeText(allText.String()))

    sort.Strings(internalLinks)
    fmt.Fprintln(f4, "from_url\tto_url\tanchor_text/img_url")
    for _, link := range internalLinks {
        fmt.Fprintln(f4, link)
    }

    sort.Strings(externalLinks)
    fmt.Fprintln(f5, "from_url\tto_url\tanchor_text/img_url")
    for _, link := range externalLinks {
        fmt.Fprintln(f5, link)
    }

    type summaryEntry struct {
        toURL string
        count int
    }
    var internalSummary []summaryEntry
    for toURL, fromURLs := range internalPairs {
        internalSummary = append(internalSummary, summaryEntry{toURL, len(fromURLs)})
    }
    sort.Slice(internalSummary, func(i, j int) bool {
        if internalSummary[i].count == internalSummary[j].count {
            return internalSummary[i].toURL < internalSummary[j].toURL
        }
        return internalSummary[i].count > internalSummary[j].count
    })
    fmt.Fprintln(f6, "to_url\tcount_uniques")
    for _, entry := range internalSummary {
        fmt.Fprintf(f6, "%s\t%d\n", entry.toURL, entry.count)
    }

    var externalSummary []summaryEntry
    for toURL, fromURLs := range externalPairs {
        externalSummary = append(externalSummary, summaryEntry{toURL, len(fromURLs)})
    }
    sort.Slice(externalSummary, func(i, j int) bool {
        if externalSummary[i].count == externalSummary[j].count {
            return externalSummary[i].toURL < externalSummary[j].toURL
        }
        return externalSummary[i].count > externalSummary[j].count
    })
    fmt.Fprintln(f7, "to_url\tcount_uniques")
    for _, entry := range externalSummary {
        fmt.Fprintf(f7, "%s\t%d\n", entry.toURL, entry.count)
    }

    var domainSummary []summaryEntry
    for domain, pairs := range domainCounts {
        domainSummary = append(domainSummary, summaryEntry{domain, len(pairs)})
    }
    sort.Slice(domainSummary, func(i, j int) bool {
        if domainSummary[i].count == domainSummary[j].count {
            return domainSummary[i].toURL < domainSummary[j].toURL
        }
        return domainSummary[i].count > domainSummary[j].count
    })
    fmt.Fprintln(f8, "domain\tcount_uniques")
    for _, entry := range domainSummary {
        fmt.Fprintf(f8, "%s\t%d\n", entry.toURL, entry.count)
    }

    return nil
}

// Run executes the crawler with the given URL
func Run(startURL string) error {
    if len(os.Args) < 2 {
        fmt.Fprintln(os.Stderr, "Usage: go run sickcrawler.go <start-url>")
        os.Exit(1)
    }

    rand.Seed(time.Now().UnixNano())
    crawler, err := NewCrawler(os.Args[1], 1000, 1000)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Init error: %v\n", err)
        os.Exit(1)
    }

    crawler.Crawl(os.Args[1])
    if err := crawler.SaveResults(); err != nil {
        fmt.Fprintf(os.Stderr, "Save error: %v\n", err)
        os.Exit(1)
    }
    fmt.Println("\nDone. Output saved to ./urls_with_text.txt, ./all_texts.txt, ./origin_metadata.tsv, ./internal_links_map.tsv, ./external_links_map.tsv, ./internal_links_map_summary.tsv, ./external_links_map_summary.tsv, and ./external_top_linked_domains.tsv")
}
