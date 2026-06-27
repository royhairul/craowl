# Craowl Results Directory

This directory contains all the extracted data from `craowl` crawls, automatically organized into a hierarchical structure by seller and product ID.

## Directory Structure

Data is stored in the following format:

```
results/
└── <seller_username>/                 # e.g., "erigostore" or "shop_123456"
    ├── seller.json                    # Detailed profile information for the seller
    ├── ratings.json                   # Seller-level ratings and reviews
    ├── reviews.csv                    # Seller-level reviews in CSV format
    │
    └── products/
        ├── <product_id>/              # e.g., "10551715113"
        │   ├── product.json           # Individual product details (price, stock, etc.)
        │   ├── ratings.json           # Product-specific ratings and reviews (JSON)
        │   └── reviews.csv            # Product-specific ratings and reviews (CSV)
        │
        └── <product_id_2>/
            └── ...
```

## How It Works

- **`craowl crawl shopee --type seller`**: Creates the `results/<seller_username>` folder and saves the extracted shop info into `seller.json`.
- **`craowl crawl shopee --type product_list`**: Creates a folder inside `products/` for each item found on the seller's page, and saves individual `product.json` files for each.
- **`craowl crawl shopee --type product_rating`**: Locates the specific product's folder and outputs `ratings.json` and `reviews.csv` containing the customer reviews.

*Note: If a direct product link is crawled and the seller username is not available, the system will use the `shop_id` (e.g., `shop_30203584`) as the root folder.*
