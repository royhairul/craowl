package platform

import (
	"os"
	"strconv"
	"time"
)

// LoadConfigFromEnv creates a Config object based on environment variables
func LoadConfigFromEnv() Config {
	config := DefaultConfig()

	if val := os.Getenv("TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			config.Timeout = d
		}
	}
	if val := os.Getenv("MAX_RETRIES"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.MaxRetries = i
		}
	}
	if val := os.Getenv("RATE_LIMIT_REQUESTS_PER_SECOND"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.RateLimit = i
		}
	}
	if val := os.Getenv("PROXY_URL"); val != "" {
		config.ProxyURL = val
	}
	if val := os.Getenv("LOG_LEVEL"); val == "debug" {
		config.Debug = true
	}

	// Platform specific overrides (Shopee)
	if val := os.Getenv("SHOPEE_BASE_URL"); val != "" {
		config.BaseURL = val
	}
	if val := os.Getenv("SHOPEE_TIMEOUT"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			config.Timeout = d
		}
	}
	if val := os.Getenv("SHOPEE_RATE_LIMIT"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			config.RateLimit = i
		}
	}
	if val := os.Getenv("SHOPEE_USER_AGENT"); val != "" {
		config.UserAgent = val
	}

	return config
}
