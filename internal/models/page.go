package models

import "time"

// Page represents a crawled web page
type Page struct {
	URL             string    `json:"url"`
	Text            string    `json:"text"`
	Links           []Link    `json:"links"`
	MetaTitle       string    `json:"meta_title"`
	MetaDescription string    `json:"meta_description"`
	ETag            string    `json:"etag"`
	Emails          []string  `json:"emails"`
	Phones          []string  `json:"phones"`
	WhatsApps       []string  `json:"whatsapps"`
	XHandles        []string  `json:"x_handles"`
	LinkedIns       []string  `json:"linkedins"`
	CrawledAt       time.Time `json:"crawled_at"`
	StatusCode      int       `json:"status_code"`
	PageRank        float64   `json:"pagerank"`
}

// Link represents a hyperlink from one page to another
type Link struct {
	ToURL      string `json:"to_url"`
	AnchorText string `json:"anchor_text"`
}

// CrawlResult contains the results of a crawl operation
type CrawlResult struct {
	Domain       string    `json:"domain"`
	Pages        []Page    `json:"pages"`
	TotalPages   int       `json:"total_pages"`
	CrawlTime    time.Time `json:"crawl_time"`
	ErrorCount   int       `json:"error_count"`
	Subdomains   []string  `json:"subdomains"`
}

// SEOReport represents a comprehensive SEO analysis report
type SEOReport struct {
	Domain           string            `json:"domain"`
	GeneratedAt      time.Time         `json:"generated_at"`
	ExecutiveSummary ExecutiveSummary  `json:"executive_summary"`
	Scores           OverallScores     `json:"scores"`
	KeyFindings      []Finding         `json:"key_findings"`
	Recommendations  []Recommendation  `json:"recommendations"`
	DataSources      []string          `json:"data_sources"`
}

// ExecutiveSummary provides high-level SEO insights
type ExecutiveSummary struct {
	OverallGrade    string   `json:"overall_grade"`
	OverallScore    float64  `json:"overall_score"`
	Strengths       []string `json:"strengths"`
	Weaknesses      []string `json:"weaknesses"`
	TopPriorities   []string `json:"top_priorities"`
	EstimatedImpact string   `json:"estimated_impact"`
}

// OverallScores contains various SEO metric scores
type OverallScores struct {
	Technical   float64 `json:"technical"`
	Content     float64 `json:"content"`
	Performance float64 `json:"performance"`
	Security    float64 `json:"security"`
	Overall     float64 `json:"overall"`
}

// Finding represents an SEO issue or observation
type Finding struct {
	Category    string `json:"category"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Details     string `json:"details,omitempty"`
}

// Recommendation represents an actionable SEO improvement
type Recommendation struct {
	Priority    string `json:"priority"`
	Category    string `json:"category"`
	Action      string `json:"action"`
	Impact      string `json:"impact"`
	Effort      string `json:"effort"`
	Description string `json:"description"`
}