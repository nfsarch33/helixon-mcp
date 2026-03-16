// Package main is the entry point for the IronClaw MCP server.
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

	"github.com/nfsarch33/ironclaw-mcp/internal/config"
	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
	"github.com/nfsarch33/ironclaw-mcp/internal/server"
	"github.com/nfsarch33/ironclaw-mcp/internal/tools"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const version = "0.1.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	logger := buildLogger(cfg.LogLevel)

	logger.Info("starting ironclaw-mcp",
		"version", version,
		"ironclaw_url", cfg.IronclawBaseURL,
		"transport", cfg.Transport,
		"auth_configured", cfg.APIKey != "",
	)

	client := ironclaw.NewClient(cfg.IronclawBaseURL, cfg.APIKey, cfg.Timeout)

	var prom tools.PrometheusQuerier
	if cfg.PrometheusURL != "" {
		prom = tools.NewHTTPPrometheusQuerier(cfg.PrometheusURL)
		logger.Info("prometheus enabled", "url", cfg.PrometheusURL)
	}

	var cli tools.CLIRunner
	if bin := config.MCCLIPath(); bin != "" {
		cli = tools.NewExecCLIRunner(bin)
	}

	gws := tools.NewExecCLIRunner("gws")

	srv := server.New(client, prom, cli, gws, logger, version)

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
