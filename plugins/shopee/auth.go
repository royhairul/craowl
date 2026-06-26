package shopee

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/royhairul/craowl/internal/core/platform"
)

// manualLogin performs manual login using browser - user handles captcha
func (p *ShopeePlatform) manualLogin(ctx context.Context) (*platform.Session, error) {
	fmt.Println("Starting manual login process...")
	fmt.Println("A browser window will open. Please login manually.")
	fmt.Println("Handle any captcha if presented.")
	fmt.Println("Once logged in, the session will be saved automatically.")
	fmt.Println()

	// Create browser context with visible browser
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false), // Visible browser for manual login
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	allocCtx, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithErrorf(func(string, ...interface{}) {}))
	defer cancel()

	// Navigate to login page
	loginURL := fmt.Sprintf("%s/buyer/login", p.config.BaseURL)

	var cookies []*http.Cookie

	err := chromedp.Run(browserCtx,
		chromedp.Navigate(loginURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Browser opened. Please login manually...")
			return nil
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to open browser: %w", err)
	}

	// Wait for user to login - check every 3 seconds
	fmt.Println("Waiting for login completion...")
	fmt.Println("Checking login status every 3 seconds...")

	maxWaitTime := 5 * time.Minute
	checkInterval := 3 * time.Second
	startTime := time.Now()

	for {
		if time.Since(startTime) > maxWaitTime {
			return nil, fmt.Errorf("login timeout: exceeded %v", maxWaitTime)
		}

		// Check if user is logged in
		var isLoggedIn bool
		err = chromedp.Run(browserCtx,
			chromedp.Evaluate(`!!document.cookie.match(/SPC_EC/)`, &isLoggedIn),
		)

		if err != nil {
			return nil, fmt.Errorf("failed to check login status: %w", err)
		}

		if isLoggedIn {
			fmt.Println("✓ Login detected!")
			break
		}

		fmt.Println("Not logged in yet... waiting...")
		time.Sleep(checkInterval)
	}

	// Get cookies after successful login
	err = chromedp.Run(browserCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Use cdproto network.GetCookies
			chromedpCookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return err
			}

			// Convert chromedp cookies to http.Cookie
			for _, c := range chromedpCookies {
				cookies = append(cookies, &http.Cookie{
					Name:     c.Name,
					Value:    c.Value,
					Path:     c.Path,
					Domain:   c.Domain,
					Expires:  time.Unix(int64(c.Expires), 0),
					Secure:   c.Secure,
					HttpOnly: c.HTTPOnly,
				})
			}
			return nil
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	fmt.Printf("✓ Session captured with %d cookies\n", len(cookies))

	// Create session
	session := &platform.Session{
		ID:        generateSessionID(),
		Platform:  p.Name(),
		Cookies:   cookies,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour), // 30 days
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save session to file
	if err := saveSessionToFile(session); err != nil {
		fmt.Printf("Warning: Failed to save session to file: %v\n", err)
	} else {
		fmt.Println("✓ Session saved to file")
	}

	fmt.Println("✓ Login completed successfully!")
	fmt.Println()

	return session, nil
}

// loadCookiesFromFile loads cookies from a JSON file
func loadCookiesFromFile(filePath string) ([]*http.Cookie, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cookie file: %w", err)
	}

	var cookies []*http.Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, fmt.Errorf("failed to parse cookie file: %w", err)
	}

	return cookies, nil
}

// saveCookiesToFile saves cookies to a JSON file
func saveCookiesToFile(cookies []*http.Cookie, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cookie file: %w", err)
	}

	return nil
}

// saveSessionToFile saves session to default location
func saveSessionToFile(session *platform.Session) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sessionDir := filepath.Join(homeDir, ".craowl", "sessions")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return err
	}

	sessionFile := filepath.Join(sessionDir, fmt.Sprintf("%s_%s.json", session.Platform, session.ID))

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionFile, data, 0600)
}

// loadSessionFromFile loads session from default location
func loadSessionFromFile(platformName, sessionID string) (*platform.Session, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sessionFile := filepath.Join(homeDir, ".craowl", "sessions", fmt.Sprintf("%s_%s.json", platformName, sessionID))

	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return nil, err
	}

	var session platform.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

// listSessions lists all saved sessions for a platform
func listSessions(platformName string) ([]string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sessionDir := filepath.Join(homeDir, ".craowl", "sessions")
	pattern := filepath.Join(sessionDir, fmt.Sprintf("%s_*.json", platformName))

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	sessionIDs := make([]string, 0, len(matches))
	for _, match := range matches {
		base := filepath.Base(match)
		// Extract session ID from filename: platform_sessionID.json
		sessionID := base[len(platformName)+1 : len(base)-5]
		sessionIDs = append(sessionIDs, sessionID)
	}

	return sessionIDs, nil
}
