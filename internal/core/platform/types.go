package platform

import (
	"context"
	"net/http"
	"time"
)

// AcquisitionMethod defines how data is acquired
type AcquisitionMethod string

const (
	MethodAPI       AcquisitionMethod = "api"
	MethodBrowser   AcquisitionMethod = "browser"
	MethodGraphQL   AcquisitionMethod = "graphql"
	MethodWebSocket AcquisitionMethod = "websocket"
	MethodHTML      AcquisitionMethod = "html"
)

// Platform defines the interface all platform plugins must implement
type Platform interface {
	// Metadata
	Name() string
	Version() string

	// Capabilities
	Supports(method AcquisitionMethod) bool

	// Authentication
	Login(ctx context.Context, creds Credentials) (*Session, error)
	ValidateSession(ctx context.Context, session *Session) error
	RefreshSession(ctx context.Context, session *Session) error

	// Data acquisition
	Crawl(ctx context.Context, target Target, opts CrawlOptions) (*Result, error)
	Extract(ctx context.Context, data []byte) (interface{}, error)

	// Configuration
	DefaultConfig() Config
}

// Credentials holds authentication information
type Credentials struct {
	Method   string                 `json:"method"`   // cookie, token, username_password
	Cookies  []*http.Cookie         `json:"cookies"`
	Token    string                 `json:"token"`
	Username string                 `json:"username"`
	Password string                 `json:"password"`
	Extra    map[string]interface{} `json:"extra"`
}

// Session represents an authenticated session
type Session struct {
	ID           string         `json:"id"`
	Platform     string         `json:"platform"`
	AccountID    string         `json:"account_id"`
	Cookies      []*http.Cookie `json:"cookies"`
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	ExpiresAt    time.Time      `json:"expires_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// Target defines what to crawl
type Target struct {
	Type string                 `json:"type"` // seller, product, rating, etc.
	ID   string                 `json:"id"`   // shop username, product ID, etc.
	URL  string                 `json:"url"`  // direct URL
	Meta map[string]interface{} `json:"meta"` // additional metadata
}

// CrawlOptions defines crawling parameters
type CrawlOptions struct {
	Method        AcquisitionMethod      `json:"method"`
	Session       *Session               `json:"session"`
	Timeout       time.Duration          `json:"timeout"`
	MaxRetries    int                    `json:"max_retries"`
	RateLimit     int                    `json:"rate_limit"`
	ProxyURL      string                 `json:"proxy_url"`
	UserAgent     string                 `json:"user_agent"`
	ExtraHeaders  map[string]string      `json:"extra_headers"`
	ExtraParams   map[string]interface{} `json:"extra_params"`
	OutputFormat  string                 `json:"output_format"` // json, csv, excel
	OutputFile    string                 `json:"output_file"`
	FollowPagination bool                `json:"follow_pagination"`
}

// Result holds crawl results
type Result struct {
	Platform   string                 `json:"platform"`
	Target     Target                 `json:"target"`
	Data       interface{}            `json:"data"`
	Method     AcquisitionMethod      `json:"method"`
	StatusCode int                    `json:"status_code"`
	Success    bool                   `json:"success"`
	Error      string                 `json:"error,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	CrawledAt  time.Time              `json:"crawled_at"`
	Duration   time.Duration          `json:"duration"`
}

// Config holds platform configuration
type Config struct {
	BaseURL       string                 `json:"base_url"`
	Timeout       time.Duration          `json:"timeout"`
	MaxRetries    int                    `json:"max_retries"`
	RateLimit     int                    `json:"rate_limit"`
	UserAgent     string                 `json:"user_agent"`
	ProxyURL      string                 `json:"proxy_url"`
	Debug         bool                   `json:"debug"`
	Extra         map[string]interface{} `json:"extra"`
}

// DefaultCrawlOptions returns default crawling options
func DefaultCrawlOptions() CrawlOptions {
	return CrawlOptions{
		Method:           MethodAPI,
		Timeout:          30 * time.Second,
		MaxRetries:       3,
		RateLimit:        10,
		UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		ExtraHeaders:     make(map[string]string),
		ExtraParams:      make(map[string]interface{}),
		OutputFormat:     "json",
		FollowPagination: false,
	}
}

// DefaultConfig returns default platform configuration
func DefaultConfig() Config {
	return Config{
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RateLimit:  10,
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		Debug:      false,
		Extra:      make(map[string]interface{}),
	}
}
