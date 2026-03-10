package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
)

// MemoryHandler handles memory-related MCP tools.
type MemoryHandler struct {
	client IronclawClient
}

// NewMemoryHandler creates a new MemoryHandler.
func NewMemoryHandler(client IronclawClient) *MemoryHandler {
	return &MemoryHandler{client: client}
}

// Tool returns the ironclaw_search_memory tool definition.
func (h *MemoryHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_search_memory",
		mcp.WithDescription("Search IronClaw's persistent workspace memory using semantic search. Returns relevant notes, context, and previously stored information."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query to find relevant memory entries."),
		),
		mcp.WithString("limit",
			mcp.Description("Maximum number of results to return (default: 10)."),
		),
	)
}

// Handle executes the search memory tool.
func (h *MemoryHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := requiredString(req, "query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	limit := 10
	if limitStr := optionalString(req, "limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	resp, err := h.client.SearchMemory(ctx, ironclaw.MemorySearchRequest{
		Query: query,
		Limit: limit,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("searching memory: %v", err)), nil
	}
	return jsonResult(resp)
}
