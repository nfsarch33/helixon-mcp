// Package tools defines all MCP tool handlers for IronClaw.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
)

// ChatHandler handles the ironclaw_chat MCP tool.
type ChatHandler struct {
	client IronclawClient
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(client IronclawClient) *ChatHandler {
	return &ChatHandler{client: client}
}

// Tool returns the MCP tool definition.
func (h *ChatHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_chat",
		mcp.WithDescription("Send a message to your IronClaw AI assistant and receive a response. Use this for general queries, commands, and interactions with IronClaw."),
		mcp.WithString("message",
			mcp.Required(),
			mcp.Description("The message to send to IronClaw."),
		),
		mcp.WithString("session_id",
			mcp.Description("Optional session ID to maintain conversation context across calls."),
		),
	)
}

// Handle executes the chat tool.
func (h *ChatHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	message, err := requiredString(req, "message")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	sessionID := optionalString(req, "session_id")

	resp, err := h.client.Chat(ctx, ironclaw.ChatRequest{
		Message:   message,
		SessionID: sessionID,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("IronClaw chat error: %v", err)), nil
	}

	result := map[string]any{
		"response":   resp.Response,
		"message_id": resp.MessageID,
		"session_id": resp.SessionID,
		"status":     resp.Status,
	}
	return jsonResult(result)
}

// jsonResult marshals v into a text MCP tool result.
func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("encoding result: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}
