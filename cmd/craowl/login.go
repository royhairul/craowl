package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/royhairul/craowl/internal/core/platform"
	"github.com/royhairul/craowl/plugins/shopee"
	"github.com/spf13/cobra"
)

func runLogin(cmd *cobra.Command, args []string) error {
	platformName := args[0]

	if platformName != "shopee" {
		return fmt.Errorf("unsupported platform: %s (currently only 'shopee' is supported)", platformName)
	}

	// Create platform instance
	config := platform.LoadConfigFromEnv()
	if verbose {
		config.Debug = true
	}
	shopeePlatform := shopee.New(config)

	ctx := context.Background()
	var session *platform.Session
	var err error

	switch loginMethod {
	case "cookie":
		if cookieFile == "" && cookieString == "" {
			return fmt.Errorf("either --cookie-file or --cookie-string is required for cookie login method")
		}

		var cookies []*http.Cookie
		var err error

		if cookieFile != "" {
			cookies, err = loadCookiesFromFile(cookieFile)
			if err != nil {
				return fmt.Errorf("failed to load cookies from file: %w", err)
			}
		} else if cookieString != "" {
			cookies = parseCookieString(cookieString, platformName)
		}

		creds := platform.Credentials{
			Method:  "cookie",
			Cookies: cookies,
		}

		session, err = shopeePlatform.Login(ctx, creds)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

	case "manual":
		fmt.Println("Starting manual login...")
		fmt.Println("A browser window will open for you to login.")
		fmt.Println()

		creds := platform.Credentials{
			Method: "manual",
		}

		session, err = shopeePlatform.Login(ctx, creds)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

	default:
		return fmt.Errorf("unsupported login method: %s", loginMethod)
	}

	fmt.Printf("✓ Login successful!\n")
	fmt.Printf("Session ID: %s\n", session.ID)
	fmt.Printf("Expires: %s\n", session.ExpiresAt.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("Session saved. You can now use 'craowl crawl' commands.")

	return nil
}

func loadCookiesFromFile(filePath string) ([]*http.Cookie, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cookies []*http.Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return nil, err
	}

	return cookies, nil
}

func parseCookieString(cookieStr string, platformName string) []*http.Cookie {
	var cookies []*http.Cookie

	domain := ""
	if platformName == "shopee" {
		domain = ".shopee.co.id"
	}

	parts := strings.Split(cookieStr, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		cookie := &http.Cookie{
			Name:   strings.TrimSpace(kv[0]),
			Value:  strings.TrimSpace(kv[1]),
			Domain: domain,
			Path:   "/",
		}
		cookies = append(cookies, cookie)
	}

	return cookies
}
