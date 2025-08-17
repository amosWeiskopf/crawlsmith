# CrawlSmith 🔨

[![CI](https://github.com/amosWeiskopf/crawlsmith/actions/workflows/ci.yml/badge.svg)](https://github.com/amosWeiskopf/crawlsmith/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/amosWeiskopf/crawlsmith)](https://goreportcard.com/report/github.com/amosWeiskopf/crawlsmith)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A comprehensive SEO analysis and web crawling suite written in Go. CrawlSmith provides deep website analysis, content extraction, and AI-powered insights for SEO optimization.

## Features

- 🕷️ **Advanced Web Crawling**: Subdomain discovery, robots.txt compliance, rate limiting
- 📊 **SEO Analysis**: PageRank calculation, internal link mapping, content scoring
- 🤖 **AI-Powered Insights**: Question generation, adversarial testing, content analysis (OpenAI integration)
- 📱 **Contact Extraction**: Email, phone, social media handle detection
- 📈 **External SEO Metrics**: Integration with DataForSEO and SerpAPI
- 🔍 **Content Processing**: Text extraction, keyword analysis, profanity filtering
- 📝 **Comprehensive Reporting**: JSON/TSV exports, McKinsey 7S framework analysis

## Installation

```bash
go get github.com/amosWeiskopf/crawlsmith
```

Or clone and build:

```bash
git clone https://github.com/amosWeiskopf/crawlsmith.git
cd crawlsmith
make build
```

## Quick Start

```bash
# Basic crawl
crawlsmith crawl https://example.com

# Full SEO analysis pipeline
crawlsmith analyze https://example.com --full

# Generate SEO report
crawlsmith report example.com
```

## Configuration

Set the following environment variables:

```bash
export OPENAI_API_KEY="your-key"           # For AI features
export DATAFORSEO_LOGIN="your-login"       # For SEO metrics
export DATAFORSEO_PASSWORD="your-password" # For SEO metrics
export SERPAPI_API_KEY="your-key"          # For SERP analysis
```

## Project Structure

```
crawlsmith/
├── cmd/crawlsmith/       # Main application entry point
├── pkg/
│   ├── crawler/          # Web crawling and subdomain discovery
│   ├── extractor/        # Content and contact extraction
│   ├── analyzer/         # SEO and content analysis
│   ├── reporter/         # Report generation
│   └── utils/            # Common utilities
├── internal/
│   ├── config/           # Configuration management
│   └── models/           # Data models
├── test/                 # Integration tests
├── docs/                 # Documentation
└── scripts/              # Build and deployment scripts
```

## Usage Examples

### Crawl a Website

```go
import "github.com/amosWeiskopf/crawlsmith/pkg/crawler"

c := crawler.New("https://example.com")
pages, err := c.Crawl()
```

### Generate SEO Report

```go
import "github.com/amosWeiskopf/crawlsmith/pkg/reporter"

r := reporter.New()
report, err := r.GenerateSEOReport(domain, crawlData)
```

### Extract Contact Information

```go
import "github.com/amosWeiskopf/crawlsmith/pkg/extractor"

e := extractor.New()
contacts := e.ExtractContacts(content)
```

## Development

### Running Tests

```bash
make test           # Run all tests
make test-coverage  # Run tests with coverage
make test-race      # Run tests with race detection
```

### Building

```bash
make build          # Build binary
make docker         # Build Docker image
make clean          # Clean build artifacts
```

### Linting

```bash
make lint           # Run golangci-lint
make fmt            # Format code
```

## CI/CD

This project uses GitHub Actions for continuous integration and deployment:

- **Tests**: Run on every push and pull request
- **Coverage**: Reports sent to Codecov
- **Linting**: Enforced code quality standards
- **Releases**: Automated binary releases on tags

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

- OpenAI for GPT integration
- DataForSEO for SEO metrics
- SerpAPI for SERP data
- Go community for excellent libraries

## Support

For issues, questions, or suggestions, please open an issue on GitHub.