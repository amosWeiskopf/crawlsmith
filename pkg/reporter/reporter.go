package reporter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"time"

	"github.com/amosWeiskopf/crawlsmith/internal/models"
)

// Reporter handles report generation in various formats
type Reporter struct {
	templateDir string
}

// New creates a new Reporter instance
func New() *Reporter {
	return &Reporter{
		templateDir: "templates/",
	}
}

// GenerateReport creates a report in the specified format
func (r *Reporter) GenerateReport(domain string, format string) (string, error) {
	// Load data for domain
	report, err := r.loadReportData(domain)
	if err != nil {
		return "", fmt.Errorf("failed to load report data: %w", err)
	}

	switch format {
	case "json":
		return r.generateJSON(report)
	case "html":
		return r.generateHTML(report)
	case "markdown":
		return r.generateMarkdown(report)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// GenerateSEOReport creates a comprehensive SEO report
func (r *Reporter) GenerateSEOReport(report *models.SEOReport) (string, error) {
	return r.generateJSON(report)
}

// generateJSON creates a JSON formatted report
func (r *Reporter) generateJSON(report *models.SEOReport) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal report: %w", err)
	}
	return string(data), nil
}

// generateHTML creates an HTML formatted report
func (r *Reporter) generateHTML(report *models.SEOReport) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>SEO Report - {{.Domain}}</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 2rem;
            border-radius: 10px;
            margin-bottom: 2rem;
        }
        .score-card {
            background: white;
            border-radius: 10px;
            padding: 1.5rem;
            margin-bottom: 1.5rem;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .score-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin: 1rem 0;
        }
        .score-item {
            text-align: center;
            padding: 1rem;
            background: #f8f9fa;
            border-radius: 8px;
        }
        .score-value {
            font-size: 2rem;
            font-weight: bold;
            color: #667eea;
        }
        .score-label {
            color: #666;
            font-size: 0.9rem;
            margin-top: 0.5rem;
        }
        .grade {
            display: inline-block;
            padding: 0.5rem 1rem;
            background: #28a745;
            color: white;
            border-radius: 5px;
            font-weight: bold;
            font-size: 1.2rem;
        }
        .finding {
            background: white;
            border-left: 4px solid #ffc107;
            padding: 1rem;
            margin: 1rem 0;
            border-radius: 4px;
        }
        .finding.high {
            border-left-color: #dc3545;
        }
        .finding.medium {
            border-left-color: #ffc107;
        }
        .finding.low {
            border-left-color: #28a745;
        }
        .recommendation {
            background: white;
            padding: 1.5rem;
            margin: 1rem 0;
            border-radius: 8px;
            box-shadow: 0 2px 5px rgba(0,0,0,0.1);
        }
        .priority-badge {
            display: inline-block;
            padding: 0.25rem 0.75rem;
            border-radius: 4px;
            font-size: 0.85rem;
            font-weight: bold;
            margin-right: 0.5rem;
        }
        .priority-critical {
            background: #dc3545;
            color: white;
        }
        .priority-high {
            background: #fd7e14;
            color: white;
        }
        .priority-medium {
            background: #ffc107;
            color: #333;
        }
        .priority-low {
            background: #28a745;
            color: white;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>SEO Report for {{.Domain}}</h1>
        <p>Generated on {{.GeneratedAt.Format "January 2, 2006"}}</p>
    </div>

    <div class="score-card">
        <h2>Executive Summary</h2>
        <p>Overall Grade: <span class="grade">{{.ExecutiveSummary.OverallGrade}}</span></p>
        
        <div class="score-grid">
            <div class="score-item">
                <div class="score-value">{{printf "%.0f" .Scores.Technical}}</div>
                <div class="score-label">Technical SEO</div>
            </div>
            <div class="score-item">
                <div class="score-value">{{printf "%.0f" .Scores.Content}}</div>
                <div class="score-label">Content Quality</div>
            </div>
            <div class="score-item">
                <div class="score-value">{{printf "%.0f" .Scores.Performance}}</div>
                <div class="score-label">Performance</div>
            </div>
            <div class="score-item">
                <div class="score-value">{{printf "%.0f" .Scores.Overall}}</div>
                <div class="score-label">Overall Score</div>
            </div>
        </div>

        {{if .ExecutiveSummary.Strengths}}
        <h3>Strengths</h3>
        <ul>
            {{range .ExecutiveSummary.Strengths}}
            <li>{{.}}</li>
            {{end}}
        </ul>
        {{end}}

        {{if .ExecutiveSummary.Weaknesses}}
        <h3>Areas for Improvement</h3>
        <ul>
            {{range .ExecutiveSummary.Weaknesses}}
            <li>{{.}}</li>
            {{end}}
        </ul>
        {{end}}
    </div>

    {{if .KeyFindings}}
    <div class="score-card">
        <h2>Key Findings</h2>
        {{range .KeyFindings}}
        <div class="finding {{.Severity}}">
            <h4>{{.Type}}</h4>
            <p>{{.Description}}</p>
            {{if .Details}}<p><small>{{.Details}}</small></p>{{end}}
        </div>
        {{end}}
    </div>
    {{end}}

    {{if .Recommendations}}
    <div class="score-card">
        <h2>Recommendations</h2>
        {{range .Recommendations}}
        <div class="recommendation">
            <span class="priority-badge priority-{{.Priority}}">{{.Priority}} Priority</span>
            <h4>{{.Action}}</h4>
            <p>{{.Description}}</p>
            <p><small>Impact: {{.Impact}} | Effort: {{.Effort}}</small></p>
        </div>
        {{end}}
    </div>
    {{end}}
</body>
</html>
`

	t, err := template.New("report").Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, report); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// generateMarkdown creates a Markdown formatted report
func (r *Reporter) generateMarkdown(report *models.SEOReport) (string, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "# SEO Report for %s\n\n", report.Domain)
	fmt.Fprintf(&buf, "*Generated on %s*\n\n", report.GeneratedAt.Format("January 2, 2006"))

	fmt.Fprintf(&buf, "## Executive Summary\n\n")
	fmt.Fprintf(&buf, "**Overall Grade:** %s (%.0f/100)\n\n", 
		report.ExecutiveSummary.OverallGrade, 
		report.ExecutiveSummary.OverallScore)

	fmt.Fprintf(&buf, "### Scores\n\n")
	fmt.Fprintf(&buf, "| Metric | Score |\n")
	fmt.Fprintf(&buf, "|--------|-------|\n")
	fmt.Fprintf(&buf, "| Technical SEO | %.0f |\n", report.Scores.Technical)
	fmt.Fprintf(&buf, "| Content Quality | %.0f |\n", report.Scores.Content)
	fmt.Fprintf(&buf, "| Performance | %.0f |\n", report.Scores.Performance)
	fmt.Fprintf(&buf, "| **Overall** | **%.0f** |\n\n", report.Scores.Overall)

	if len(report.ExecutiveSummary.Strengths) > 0 {
		fmt.Fprintf(&buf, "### Strengths\n\n")
		for _, strength := range report.ExecutiveSummary.Strengths {
			fmt.Fprintf(&buf, "- %s\n", strength)
		}
		fmt.Fprintf(&buf, "\n")
	}

	if len(report.ExecutiveSummary.Weaknesses) > 0 {
		fmt.Fprintf(&buf, "### Areas for Improvement\n\n")
		for _, weakness := range report.ExecutiveSummary.Weaknesses {
			fmt.Fprintf(&buf, "- %s\n", weakness)
		}
		fmt.Fprintf(&buf, "\n")
	}

	if len(report.KeyFindings) > 0 {
		fmt.Fprintf(&buf, "## Key Findings\n\n")
		for _, finding := range report.KeyFindings {
			fmt.Fprintf(&buf, "### %s\n", finding.Type)
			fmt.Fprintf(&buf, "- **Category:** %s\n", finding.Category)
			fmt.Fprintf(&buf, "- **Severity:** %s\n", finding.Severity)
			fmt.Fprintf(&buf, "- **Description:** %s\n", finding.Description)
			if finding.Details != "" {
				fmt.Fprintf(&buf, "- **Details:** %s\n", finding.Details)
			}
			fmt.Fprintf(&buf, "\n")
		}
	}

	if len(report.Recommendations) > 0 {
		fmt.Fprintf(&buf, "## Recommendations\n\n")
		for i, rec := range report.Recommendations {
			fmt.Fprintf(&buf, "### %d. %s\n", i+1, rec.Action)
			fmt.Fprintf(&buf, "- **Priority:** %s\n", rec.Priority)
			fmt.Fprintf(&buf, "- **Category:** %s\n", rec.Category)
			fmt.Fprintf(&buf, "- **Impact:** %s\n", rec.Impact)
			fmt.Fprintf(&buf, "- **Effort:** %s\n", rec.Effort)
			fmt.Fprintf(&buf, "- **Description:** %s\n", rec.Description)
			fmt.Fprintf(&buf, "\n")
		}
	}

	return buf.String(), nil
}

// loadReportData loads existing report data for a domain
func (r *Reporter) loadReportData(domain string) (*models.SEOReport, error) {
	// This would load from database or file system
	// For now, return a sample report
	return &models.SEOReport{
		Domain:      domain,
		GeneratedAt: time.Now(),
		ExecutiveSummary: models.ExecutiveSummary{
			OverallGrade: "B",
			OverallScore: 82.5,
			Strengths:    []string{"Good technical foundation", "Strong content quality"},
			Weaknesses:   []string{"Performance could be improved"},
		},
		Scores: models.OverallScores{
			Technical:   85,
			Content:     88,
			Performance: 75,
			Overall:     82.5,
		},
	}, nil
}