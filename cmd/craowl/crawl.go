package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/royhairul/craowl/internal/core/platform"
	"github.com/royhairul/craowl/plugins/shopee"
	"github.com/spf13/cobra"
)

func runCrawl(cmd *cobra.Command, args []string) error {
	platformName := args[0]

	if platformName != "shopee" {
		return fmt.Errorf("unsupported platform: %s (currently only 'shopee' is supported)", platformName)
	}

	if targetType == "" {
		if targetURL != "" {
			if u, err := url.Parse(targetURL); err == nil && u.Fragment != "" {
				validTypes := map[string]bool{
					"seller":         true,
					"product_list":   true,
					"seller_rating":  true,
					"product_detail": true,
					"product_rating": true,
				}
				if validTypes[u.Fragment] {
					targetType = u.Fragment
				}
			}
		}

		if targetType == "" {
			return fmt.Errorf("--type is required (or must be inferrable from URL fragment e.g., #product_list)")
		}
	}

	if targetID == "" && targetURL == "" {
		return fmt.Errorf("either --id or --url is required")
	}

	// Create platform instance
	config := platform.LoadConfigFromEnv()
	if verbose {
		config.Debug = true
	}
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

	// Load session if exists
	session, err := loadLatestSession(platformName)
	if err == nil && session != nil {
		opts.Session = session
		if verbose {
			fmt.Printf("Loaded session %s (expires %s)\n", session.ID, session.ExpiresAt.Format(time.RFC3339))
		}
	} else if verbose {
		fmt.Printf("No saved session loaded (crawling as guest): %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

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
	if outputFile == "" {
		if err := os.MkdirAll("results", 0755); err != nil {
			return fmt.Errorf("failed to create results directory: %w", err)
		}
		timestamp := time.Now().Format("20060102_150405")
		outputFile = fmt.Sprintf("results/%s_%s_%s.%s", platformName, targetType, timestamp, outputFormat)
	}

	if err := saveResultToFile(result, outputFile, outputFormat); err != nil {
		return fmt.Errorf("failed to save result: %w", err)
	}
	fmt.Printf("✓ Result saved to: %s\n", outputFile)

	fmt.Println()
	fmt.Printf("✓ Crawl completed in %v\n", result.Duration)
	fmt.Printf("Platform: %s\n", result.Platform)
	fmt.Printf("Method: %s\n", result.Method)

	return nil
}

func saveResultToFile(result *platform.Result, filePath, format string) error {
	// Identify seller name
	sellerName := result.Target.ID
	if sellerName == "" && result.Target.URL != "" {
		sellerName = shopee.ExtractShopUsername(result.Target.URL)
	}
	if sellerName == "" {
		sellerName = "unknown_seller"
	}

	baseDir := filepath.Join("results", sellerName)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	switch v := result.Data.(type) {
	case *shopee.SellerInfo:
		// results/<seller_name>/seller.json
		sellerFile := filepath.Join(baseDir, "seller.json")
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(sellerFile, data, 0644)

	case *shopee.ProductListResponse:
		// results/<seller_name>/products/<product_id>/product.json
		productsDir := filepath.Join(baseDir, "products")
		for _, p := range v.Products {
			productDir := filepath.Join(productsDir, fmt.Sprintf("%d", p.ItemID))
			if err := os.MkdirAll(productDir, 0755); err != nil {
				return err
			}

			productFile := filepath.Join(productDir, "product.json")
			data, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(productFile, data, 0644); err != nil {
				return err
			}
		}
		return nil

	case *shopee.ProductRatingResponse:
		// results/<seller_name>/products/<product_id>/ratings.json and reviews.csv
		productID := ""
		// Extract product ID from URL if possible
		reItem := regexp.MustCompile(`-i\.\d+\.(\d+)`)
		if matches := reItem.FindStringSubmatch(result.Target.URL); len(matches) > 1 {
			productID = matches[1]
		} else {
			productID = "unknown_product"
		}

		productDir := filepath.Join(baseDir, "products", productID)
		if err := os.MkdirAll(productDir, 0755); err != nil {
			return err
		}

		// Save ratings.json
		ratingsFile := filepath.Join(productDir, "ratings.json")
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(ratingsFile, data, 0644); err != nil {
			return err
		}

		// Optionally save reviews.csv
		csvFile := filepath.Join(productDir, "reviews.csv")
		file, err := os.Create(csvFile)
		if err != nil {
			return err
		}
		defer file.Close()
		writer := csv.NewWriter(file)
		defer writer.Flush()
		writer.Write([]string{"Link", "ProductName", "Username", "Rating", "Comment", "Variant", "Date", "Images"})
		for _, r := range v.Ratings {
			imagesStr := strings.Join(r.Images, ";")
			writer.Write([]string{r.Link, r.ProductName, r.Username, fmt.Sprintf("%d", r.Rating), r.Comment, r.Variant, r.Date, imagesStr})
		}
		return nil

	case *shopee.SellerRatingResponse:
		// Similar to product rating, but at the seller root maybe?
		ratingsFile := filepath.Join(baseDir, "ratings.json")
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(ratingsFile, data, 0644); err != nil {
			return err
		}

		// Optionally save reviews.csv
		csvFile := filepath.Join(baseDir, "reviews.csv")
		file, err := os.Create(csvFile)
		if err != nil {
			return err
		}
		defer file.Close()
		writer := csv.NewWriter(file)
		defer writer.Flush()
		writer.Write([]string{"Link", "ProductName", "Username", "Rating", "Comment", "Variant", "Date", "Images"})
		for _, r := range v.Ratings {
			imagesStr := strings.Join(r.Images, ";")
			writer.Write([]string{r.Link, r.ProductName, r.Username, fmt.Sprintf("%d", r.Rating), r.Comment, r.Variant, r.Date, imagesStr})
		}
		return nil

	default:
		return fmt.Errorf("hierarchical save not implemented for this data type")
	}
}
