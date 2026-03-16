package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// GovernanceHandler exposes governance operations as an MCP tool, delegating to mc-cli governance.
type GovernanceHandler struct {
	cli CLIRunner
}

func NewGovernanceHandler(cli CLIRunner) *GovernanceHandler {
	return &GovernanceHandler{cli: cli}
}

func (h *GovernanceHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_governance",
		mcp.WithDescription("Manage governance approvals: list pending requests, approve or deny operations, classify risk levels."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: list-pending, approve, deny, risk-classify")),
		mcp.WithString("request_id", mcp.Description("Approval request ID (required for approve/deny)")),
		mcp.WithString("approved_by", mcp.Description("Identity of the approver (required for approve/deny)")),
		mcp.WithString("reason", mcp.Description("Reason for approval or denial")),
		mcp.WithString("tool", mcp.Description("Tool name for risk classification")),
		mcp.WithString("tool_action", mcp.Description("Tool action for risk classification")),
	)
}

func (h *GovernanceHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	action, err := requiredString(req, "action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"governance", action}
	if rid := optionalString(req, "request_id"); rid != "" {
		args = append(args, "--id", rid)
	}
	if ab := optionalString(req, "approved_by"); ab != "" {
		args = append(args, "--by", ab)
	}
	if reason := optionalString(req, "reason"); reason != "" {
		args = append(args, "--reason", reason)
	}
	if tool := optionalString(req, "tool"); tool != "" {
		args = append(args, "--tool", tool)
	}
	if ta := optionalString(req, "tool_action"); ta != "" {
		args = append(args, "--action", ta)
	}

	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		dualToolOpsTotal.WithLabelValues("governance", action, "error").Inc()
		return mcp.NewToolResultError(fmt.Sprintf("governance %s: %v\n%s", action, err, out)), nil
	}
	dualToolOpsTotal.WithLabelValues("governance", action, "success").Inc()
	return mcp.NewToolResultText(out), nil
}
