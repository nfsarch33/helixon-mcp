package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// HealthHandler handles the helixon_health MCP tool.
type HealthHandler struct {
	client IronclawClient
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(client IronclawClient) *HealthHandler {
	return &HealthHandler{client: client}
}

// Tool returns the helixon_health MCP tool definition.
func (h *HealthHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"helixon_health",
		mcp.WithDescription("Check the health and availability of the Helixon instance. Returns status and version."),
	)
}

// Handle executes the health check.
func (h *HealthHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.Health(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("health check failed: %v", err)), nil
	}
	return jsonResult(resp)
}
