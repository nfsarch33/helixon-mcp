package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
)

// IronclawClient defines the interface the tool handlers need.
// This allows easy mocking in tests.
type IronclawClient interface {
	Health(ctx context.Context) (*ironclaw.HealthResponse, error)
	Chat(ctx context.Context, req ironclaw.ChatRequest) (*ironclaw.ChatResponse, error)
	ListJobs(ctx context.Context) (*ironclaw.JobsResponse, error)
	GetJob(ctx context.Context, jobID string) (*ironclaw.Job, error)
	CancelJob(ctx context.Context, jobID string) error
	SearchMemory(ctx context.Context, req ironclaw.MemorySearchRequest) (*ironclaw.MemorySearchResponse, error)
	ListRoutines(ctx context.Context) (*ironclaw.RoutinesResponse, error)
	CreateRoutine(ctx context.Context, req ironclaw.CreateRoutineRequest) (*ironclaw.Routine, error)
	DeleteRoutine(ctx context.Context, routineID string) error
	ListTools(ctx context.Context) (*ironclaw.ToolsResponse, error)
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
