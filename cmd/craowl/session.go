package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/royhairul/craowl/internal/core/platform"
	"github.com/spf13/cobra"
)

func runSessionInfo(cmd *cobra.Command, args []string) error {
	platformName := args[0]
	if platformName != "shopee" {
		return fmt.Errorf("unsupported platform: %s (currently only 'shopee' is supported)", platformName)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	sessionDir := filepath.Join(homeDir, ".craowl", "sessions")

	fmt.Printf("Platform: %s\n", platformName)

	files, err := os.ReadDir(sessionDir)
	if err != nil {
		fmt.Println("Status: No session directory found.")
		return nil
	}

	var latestFile string
	var latestTime time.Time
	prefix := platformName + "_"

	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ".json") {
			info, err := f.Info()
			if err == nil && info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestFile = filepath.Join(sessionDir, f.Name())
			}
		}
	}

	if latestFile == "" {
		fmt.Println("Status: No saved session found.")
		return nil
	}

	data, err := os.ReadFile(latestFile)
	if err != nil {
		return fmt.Errorf("failed to read session file: %w", err)
	}

	var session platform.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return fmt.Errorf("failed to parse session file: %w", err)
	}

	fmt.Printf("Status: Session found\n")
	fmt.Printf("Session ID: %s\n", session.ID)
	fmt.Printf("Saved Path: %s\n", latestFile)
	fmt.Printf("Last Modified: %s\n", latestTime.Format(time.RFC1123))
	fmt.Printf("Expires At: %s\n", session.ExpiresAt.Format(time.RFC1123))

	if time.Now().After(session.ExpiresAt) {
		fmt.Println("\nWarning: This session has expired!")
	} else {
		fmt.Println("\nSession is currently active.")
	}

	return nil
}

func runTestSession(cmd *cobra.Command, args []string) error {
	platformName := args[0]
	if platformName != "shopee" {
		return fmt.Errorf("unsupported platform")
	}

	session, err := loadLatestSession(platformName)
	if err != nil {
		return fmt.Errorf("failed to load session: %w", err)
	}

	fmt.Printf("Testing session %s...\n", session.ID)

	req, err := http.NewRequest("GET", "https://shopee.co.id/api/v4/account/basic/get_account_info", nil)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	var hasCsrf bool
	for _, c := range session.Cookies {
		req.AddCookie(c)
		if c.Name == "csrftoken" {
			hasCsrf = true
			req.Header.Set("x-csrftoken", c.Value)
		}
	}

	if !hasCsrf {
		fmt.Println("⚠️  Warning: Your session cookies do NOT contain 'csrftoken'. API requests requiring it will fail with 403 Forbidden.")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 && strings.Contains(string(body), `"error":0`) {
		fmt.Println("✅ Session is VALID and active!")
		return nil
	}

	fmt.Printf("❌ Session test failed! Status: %d\n", resp.StatusCode)
	fmt.Printf("Response: %s\n", string(body))
	return nil
}

func loadLatestSession(platformName string) (*platform.Session, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	sessionDir := filepath.Join(homeDir, ".craowl", "sessions")
	files, err := os.ReadDir(sessionDir)
	if err != nil {
		return nil, err
	}
	var latestFile string
	var latestTime time.Time
	prefix := platformName + "_"
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ".json") {
			info, err := f.Info()
			if err == nil && info.ModTime().After(latestTime) {
				latestTime = info.ModTime()
				latestFile = filepath.Join(sessionDir, f.Name())
			}
		}
	}
	if latestFile == "" {
		return nil, fmt.Errorf("no session found for platform %s", platformName)
	}
	data, err := os.ReadFile(latestFile)
	if err != nil {
		return nil, err
	}
	var session platform.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}
