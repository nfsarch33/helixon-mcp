package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// RoutinesHandler handles routine-related MCP tools.
type RoutinesHandler struct {
	client IronclawClient
}

// NewRoutinesHandler creates a new RoutinesHandler.
func NewRoutinesHandler(client IronclawClient) *RoutinesHandler {
	return &RoutinesHandler{client: client}
}

// ListRoutinesTool returns the helixon_list_routines tool definition.
func (h *RoutinesHandler) ListRoutinesTool() mcp.Tool {
	return mcp.NewTool(
		"helixon_list_routines",
		mcp.WithDescription("List all scheduled routines in Helixon (cron jobs, event triggers)."),
	)
}

// DeleteRoutineTool returns the helixon_delete_routine tool definition.
func (h *RoutinesHandler) DeleteRoutineTool() mcp.Tool {
	return mcp.NewTool(
		"helixon_delete_routine",
		mcp.WithDescription("Delete a scheduled routine from Helixon by ID."),
		mcp.WithString("routine_id",
			mcp.Required(),
			mcp.Description("The routine ID to delete."),
		),
	)
}

// HandleListRoutines handles the list routines tool call.
func (h *RoutinesHandler) HandleListRoutines(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.ListRoutines(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("listing routines: %v", err)), nil
	}
	return jsonResult(resp)
}

// HandleDeleteRoutine handles the delete routine tool call.
func (h *RoutinesHandler) HandleDeleteRoutine(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	routineID, err := requiredString(req, "routine_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := h.client.DeleteRoutine(ctx, routineID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("deleting routine %q: %v", routineID, err)), nil
	}
	return jsonResult(map[string]string{"status": "deleted", "routine_id": routineID})
}
