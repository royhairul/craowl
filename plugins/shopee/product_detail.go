package shopee

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/royhairul/craowl/internal/core/platform"
)

// ProductDetail holds detailed product information
type ProductDetail struct {
	ItemID        int64    `json:"item_id"`
	ShopID        int64    `json:"shop_id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Price         float64  `json:"price"`
	OriginalPrice float64  `json:"original_price,omitempty"`
	Discount      string   `json:"discount,omitempty"`
	Stock         int      `json:"stock"`
	Sold          int      `json:"sold"`
	Images        []string `json:"images"`
	Rating        float64  `json:"rating"`
	RatingCount   int      `json:"rating_count"`
	Category      string   `json:"category"`
	Brand         string   `json:"brand,omitempty"`
	Condition     string   `json:"condition"`
	Weight        string   `json:"weight,omitempty"`
	Dimensions    string   `json:"dimensions,omitempty"`
	URL           string   `json:"url"`
}

// ProductRatingResponse holds product ratings
type ProductRatingResponse struct {
	Ratings         []Rating    `json:"ratings"`
	TotalCount      int         `json:"total_count"`
	Page            int         `json:"page"`
	Limit           int         `json:"limit"`
	HasMore         bool        `json:"has_more"`
	AverageRating   float64     `json:"average_rating"`
	RatingBreakdown map[int]int `json:"rating_breakdown"`
}

// crawlProductDetail crawls detailed product information
func (p *ShopeePlatform) crawlProductDetail(ctx context.Context, target platform.Target, opts platform.CrawlOptions) (*ProductDetail, error) {
	// Extract shop_id and item_id from target
	var shopID, itemID int64
	var err error

	if target.Meta != nil {
		if sid, ok := target.Meta["shop_id"].(int64); ok {
			shopID = sid
		}
		if iid, ok := target.Meta["item_id"].(int64); ok {
			itemID = iid
		}
	}

	// Try to extract from URL if not provided
	if (shopID == 0 || itemID == 0) && target.URL != "" {
		shopID, itemID, err = extractProductIDs(target.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to extract product IDs: %w", err)
		}
	}

	if shopID == 0 || itemID == 0 {
		return nil, fmt.Errorf("shop_id and item_id are required")
	}

	// Try API first
	if opts.Method == platform.MethodAPI || opts.Method == "" {
		detail, err := p.crawlProductDetailAPI(ctx, shopID, itemID, opts)
		if err == nil {
			return detail, nil
		}
	}

	// Fallback to browser
	return p.crawlProductDetailBrowser(ctx, shopID, itemID, opts)
}

// crawlProductDetailAPI fetches product detail via API
func (p *ShopeePlatform) crawlProductDetailAPI(ctx context.Context, shopID, itemID int64, opts platform.CrawlOptions) (*ProductDetail, error) {
	apiURL := fmt.Sprintf("%s/api/v4/item/get?itemid=%d&shopid=%d", p.config.BaseURL, itemID, shopID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", opts.UserAgent)
	req.Header.Set("Referer", p.config.BaseURL)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

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
		Item struct {
			Itemid              int64    `json:"itemid"`
			Shopid              int64    `json:"shopid"`
			Name                string   `json:"name"`
			Description         string   `json:"description"`
			Price               int64    `json:"price"`
			PriceBeforeDiscount int64    `json:"price_before_discount"`
			Stock               int      `json:"stock"`
			Sold                int      `json:"sold"`
			Images              []string `json:"images"`
			ItemRating          struct {
				RatingStar  float64 `json:"rating_star"`
				RatingCount []int   `json:"rating_count"`
			} `json:"item_rating"`
			CatID      int    `json:"catid"`
			Brand      string `json:"brand"`
			Condition  int    `json:"condition"`
			ItemWeight string `json:"item_weight"`
		} `json:"item"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	item := apiResp.Item
	price := float64(item.Price) / 100000
	originalPrice := float64(item.PriceBeforeDiscount) / 100000

	discount := ""
	if originalPrice > price {
		discountPercent := ((originalPrice - price) / originalPrice) * 100
		discount = fmt.Sprintf("%.0f%%", discountPercent)
	}

	images := make([]string, 0, len(item.Images))
	for _, img := range item.Images {
		images = append(images, fmt.Sprintf("https://cf.shopee.co.id/file/%s", img))
	}

	condition := "New"
	if item.Condition == 1 {
		condition = "Used"
	}

	totalRatingCount := 0
	for _, count := range item.ItemRating.RatingCount {
		totalRatingCount += count
	}

	detail := &ProductDetail{
		ItemID:        item.Itemid,
		ShopID:        item.Shopid,
		Name:          item.Name,
		Description:   item.Description,
		Price:         price,
		OriginalPrice: originalPrice,
		Discount:      discount,
		Stock:         item.Stock,
		Sold:          item.Sold,
		Images:        images,
		Rating:        item.ItemRating.RatingStar,
		RatingCount:   totalRatingCount,
		Brand:         item.Brand,
		Condition:     condition,
		Weight:        item.ItemWeight,
		URL:           fmt.Sprintf("%s/product-%d-%d", p.config.BaseURL, item.Shopid, item.Itemid),
	}

	return detail, nil
}

// crawlProductDetailBrowser fetches product detail using browser
func (p *ShopeePlatform) crawlProductDetailBrowser(ctx context.Context, shopID, itemID int64, opts platform.CrawlOptions) (*ProductDetail, error) {
	productURL := fmt.Sprintf("%s/product-%d-%d", p.config.BaseURL, shopID, itemID)

	allocCtx, cancel := chromedp.NewContext(ctx, chromedp.WithErrorf(func(string, ...interface{}) {}))
	defer cancel()

	var htmlContent string
	err := chromedp.Run(allocCtx,
		chromedp.Navigate(productURL),
		chromedp.WaitVisible(`.page-product`, chromedp.ByQuery),
		chromedp.Sleep(2000),
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("browser automation failed: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	detail := &ProductDetail{
		ItemID: itemID,
		ShopID: shopID,
		URL:    productURL,
	}

	// Extract product name
	if name := doc.Find(".product-name").Text(); name != "" {
		detail.Name = strings.TrimSpace(name)
	}

	// Extract price
	if priceText := doc.Find(".product-price").Text(); priceText != "" {
		priceText = strings.ReplaceAll(priceText, "Rp", "")
		priceText = strings.ReplaceAll(priceText, ".", "")
		priceText = strings.TrimSpace(priceText)
		var price float64
		fmt.Sscanf(priceText, "%f", &price)
		detail.Price = price
	}

	return detail, nil
}

// extractProductIDs extracts shop_id and item_id from product URL
func extractProductIDs(productURL string) (shopID int64, itemID int64, err error) {
	// URL format: https://shopee.co.id/product-{shopid}-{itemid}
	parts := strings.Split(productURL, "-")
	if len(parts) < 3 {
		return 0, 0, fmt.Errorf("invalid product URL format")
	}

	shopID, err = strconv.ParseInt(parts[len(parts)-2], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse shop_id: %w", err)
	}

	itemID, err = strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse item_id: %w", err)
	}

	return shopID, itemID, nil
}
