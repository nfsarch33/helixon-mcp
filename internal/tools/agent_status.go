package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// AgentStatusHandler handles the ironclaw_agent_status MCP tool.
type AgentStatusHandler struct {
	client IronclawClient
}

// NewAgentStatusHandler creates a new AgentStatusHandler.
func NewAgentStatusHandler(client IronclawClient) *AgentStatusHandler {
	return &AgentStatusHandler{client: client}
}

// Tool returns the ironclaw_agent_status MCP tool definition.
func (h *AgentStatusHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_agent_status",
		mcp.WithDescription("Get the current IronClaw agent status including thread states, active/total job counts, and last heartbeat time."),
	)
}

// Handle executes the agent status tool.
func (h *AgentStatusHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.AgentStatus(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("agent status check failed: %v", err)), nil
	}
	return jsonResult(resp)
}
