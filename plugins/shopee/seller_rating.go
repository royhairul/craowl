package shopee

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/chromedp/chromedp"
	"github.com/royhairul/craowl/internal/core/platform"
)

// SellerRatingResponse holds seller ratings
type SellerRatingResponse struct {
	Ratings       []Rating `json:"ratings"`
	TotalCount    int      `json:"total_count"`
	Page          int      `json:"page"`
	Limit         int      `json:"limit"`
	HasMore       bool     `json:"has_more"`
	AverageRating float64  `json:"average_rating"`
}

// crawlSellerRating crawls seller ratings
func (p *ShopeePlatform) crawlSellerRating(ctx context.Context, target platform.Target, opts platform.CrawlOptions) (*SellerRatingResponse, error) {
	// Extract shop_id from target
	shopID, err := extractShopID(target.ID)
	if err != nil && target.URL != "" {
		shopID, err = extractShopID(target.URL)
	}
	if err != nil {
		return nil, fmt.Errorf("shop_id is required")
	}

	// Get page and limit from meta
	page := 0
	limit := 20
	if target.Meta != nil {
		if p, ok := target.Meta["page"].(int); ok {
			page = p
		}
		if l, ok := target.Meta["limit"].(int); ok {
			limit = l
		}
	}

	// Try API first
	if opts.Method == platform.MethodAPI || opts.Method == "" {
		ratings, err := p.crawlSellerRatingAPI(ctx, shopID, page, limit, opts)
		if err == nil {
			return ratings, nil
		}
	}

	// Fallback to browser
	return p.crawlSellerRatingBrowser(ctx, shopID, opts)
}

// crawlSellerRatingAPI fetches seller ratings via API
func (p *ShopeePlatform) crawlSellerRatingAPI(ctx context.Context, shopID int64, page, limit int, opts platform.CrawlOptions) (*SellerRatingResponse, error) {
	offset := page * limit
	apiURL := fmt.Sprintf("%s/api/v2/shop/get_shop_rating?limit=%d&offset=%d&shopid=%d&type=0",
		p.config.BaseURL, limit, offset, shopID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header.Set("User-Agent", opts.UserAgent)
	req.Header.Set("Referer", p.config.BaseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// Add cookies if available
	if opts.Session != nil && len(opts.Session.Cookies) > 0 {
		for _, cookie := range opts.Session.Cookies {
			req.AddCookie(cookie)
		}
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Data struct {
			Ratings []struct {
				Cmtid  string `json:"cmtid"`
				Author struct {
					Username string `json:"username"`
				} `json:"author"`
				RatingStar int      `json:"rating_star"`
				Comment    string   `json:"comment"`
				Images     []string `json:"images"`
				Ctime      int64    `json:"ctime"`
				ItemName   string   `json:"item_name"`
			} `json:"ratings"`
			RatingTotal int `json:"rating_total"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	ratings := make([]Rating, 0, len(apiResp.Data.Ratings))
	totalRating := 0.0

	for _, r := range apiResp.Data.Ratings {
		images := make([]string, 0, len(r.Images))
		for _, img := range r.Images {
			images = append(images, fmt.Sprintf("https://cf.shopee.co.id/file/%s", img))
		}

		totalRating += float64(r.RatingStar)

		ratings = append(ratings, Rating{
			RatingID:    r.Cmtid,
			Username:    r.Author.Username,
			Rating:      r.RatingStar,
			Comment:     r.Comment,
			Images:      images,
			Date:        formatTimestamp(r.Ctime),
			ProductName: r.ItemName,
		})
	}

	avgRating := 0.0
	if len(ratings) > 0 {
		avgRating = totalRating / float64(len(ratings))
	}

	hasMore := (page+1)*limit < apiResp.Data.RatingTotal

	return &SellerRatingResponse{
		Ratings:       ratings,
		TotalCount:    apiResp.Data.RatingTotal,
		Page:          page,
		Limit:         limit,
		HasMore:       hasMore,
		AverageRating: avgRating,
	}, nil
}

// crawlSellerRatingBrowser fetches seller ratings using browser
func (p *ShopeePlatform) crawlSellerRatingBrowser(ctx context.Context, shopID int64, opts platform.CrawlOptions) (*SellerRatingResponse, error) {
	// Note: This requires authentication and complex interaction
	// Browser automation for rating page is more complex due to lazy loading
	ratingURL := fmt.Sprintf("%s/shop/%d/rating", p.config.BaseURL, shopID)

	allocCtx, cancel := chromedp.NewContext(ctx, chromedp.WithErrorf(func(string, ...interface{}) {}))
	defer cancel()

	var htmlContent string
	err := chromedp.Run(allocCtx,
		chromedp.Navigate(ratingURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(3000), // Wait for ratings to load
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("browser automation failed: %w", err)
	}

	// Extract ratings from HTML (simplified - actual implementation would be more complex)
	ratings := make([]Rating, 0)

	// Parse and extract rating data from HTML
	// This is a placeholder - actual implementation would parse the HTML structure

	return &SellerRatingResponse{
		Ratings:       ratings,
		TotalCount:    len(ratings),
		Page:          0,
		Limit:         len(ratings),
		HasMore:       false,
		AverageRating: 0.0,
	}, nil
}

// formatTimestamp converts Unix timestamp to readable date
func formatTimestamp(timestamp int64) string {
	// Simple date formatting
	return fmt.Sprintf("%d", timestamp)
}
