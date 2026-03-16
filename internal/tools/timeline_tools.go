package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// TimelineHandler exposes activity timeline operations as an MCP tool, delegating to mc-cli timeline.
type TimelineHandler struct {
	cli CLIRunner
}

func NewTimelineHandler(cli CLIRunner) *TimelineHandler {
	return &TimelineHandler{cli: cli}
}

func (h *TimelineHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_timeline",
		mcp.WithDescription("Query and manage the activity timeline: list events, export for crash recovery, import from backup."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: list, export, import")),
		mcp.WithString("type", mcp.Description("Filter by event type (spawn, tool_invoke, approval, task_update, gpu_alert, etc.)")),
		mcp.WithString("actor", mcp.Description("Filter by actor name")),
		mcp.WithString("limit", mcp.Description("Maximum number of events to return (default: 50)")),
		mcp.WithString("file", mcp.Description("File path for export/import operations")),
	)
}

func (h *TimelineHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	action, err := requiredString(req, "action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"timeline", action}
	if t := optionalString(req, "type"); t != "" {
		args = append(args, "--type", t)
	}
	if a := optionalString(req, "actor"); a != "" {
		args = append(args, "--actor", a)
	}
	if l := optionalString(req, "limit"); l != "" {
		args = append(args, "--limit", l)
	}
	if f := optionalString(req, "file"); f != "" {
		args = append(args, "--file", f)
	}

	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		dualToolOpsTotal.WithLabelValues("timeline", action, "error").Inc()
		return mcp.NewToolResultError(fmt.Sprintf("timeline %s: %v\n%s", action, err, out)), nil
	}
	dualToolOpsTotal.WithLabelValues("timeline", action, "success").Inc()
	return mcp.NewToolResultText(out), nil
}
