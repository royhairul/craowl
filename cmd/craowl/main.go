package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/royhairul/craowl/internal/core/platform"
	"github.com/royhairul/craowl/plugins/shopee"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose      bool
	outputFile   string
	outputFormat string

	// Login flags
	cookieFile  string
	loginMethod string

	// Crawl flags
	targetType string
	targetID   string
	targetURL  string
	page       int
	limit      int
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	rootCmd := &cobra.Command{
		Use:   "craowl",
		Short: "Craowl - Universal Data Acquisition Platform",
		Long:  "Craowl is a high-performance data acquisition engine for scraping data from multiple platforms.",
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")

	// Login command
	loginCmd := &cobra.Command{
		Use:   "login <platform>",
		Short: "Authenticate with a platform",
		Args:  cobra.ExactArgs(1),
		RunE:  runLogin,
	}
	loginCmd.Flags().StringVar(&cookieFile, "cookie-file", "", "Path to cookie JSON file")
	loginCmd.Flags().StringVar(&loginMethod, "method", "manual", "Login method: cookie, manual")

	// Crawl command
	crawlCmd := &cobra.Command{
		Use:   "crawl <platform>",
		Short: "Crawl data from a platform",
		Args:  cobra.ExactArgs(1),
		RunE:  runCrawl,
	}
	crawlCmd.Flags().StringVar(&targetType, "type", "", "Target type: seller, product_list, seller_rating, product_detail, product_rating")
	crawlCmd.Flags().StringVar(&targetID, "id", "", "Target ID (e.g., shop username, product ID)")
	crawlCmd.Flags().StringVar(&targetURL, "url", "", "Target URL")
	crawlCmd.Flags().IntVar(&page, "page", 0, "Page number for pagination")
	crawlCmd.Flags().IntVar(&limit, "limit", 20, "Items per page")
	crawlCmd.Flags().StringVarP(&outputFormat, "output", "o", "json", "Output format: json, csv, excel")
	crawlCmd.Flags().StringVarP(&outputFile, "file", "f", "", "Output file path")
	crawlCmd.MarkFlagRequired("type")

	// Platforms command
	platformsCmd := &cobra.Command{
		Use:   "platforms",
		Short: "List supported platforms",
		RunE:  runPlatforms,
	}

	rootCmd.AddCommand(loginCmd, crawlCmd, platformsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runLogin(cmd *cobra.Command, args []string) error {
	platformName := args[0]

	if platformName != "shopee" {
		return fmt.Errorf("unsupported platform: %s (currently only 'shopee' is supported)", platformName)
	}

	// Create platform instance
	config := platform.LoadConfigFromEnv()
	shopeePlatform := shopee.New(config)

	ctx := context.Background()
	var session *platform.Session
	var err error

	switch loginMethod {
	case "cookie":
		if cookieFile == "" {
			return fmt.Errorf("--cookie-file is required for cookie login method")
		}

		cookies, err := loadCookiesFromFile(cookieFile)
		if err != nil {
			return fmt.Errorf("failed to load cookies: %w", err)
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

func runCrawl(cmd *cobra.Command, args []string) error {
	platformName := args[0]

	if platformName != "shopee" {
		return fmt.Errorf("unsupported platform: %s (currently only 'shopee' is supported)", platformName)
	}

	if targetType == "" {
		return fmt.Errorf("--type is required")
	}

	if targetID == "" && targetURL == "" {
		return fmt.Errorf("either --id or --url is required")
	}

	// Create platform instance
	config := platform.LoadConfigFromEnv()
	shopeePlatform := shopee.New(config)

	// Create target
	target := platform.Target{
		Type: targetType,
		ID:   targetID,
		URL:  targetURL,
		Meta: map[string]interface{}{
			"page":  page,
			"limit": limit,
		},
	}

	// Create crawl options
	opts := platform.DefaultCrawlOptions()
	opts.OutputFormat = outputFormat

	// Load session if exists (optional for some operations)
	// For now, we'll crawl without session (public data)

	ctx := context.Background()

	fmt.Printf("Crawling %s from %s...\n", targetType, platformName)
	fmt.Println()

	result, err := shopeePlatform.Crawl(ctx, target, opts)
	if err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("crawl failed: %s", result.Error)
	}

	// Output result
	if outputFile != "" {
		if err := saveResultToFile(result, outputFile, outputFormat); err != nil {
			return fmt.Errorf("failed to save result: %w", err)
		}
		fmt.Printf("✓ Result saved to: %s\n", outputFile)
	} else {
		// Print to stdout
		data, err := json.MarshalIndent(result.Data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format result: %w", err)
		}
		fmt.Println(string(data))
	}

	fmt.Println()
	fmt.Printf("✓ Crawl completed in %v\n", result.Duration)
	fmt.Printf("Platform: %s\n", result.Platform)
	fmt.Printf("Method: %s\n", result.Method)

	return nil
}

func runPlatforms(cmd *cobra.Command, args []string) error {
	fmt.Println("Supported Platforms:")
	fmt.Println()
	fmt.Println("  shopee - Shopee Indonesia")
	fmt.Println("    Types: seller, product_list, seller_rating, product_detail, product_rating")
	fmt.Println()
	fmt.Println("More platforms coming soon!")
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

func saveResultToFile(result *platform.Result, filePath, format string) error {
	var data []byte
	var err error

	switch format {
	case "json":
		data, err = json.MarshalIndent(result.Data, "", "  ")
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}
