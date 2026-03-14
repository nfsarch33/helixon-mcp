// Package config handles loading and validation of server configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the MCP server.
type Config struct {
	// IronclawBaseURL is the base URL of the running IronClaw instance.
	// Default: http://localhost:3000
	IronclawBaseURL string

	// APIKey is the optional bearer token for IronClaw authentication.
	APIKey string

	// Timeout is the HTTP client timeout for IronClaw API calls.
	Timeout time.Duration

	// Transport is the MCP transport: "stdio" or "sse".
	Transport string

	// SSEAddr is the address to listen on when Transport == "sse".
	SSEAddr string

	// LogLevel controls verbosity: debug, info, warn, error.
	LogLevel string

	// PrometheusURL is the optional base URL for Prometheus metric queries.
	// If empty, the ironclaw_get_metrics tool is not registered.
	PrometheusURL string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		IronclawBaseURL: envOrDefault("IRONCLAW_BASE_URL", "http://localhost:3000"),
		APIKey:          os.Getenv("IRONCLAW_API_KEY"),
		Transport:       envOrDefault("MCP_TRANSPORT", "stdio"),
		SSEAddr:         envOrDefault("MCP_SSE_ADDR", ":8080"),
		LogLevel:        envOrDefault("LOG_LEVEL", "info"),
		PrometheusURL:   os.Getenv("PROMETHEUS_URL"),
	}

	timeoutSec := envOrDefault("IRONCLAW_TIMEOUT_SECONDS", "30")
	secs, err := strconv.Atoi(timeoutSec)
	if err != nil {
		return nil, fmt.Errorf("invalid IRONCLAW_TIMEOUT_SECONDS %q: %w", timeoutSec, err)
	}
	cfg.Timeout = time.Duration(secs) * time.Second

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.IronclawBaseURL == "" {
		return fmt.Errorf("IRONCLAW_BASE_URL must not be empty")
	}
	if c.Transport != "stdio" && c.Transport != "sse" {
		return fmt.Errorf("MCP_TRANSPORT must be \"stdio\" or \"sse\", got %q", c.Transport)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("IRONCLAW_TIMEOUT_SECONDS must be positive")
	}
	return nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
