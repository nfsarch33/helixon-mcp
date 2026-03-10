package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
)

// RoutinesHandler handles routine-related MCP tools.
type RoutinesHandler struct {
	client IronclawClient
}

// NewRoutinesHandler creates a new RoutinesHandler.
func NewRoutinesHandler(client IronclawClient) *RoutinesHandler {
	return &RoutinesHandler{client: client}
}

// ListRoutinesTool returns the ironclaw_list_routines tool definition.
func (h *RoutinesHandler) ListRoutinesTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_list_routines",
		mcp.WithDescription("List all scheduled routines in IronClaw (cron jobs, event triggers)."),
	)
}

// CreateRoutineTool returns the ironclaw_create_routine tool definition.
func (h *RoutinesHandler) CreateRoutineTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_create_routine",
		mcp.WithDescription("Create a new scheduled routine in IronClaw. Routines run a prompt on a cron schedule."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Unique name for the routine."),
		),
		mcp.WithString("schedule",
			mcp.Required(),
			mcp.Description("Cron schedule expression (e.g. '0 9 * * *' for 9am daily)."),
		),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("The prompt to execute on the schedule."),
		),
	)
}

// DeleteRoutineTool returns the ironclaw_delete_routine tool definition.
func (h *RoutinesHandler) DeleteRoutineTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_delete_routine",
		mcp.WithDescription("Delete a scheduled routine from IronClaw by ID."),
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

// HandleCreateRoutine handles the create routine tool call.
func (h *RoutinesHandler) HandleCreateRoutine(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := requiredString(req, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	schedule, err := requiredString(req, "schedule")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	prompt, err := requiredString(req, "prompt")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	routine, err := h.client.CreateRoutine(ctx, ironclaw.CreateRoutineRequest{
		Name:     name,
		Schedule: schedule,
		Prompt:   prompt,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("creating routine: %v", err)), nil
	}
	return jsonResult(routine)
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
