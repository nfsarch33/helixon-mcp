package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

// StackStatusHandler handles the ironclaw_stack_status MCP tool.
type StackStatusHandler struct {
	client    IronclawClient
	routerURL string
}

// NewStackStatusHandler creates a new StackStatusHandler.
// routerURL is read from IRONCLAW_ROUTER_URL env var, defaulting to http://127.0.0.1:8080.
func NewStackStatusHandler(client IronclawClient) *StackStatusHandler {
	routerURL := os.Getenv("IRONCLAW_ROUTER_URL")
	if routerURL == "" {
		routerURL = "http://127.0.0.1:8080"
	}
	return &StackStatusHandler{client: client, routerURL: routerURL}
}

// Tool returns the ironclaw_stack_status MCP tool definition.
func (h *StackStatusHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_stack_status",
		mcp.WithDescription("Get combined health status of the IronClaw stack: LLM router nodes, GPU availability, gateway status, queue depth, and inflight requests."),
	)
}

// Handle executes the stack status check.
func (h *StackStatusHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.StackStatus(ctx, h.routerURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("stack status failed: %v", err)), nil
	}
	return jsonResult(resp)
}
