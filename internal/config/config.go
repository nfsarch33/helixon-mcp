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
	// IronclawBaseURL is the base URL of the running Helixon instance.
	// Default: http://localhost:3000
	IronclawBaseURL string

	// APIKey is the optional bearer token for Helixon authentication.
	APIKey string

	// Timeout is the HTTP client timeout for Helixon API calls.
	Timeout time.Duration

	// Transport is the MCP transport: "stdio" or "sse".
	Transport string

	// SSEAddr is the address to listen on when Transport == "sse".
	SSEAddr string

	// LogLevel controls verbosity: debug, info, warn, error.
	LogLevel string

	// AllowNonLocalhost permits HELIXON_BASE_URL to point to non-loopback hosts.
	AllowNonLocalhost bool

	// PrometheusURL is the optional base URL for Prometheus metric queries.
	// If empty, the helixon_get_metrics tool is not registered.
	PrometheusURL string

	// PrometheusMetricsPort is the optional port to expose /metrics.
	PrometheusMetricsPort string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		IronclawBaseURL:       envOrDefault("HELIXON_BASE_URL", "http://localhost:3000"),
		APIKey:                os.Getenv("HELIXON_API_KEY"),
		Transport:             envOrDefault("MCP_TRANSPORT", "stdio"),
		SSEAddr:               envOrDefault("MCP_SSE_ADDR", ":8080"),
		LogLevel:              envOrDefault("LOG_LEVEL", "info"),
		AllowNonLocalhost:     envOrDefault("HELIXON_ALLOW_NON_LOCALHOST", "") == "true",
		PrometheusURL:         os.Getenv("PROMETHEUS_URL"),
		PrometheusMetricsPort: os.Getenv("PROMETHEUS_METRICS_PORT"),
	}

	timeoutSec := envOrDefault("HELIXON_TIMEOUT_SECONDS", "30")
	secs, err := strconv.Atoi(timeoutSec)
	if err != nil {
		return nil, fmt.Errorf("invalid HELIXON_TIMEOUT_SECONDS %q: %w", timeoutSec, err)
	}
	cfg.Timeout = time.Duration(secs) * time.Second

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.IronclawBaseURL == "" {
		return fmt.Errorf("HELIXON_BASE_URL must not be empty")
	}
	if err := validateBaseURL(c.IronclawBaseURL, c.AllowNonLocalhost); err != nil {
		return err
	}
	if c.Transport != "stdio" && c.Transport != "sse" {
		return fmt.Errorf("MCP_TRANSPORT must be \"stdio\" or \"sse\", got %q", c.Transport)
	}
	if c.Timeout < minTimeoutSec*time.Second {
		return fmt.Errorf("HELIXON_TIMEOUT_SECONDS must be at least %d", minTimeoutSec)
	}
	if c.Timeout > maxTimeoutSec*time.Second {
		return fmt.Errorf("HELIXON_TIMEOUT_SECONDS must be at most %d", maxTimeoutSec)
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
		// valid
	default:
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error, got %q", c.LogLevel)
	}
	return nil
}

// validateBaseURL checks that the URL is well-formed, uses http(s), and
// optionally restricts to loopback. Thin orchestrator on top of the helpers
// below; the heavy lifting is in checkScheme / extractHost / requireLocalhost.
func validateBaseURL(raw string, allowNonLocalhost bool) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("HELIXON_BASE_URL malformed: %w", err)
	}
	if err := checkScheme(u.Scheme); err != nil {
		return err
	}
	if u.Host == "" {
		return fmt.Errorf("HELIXON_BASE_URL must have a host")
	}
	host := extractHost(u.Host)
	if allowNonLocalhost {
		return nil
	}
	return requireLocalhost(host)
}

// checkScheme reports whether the given scheme is http or https.
func checkScheme(scheme string) error {
	switch scheme {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("HELIXON_BASE_URL must use http or https scheme, got %q", scheme)
	}
}

// extractHost returns the hostname portion of a host[:port] string. If the
// input is a bare hostname (no port) it is returned unchanged.
func extractHost(hostPort string) string {
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		return hostPort
	}
	return host
}

// requireLocalhost returns nil only if host is a loopback IP or one of the
// well-known localhost names. Bracketed IPv6 literals are stripped before
// parsing.
func requireLocalhost(host string) error {
	stripped := strings.Trim(host, "[]")
	if ip := net.ParseIP(stripped); ip != nil {
		if !ip.IsLoopback() {
			return fmt.Errorf("HELIXON_BASE_URL host %q is not loopback; set HELIXON_ALLOW_NON_LOCALHOST=true to allow", host)
		}
		return nil
	}
	switch strings.ToLower(stripped) {
	case "localhost", "127.0.0.1", "::1":
		return nil
	default:
		return fmt.Errorf("HELIXON_BASE_URL host %q is not localhost; set HELIXON_ALLOW_NON_LOCALHOST=true to allow", host)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
