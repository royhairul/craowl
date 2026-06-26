package shopee

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/royhairul/craowl/internal/core/platform"
)

// crawlProductRating crawls product ratings
func (p *ShopeePlatform) crawlProductRating(ctx context.Context, target platform.Target, opts platform.CrawlOptions) (*ProductRatingResponse, error) {
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

	if (shopID == 0 || itemID == 0) && target.URL != "" {
		shopID, itemID, err = extractProductIDs(target.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to extract product IDs: %w", err)
		}
	}

	if shopID == 0 || itemID == 0 {
		return nil, fmt.Errorf("shop_id and item_id are required")
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

	// Only API method is available for product ratings
	return p.crawlProductRatingAPI(ctx, shopID, itemID, page, limit, opts)
}

// crawlProductRatingAPI fetches product ratings via API
func (p *ShopeePlatform) crawlProductRatingAPI(ctx context.Context, shopID, itemID int64, page, limit int, opts platform.CrawlOptions) (*ProductRatingResponse, error) {
	offset := page * limit
	apiURL := fmt.Sprintf("%s/api/v2/item/get_ratings?filter=0&flag=1&itemid=%d&limit=%d&offset=%d&shopid=%d&type=0",
		p.config.BaseURL, itemID, limit, offset, shopID)

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
			} `json:"ratings"`
			ItemRatingCount []int `json:"item_rating_count"`
		} `json:"data"`
		Error   int    `json:"error"`
		Message string `json:"error_msg"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	if apiResp.Error != 0 {
		return nil, fmt.Errorf("API error: %s", apiResp.Message)
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
			RatingID: r.Cmtid,
			Username: r.Author.Username,
			Rating:   r.RatingStar,
			Comment:  r.Comment,
			Images:   images,
			Date:     formatTimestamp(r.Ctime),
		})
	}

	avgRating := 0.0
	if len(ratings) > 0 {
		avgRating = totalRating / float64(len(ratings))
	}

	// Calculate rating breakdown
	ratingBreakdown := make(map[int]int)
	for i, count := range apiResp.Data.ItemRatingCount {
		ratingBreakdown[i+1] = count
	}

	totalCount := 0
	for _, count := range apiResp.Data.ItemRatingCount {
		totalCount += count
	}

	hasMore := (page+1)*limit < totalCount

	return &ProductRatingResponse{
		Ratings:         ratings,
		TotalCount:      totalCount,
		Page:            page,
		Limit:           limit,
		HasMore:         hasMore,
		AverageRating:   avgRating,
		RatingBreakdown: ratingBreakdown,
	}, nil
}
