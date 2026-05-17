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
		{"helixon_health", NewHealthHandler(client).Tool()},
		{"helixon_chat", NewChatHandler(client).Tool()},
		{"helixon_list_jobs", NewJobsHandler(client).ListJobsTool()},
		{"helixon_get_job", NewJobsHandler(client).GetJobTool()},
		{"helixon_cancel_job", NewJobsHandler(client).CancelJobTool()},
		{"helixon_search_memory", NewMemoryHandler(client).Tool()},
		{"memory_search", NewWorkspaceMemoryHandler(client).SearchTool()},
		{"memory_write", NewWorkspaceMemoryHandler(client).WriteTool()},
		{"memory_read", NewWorkspaceMemoryHandler(client).ReadTool()},
		{"memory_tree", NewWorkspaceMemoryHandler(client).TreeTool()},
		{"helixon_list_routines", NewRoutinesHandler(client).ListRoutinesTool()},
		{"helixon_delete_routine", NewRoutinesHandler(client).DeleteRoutineTool()},
		{"helixon_list_tools", NewToolsListHandler(client).Tool()},
		{"helixon_stack_status", NewStackStatusHandler(client).Tool()},
		{"helixon_spawn_agent", NewSpawnAgentHandler(client).Tool()},
		{"helixon_send_task", NewSendTaskHandler(client).Tool()},
		{"helixon_agent_status", NewAgentStatusHandler(client).Tool()},
		{"helixon_get_metrics", NewGetMetricsHandler(prom).Tool()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.name, tt.tool.Name)
			assert.NotEmpty(t, tt.tool.Description)
			assert.Equal(t, "object", tt.tool.InputSchema.Type)
		})
	}
}
