package shopee

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	"github.com/royhairul/craowl/internal/core/platform"
)

// SellerInfo holds seller page information
type SellerInfo struct {
	ShopID       int64   `json:"shop_id"`
	Username     string  `json:"username"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Logo         string  `json:"logo"`
	Cover        string  `json:"cover"`
	Rating       float64 `json:"rating"`
	Followers    int     `json:"followers"`
	Products     int     `json:"products"`
	JoinedDate   string  `json:"joined_date"`
	Location     string  `json:"location"`
	ResponseRate float64 `json:"response_rate"`
	ResponseTime string  `json:"response_time"`
}

// Product holds product information
type Product struct {
	ItemID        int64   `json:"item_id"`
	ShopID        int64   `json:"shop_id"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	OriginalPrice float64 `json:"original_price,omitempty"`
	Discount      string  `json:"discount,omitempty"`
	Stock         int     `json:"stock"`
	Sold          int     `json:"sold"`
	Image         string  `json:"image"`
	Rating        float64 `json:"rating"`
	URL           string  `json:"url"`
}

// Rating holds rating information
type Rating struct {
	RatingID    string   `json:"rating_id"`
	Username    string   `json:"username"`
	Rating      int      `json:"rating"`
	Comment     string   `json:"comment"`
	Images      []string `json:"images,omitempty"`
	Date        string   `json:"date"`
	ProductName string   `json:"product_name,omitempty"`
}

// crawlSellerPage crawls seller page information
func (p *ShopeePlatform) crawlSellerPage(ctx context.Context, target platform.Target, opts platform.CrawlOptions) (*SellerInfo, error) {
	shopUsername := target.ID
	if shopUsername == "" && target.URL != "" {
		shopUsername = extractShopUsername(target.URL)
	}

	if shopUsername == "" {
		return nil, fmt.Errorf("shop username is required")
	}

	// Try API first
	if opts.Method == platform.MethodAPI || opts.Method == "" {
		sellerInfo, err := p.crawlSellerPageAPI(ctx, shopUsername, opts)
		if err == nil {
			return sellerInfo, nil
		}
		// Fallback to browser if API fails
	}

	// Use browser automation
	return p.crawlSellerPageBrowser(ctx, shopUsername, opts)
}

// crawlSellerPageAPI fetches seller info via Shopee API
func (p *ShopeePlatform) crawlSellerPageAPI(ctx context.Context, shopUsername string, opts platform.CrawlOptions) (*SellerInfo, error) {
	apiURL := fmt.Sprintf("%s/api/v4/shop/get_shop_detail?username=%s", p.config.BaseURL, url.QueryEscape(shopUsername))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header.Set("User-Agent", opts.UserAgent)
	req.Header.Set("Referer", fmt.Sprintf("%s/%s", p.config.BaseURL, shopUsername))
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
			Shopid  int64 `json:"shopid"`
			Account struct {
				Username string `json:"username"`
			} `json:"account"`
			Name         string  `json:"name"`
			Description  string  `json:"description"`
			ShopLogo     string  `json:"shop_logo"`
			ShopCover    string  `json:"shop_cover"`
			Rating       float64 `json:"rating_normal"`
			Follower     int     `json:"follower_count"`
			ItemCount    int     `json:"item_count"`
			Ctime        int64   `json:"ctime"`
			Country      string  `json:"country"`
			ResponseRate float64 `json:"response_rate"`
			ResponseTime int     `json:"response_time"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	sellerInfo := &SellerInfo{
		ShopID:       apiResp.Data.Shopid,
		Username:     apiResp.Data.Account.Username,
		Name:         apiResp.Data.Name,
		Description:  apiResp.Data.Description,
		Logo:         apiResp.Data.ShopLogo,
		Cover:        apiResp.Data.ShopCover,
		Rating:       apiResp.Data.Rating,
		Followers:    apiResp.Data.Follower,
		Products:     apiResp.Data.ItemCount,
		Location:     apiResp.Data.Country,
		ResponseRate: apiResp.Data.ResponseRate,
		ResponseTime: fmt.Sprintf("%d minutes", apiResp.Data.ResponseTime/60),
	}

	return sellerInfo, nil
}

// crawlSellerPageBrowser fetches seller info using browser automation
func (p *ShopeePlatform) crawlSellerPageBrowser(ctx context.Context, shopUsername string, opts platform.CrawlOptions) (*SellerInfo, error) {
	shopURL := fmt.Sprintf("%s/%s", p.config.BaseURL, shopUsername)

	// Create browser context
	allocCtx, cancel := chromedp.NewContext(ctx, chromedp.WithErrorf(func(string, ...interface{}) {}))
	defer cancel()

	var htmlContent string
	err := chromedp.Run(allocCtx,
		chromedp.Navigate(shopURL),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(2000), // Wait for dynamic content
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

	// Extract seller info from HTML
	sellerInfo := &SellerInfo{
		Username: shopUsername,
	}

	// Try to extract data from script tags (Shopee embeds data in JSON)
	doc.Find("script").Each(func(i int, s *goquery.Selection) {
		scriptContent := s.Text()
		if strings.Contains(scriptContent, "window.__INITIAL_STATE__") {
			// Extract and parse JSON data
			re := regexp.MustCompile(`window.__INITIAL_STATE__\s*=\s*({.*?});`)
			matches := re.FindStringSubmatch(scriptContent)
			if len(matches) > 1 {
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(matches[1]), &data); err == nil {
					// Extract seller info from parsed data
					if shop, ok := data["shop"].(map[string]interface{}); ok {
						if name, ok := shop["name"].(string); ok {
							sellerInfo.Name = name
						}
						if desc, ok := shop["description"].(string); ok {
							sellerInfo.Description = desc
						}
						if rating, ok := shop["rating"].(float64); ok {
							sellerInfo.Rating = rating
						}
						if followers, ok := shop["follower_count"].(float64); ok {
							sellerInfo.Followers = int(followers)
						}
						if products, ok := shop["item_count"].(float64); ok {
							sellerInfo.Products = int(products)
						}
					}
				}
			}
		}
	})

	return sellerInfo, nil
}

// extractShopUsername extracts shop username from URL
func extractShopUsername(shopURL string) string {
	re := regexp.MustCompile(`shopee\.co\.id/([^/?]+)`)
	matches := re.FindStringSubmatch(shopURL)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractShopID extracts shop ID from URL or string
func extractShopID(input string) (int64, error) {
	re := regexp.MustCompile(`shop_id=(\d+)`)
	matches := re.FindStringSubmatch(input)
	if len(matches) > 1 {
		return strconv.ParseInt(matches[1], 10, 64)
	}
	return 0, fmt.Errorf("shop_id not found")
}
