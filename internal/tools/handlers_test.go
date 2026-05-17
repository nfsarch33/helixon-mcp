package tools

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nfsarch33/helixon-mcp/internal/helixon"
)

func makeReq(args map[string]any) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Arguments = args
	return req
}

// --- Health ---

func TestHealthHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Health", context.Background()).Return(&helixon.HealthResponse{Status: "ok", Channel: "gateway"}, nil)
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
	m.On("Health", context.Background()).Return(&helixon.HealthResponse{}, errors.New("connection refused"))
	h := NewHealthHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Chat ---

func TestChatHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Chat", context.Background(), helixon.ChatRequest{Message: "hello", SessionID: ""}).
		Return(&helixon.ChatResponse{Response: "hi", MessageID: "m1", Status: "completed"}, nil)
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
	m.On("Chat", context.Background(), helixon.ChatRequest{Message: "hi", SessionID: "sess-1"}).
		Return(&helixon.ChatResponse{Response: "hello", SessionID: "sess-1", Status: "completed"}, nil)
	h := NewChatHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "hi", "session_id": "sess-1"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestChatHandler_ClientError(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Chat", context.Background(), helixon.ChatRequest{Message: "hello", SessionID: ""}).
		Return(&helixon.ChatResponse{}, errors.New("timeout"))
	h := NewChatHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "hello"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestChatHandler_BackendFailureDetail(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Chat", context.Background(), helixon.ChatRequest{Message: "hello", SessionID: ""}).
		Return(&helixon.ChatResponse{}, errors.New("backend turn failed: OpenAIToolParser requires token IDs"))
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
		Return(&helixon.JobsResponse{Jobs: []helixon.Job{{ID: "j1", State: "in_progress"}}}, nil)
	h := NewJobsHandler(m)
	res, err := h.HandleListJobs(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestJobsHandler_ListJobs_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListJobs", context.Background()).Return(&helixon.JobsResponse{}, errors.New("db error"))
	h := NewJobsHandler(m)
	res, err := h.HandleListJobs(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestJobsHandler_GetJob_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("GetJob", context.Background(), "j42").Return(&helixon.Job{ID: "j42", State: "completed"}, nil)
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
	m.On("SearchMemory", context.Background(), helixon.MemorySearchRequest{Query: "golang", Limit: 10}).
		Return(&helixon.MemorySearchResponse{Results: []helixon.MemoryEntry{{Path: "go.md", Content: "tips"}}}, nil)
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
	m.On("SearchMemory", context.Background(), helixon.MemorySearchRequest{Query: "tasks", Limit: 5}).
		Return(&helixon.MemorySearchResponse{}, nil)
	h := NewMemoryHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"query": "tasks", "limit": "5"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

// --- Routines ---

func TestRoutinesHandler_ListRoutines_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListRoutines", context.Background()).
		Return(&helixon.RoutinesResponse{Routines: []helixon.Routine{{ID: "r1", Name: "daily"}}}, nil)
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
		Return(&helixon.ToolsResponse{Tools: []helixon.ToolInfo{{Name: "search", Description: "web"}}}, nil)
	h := NewToolsListHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestToolsListHandler_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListTools", context.Background()).Return(&helixon.ToolsResponse{}, errors.New("unavailable"))
	h := NewToolsListHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Stack Status ---

func TestStackStatusHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("StackStatus", context.Background(), "http://127.0.0.1:8080").
		Return(&helixon.StackStatusResponse{
			Router: &helixon.RouterHealthResponse{
				OK:           true,
				HealthyNodes: 2,
				TotalNodes:   2,
				Nodes: []helixon.RouterNode{
					{Name: "example-agent", Tier: "agent", Healthy: true},
					{Name: "example-fast", Tier: "fast", Healthy: true},
				},
			},
			Gateway: &helixon.GatewayStatusResponse{
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
	assert.Contains(t, text, "example-agent")
	assert.Contains(t, text, "ok")
}

func TestStackStatusHandler_RouterOnly(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("StackStatus", context.Background(), "http://127.0.0.1:8080").
		Return(&helixon.StackStatusResponse{
			Router: &helixon.RouterHealthResponse{OK: true, HealthyNodes: 1},
		}, nil)
	h := &StackStatusHandler{client: m, routerURL: "http://127.0.0.1:8080"}
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestStackStatusHandler_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("StackStatus", context.Background(), "http://127.0.0.1:8080").
		Return(&helixon.StackStatusResponse{}, errors.New("connection refused"))
	h := &StackStatusHandler{client: m, routerURL: "http://127.0.0.1:8080"}
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Spawn Agent ---

func TestSpawnAgentHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("SpawnAgent", context.Background(), helixon.SpawnAgentRequest{Name: "worker", Model: "example-model", Tier: "agent"}).
		Return(&helixon.SpawnAgentResponse{JobID: "j42", Status: "accepted", Model: "example-model"}, nil)
	h := NewSpawnAgentHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"name": "worker", "model": "example-model", "tier": "agent"}))
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
	m.On("SpawnAgent", context.Background(), helixon.SpawnAgentRequest{Name: "test"}).
		Return(&helixon.SpawnAgentResponse{}, errors.New("gateway unreachable"))
	h := NewSpawnAgentHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"name": "test"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- SendTask ---

func TestSendTaskHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("SendTask", context.Background(), helixon.SendTaskRequest{Message: "deploy service", SessionID: ""}).
		Return(&helixon.SendTaskResponse{JobID: "j100", Status: "accepted"}, nil)
	h := NewSendTaskHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "deploy service"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out))
	assert.Equal(t, "j100", out["job_id"])
	assert.Equal(t, "accepted", out["status"])
}

func TestSendTaskHandler_WithSession(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("SendTask", context.Background(), helixon.SendTaskRequest{Message: "review PR", SessionID: "s-42"}).
		Return(&helixon.SendTaskResponse{JobID: "j101", SessionID: "s-42", Status: "accepted"}, nil)
	h := NewSendTaskHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "review PR", "session_id": "s-42"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestSendTaskHandler_MissingMessage(t *testing.T) {
	h := NewSendTaskHandler(new(MockIronclawClient))
	res, err := h.Handle(context.Background(), makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestSendTaskHandler_ClientError(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("SendTask", context.Background(), helixon.SendTaskRequest{Message: "deploy", SessionID: ""}).
		Return(&helixon.SendTaskResponse{}, errors.New("connection refused"))
	h := NewSendTaskHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "deploy"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- GetMetrics ---

func TestGetMetricsHandler_DefaultQueries(t *testing.T) {
	p := new(MockPrometheusQuerier)
	for _, query := range defaultMetricQueries {
		p.On("Query", context.Background(), query).Return(`{"status":"success"}`, nil)
	}
	h := NewGetMetricsHandler(p)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out))
	metrics, ok := out["metrics"].(map[string]any)
	require.True(t, ok)
	assert.Len(t, metrics, len(defaultMetricQueries))
}

func TestGetMetricsHandler_CustomQuery(t *testing.T) {
	p := new(MockPrometheusQuerier)
	p.On("Query", context.Background(), "up{job=\"test\"}").Return(`{"status":"success","data":[]}`, nil)
	h := NewGetMetricsHandler(p)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"query": "up{job=\"test\"}"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out))
	assert.Equal(t, "up{job=\"test\"}", out["query"])
}

func TestGetMetricsHandler_QueryError(t *testing.T) {
	p := new(MockPrometheusQuerier)
	p.On("Query", context.Background(), "bad_query").Return("", errors.New("bad query"))
	h := NewGetMetricsHandler(p)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"query": "bad_query"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestGetMetricsHandler_PartialFailure(t *testing.T) {
	p := new(MockPrometheusQuerier)
	firstCall := true
	for _, query := range defaultMetricQueries {
		if firstCall {
			p.On("Query", context.Background(), query).Return("", errors.New("timeout")).Once()
			firstCall = false
		} else {
			p.On("Query", context.Background(), query).Return(`{"status":"success"}`, nil)
		}
	}
	h := NewGetMetricsHandler(p)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out))
	assert.Contains(t, out, "errors")
}

// --- AgentStatus ---

func TestAgentStatusHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("AgentStatus", context.Background()).Return(&helixon.AgentStatusResponse{
		Status:        "running",
		ActiveJobs:    3,
		TotalJobs:     42,
		LastHeartbeat: "2026-03-10T10:00:00Z",
		Threads: []helixon.ThreadStatus{
			{ID: "t1", State: "idle"},
			{ID: "t2", State: "busy", JobID: "j50"},
		},
	}, nil)
	h := NewAgentStatusHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out))
	assert.Equal(t, "running", out["status"])
	assert.Equal(t, float64(3), out["active_jobs"])
	assert.Equal(t, float64(42), out["total_jobs"])
}

func TestAgentStatusHandler_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("AgentStatus", context.Background()).Return(&helixon.AgentStatusResponse{}, errors.New("connection refused"))
	h := NewAgentStatusHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}
