package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
)

// SendTaskHandler handles the ironclaw_send_task MCP tool.
type SendTaskHandler struct {
	client IronclawClient
}

// NewSendTaskHandler creates a new SendTaskHandler.
func NewSendTaskHandler(client IronclawClient) *SendTaskHandler {
	return &SendTaskHandler{client: client}
}

// Tool returns the ironclaw_send_task MCP tool definition.
func (h *SendTaskHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_send_task",
		mcp.WithDescription("Send a strategic task to IronClaw for background execution via POST /api/chat/send with Bearer token auth. Returns the job ID for tracking."),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("The strategic task or instruction to send to IronClaw."),
		),
		mcp.WithString("session_id",
			mcp.Description("Optional session ID to maintain task context."),
		),
	)
}

// Handle executes the send task tool.
func (h *SendTaskHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	message, err := requiredString(req, "message")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	sessionID := optionalString(req, "session_id")

	resp, err := h.client.SendTask(ctx, ironclaw.SendTaskRequest{
		Message:   message,
		SessionID: sessionID,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("sending task: %v", err)), nil
	}
	return jsonResult(resp)
}
