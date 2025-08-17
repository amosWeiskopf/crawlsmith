package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Server ServerConfig `mapstructure:"server"`
	
	// Crawler configuration
	Crawler CrawlerConfig `mapstructure:"crawler"`
	
	// API Keys
	APIs APIConfig `mapstructure:"apis"`
	
	// Storage configuration
	Storage StorageConfig `mapstructure:"storage"`
	
	// Logging configuration
	Logging LoggingConfig `mapstructure:"logging"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// CrawlerConfig holds crawler-specific configuration
type CrawlerConfig struct {
	MaxDepth          int           `mapstructure:"max_depth"`
	MaxPagesPerDomain int           `mapstructure:"max_pages_per_domain"`
	RequestsPerSecond int           `mapstructure:"requests_per_second"`
	UserAgent         string        `mapstructure:"user_agent"`
	Timeout           time.Duration `mapstructure:"timeout"`
	FollowRobotsTxt   bool          `mapstructure:"follow_robots_txt"`
	ExtractContacts   bool          `mapstructure:"extract_contacts"`
	EnableJavaScript  bool          `mapstructure:"enable_javascript"`
	MaxWorkers        int           `mapstructure:"max_workers"`
}

// APIConfig holds API keys and endpoints
type APIConfig struct {
	OpenAI      OpenAIConfig      `mapstructure:"openai"`
	DataForSEO  DataForSEOConfig  `mapstructure:"dataforseo"`
	SerpAPI     SerpAPIConfig     `mapstructure:"serpapi"`
}

// OpenAIConfig holds OpenAI API configuration
type OpenAIConfig struct {
	APIKey      string `mapstructure:"api_key"`
	Model       string `mapstructure:"model"`
	MaxTokens   int    `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
}

// DataForSEOConfig holds DataForSEO API configuration
type DataForSEOConfig struct {
	Login    string `mapstructure:"login"`
	Password string `mapstructure:"password"`
	Endpoint string `mapstructure:"endpoint"`
}

// SerpAPIConfig holds SerpAPI configuration
type SerpAPIConfig struct {
	APIKey string `mapstructure:"api_key"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Type      string `mapstructure:"type"` // "file", "database", "s3"
	Path      string `mapstructure:"path"`
	BatchSize int    `mapstructure:"batch_size"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // "json" or "text"
	OutputPath string `mapstructure:"output_path"`
}

var (
	defaultConfig *Config
	configLoaded  bool
)

// Load loads configuration from file and environment
func Load(configPath string) (*Config, error) {
	if configLoaded && defaultConfig != nil {
		return defaultConfig, nil
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath("$HOME/.crawlsmith")
	}

	// Set defaults
	setDefaults()

	// Bind environment variables
	bindEnvVars()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is not an error, we'll use defaults and env
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Override with environment variables
	loadFromEnv(&config)

	defaultConfig = &config
	configLoaded = true

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")

	// Crawler defaults
	viper.SetDefault("crawler.max_depth", 10)
	viper.SetDefault("crawler.max_pages_per_domain", 1000)
	viper.SetDefault("crawler.requests_per_second", 10)
	viper.SetDefault("crawler.user_agent", "CrawlSmith/1.0")
	viper.SetDefault("crawler.timeout", "30s")
	viper.SetDefault("crawler.follow_robots_txt", true)
	viper.SetDefault("crawler.extract_contacts", true)
	viper.SetDefault("crawler.enable_javascript", false)
	viper.SetDefault("crawler.max_workers", 10)

	// API defaults
	viper.SetDefault("apis.openai.model", "gpt-4")
	viper.SetDefault("apis.openai.max_tokens", 2000)
	viper.SetDefault("apis.openai.temperature", 0.7)
	viper.SetDefault("apis.dataforseo.endpoint", "https://api.dataforseo.com")

	// Storage defaults
	viper.SetDefault("storage.type", "file")
	viper.SetDefault("storage.path", "./data")
	viper.SetDefault("storage.batch_size", 100)

	// Logging defaults
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.output_path", "stdout")
}

// bindEnvVars binds environment variables
func bindEnvVars() {
	viper.SetEnvPrefix("CRAWLSMITH")
	viper.AutomaticEnv()

	// Bind specific env vars
	viper.BindEnv("apis.openai.api_key", "OPENAI_API_KEY")
	viper.BindEnv("apis.dataforseo.login", "DATAFORSEO_LOGIN")
	viper.BindEnv("apis.dataforseo.password", "DATAFORSEO_PASSWORD")
	viper.BindEnv("apis.serpapi.api_key", "SERPAPI_API_KEY")
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *Config) {
	// OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.APIs.OpenAI.APIKey = apiKey
	}

	// DataForSEO
	if login := os.Getenv("DATAFORSEO_LOGIN"); login != "" {
		config.APIs.DataForSEO.Login = login
	}
	if password := os.Getenv("DATAFORSEO_PASSWORD"); password != "" {
		config.APIs.DataForSEO.Password = password
	}

	// SerpAPI
	if apiKey := os.Getenv("SERPAPI_API_KEY"); apiKey != "" {
		config.APIs.SerpAPI.APIKey = apiKey
	}
}

// Get returns the current configuration
func Get() *Config {
	if !configLoaded || defaultConfig == nil {
		// Load with defaults if not already loaded
		config, _ := Load("")
		return config
	}
	return defaultConfig
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate required fields
	if c.Crawler.MaxDepth <= 0 {
		return fmt.Errorf("crawler.max_depth must be positive")
	}
	if c.Crawler.RequestsPerSecond <= 0 {
		return fmt.Errorf("crawler.requests_per_second must be positive")
	}
	if c.Crawler.MaxWorkers <= 0 {
		return fmt.Errorf("crawler.max_workers must be positive")
	}

	// Validate API keys if features are enabled
	if c.APIs.OpenAI.APIKey == "" {
		// Not an error, just means AI features won't be available
		fmt.Println("Warning: OpenAI API key not set. AI features will be disabled.")
	}

	return nil
}