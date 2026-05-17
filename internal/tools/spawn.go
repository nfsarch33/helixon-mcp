package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/nfsarch33/helixon-mcp/internal/helixon"
)

// SpawnAgentHandler handles the helixon_spawn_agent MCP tool.
type SpawnAgentHandler struct {
	client IronclawClient
}

// NewSpawnAgentHandler creates a new SpawnAgentHandler.
func NewSpawnAgentHandler(client IronclawClient) *SpawnAgentHandler {
	return &SpawnAgentHandler{client: client}
}

// Tool returns the helixon_spawn_agent MCP tool definition.
func (h *SpawnAgentHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"helixon_spawn_agent",
		mcp.WithDescription("Spawn a new Helixon agent job. Returns the job ID and initial status."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name for the new agent")),
		mcp.WithString("model", mcp.Description("LLM model to use (default: instance config)")),
		mcp.WithString("tier", mcp.Description("Routing tier: agent, fast, reasoning")),
	)
}

// Handle executes the spawn operation.
func (h *SpawnAgentHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := requiredString(req, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	spawnReq := helixon.SpawnAgentRequest{
		Name:  name,
		Model: optionalString(req, "model"),
		Tier:  optionalString(req, "tier"),
	}

	resp, err := h.client.SpawnAgent(ctx, spawnReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("spawn agent failed: %v", err)), nil
	}
	return jsonResult(resp)
}
