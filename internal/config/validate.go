package config

import (
	"fmt"
	"net/url"
)

// Validate checks the configuration for invalid values.
func Validate(cfg *Config) error {
	if cfg.Engine.Concurrency < 1 {
		return fmt.Errorf("engine.concurrency must be >= 1, got %d", cfg.Engine.Concurrency)
	}
	if cfg.Engine.Concurrency > 1000 {
		return fmt.Errorf("engine.concurrency must be <= 1000, got %d", cfg.Engine.Concurrency)
	}
	if cfg.Engine.MaxDepth < 0 {
		return fmt.Errorf("engine.max_depth must be >= 0, got %d", cfg.Engine.MaxDepth)
	}
	if cfg.Engine.RequestTimeout <= 0 {
		return fmt.Errorf("engine.request_timeout must be > 0")
	}
	if cfg.Engine.PolitenessDelay < 0 {
		return fmt.Errorf("engine.politeness_delay must be >= 0")
	}
	if cfg.Engine.MaxRetries < 0 {
		return fmt.Errorf("engine.max_retries must be >= 0, got %d", cfg.Engine.MaxRetries)
	}

	if cfg.Fetcher.MaxBodySize <= 0 {
		return fmt.Errorf("fetcher.max_body_size must be > 0")
	}
	if cfg.Fetcher.MaxRedirects < 0 {
		return fmt.Errorf("fetcher.max_redirects must be >= 0")
	}
	if cfg.Fetcher.Type != "http" && cfg.Fetcher.Type != "browser" {
		return fmt.Errorf("fetcher.type must be 'http' or 'browser', got %q", cfg.Fetcher.Type)
	}

	if cfg.Proxy.Enabled {
		if cfg.Proxy.Rotation != "round_robin" && cfg.Proxy.Rotation != "random" {
			return fmt.Errorf("proxy.rotation must be 'round_robin' or 'random', got %q", cfg.Proxy.Rotation)
		}
		for _, proxyURL := range cfg.Proxy.URLs {
			if _, err := url.Parse(proxyURL); err != nil {
				return fmt.Errorf("invalid proxy URL %q: %w", proxyURL, err)
			}
		}
	}

	validStorageTypes := map[string]bool{
		"json": true, "jsonl": true, "csv": true,
	}
	if !validStorageTypes[cfg.Storage.Type] {
		return fmt.Errorf("storage.type %q is not supported (valid: json, jsonl, csv)", cfg.Storage.Type)
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[cfg.Logging.Level] {
		return fmt.Errorf("logging.level must be debug/info/warn/error, got %q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" && cfg.Logging.Format != "json" {
		return fmt.Errorf("logging.format must be 'text' or 'json', got %q", cfg.Logging.Format)
	}

	if cfg.Metrics.Enabled {
		if cfg.Metrics.Port < 1 || cfg.Metrics.Port > 65535 {
			return fmt.Errorf("metrics.port must be 1-65535, got %d", cfg.Metrics.Port)
		}
	}

	return nil
}

// ValidateURL checks if a URL string is valid for crawling.
func ValidateURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("URL must have a host")
	}
	return nil
}
