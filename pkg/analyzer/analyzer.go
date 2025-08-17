package analyzer

import (
	"fmt"
	"math"
	"sort"
	"strings"
	
	"github.com/amosWeiskopf/crawlsmith/internal/models"
)

// Analyzer performs SEO and content analysis
type Analyzer struct {
	config *Config
}

// Config holds analyzer configuration
type Config struct {
	EnableAI           bool
	OpenAIKey          string
	AnalyzePageRank    bool
	AnalyzeContent     bool
	AnalyzeTechnical   bool
	AnalyzePerformance bool
}

// New creates a new Analyzer instance
func New() *Analyzer {
	return &Analyzer{
		config: &Config{
			AnalyzePageRank:    true,
			AnalyzeContent:     true,
			AnalyzeTechnical:   true,
			AnalyzePerformance: true,
		},
	}
}

// NewWithConfig creates an Analyzer with custom configuration
func NewWithConfig(config *Config) *Analyzer {
	return &Analyzer{config: config}
}

// Analyze performs comprehensive SEO analysis on crawl results
func (a *Analyzer) Analyze(crawlResult *models.CrawlResult, full bool) (*models.SEOReport, error) {
	report := &models.SEOReport{
		Domain:      crawlResult.Domain,
		GeneratedAt: crawlResult.CrawlTime,
	}
	
	// Calculate PageRank if enabled
	if a.config.AnalyzePageRank {
		a.calculatePageRank(crawlResult)
	}
	
	// Analyze content
	if a.config.AnalyzeContent {
		contentScore := a.analyzeContent(crawlResult)
		report.Scores.Content = contentScore
	}
	
	// Technical SEO analysis
	if a.config.AnalyzeTechnical {
		technicalScore := a.analyzeTechnical(crawlResult)
		report.Scores.Technical = technicalScore
	}
	
	// Performance analysis
	if a.config.AnalyzePerformance {
		performanceScore := a.analyzePerformance(crawlResult)
		report.Scores.Performance = performanceScore
	}
	
	// Calculate overall score
	report.Scores.Overall = a.calculateOverallScore(report.Scores)
	
	// Generate findings and recommendations
	report.KeyFindings = a.generateFindings(crawlResult)
	report.Recommendations = a.generateRecommendations(report.KeyFindings)
	
	// Generate executive summary
	report.ExecutiveSummary = a.generateExecutiveSummary(report)
	
	return report, nil
}

// calculatePageRank implements the PageRank algorithm
func (a *Analyzer) calculatePageRank(crawlResult *models.CrawlResult) {
	const (
		dampingFactor = 0.85
		iterations    = 100
	)
	
	// Build link graph
	linkGraph := make(map[string][]string)
	inboundLinks := make(map[string][]string)
	
	for _, page := range crawlResult.Pages {
		for _, link := range page.Links {
			linkGraph[page.URL] = append(linkGraph[page.URL], link.ToURL)
			inboundLinks[link.ToURL] = append(inboundLinks[link.ToURL], page.URL)
		}
	}
	
	// Initialize PageRank values
	pageCount := float64(len(crawlResult.Pages))
	pageRank := make(map[string]float64)
	for _, page := range crawlResult.Pages {
		pageRank[page.URL] = 1.0 / pageCount
	}
	
	// Iterate PageRank calculation
	for i := 0; i < iterations; i++ {
		newPageRank := make(map[string]float64)
		
		for _, page := range crawlResult.Pages {
			rank := (1.0 - dampingFactor) / pageCount
			
			for _, inbound := range inboundLinks[page.URL] {
				outboundCount := float64(len(linkGraph[inbound]))
				if outboundCount > 0 {
					rank += dampingFactor * pageRank[inbound] / outboundCount
				}
			}
			
			newPageRank[page.URL] = rank
		}
		
		pageRank = newPageRank
	}
	
	// Update pages with PageRank scores
	for i := range crawlResult.Pages {
		crawlResult.Pages[i].PageRank = pageRank[crawlResult.Pages[i].URL]
	}
}

// analyzeContent evaluates content quality
func (a *Analyzer) analyzeContent(crawlResult *models.CrawlResult) float64 {
	score := 0.0
	factors := 0
	
	for _, page := range crawlResult.Pages {
		// Check meta title
		if len(page.MetaTitle) > 0 && len(page.MetaTitle) <= 60 {
			score += 1.0
		} else if len(page.MetaTitle) > 0 {
			score += 0.5
		}
		factors++
		
		// Check meta description
		if len(page.MetaDescription) >= 120 && len(page.MetaDescription) <= 160 {
			score += 1.0
		} else if len(page.MetaDescription) > 0 {
			score += 0.5
		}
		factors++
		
		// Check content length
		wordCount := len(strings.Fields(page.Text))
		if wordCount >= 300 {
			score += 1.0
		} else if wordCount >= 100 {
			score += 0.5
		}
		factors++
	}
	
	if factors == 0 {
		return 0
	}
	
	return (score / float64(factors)) * 100
}

// analyzeTechnical evaluates technical SEO factors
func (a *Analyzer) analyzeTechnical(crawlResult *models.CrawlResult) float64 {
	score := 0.0
	factors := 0
	
	// Check for duplicate titles
	titles := make(map[string]int)
	for _, page := range crawlResult.Pages {
		titles[page.MetaTitle]++
	}
	
	duplicateTitles := 0
	for _, count := range titles {
		if count > 1 {
			duplicateTitles++
		}
	}
	
	if duplicateTitles == 0 {
		score += 1.0
	} else {
		score += math.Max(0, 1.0-float64(duplicateTitles)/float64(len(titles)))
	}
	factors++
	
	// Check for broken links (simplified - would need actual HTTP checks)
	brokenLinks := 0
	for _, page := range crawlResult.Pages {
		if page.StatusCode >= 400 {
			brokenLinks++
		}
	}
	
	if brokenLinks == 0 {
		score += 1.0
	} else {
		score += math.Max(0, 1.0-float64(brokenLinks)/float64(len(crawlResult.Pages)))
	}
	factors++
	
	// Check for proper URL structure
	for _, page := range crawlResult.Pages {
		if !strings.Contains(page.URL, "?") && !strings.Contains(page.URL, "#") {
			score += 1.0
		} else {
			score += 0.5
		}
		factors++
		break // Sample check
	}
	
	if factors == 0 {
		return 0
	}
	
	return (score / float64(factors)) * 100
}

// analyzePerformance evaluates site performance
func (a *Analyzer) analyzePerformance(crawlResult *models.CrawlResult) float64 {
	// Simplified performance scoring
	// In a real implementation, this would measure:
	// - Page load times
	// - Resource optimization
	// - Image optimization
	// - Caching headers
	
	score := 75.0 // Default moderate score
	
	// Penalize for too many pages (site might be bloated)
	if crawlResult.TotalPages > 1000 {
		score -= 10
	}
	
	// Bonus for having contact information
	hasContact := false
	for _, page := range crawlResult.Pages {
		if len(page.Emails) > 0 || len(page.Phones) > 0 {
			hasContact = true
			break
		}
	}
	if hasContact {
		score += 5
	}
	
	return math.Max(0, math.Min(100, score))
}

// calculateOverallScore computes the weighted average of all scores
func (a *Analyzer) calculateOverallScore(scores models.OverallScores) float64 {
	weights := map[string]float64{
		"technical":   0.3,
		"content":     0.3,
		"performance": 0.2,
		"security":    0.2,
	}
	
	totalScore := scores.Technical*weights["technical"] +
		scores.Content*weights["content"] +
		scores.Performance*weights["performance"] +
		scores.Security*weights["security"]
	
	return totalScore
}

// generateFindings creates a list of SEO findings
func (a *Analyzer) generateFindings(crawlResult *models.CrawlResult) []models.Finding {
	findings := []models.Finding{}
	
	// Check for missing meta descriptions
	missingDesc := 0
	for _, page := range crawlResult.Pages {
		if page.MetaDescription == "" {
			missingDesc++
		}
	}
	if missingDesc > 0 {
		findings = append(findings, models.Finding{
			Category:    "Content",
			Type:        "Missing Meta Descriptions",
			Description: fmt.Sprintf("%d pages lack meta descriptions", missingDesc),
			Severity:    "medium",
		})
	}
	
	// Check for duplicate titles
	titles := make(map[string][]string)
	for _, page := range crawlResult.Pages {
		titles[page.MetaTitle] = append(titles[page.MetaTitle], page.URL)
	}
	for title, urls := range titles {
		if len(urls) > 1 && title != "" {
			findings = append(findings, models.Finding{
				Category:    "Technical",
				Type:        "Duplicate Title",
				Description: fmt.Sprintf("Title '%s' used on %d pages", title, len(urls)),
				Severity:    "high",
			})
		}
	}
	
	// Check for thin content
	thinContent := 0
	for _, page := range crawlResult.Pages {
		if len(strings.Fields(page.Text)) < 100 {
			thinContent++
		}
	}
	if thinContent > 0 {
		findings = append(findings, models.Finding{
			Category:    "Content",
			Type:        "Thin Content",
			Description: fmt.Sprintf("%d pages have less than 100 words", thinContent),
			Severity:    "medium",
		})
	}
	
	return findings
}

// generateRecommendations creates actionable recommendations based on findings
func (a *Analyzer) generateRecommendations(findings []models.Finding) []models.Recommendation {
	recommendations := []models.Recommendation{}
	
	// Sort findings by severity
	sort.Slice(findings, func(i, j int) bool {
		severityOrder := map[string]int{"critical": 0, "high": 1, "medium": 2, "low": 3}
		return severityOrder[findings[i].Severity] < severityOrder[findings[j].Severity]
	})
	
	for _, finding := range findings {
		var rec models.Recommendation
		
		switch finding.Type {
		case "Missing Meta Descriptions":
			rec = models.Recommendation{
				Priority:    "high",
				Category:    "Content",
				Action:      "Add unique meta descriptions",
				Impact:      "high",
				Effort:      "low",
				Description: "Write unique, compelling meta descriptions (120-160 characters) for all pages",
			}
		case "Duplicate Title":
			rec = models.Recommendation{
				Priority:    "critical",
				Category:    "Technical",
				Action:      "Fix duplicate titles",
				Impact:      "high",
				Effort:      "low",
				Description: "Ensure each page has a unique, descriptive title tag",
			}
		case "Thin Content":
			rec = models.Recommendation{
				Priority:    "medium",
				Category:    "Content",
				Action:      "Expand content",
				Impact:      "medium",
				Effort:      "medium",
				Description: "Add more valuable, relevant content to pages with less than 300 words",
			}
		default:
			continue
		}
		
		recommendations = append(recommendations, rec)
	}
	
	return recommendations
}

// generateExecutiveSummary creates a high-level summary
func (a *Analyzer) generateExecutiveSummary(report *models.SEOReport) models.ExecutiveSummary {
	summary := models.ExecutiveSummary{
		OverallScore: report.Scores.Overall,
	}
	
	// Determine grade
	switch {
	case summary.OverallScore >= 90:
		summary.OverallGrade = "A"
	case summary.OverallScore >= 80:
		summary.OverallGrade = "B"
	case summary.OverallScore >= 70:
		summary.OverallGrade = "C"
	case summary.OverallScore >= 60:
		summary.OverallGrade = "D"
	default:
		summary.OverallGrade = "F"
	}
	
	// Identify strengths and weaknesses
	if report.Scores.Technical >= 80 {
		summary.Strengths = append(summary.Strengths, "Strong technical SEO foundation")
	}
	if report.Scores.Content >= 80 {
		summary.Strengths = append(summary.Strengths, "High-quality content optimization")
	}
	if report.Scores.Performance >= 80 {
		summary.Strengths = append(summary.Strengths, "Excellent site performance")
	}
	
	if report.Scores.Technical < 60 {
		summary.Weaknesses = append(summary.Weaknesses, "Technical SEO issues need attention")
	}
	if report.Scores.Content < 60 {
		summary.Weaknesses = append(summary.Weaknesses, "Content optimization required")
	}
	if report.Scores.Performance < 60 {
		summary.Weaknesses = append(summary.Weaknesses, "Performance improvements needed")
	}
	
	// Top priorities
	for i, rec := range report.Recommendations {
		if i >= 3 {
			break
		}
		summary.TopPriorities = append(summary.TopPriorities, rec.Action)
	}
	
	// Estimated impact
	if len(report.Recommendations) > 0 {
		highPriority := 0
		for _, rec := range report.Recommendations {
			if rec.Priority == "critical" || rec.Priority == "high" {
				highPriority++
			}
		}
		if highPriority > 5 {
			summary.EstimatedImpact = "Significant improvements possible with focused effort"
		} else {
			summary.EstimatedImpact = "Moderate improvements achievable with targeted optimizations"
		}
	}
	
	return summary
}