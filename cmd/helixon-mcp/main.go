// Package main is the entry point for the Helixon MCP server.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nfsarch33/helixon-mcp/internal/config"
	"github.com/nfsarch33/helixon-mcp/internal/helixon"
	"github.com/nfsarch33/helixon-mcp/internal/server"
	"github.com/nfsarch33/helixon-mcp/internal/tools"
)

const version = "0.5.1"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-version", "version":
			fmt.Printf("helixon-mcp %s\n", version)
			return nil
		case "--help", "-h", "help":
			printUsage()
			return nil
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := buildLogger(cfg.LogLevel)

	logger.Info("starting helixon-mcp",
		"version", version,
		"helixon_url", cfg.IronclawBaseURL,
		"transport", cfg.Transport,
		"auth_configured", cfg.APIKey != "",
	)

	client := helixon.NewClient(cfg.IronclawBaseURL, cfg.APIKey, cfg.Timeout)

	var prom tools.PrometheusQuerier
	if cfg.PrometheusURL != "" {
		prom = tools.NewHTTPPrometheusQuerier(cfg.PrometheusURL)
		logger.Info("prometheus enabled", "url", cfg.PrometheusURL)
	}

	srv := server.New(client, prom, logger, version)

	if cfg.PrometheusMetricsPort != "" {
		go func() {
			http.Handle("/metrics", promhttp.Handler())
			logger.Info("starting metrics server", "port", cfg.PrometheusMetricsPort)
			if err := http.ListenAndServe(":"+cfg.PrometheusMetricsPort, nil); err != nil {
				logger.Error("metrics server failed", "error", err)
			}
		}()
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return srv.Run(ctx, cfg.Transport)
}

func buildLogger(level string) *slog.Logger {
	var slevel slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		slevel = slog.LevelDebug
	case "warn", "warning":
		slevel = slog.LevelWarn
	case "error":
		slevel = slog.LevelError
	default:
		slevel = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slevel}))
}

func printUsage() {
	fmt.Printf(`helixon-mcp %s

A general-purpose MCP server bridging Helixon with MCP-compatible AI clients.

Usage:
  helixon-mcp [--version | --help]

The server is configured entirely through environment variables; see
docs/configuration.md or README.md for the full reference. Most common:

  HELIXON_BASE_URL              Helixon gateway URL (default http://localhost:3000)
  HELIXON_API_KEY               Bearer token when GATEWAY_AUTH_TOKEN is set
  HELIXON_ALLOW_NON_LOCALHOST   Allow non-loopback HELIXON_BASE_URL (default false)
  MCP_TRANSPORT                  stdio | sse (default stdio)
  MCP_SSE_ADDR                   bind address for sse (default :8080)
  LOG_LEVEL                      debug | info | warn | error (default info)
  PROMETHEUS_URL                 enables helixon_get_metrics tool

Source: https://github.com/nfsarch33/helixon-mcp
`, version)
}
