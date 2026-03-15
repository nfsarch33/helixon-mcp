// Package main is the entry point for the IronClaw MCP server.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nfsarch33/ironclaw-mcp/internal/config"
	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
	"github.com/nfsarch33/ironclaw-mcp/internal/server"
	"github.com/nfsarch33/ironclaw-mcp/internal/tools"
	"go.uber.org/zap"
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

	logger, err := buildLogger(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("building logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	logger.Info("starting ironclaw-mcp",
		zap.String("version", version),
		zap.String("ironclaw_url", cfg.IronclawBaseURL),
		zap.String("transport", cfg.Transport),
		zap.Bool("auth_configured", cfg.APIKey != ""),
	)

	client := ironclaw.NewClient(cfg.IronclawBaseURL, cfg.APIKey, cfg.Timeout)

	var cli tools.CLIRunner
	if bin := config.MCCLIPath(); bin != "" {
		cli = tools.NewExecCLIRunner(bin)
	}
	srv := server.New(client, cli, logger, version)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	return srv.Run(ctx, cfg.Transport)
}

func buildLogger(level string) (*zap.Logger, error) {
	var cfg zap.Config
	if level == "debug" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	switch level {
	case "debug":
		cfg.Level.SetLevel(zap.DebugLevel)
	case "warn":
		cfg.Level.SetLevel(zap.WarnLevel)
	case "error":
		cfg.Level.SetLevel(zap.ErrorLevel)
	default:
		cfg.Level.SetLevel(zap.InfoLevel)
	}
	return cfg.Build()
}
