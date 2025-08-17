package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/amosWeiskopf/crawlsmith/internal/config"
	"github.com/amosWeiskopf/crawlsmith/pkg/analyzer"
	"github.com/amosWeiskopf/crawlsmith/pkg/crawler"
	"github.com/amosWeiskopf/crawlsmith/pkg/reporter"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "crawlsmith",
	Short: "CrawlSmith - Comprehensive SEO Analysis Suite",
	Long: `CrawlSmith is a powerful SEO analysis and web crawling tool
that provides deep website insights, content extraction, and AI-powered analysis.`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
}

var crawlCmd = &cobra.Command{
	Use:   "crawl [URL]",
	Short: "Crawl a website and extract content",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		maxPerPath, _ := cmd.Flags().GetInt("max-per-path")
		maxPathTypes, _ := cmd.Flags().GetInt("max-path-types")
		
		c, err := crawler.New(url, maxPerPath, maxPathTypes)
		if err != nil {
			return fmt.Errorf("failed to create crawler: %w", err)
		}
		
		result, err := c.Crawl()
		if err != nil {
			return fmt.Errorf("crawl failed: %w", err)
		}
		
		fmt.Printf("Crawled %d pages from %s\n", result.TotalPages, result.Domain)
		return nil
	},
}

var analyzeCmd = &cobra.Command{
	Use:   "analyze [URL]",
	Short: "Perform comprehensive SEO analysis",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		url := args[0]
		full, _ := cmd.Flags().GetBool("full")
		
		// First crawl
		c, err := crawler.New(url, 50, 100)
		if err != nil {
			return fmt.Errorf("failed to create crawler: %w", err)
		}
		
		crawlResult, err := c.Crawl()
		if err != nil {
			return fmt.Errorf("crawl failed: %w", err)
		}
		
		// Then analyze
		a := analyzer.New()
		analysis, err := a.Analyze(crawlResult, full)
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}
		
		fmt.Printf("Analysis complete: %v\n", analysis)
		return nil
	},
}

var reportCmd = &cobra.Command{
	Use:   "report [DOMAIN]",
	Short: "Generate SEO report for a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		format, _ := cmd.Flags().GetString("format")
		output, _ := cmd.Flags().GetString("output")
		
		r := reporter.New()
		report, err := r.GenerateReport(domain, format)
		if err != nil {
			return fmt.Errorf("report generation failed: %w", err)
		}
		
		if output != "" {
			err = os.WriteFile(output, []byte(report), 0644)
			if err != nil {
				return fmt.Errorf("failed to write report: %w", err)
			}
			fmt.Printf("Report saved to %s\n", output)
		} else {
			fmt.Println(report)
		}
		
		return nil
	},
}

func init() {
	// Crawl command flags
	crawlCmd.Flags().Int("max-per-path", 50, "Maximum pages per path pattern")
	crawlCmd.Flags().Int("max-path-types", 100, "Maximum number of path types")
	crawlCmd.Flags().String("output", "", "Output file for crawl results")
	
	// Analyze command flags
	analyzeCmd.Flags().Bool("full", false, "Perform full analysis including AI features")
	analyzeCmd.Flags().String("output", "", "Output file for analysis results")
	
	// Report command flags
	reportCmd.Flags().String("format", "json", "Report format (json, html, markdown)")
	reportCmd.Flags().String("output", "", "Output file for report")
	
	// Add commands to root
	rootCmd.AddCommand(crawlCmd)
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(reportCmd)
	
	// Global flags
	rootCmd.PersistentFlags().String("config", "", "Config file path")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}