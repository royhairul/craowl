package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	verbose      bool
	logFile      string
	outputFile   string
	outputFormat string

	// Login flags
	cookieFile   string
	cookieString string
	loginMethod  string

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
	const asciiArt = `
    ,___,
    [0.0]
    /)__)
    -"--"-
`

	rootCmd := &cobra.Command{
		Use:   "craowl",
		Short: "Craowl - Universal Data Acquisition Platform",
		Long:  asciiArt + "\nCraowl is a high-performance data acquisition engine for scraping data from multiple platforms.",
	}

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&logFile, "log-file", "", "Path to save log output to a file")

	// Set up log file redirection if provided
	cobra.OnInitialize(func() {
		if logFile != "" {
			f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				os.Stdout = f
				os.Stderr = f
			} else {
				fmt.Printf("Warning: Failed to open log file %s: %v\n", logFile, err)
			}
		}
	})

	// Login command
	loginCmd := &cobra.Command{
		Use:   "login <platform>",
		Short: "Authenticate with a platform",
		Args:  cobra.ExactArgs(1),
		RunE:  runLogin,
	}
	loginCmd.Flags().StringVar(&cookieFile, "cookie-file", "", "Path to a JSON file containing cookies")
	loginCmd.Flags().StringVar(&cookieString, "cookie-string", "", "Raw cookie string (e.g. 'key1=value1; key2=value2')")
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
	crawlCmd.Flags().IntVar(&limit, "limit", 0, "Maximum total items to fetch (0 for all)")
	crawlCmd.Flags().StringVarP(&outputFormat, "output", "o", "csv", "Output format: json, csv, excel")
	crawlCmd.Flags().StringVarP(&outputFile, "file", "f", "", "Output file path")

	// Platforms command
	platformsCmd := &cobra.Command{
		Use:   "platforms",
		Short: "List supported platforms",
		RunE:  runPlatforms,
	}

	// Session commands
	sessionCmd := &cobra.Command{
		Use:   "session",
		Short: "Manage platform sessions",
	}

	sessionInfoCmd := &cobra.Command{
		Use:   "info <platform>",
		Short: "Show information about the saved session for a platform",
		Args:  cobra.ExactArgs(1),
		RunE:  runSessionInfo,
	}

	testSessionCmd := &cobra.Command{
		Use:   "test <platform>",
		Short: "Test if the currently saved session is valid",
		Args:  cobra.ExactArgs(1),
		RunE:  runTestSession,
	}

	sessionCmd.AddCommand(sessionInfoCmd, testSessionCmd)

	// Doctor command
	doctorCmd := &cobra.Command{
		Use:   "doctor",
		Short: "Run diagnostics on the craowl environment",
	}

	doctorBrowserCmd := &cobra.Command{
		Use:   "browser",
		Short: "Run forensic browser diagnostics",
		RunE:  runDoctorBrowser,
	}
	doctorCmd.AddCommand(doctorBrowserCmd)

	rootCmd.AddCommand(loginCmd, crawlCmd, platformsCmd, sessionCmd, doctorCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
