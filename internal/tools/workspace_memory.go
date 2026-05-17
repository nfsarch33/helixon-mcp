package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nfsarch33/helixon-mcp/internal/helixon"
)

// WorkspaceMemoryHandler exposes Helixon workspace memory as the four generic
// memory tools expected by Cursor and other MCP clients.
type WorkspaceMemoryHandler struct {
	client IronclawClient
}

// NewWorkspaceMemoryHandler creates a workspace memory tool handler.
func NewWorkspaceMemoryHandler(client IronclawClient) *WorkspaceMemoryHandler {
	return &WorkspaceMemoryHandler{client: client}
}

// SearchTool returns the generic memory_search tool definition.
func (h *WorkspaceMemoryHandler) SearchTool() mcp.Tool {
	return mcp.NewTool(
		"memory_search",
		mcp.WithDescription("Search Helixon workspace memory using hybrid FTS/vector RRF ranking."),
		mcp.WithString("query", mcp.Required(), mcp.Description("Search query.")),
		mcp.WithString("limit", mcp.Description("Maximum result count (default 10).")),
	)
}

// WriteTool returns the generic memory_write tool definition.
func (h *WorkspaceMemoryHandler) WriteTool() mcp.Tool {
	return mcp.NewTool(
		"memory_write",
		mcp.WithDescription("Write or replace an Helixon workspace memory entry by path."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Workspace memory path.")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Markdown or text content to store.")),
	)
}

// ReadTool returns the generic memory_read tool definition.
func (h *WorkspaceMemoryHandler) ReadTool() mcp.Tool {
	return mcp.NewTool(
		"memory_read",
		mcp.WithDescription("Read a single Helixon workspace memory entry by path."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Workspace memory path.")),
	)
}

// TreeTool returns the generic memory_tree tool definition.
func (h *WorkspaceMemoryHandler) TreeTool() mcp.Tool {
	return mcp.NewTool(
		"memory_tree",
		mcp.WithDescription("List Helixon workspace memory paths below a prefix."),
		mcp.WithString("prefix", mcp.Description("Optional path prefix.")),
	)
}

// HandleSearch executes memory_search.
func (h *WorkspaceMemoryHandler) HandleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := requiredString(req, "query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	limit := 10
	if limitStr := optionalString(req, "limit"); limitStr != "" {
		n, err := strconv.Atoi(limitStr)
		if err != nil || n <= 0 {
			return mcp.NewToolResultError("argument \"limit\" must be a positive integer"), nil
		}
		limit = n
	}
	resp, err := h.client.SearchMemory(ctx, helixon.MemorySearchRequest{Query: query, Limit: limit})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("searching memory: %v", err)), nil
	}
	return jsonResult(resp)
}

// HandleWrite executes memory_write.
func (h *WorkspaceMemoryHandler) HandleWrite(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := requiredString(req, "path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	content, err := requiredString(req, "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	resp, err := h.client.WriteMemory(ctx, helixon.MemoryWriteRequest{Path: path, Content: content})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("writing memory: %v", err)), nil
	}
	return jsonResult(resp)
}

// HandleRead executes memory_read.
func (h *WorkspaceMemoryHandler) HandleRead(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := requiredString(req, "path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	resp, err := h.client.ReadMemory(ctx, helixon.MemoryReadRequest{Path: path})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("reading memory: %v", err)), nil
	}
	return jsonResult(resp)
}

// HandleTree executes memory_tree.
func (h *WorkspaceMemoryHandler) HandleTree(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.TreeMemory(ctx, helixon.MemoryTreeRequest{Prefix: optionalString(req, "prefix")})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("listing memory tree: %v", err)), nil
	}
	return jsonResult(resp)
}
