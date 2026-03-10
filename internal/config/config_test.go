package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("IRONCLAW_BASE_URL", "")
	t.Setenv("MCP_TRANSPORT", "")
	t.Setenv("IRONCLAW_TIMEOUT_SECONDS", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:3000", cfg.IronclawBaseURL)
	assert.Equal(t, "stdio", cfg.Transport)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("IRONCLAW_BASE_URL", "http://myhost:4000")
	t.Setenv("IRONCLAW_API_KEY", "secret123")
	t.Setenv("MCP_TRANSPORT", "sse")
	t.Setenv("MCP_SSE_ADDR", ":9090")
	t.Setenv("IRONCLAW_TIMEOUT_SECONDS", "60")
	t.Setenv("LOG_LEVEL", "debug")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "http://myhost:4000", cfg.IronclawBaseURL)
	assert.Equal(t, "secret123", cfg.APIKey)
	assert.Equal(t, "sse", cfg.Transport)
	assert.Equal(t, ":9090", cfg.SSEAddr)
	assert.Equal(t, 60*time.Second, cfg.Timeout)
	assert.Equal(t, "debug", cfg.LogLevel)
}

func TestLoad_InvalidTimeout(t *testing.T) {
	t.Setenv("IRONCLAW_TIMEOUT_SECONDS", "not-a-number")
	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IRONCLAW_TIMEOUT_SECONDS")
}

func TestLoad_InvalidTransport(t *testing.T) {
	t.Setenv("IRONCLAW_TIMEOUT_SECONDS", "30")
	t.Setenv("MCP_TRANSPORT", "grpc")
	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MCP_TRANSPORT")
}

func TestLoad_EmptyBaseURL(t *testing.T) {
	t.Setenv("IRONCLAW_BASE_URL", "cannot-be-set-empty-via-setenv")
	// validate() checks empty string but setenv above prevents empty;
	// test validate directly:
	c := &Config{IronclawBaseURL: "", Transport: "stdio", Timeout: 30 * time.Second}
	err := c.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "IRONCLAW_BASE_URL")
}
