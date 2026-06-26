# Craowl Usage Examples

This document provides examples of how to use Craowl to scrape Shopee data.

## Installation

```bash
# Install Craowl CLI
go install github.com/royhairul/craowl/cmd/craowl@latest

# Or build from source
git clone https://github.com/royhairul/craowl
cd craowl
go build -o craowl cmd/craowl/main.go
```

## Authentication

### Method 1: Manual Login (Recommended for first-time setup)

This method opens a browser window where you can login manually and handle captcha:

```bash
craowl login shopee --method=manual
```

**What happens:**
1. Browser window opens with Shopee login page
2. You login manually (username/password)
3. Handle captcha if presented
4. Session is automatically captured and saved
5. Session is stored in `~/.craowl/sessions/`

### Method 2: Using Cookie File

If you already have cookies from a previous session:

```bash
craowl login shopee --method=cookie --cookie-file=cookies.json
```

**Cookie file format (cookies.json):**
```json
[
  {
    "name": "SPC_EC",
    "value": "your_cookie_value",
    "domain": ".shopee.co.id",
    "path": "/",
    "expires": "2026-07-24T00:00:00Z"
  }
]
```

## Scraping Operations

### 1. Scrape Seller Page

Get seller information (shop name, rating, followers, etc.):

```bash
# Using shop username
craowl crawl shopee --type=seller --id=3secondshop

# Using shop URL
craowl crawl shopee --type=seller --url=https://shopee.co.id/3secondshop

# Save to file
craowl crawl shopee --type=seller --id=3secondshop --output=json --file=seller.json
```

**Output example:**
```json
{
  "shop_id": 18358818,
  "username": "3secondshop",
  "name": "3SECOND Official Shop",
  "description": "Official Store 3SECOND Indonesia",
  "rating": 4.8,
  "followers": 125000,
  "products": 450,
  "location": "Jakarta",
  "response_rate": 95.5,
  "response_time": "30 minutes"
}
```

### 2. Scrape Product List

Get all products from a seller:

```bash
# Get first page (default: 30 items)
craowl crawl shopee --type=product_list --id=3secondshop

# Get specific page with custom limit
craowl crawl shopee --type=product_list --id=3secondshop --page=0 --limit=50

# Save to file
craowl crawl shopee --type=product_list --id=3secondshop --file=products.json
```

**Output example:**
```json
{
  "products": [
    {
      "item_id": 12345678,
      "shop_id": 18358818,
      "name": "Kaos 3SECOND Premium Cotton",
      "price": 149000,
      "original_price": 199000,
      "discount": "25%",
      "stock": 150,
      "sold": 2500,
      "rating": 4.7,
      "url": "https://shopee.co.id/product-18358818-12345678"
    }
  ],
  "total_count": 450,
  "page": 0,
  "limit": 30,
  "has_more": true
}
```

### 3. Scrape Seller Rating

Get ratings and reviews for a seller:

```bash
# Note: shop_id required (get it from seller page first)
craowl crawl shopee --type=seller_rating --id="shop_id=18358818"

# With pagination
craowl crawl shopee --type=seller_rating --id="shop_id=18358818" --page=0 --limit=20

# Using URL
craowl crawl shopee --type=seller_rating --url="https://shopee.co.id/buyer/18360154/rating?shop_id=18358818"
```

**Output example:**
```json
{
  "ratings": [
    {
      "rating_id": "abc123",
      "username": "user***123",
      "rating": 5,
      "comment": "Kualitas bagus, pengiriman cepat!",
      "images": ["https://cf.shopee.co.id/file/image1.jpg"],
      "date": "1719273600",
      "product_name": "Kaos 3SECOND Premium"
    }
  ],
  "total_count": 15000,
  "page": 0,
  "limit": 20,
  "has_more": true,
  "average_rating": 4.8
}
```

### 4. Scrape Product Detail

Get detailed information about a specific product:

```bash
# Using product URL
craowl crawl shopee --type=product_detail --url="https://shopee.co.id/product-18358818-12345678"

# Save to file
craowl crawl shopee --type=product_detail --url="https://shopee.co.id/product-18358818-12345678" --file=product_detail.json
```

**Output example:**
```json
{
  "item_id": 12345678,
  "shop_id": 18358818,
  "name": "Kaos 3SECOND Premium Cotton",
  "description": "Kaos premium dengan bahan cotton 100%...",
  "price": 149000,
  "original_price": 199000,
  "discount": "25%",
  "stock": 150,
  "sold": 2500,
  "images": [
    "https://cf.shopee.co.id/file/image1.jpg",
    "https://cf.shopee.co.id/file/image2.jpg"
  ],
  "rating": 4.7,
  "rating_count": 850,
  "brand": "3SECOND",
  "condition": "New",
  "weight": "200g",
  "url": "https://shopee.co.id/product-18358818-12345678"
}
```

### 5. Scrape Product Rating

Get ratings and reviews for a specific product:

```bash
# Using product URL
craowl crawl shopee --type=product_rating --url="https://shopee.co.id/product-18358818-12345678"

# With pagination
craowl crawl shopee --type=product_rating --url="https://shopee.co.id/product-18358818-12345678" --page=0 --limit=20
```

**Output example:**
```json
{
  "ratings": [
    {
      "rating_id": "xyz789",
      "username": "buyer***456",
      "rating": 5,
      "comment": "Sesuai deskripsi, kualitas bagus!",
      "images": ["https://cf.shopee.co.id/file/review1.jpg"],
      "date": "1719273600"
    }
  ],
  "total_count": 850,
  "page": 0,
  "limit": 20,
  "has_more": true,
  "average_rating": 4.7,
  "rating_breakdown": {
    "1": 10,
    "2": 15,
    "3": 50,
    "4": 200,
    "5": 575
  }
}
```

## Advanced Usage

### Complete Workflow Example

```bash
# 1. Login first
craowl login shopee --method=manual

# 2. Get seller info
craowl crawl shopee --type=seller --id=3secondshop --file=seller.json

# 3. Get all products
craowl crawl shopee --type=product_list --id=3secondshop --file=products.json

# 4. Get seller ratings
craowl crawl shopee --type=seller_rating --id="shop_id=18358818" --file=seller_ratings.json

# 5. Get specific product detail
craowl crawl shopee --type=product_detail --url="https://shopee.co.id/product-18358818-12345678" --file=product_detail.json

# 6. Get product ratings
craowl crawl shopee --type=product_rating --url="https://shopee.co.id/product-18358818-12345678" --file=product_ratings.json
```

## Tips

1. **Rate Limiting:** Craowl respects rate limits automatically. For large-scale scraping, consider adding delays between requests.

2. **Session Management:** Sessions are saved automatically in `~/.craowl/sessions/`. They typically last 30 days.

3. **Pagination:** Use `--page` and `--limit` flags to control pagination for product lists and ratings.

4. **Error Handling:** If a crawl fails, Craowl will automatically retry up to 3 times with exponential backoff.

5. **Verbose Mode:** Add `--verbose` flag to see detailed logs: `craowl crawl shopee --type=seller --id=3secondshop --verbose`

## Troubleshooting

**Problem:** Login fails with "session validation failed"
**Solution:** Try manual login again: `craowl login shopee --method=manual`

**Problem:** API returns 401 Unauthorized
**Solution:** Session expired. Re-authenticate with `craowl login shopee --method=manual`

**Problem:** Captcha appears during manual login
**Solution:** Solve the captcha manually in the browser window. Craowl will wait for you to complete it.

**Problem:** Rate limit errors
**Solution:** Reduce request frequency or use `--limit` to reduce items per page.

## Next Steps

- Check the [API Reference](docs/api-reference.md) for programmatic usage
- See [Plugin Development Guide](docs/plugin-development.md) to add new platforms
- Read [Architecture](docs/architecture.md) to understand how Craowl works
