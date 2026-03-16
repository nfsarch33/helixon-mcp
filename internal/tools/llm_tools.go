package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// LLMRouteHandler exposes hybrid LLM routing as an MCP tool, delegating to mc-cli llm-route.
type LLMRouteHandler struct {
	cli CLIRunner
}

func NewLLMRouteHandler(cli CLIRunner) *LLMRouteHandler {
	return &LLMRouteHandler{cli: cli}
}

func (h *LLMRouteHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_llm_route",
		mcp.WithDescription("Route an LLM request to the optimal provider (local GPU or cloud API) based on complexity, context length, and budget."),
		mcp.WithString("complexity", mcp.Description("Task complexity: simple, moderate, complex (default: simple)")),
		mcp.WithString("context_len", mcp.Description("Estimated context length in tokens (default: 2000)")),
		mcp.WithString("require_large_context", mcp.Description("Set to true if task requires large context window")),
	)
}

func (h *LLMRouteHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}

	args := []string{"llm-route"}
	if c := optionalString(req, "complexity"); c != "" {
		args = append(args, "--complexity", c)
	}
	if cl := optionalString(req, "context_len"); cl != "" {
		args = append(args, "--context-len", cl)
	}
	if rlc := optionalString(req, "require_large_context"); rlc == "true" {
		args = append(args, "--gpus")
	}

	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		dualToolOpsTotal.WithLabelValues("llm_route", "route", "error").Inc()
		return mcp.NewToolResultError(fmt.Sprintf("llm-route: %v\n%s", err, out)), nil
	}
	dualToolOpsTotal.WithLabelValues("llm_route", "route", "success").Inc()
	return mcp.NewToolResultText(out), nil
}

// LLMUsageHandler exposes LLM token usage summary as an MCP tool, delegating to mc-cli llm-usage.
type LLMUsageHandler struct {
	cli CLIRunner
}

func NewLLMUsageHandler(cli CLIRunner) *LLMUsageHandler {
	return &LLMUsageHandler{cli: cli}
}

func (h *LLMUsageHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_llm_usage",
		mcp.WithDescription("Get LLM token usage summary grouped by provider, including costs and request counts."),
	)
}

func (h *LLMUsageHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}

	out, err := h.cli.Run(ctx, "llm-usage")
	if err != nil {
		dualToolOpsTotal.WithLabelValues("llm_usage", "summary", "error").Inc()
		return mcp.NewToolResultError(fmt.Sprintf("llm-usage: %v\n%s", err, out)), nil
	}
	dualToolOpsTotal.WithLabelValues("llm_usage", "summary", "success").Inc()
	return mcp.NewToolResultText(out), nil
}

// LLMBudgetHandler exposes LLM budget status as an MCP tool, delegating to mc-cli llm-budget.
type LLMBudgetHandler struct {
	cli CLIRunner
}

func NewLLMBudgetHandler(cli CLIRunner) *LLMBudgetHandler {
	return &LLMBudgetHandler{cli: cli}
}

func (h *LLMBudgetHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_llm_budget",
		mcp.WithDescription("Check daily LLM budget status: total budget, amount spent, remaining balance, and whether budget is exhausted."),
	)
}

func (h *LLMBudgetHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}

	out, err := h.cli.Run(ctx, "llm-budget")
	if err != nil {
		dualToolOpsTotal.WithLabelValues("llm_budget", "check", "error").Inc()
		return mcp.NewToolResultError(fmt.Sprintf("llm-budget: %v\n%s", err, out)), nil
	}
	dualToolOpsTotal.WithLabelValues("llm_budget", "check", "success").Inc()
	return mcp.NewToolResultText(out), nil
}
