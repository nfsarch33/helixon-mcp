package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/nfsarch33/helixon-mcp/internal/helixon"
)

// IronclawClient defines the interface the tool handlers need.
// This allows easy mocking in tests.
type IronclawClient interface {
	Health(ctx context.Context) (*helixon.HealthResponse, error)
	Chat(ctx context.Context, req helixon.ChatRequest) (*helixon.ChatResponse, error)
	ListJobs(ctx context.Context) (*helixon.JobsResponse, error)
	GetJob(ctx context.Context, jobID string) (*helixon.Job, error)
	CancelJob(ctx context.Context, jobID string) error
	SearchMemory(ctx context.Context, req helixon.MemorySearchRequest) (*helixon.MemorySearchResponse, error)
	WriteMemory(ctx context.Context, req helixon.MemoryWriteRequest) (*helixon.MemoryWriteResponse, error)
	ReadMemory(ctx context.Context, req helixon.MemoryReadRequest) (*helixon.MemoryReadResponse, error)
	TreeMemory(ctx context.Context, req helixon.MemoryTreeRequest) (*helixon.MemoryTreeResponse, error)
	ListRoutines(ctx context.Context) (*helixon.RoutinesResponse, error)
	DeleteRoutine(ctx context.Context, routineID string) error
	ListTools(ctx context.Context) (*helixon.ToolsResponse, error)
	StackStatus(ctx context.Context, routerURL string) (*helixon.StackStatusResponse, error)
	SpawnAgent(ctx context.Context, req helixon.SpawnAgentRequest) (*helixon.SpawnAgentResponse, error)
	SendTask(ctx context.Context, req helixon.SendTaskRequest) (*helixon.SendTaskResponse, error)
	AgentStatus(ctx context.Context) (*helixon.AgentStatusResponse, error)
}

// PrometheusQuerier queries Prometheus for metrics.
type PrometheusQuerier interface {
	Query(ctx context.Context, query string) (string, error)
}

// requiredString extracts a required string argument from a tool call request.
func requiredString(req mcp.CallToolRequest, key string) (string, error) {
	v, ok := req.Params.Arguments[key]
	if !ok {
		return "", fmt.Errorf("missing required argument %q", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %q must be a string, got %T", key, v)
	}
	if s == "" {
		return "", fmt.Errorf("argument %q must not be empty", key)
	}
	return s, nil
}

// optionalString extracts an optional string argument, returning "" if absent.
func optionalString(req mcp.CallToolRequest, key string) string {
	v, ok := req.Params.Arguments[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}
