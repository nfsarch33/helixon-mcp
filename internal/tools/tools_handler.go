package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// ToolsListHandler handles the helixon_list_tools MCP tool.
type ToolsListHandler struct {
	client IronclawClient
}

// NewToolsListHandler creates a new ToolsListHandler.
func NewToolsListHandler(client IronclawClient) *ToolsListHandler {
	return &ToolsListHandler{client: client}
}

// Tool returns the helixon_list_tools MCP tool definition.
func (h *ToolsListHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"helixon_list_tools",
		mcp.WithDescription("List all tools registered in Helixon, including built-in, WASM, and MCP-connected tools."),
	)
}

// Handle executes the list tools call.
func (h *ToolsListHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.ListTools(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("listing tools: %v", err)), nil
	}
	return jsonResult(resp)
}
