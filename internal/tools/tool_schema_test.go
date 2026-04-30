package tools

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestDefaultToolSchemas(t *testing.T) {
	client := new(MockIronclawClient)
	prom := new(MockPrometheusQuerier)

	tests := []struct {
		name string
		tool mcp.Tool
	}{
		{"ironclaw_health", NewHealthHandler(client).Tool()},
		{"ironclaw_chat", NewChatHandler(client).Tool()},
		{"ironclaw_list_jobs", NewJobsHandler(client).ListJobsTool()},
		{"ironclaw_get_job", NewJobsHandler(client).GetJobTool()},
		{"ironclaw_cancel_job", NewJobsHandler(client).CancelJobTool()},
		{"ironclaw_search_memory", NewMemoryHandler(client).Tool()},
		{"ironclaw_list_routines", NewRoutinesHandler(client).ListRoutinesTool()},
		{"ironclaw_delete_routine", NewRoutinesHandler(client).DeleteRoutineTool()},
		{"ironclaw_list_tools", NewToolsListHandler(client).Tool()},
		{"ironclaw_stack_status", NewStackStatusHandler(client).Tool()},
		{"ironclaw_spawn_agent", NewSpawnAgentHandler(client).Tool()},
		{"ironclaw_send_task", NewSendTaskHandler(client).Tool()},
		{"ironclaw_agent_status", NewAgentStatusHandler(client).Tool()},
		{"ironclaw_get_metrics", NewGetMetricsHandler(prom).Tool()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.name, tt.tool.Name)
			assert.NotEmpty(t, tt.tool.Description)
			assert.Equal(t, "object", tt.tool.InputSchema.Type)
		})
	}
}
