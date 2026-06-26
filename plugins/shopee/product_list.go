package shopee

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/royhairul/craowl/internal/core/platform"
)

// ProductListResponse holds paginated product list
type ProductListResponse struct {
	Products   []Product `json:"products"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	Limit      int       `json:"limit"`
	HasMore    bool      `json:"has_more"`
}

// crawlProductList crawls product list from a seller
func (p *ShopeePlatform) crawlProductList(ctx context.Context, target platform.Target, opts platform.CrawlOptions) (*ProductListResponse, error) {
	shopUsername := target.ID
	if shopUsername == "" && target.URL != "" {
		shopUsername = extractShopUsername(target.URL)
	}

	if shopUsername == "" {
		return nil, fmt.Errorf("shop username is required")
	}

	// Get shop ID first
	sellerInfo, err := p.crawlSellerPageAPI(ctx, shopUsername, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get shop info: %w", err)
	}

	// Get page and limit from meta
	page := 0
	limit := 30
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
		products, err := p.crawlProductListAPI(ctx, sellerInfo.ShopID, page, limit, opts)
		if err == nil {
			return products, nil
		}
		// Fallback to browser if API fails
	}

	// Use browser automation
	return p.crawlProductListBrowser(ctx, shopUsername, opts)
}

// crawlProductListAPI fetches product list via Shopee API
func (p *ShopeePlatform) crawlProductListAPI(ctx context.Context, shopID int64, page, limit int, opts platform.CrawlOptions) (*ProductListResponse, error) {
	offset := page * limit
	apiURL := fmt.Sprintf("%s/api/v4/shop/search_items?limit=%d&newest=%d&offset=%d&shopid=%d",
		p.config.BaseURL, limit, offset, offset, shopID)

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
		Items []struct {
			Itemid              int64   `json:"itemid"`
			Shopid              int64   `json:"shopid"`
			Name                string  `json:"name"`
			Price               int64   `json:"price"`
			PriceBeforeDiscount int64   `json:"price_before_discount"`
			Stock               int     `json:"stock"`
			Sold                int     `json:"sold"`
			Image               string  `json:"image"`
			RatingStar          float64 `json:"item_rating"`
		} `json:"items"`
		TotalCount int `json:"total_count"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	products := make([]Product, 0, len(apiResp.Items))
	for _, item := range apiResp.Items {
		price := float64(item.Price) / 100000 // Shopee stores price in special format
		originalPrice := float64(item.PriceBeforeDiscount) / 100000

		discount := ""
		if originalPrice > price {
			discountPercent := ((originalPrice - price) / originalPrice) * 100
			discount = fmt.Sprintf("%.0f%%", discountPercent)
		}

		products = append(products, Product{
			ItemID:        item.Itemid,
			ShopID:        item.Shopid,
			Name:          item.Name,
			Price:         price,
			OriginalPrice: originalPrice,
			Discount:      discount,
			Stock:         item.Stock,
			Sold:          item.Sold,
			Image:         fmt.Sprintf("https://cf.shopee.co.id/file/%s", item.Image),
			Rating:        item.RatingStar,
			URL:           fmt.Sprintf("%s/product-%d-%d", p.config.BaseURL, item.Shopid, item.Itemid),
		})
	}

	hasMore := (page+1)*limit < apiResp.TotalCount

	return &ProductListResponse{
		Products:   products,
		TotalCount: apiResp.TotalCount,
		Page:       page,
		Limit:      limit,
		HasMore:    hasMore,
	}, nil
}

// crawlProductListBrowser fetches product list using browser automation
func (p *ShopeePlatform) crawlProductListBrowser(ctx context.Context, shopUsername string, opts platform.CrawlOptions) (*ProductListResponse, error) {
	shopURL := fmt.Sprintf("%s/%s", p.config.BaseURL, shopUsername)

	// Create browser context
	allocCtx, cancel := chromedp.NewContext(ctx, chromedp.WithErrorf(func(string, ...interface{}) {}))
	defer cancel()

	var htmlContent string
	err := chromedp.Run(allocCtx,
		chromedp.Navigate(shopURL),
		chromedp.WaitVisible(`.shop-search-result-view`, chromedp.ByQuery),
		chromedp.Sleep(2000), // Wait for products to load
		chromedp.OuterHTML("html", &htmlContent),
	)

	if err != nil {
		return nil, fmt.Errorf("browser automation failed: %w", err)
	}

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	products := make([]Product, 0)

	// Extract products from HTML
	doc.Find(".shop-search-result-view__item").Each(func(i int, s *goquery.Selection) {
		product := Product{}

		// Extract product name
		if name := s.Find(".ie3A\\+n").Text(); name != "" {
			product.Name = strings.TrimSpace(name)
		}

		// Extract price
		if priceText := s.Find("._3c5u87").Text(); priceText != "" {
			priceText = strings.ReplaceAll(priceText, "Rp", "")
			priceText = strings.ReplaceAll(priceText, ".", "")
			priceText = strings.TrimSpace(priceText)
			// Parse price
			var price float64
			fmt.Sscanf(priceText, "%f", &price)
			product.Price = price
		}

		// Extract sold count
		if soldText := s.Find("._1cEkb\\+").Text(); soldText != "" {
			soldText = strings.ReplaceAll(soldText, "Terjual", "")
			soldText = strings.TrimSpace(soldText)
			var sold int
			fmt.Sscanf(soldText, "%d", &sold)
			product.Sold = sold
		}

		// Extract product URL
		if href, exists := s.Find("a").Attr("href"); exists {
			product.URL = p.config.BaseURL + href
		}

		// Extract image
		if imgSrc, exists := s.Find("img").Attr("src"); exists {
			product.Image = imgSrc
		}

		if product.Name != "" {
			products = append(products, product)
		}
	})

	return &ProductListResponse{
		Products:   products,
		TotalCount: len(products),
		Page:       0,
		Limit:      len(products),
		HasMore:    false,
	}, nil
}
