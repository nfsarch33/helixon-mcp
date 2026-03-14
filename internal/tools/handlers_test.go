package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeReq(args map[string]any) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Arguments = args
	return req
}

// --- Health ---

func TestHealthHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Health", context.Background()).Return(&ironclaw.HealthResponse{Status: "ok", Channel: "gateway"}, nil)
	h := NewHealthHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out))
	assert.Equal(t, "ok", out["status"])
}

func TestHealthHandler_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Health", context.Background()).Return(&ironclaw.HealthResponse{}, errors.New("connection refused"))
	h := NewHealthHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Chat ---

func TestChatHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Chat", context.Background(), ironclaw.ChatRequest{Message: "hello", SessionID: ""}).
		Return(&ironclaw.ChatResponse{Response: "hi", MessageID: "m1", Status: "completed"}, nil)
	h := NewChatHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "hello"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out))
	assert.Equal(t, "hi", out["response"])
	assert.Equal(t, "m1", out["message_id"])
	assert.Equal(t, "completed", out["status"])
}

func TestChatHandler_MissingMessage(t *testing.T) {
	h := NewChatHandler(new(MockIronclawClient))
	res, err := h.Handle(context.Background(), makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestChatHandler_EmptyMessage(t *testing.T) {
	h := NewChatHandler(new(MockIronclawClient))
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": ""}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestChatHandler_WithSession(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Chat", context.Background(), ironclaw.ChatRequest{Message: "hi", SessionID: "sess-1"}).
		Return(&ironclaw.ChatResponse{Response: "hello", SessionID: "sess-1", Status: "completed"}, nil)
	h := NewChatHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "hi", "session_id": "sess-1"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestChatHandler_ClientError(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Chat", context.Background(), ironclaw.ChatRequest{Message: "hello", SessionID: ""}).
		Return(&ironclaw.ChatResponse{}, errors.New("timeout"))
	h := NewChatHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "hello"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestChatHandler_BackendFailureDetail(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Chat", context.Background(), ironclaw.ChatRequest{Message: "hello", SessionID: ""}).
		Return(&ironclaw.ChatResponse{}, errors.New("backend turn failed: OpenAIToolParser requires token IDs"))
	h := NewChatHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "hello"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, "OpenAIToolParser requires token IDs")
}

// --- Jobs ---

func TestJobsHandler_ListJobs_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListJobs", context.Background()).
		Return(&ironclaw.JobsResponse{Jobs: []ironclaw.Job{{ID: "j1", State: "in_progress"}}}, nil)
	h := NewJobsHandler(m)
	res, err := h.HandleListJobs(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestJobsHandler_ListJobs_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListJobs", context.Background()).Return(&ironclaw.JobsResponse{}, errors.New("db error"))
	h := NewJobsHandler(m)
	res, err := h.HandleListJobs(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestJobsHandler_GetJob_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("GetJob", context.Background(), "j42").Return(&ironclaw.Job{ID: "j42", State: "completed"}, nil)
	h := NewJobsHandler(m)
	res, err := h.HandleGetJob(context.Background(), makeReq(map[string]any{"job_id": "j42"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestJobsHandler_GetJob_MissingID(t *testing.T) {
	h := NewJobsHandler(new(MockIronclawClient))
	res, err := h.HandleGetJob(context.Background(), makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestJobsHandler_CancelJob_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("CancelJob", context.Background(), "j99").Return(nil)
	h := NewJobsHandler(m)
	res, err := h.HandleCancelJob(context.Background(), makeReq(map[string]any{"job_id": "j99"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestJobsHandler_CancelJob_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("CancelJob", context.Background(), "j99").Return(errors.New("not found"))
	h := NewJobsHandler(m)
	res, err := h.HandleCancelJob(context.Background(), makeReq(map[string]any{"job_id": "j99"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Memory ---

func TestMemoryHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("SearchMemory", context.Background(), ironclaw.MemorySearchRequest{Query: "golang", Limit: 10}).
		Return(&ironclaw.MemorySearchResponse{Results: []ironclaw.MemoryEntry{{Path: "go.md", Content: "tips"}}}, nil)
	h := NewMemoryHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"query": "golang"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestMemoryHandler_MissingQuery(t *testing.T) {
	h := NewMemoryHandler(new(MockIronclawClient))
	res, err := h.Handle(context.Background(), makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestMemoryHandler_CustomLimit(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("SearchMemory", context.Background(), ironclaw.MemorySearchRequest{Query: "tasks", Limit: 5}).
		Return(&ironclaw.MemorySearchResponse{}, nil)
	h := NewMemoryHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"query": "tasks", "limit": "5"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

// --- Routines ---

func TestRoutinesHandler_ListRoutines_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListRoutines", context.Background()).
		Return(&ironclaw.RoutinesResponse{Routines: []ironclaw.Routine{{ID: "r1", Name: "daily"}}}, nil)
	h := NewRoutinesHandler(m)
	res, err := h.HandleListRoutines(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestRoutinesHandler_DeleteRoutine_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("DeleteRoutine", context.Background(), "r99").Return(nil)
	h := NewRoutinesHandler(m)
	res, err := h.HandleDeleteRoutine(context.Background(), makeReq(map[string]any{"routine_id": "r99"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

// --- Tools List ---

func TestToolsListHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListTools", context.Background()).
		Return(&ironclaw.ToolsResponse{Tools: []ironclaw.ToolInfo{{Name: "search", Description: "web"}}}, nil)
	h := NewToolsListHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestReviewedPushHandler_ReviewOnlyPass(t *testing.T) {
	h := &ReviewedPushHandler{
		run: func(_ context.Context, _ string, _ string, name string, args ...string) (string, error) {
			switch name {
			case "git":
				if len(args) >= 2 && args[0] == "branch" && args[1] == "--show-current" {
					return "feature/test\n", nil
				}
				return "diff --git a/a.go b/a.go\n+package main\n", nil
			case "gemini":
				return `{"verdict":"pass","must_fix":[],"should_fix":[],"nits":[]}`, nil
			default:
				return "", errors.New("unexpected command")
			}
		},
	}

	res, err := h.Handle(context.Background(), makeReq(map[string]any{
		"workdir":     "/tmp/repo",
		"review_only": "true",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, `"allowed": true`)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, `"pushed": false`)
}

func TestReviewedPushHandler_BlocksMustFix(t *testing.T) {
	h := &ReviewedPushHandler{
		run: func(_ context.Context, _ string, _ string, name string, args ...string) (string, error) {
			switch name {
			case "git":
				if len(args) >= 2 && args[0] == "branch" && args[1] == "--show-current" {
					return "feature/test\n", nil
				}
				return "diff --git a/main.rs b/main.rs\n+.unwrap()\n", nil
			case "gemini":
				return `{"verdict":"fail","must_fix":[{"issue":"panic-prone unwrap","file":"main.rs","line":10}],"should_fix":[],"nits":[]}`, nil
			default:
				return "", errors.New("unexpected command")
			}
		},
	}

	res, err := h.Handle(context.Background(), makeReq(map[string]any{"workdir": "/tmp/repo"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, `"allowed": false`)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, `panic-prone unwrap`)
}

func TestReviewedPushHandler_PushesOnPass(t *testing.T) {
	var pushCalled bool
	h := &ReviewedPushHandler{
		run: func(_ context.Context, _ string, _ string, name string, args ...string) (string, error) {
			switch name {
			case "git":
				if len(args) >= 2 && args[0] == "branch" && args[1] == "--show-current" {
					return "feature/test\n", nil
				}
				if len(args) >= 1 && args[0] == "push" {
					pushCalled = true
					return "pushed", nil
				}
				return "diff --git a/a.go b/a.go\n+package main\n", nil
			case "gemini":
				return `{"verdict":"pass","must_fix":[],"should_fix":[{"issue":"add test","file":"a.go","line":1}],"nits":[]}`, nil
			default:
				return "", errors.New("unexpected command")
			}
		},
	}

	res, err := h.Handle(context.Background(), makeReq(map[string]any{"workdir": "/tmp/repo"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.True(t, pushCalled)
	assert.Contains(t, res.Content[0].(mcp.TextContent).Text, `"pushed": true`)
}

func TestToolsListHandler_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListTools", context.Background()).Return(&ironclaw.ToolsResponse{}, errors.New("unavailable"))
	h := NewToolsListHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Stack Status ---

func TestStackStatusHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("StackStatus", context.Background(), "http://127.0.0.1:8080").
		Return(&ironclaw.StackStatusResponse{
			Router: &ironclaw.RouterHealthResponse{
				OK:           true,
				HealthyNodes: 2,
				TotalNodes:   2,
				Nodes: []ironclaw.RouterNode{
					{Name: "gpu-27b", Tier: "agent", Healthy: true},
					{Name: "gpu-9b", Tier: "fast", Healthy: true},
				},
			},
			Gateway: &ironclaw.GatewayStatusResponse{
				Status:      "ok",
				Uptime:      "12h",
				Connections: 5,
			},
		}, nil)
	h := &StackStatusHandler{client: m, routerURL: "http://127.0.0.1:8080"}
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	text := res.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "gpu-27b")
	assert.Contains(t, text, "ok")
}

func TestStackStatusHandler_RouterOnly(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("StackStatus", context.Background(), "http://127.0.0.1:8080").
		Return(&ironclaw.StackStatusResponse{
			Router: &ironclaw.RouterHealthResponse{OK: true, HealthyNodes: 1},
		}, nil)
	h := &StackStatusHandler{client: m, routerURL: "http://127.0.0.1:8080"}
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestStackStatusHandler_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("StackStatus", context.Background(), "http://127.0.0.1:8080").
		Return(&ironclaw.StackStatusResponse{}, errors.New("connection refused"))
	h := &StackStatusHandler{client: m, routerURL: "http://127.0.0.1:8080"}
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Spawn Agent ---

func TestSpawnAgentHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("SpawnAgent", context.Background(), ironclaw.SpawnAgentRequest{Name: "auditor", Model: "qwen3.5-27b", Tier: "agent"}).
		Return(&ironclaw.SpawnAgentResponse{JobID: "j42", Status: "accepted", Model: "qwen3.5-27b"}, nil)
	h := NewSpawnAgentHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"name": "auditor", "model": "qwen3.5-27b", "tier": "agent"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	text := res.Content[0].(mcp.TextContent).Text
	assert.Contains(t, text, "j42")
}

func TestSpawnAgentHandler_MissingName(t *testing.T) {
	h := NewSpawnAgentHandler(new(MockIronclawClient))
	res, err := h.Handle(context.Background(), makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestSpawnAgentHandler_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("SpawnAgent", context.Background(), ironclaw.SpawnAgentRequest{Name: "test"}).
		Return(&ironclaw.SpawnAgentResponse{}, errors.New("gateway unreachable"))
	h := NewSpawnAgentHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"name": "test"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}
