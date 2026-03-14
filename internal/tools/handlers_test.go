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
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

// --- Health ---

func TestHealthHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("Health", context.Background()).Return(&ironclaw.HealthResponse{Status: "ok", Version: "1.0"}, nil)
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
		Return(&ironclaw.ChatResponse{Response: "hi", JobID: "j1"}, nil)
	h := NewChatHandler(m)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "hello"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out))
	assert.Equal(t, "hi", out["response"])
	assert.Equal(t, "j1", out["job_id"])
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
		Return(&ironclaw.ChatResponse{Response: "hello", SessionID: "sess-1"}, nil)
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

// --- Jobs ---

func TestJobsHandler_ListJobs_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListJobs", context.Background()).
		Return(&ironclaw.JobsResponse{Jobs: []ironclaw.Job{{ID: "j1", Status: "running"}}}, nil)
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
	m.On("GetJob", context.Background(), "j42").Return(&ironclaw.Job{ID: "j42", Status: "done"}, nil)
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
		Return(&ironclaw.MemorySearchResponse{Entries: []ironclaw.MemoryEntry{{Path: "go.md", Content: "tips"}}}, nil)
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

func TestRoutinesHandler_CreateRoutine_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("CreateRoutine", context.Background(), ironclaw.CreateRoutineRequest{
		Name: "morning", Schedule: "0 9 * * *", Prompt: "news",
	}).Return(&ironclaw.Routine{ID: "r2", Name: "morning"}, nil)
	h := NewRoutinesHandler(m)
	res, err := h.HandleCreateRoutine(context.Background(), makeReq(map[string]any{
		"name": "morning", "schedule": "0 9 * * *", "prompt": "news",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestRoutinesHandler_CreateRoutine_MissingField(t *testing.T) {
	h := NewRoutinesHandler(new(MockIronclawClient))
	res, err := h.HandleCreateRoutine(context.Background(), makeReq(map[string]any{"name": "morning"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
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

func TestToolsListHandler_Error(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("ListTools", context.Background()).Return(&ironclaw.ToolsResponse{}, errors.New("unavailable"))
	h := NewToolsListHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- SendTask ---

func TestSendTaskHandler_OK(t *testing.T) {
	m := new(MockIronclawClient)
	m.On("SendTask", context.Background(), ironclaw.SendTaskRequest{Message: "deploy service", SessionID: ""}).
		Return(&ironclaw.SendTaskResponse{JobID: "j100", Status: "accepted"}, nil)
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
	m.On("SendTask", context.Background(), ironclaw.SendTaskRequest{Message: "review PR", SessionID: "s-42"}).
		Return(&ironclaw.SendTaskResponse{JobID: "j101", SessionID: "s-42", Status: "accepted"}, nil)
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
	m.On("SendTask", context.Background(), ironclaw.SendTaskRequest{Message: "deploy", SessionID: ""}).
		Return(&ironclaw.SendTaskResponse{}, errors.New("connection refused"))
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
	m.On("AgentStatus", context.Background()).Return(&ironclaw.AgentStatusResponse{
		Status:        "running",
		ActiveJobs:    3,
		TotalJobs:     42,
		LastHeartbeat: "2026-03-10T10:00:00Z",
		Threads: []ironclaw.ThreadStatus{
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
	m.On("AgentStatus", context.Background()).Return(&ironclaw.AgentStatusResponse{}, errors.New("connection refused"))
	h := NewAgentStatusHandler(m)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}
