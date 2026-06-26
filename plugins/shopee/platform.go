package shopee

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/royhairul/craowl/internal/core/platform"
)

// ShopeePlatform implements the Platform interface for Shopee Indonesia
type ShopeePlatform struct {
	config     platform.Config
	httpClient *http.Client
}

// New creates a new Shopee platform instance
func New(config platform.Config) *ShopeePlatform {
	if config.BaseURL == "" {
		config.BaseURL = "https://shopee.co.id"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &ShopeePlatform{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Name returns the platform name
func (p *ShopeePlatform) Name() string {
	return "shopee"
}

// Version returns the plugin version
func (p *ShopeePlatform) Version() string {
	return "1.0.0"
}

// Supports checks if the platform supports the given acquisition method
func (p *ShopeePlatform) Supports(method platform.AcquisitionMethod) bool {
	switch method {
	case platform.MethodAPI, platform.MethodBrowser, platform.MethodHTML:
		return true
	default:
		return false
	}
}

// DefaultConfig returns the default configuration
func (p *ShopeePlatform) DefaultConfig() platform.Config {
	return platform.Config{
		BaseURL:    "https://shopee.co.id",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RateLimit:  10,
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		Extra:      make(map[string]interface{}),
	}
}

// Login authenticates with Shopee using provided credentials
func (p *ShopeePlatform) Login(ctx context.Context, creds platform.Credentials) (*platform.Session, error) {
	session := &platform.Session{
		ID:        generateSessionID(),
		Platform:  p.Name(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	switch creds.Method {
	case "cookie":
		if len(creds.Cookies) == 0 {
			return nil, fmt.Errorf("no cookies provided")
		}
		session.Cookies = creds.Cookies
		session.ExpiresAt = time.Now().Add(24 * time.Hour)

	case "token":
		if creds.Token == "" {
			return nil, fmt.Errorf("no token provided")
		}
		session.AccessToken = creds.Token
		session.ExpiresAt = time.Now().Add(24 * time.Hour)

	case "manual":
		// Manual login using browser - user handles captcha
		return p.manualLogin(ctx)

	default:
		return nil, fmt.Errorf("unsupported authentication method: %s", creds.Method)
	}

	// Validate session
	if err := p.ValidateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	return session, nil
}

// ValidateSession checks if the session is still valid
func (p *ShopeePlatform) ValidateSession(ctx context.Context, session *platform.Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if time.Now().After(session.ExpiresAt) {
		return fmt.Errorf("session expired")
	}

	// Try to make a simple API call to verify session
	req, err := http.NewRequestWithContext(ctx, "GET", p.config.BaseURL+"/api/v4/pages/get_simple_profile", nil)
	if err != nil {
		return err
	}

	// Add cookies to request
	for _, cookie := range session.Cookies {
		req.AddCookie(cookie)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("session invalid: unauthorized")
	}

	return nil
}

// RefreshSession refreshes an expired session
func (p *ShopeePlatform) RefreshSession(ctx context.Context, session *platform.Session) error {
	// Shopee doesn't have a refresh token mechanism
	// User needs to re-authenticate
	return fmt.Errorf("session refresh not supported, please re-authenticate")
}

// Crawl performs the data crawling based on target type
func (p *ShopeePlatform) Crawl(ctx context.Context, target platform.Target, opts platform.CrawlOptions) (*platform.Result, error) {
	startTime := time.Now()

	result := &platform.Result{
		Platform:  p.Name(),
		Target:    target,
		Method:    opts.Method,
		CrawledAt: startTime,
		Metadata:  make(map[string]interface{}),
	}

	var data interface{}
	var err error

	switch target.Type {
	case "seller":
		data, err = p.crawlSellerPage(ctx, target, opts)
	case "product_list":
		data, err = p.crawlProductList(ctx, target, opts)
	case "seller_rating":
		data, err = p.crawlSellerRating(ctx, target, opts)
	case "product_detail":
		data, err = p.crawlProductDetail(ctx, target, opts)
	case "product_rating":
		data, err = p.crawlProductRating(ctx, target, opts)
	default:
		return nil, fmt.Errorf("unsupported target type: %s", target.Type)
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(startTime)
		return result, err
	}

	result.Data = data
	result.Success = true
	result.StatusCode = http.StatusOK
	result.Duration = time.Since(startTime)

	return result, nil
}

// Extract extracts structured data from raw response
func (p *ShopeePlatform) Extract(ctx context.Context, data []byte) (interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to extract data: %w", err)
	}
	return result, nil
}

// Helper function to generate session ID
func generateSessionID() string {
	return fmt.Sprintf("session_%d", time.Now().UnixNano())
}
