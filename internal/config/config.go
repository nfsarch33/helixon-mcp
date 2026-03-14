// Package config handles loading and validation of server configuration.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	minTimeoutSec = 1
	maxTimeoutSec = 120
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

	// AllowNonLocalhost permits IRONCLAW_BASE_URL to point to non-loopback hosts.
	AllowNonLocalhost bool

	// PrometheusURL is the optional base URL for Prometheus metric queries.
	// If empty, the ironclaw_get_metrics tool is not registered.
	PrometheusURL string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		IronclawBaseURL:   envOrDefault("IRONCLAW_BASE_URL", "http://localhost:3000"),
		APIKey:            os.Getenv("IRONCLAW_API_KEY"),
		Transport:         envOrDefault("MCP_TRANSPORT", "stdio"),
		SSEAddr:           envOrDefault("MCP_SSE_ADDR", ":8080"),
		LogLevel:          envOrDefault("LOG_LEVEL", "info"),
		AllowNonLocalhost: envOrDefault("IRONCLAW_ALLOW_NON_LOCALHOST", "") == "true",
		PrometheusURL:     os.Getenv("PROMETHEUS_URL"),
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
	if err := validateBaseURL(c.IronclawBaseURL, c.AllowNonLocalhost); err != nil {
		return err
	}
	if c.Transport != "stdio" && c.Transport != "sse" {
		return fmt.Errorf("MCP_TRANSPORT must be \"stdio\" or \"sse\", got %q", c.Transport)
	}
	if c.Timeout < minTimeoutSec*time.Second {
		return fmt.Errorf("IRONCLAW_TIMEOUT_SECONDS must be at least %d", minTimeoutSec)
	}
	if c.Timeout > maxTimeoutSec*time.Second {
		return fmt.Errorf("IRONCLAW_TIMEOUT_SECONDS must be at most %d", maxTimeoutSec)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
		// valid
	default:
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error, got %q", c.LogLevel)
	}
	return nil
}

// validateBaseURL checks that the URL is well-formed, uses http(s), and optionally restricts to loopback.
func validateBaseURL(raw string, allowNonLocalhost bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("IRONCLAW_BASE_URL malformed: %w", err)
	}
	switch u.Scheme {
	case "http", "https":
		// allowed
	default:
		return fmt.Errorf("IRONCLAW_BASE_URL must use http or https scheme, got %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("IRONCLAW_BASE_URL must have a host")
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}
	if allowNonLocalhost {
		return nil
	}
	// Local-first: only allow loopback addresses by default.
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip != nil {
		if !ip.IsLoopback() {
			return fmt.Errorf("IRONCLAW_BASE_URL host %q is not loopback; set IRONCLAW_ALLOW_NON_LOCALHOST=true to allow", host)
		}
		return nil
	}
	// Hostname: allow localhost variants only.
	lower := strings.ToLower(host)
	if lower != "localhost" && lower != "127.0.0.1" && lower != "::1" {
		return fmt.Errorf("IRONCLAW_BASE_URL host %q is not localhost; set IRONCLAW_ALLOW_NON_LOCALHOST=true to allow", host)
	}
	return nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
