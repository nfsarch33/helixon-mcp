package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeGateway simulates IronClaw API endpoints for integration testing.
// Routes are registered via a map keyed on "METHOD /path".
type fakeGateway struct {
	routes map[string]http.HandlerFunc
}

func newFakeGateway() *fakeGateway {
	return &fakeGateway{routes: make(map[string]http.HandlerFunc)}
}

func (fg *fakeGateway) handle(method, path string, fn http.HandlerFunc) {
	fg.routes[method+" "+path] = fn
}

func (fg *fakeGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	if fn, ok := fg.routes[key]; ok {
		fn(w, r)
		return
	}
	http.NotFound(w, r)
}

func (fg *fakeGateway) start() *httptest.Server {
	return httptest.NewServer(fg)
}

func jsonResponse(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// setupGateway registers all standard IronClaw endpoints returning realistic responses.
func setupGateway() *fakeGateway {
	gw := newFakeGateway()

	gw.handle("GET", "/api/health", func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, ironclaw.HealthResponse{Status: "ok", Channel: "gateway"})
	})

	gw.handle("POST", "/api/chat/send", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		jsonResponse(w, map[string]any{
			"job_id":     "j-integration-1",
			"message_id": "m-integration-1",
			"status":     "accepted",
		})
	})

	gw.handle("GET", "/api/jobs", func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, ironclaw.JobsResponse{
			Jobs: []ironclaw.Job{
				{ID: "j1", State: "completed", Title: "test-job-1"},
				{ID: "j2", State: "in_progress", Title: "test-job-2"},
			},
		})
	})

	gw.handle("GET", "/api/jobs/j1", func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, ironclaw.Job{ID: "j1", State: "completed", Title: "test-job-1"})
	})

	gw.handle("POST", "/api/jobs/j99/cancel", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "{}")
	})

	gw.handle("POST", "/api/memory/search", func(w http.ResponseWriter, r *http.Request) {
		var body ironclaw.MemorySearchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		jsonResponse(w, ironclaw.MemorySearchResponse{
			Results: []ironclaw.MemoryEntry{
				{Path: "notes/go.md", Content: "Go patterns for " + body.Query, Score: 0.95},
			},
		})
	})

	gw.handle("GET", "/api/routines", func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, ironclaw.RoutinesResponse{
			Routines: []ironclaw.Routine{
				{ID: "r1", Name: "daily-report", Enabled: true, TriggerType: "cron"},
			},
		})
	})

	gw.handle("DELETE", "/api/routines/r99", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "{}")
	})

	gw.handle("GET", "/api/extensions/tools", func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, ironclaw.ToolsResponse{
			Tools: []ironclaw.ToolInfo{
				{Name: "web_fetch", Description: "Fetch a URL"},
				{Name: "shell", Description: "Execute shell command"},
			},
		})
	})

	gw.handle("GET", "/api/gateway/status", func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, ironclaw.GatewayStatusResponse{
			Status:      "ok",
			Uptime:      "24h",
			Connections: 3,
		})
	})

	return gw
}

// setupRouter registers the /healthz endpoint simulating llm-cluster-router.
func setupRouter() *fakeGateway {
	router := newFakeGateway()
	router.handle("GET", "/healthz", func(w http.ResponseWriter, _ *http.Request) {
		jsonResponse(w, ironclaw.RouterHealthResponse{
			OK:           true,
			HealthyNodes: 2,
			TotalNodes:   2,
			Nodes: []ironclaw.RouterNode{
				{Name: "gpu-27b", Tier: "agent", Healthy: true},
				{Name: "gpu-9b", Tier: "fast", Healthy: true},
			},
		})
	})
	return router
}

func newIntegrationClient(baseURL string) *ironclaw.Client {
	return ironclaw.NewClientWithHTTP(baseURL, "test-api-key", http.DefaultClient)
}

func extractJSON(t *testing.T, res *mcp.CallToolResult) map[string]any {
	t.Helper()
	require.NotEmpty(t, res.Content)
	text := res.Content[0].(mcp.TextContent).Text
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &out))
	return out
}

// --- Integration: Health ---

func TestIntegration_Health_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewHealthHandler(client)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)
	assert.Equal(t, "ok", out["status"])
	assert.Equal(t, "gateway", out["channel"])
}

func TestIntegration_Health_ServerDown(t *testing.T) {
	client := ironclaw.NewClientWithHTTP("http://127.0.0.1:1", "key", http.DefaultClient)
	h := NewHealthHandler(client)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestIntegration_Health_500(t *testing.T) {
	gw := newFakeGateway()
	gw.handle("GET", "/api/health", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewHealthHandler(client)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Integration: SendTask ---

func TestIntegration_SendTask_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewSendTaskHandler(client)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{
		"message": "deploy staging",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)
	assert.Equal(t, "j-integration-1", out["job_id"])
	assert.Equal(t, "accepted", out["status"])
}

func TestIntegration_SendTask_ContentFieldMapping(t *testing.T) {
	var receivedField string
	gw := newFakeGateway()
	gw.handle("POST", "/api/chat/send", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		if _, ok := body["content"]; ok {
			receivedField = "content"
		}
		if _, ok := body["message"]; ok {
			receivedField = "message"
		}
		jsonResponse(w, map[string]any{"job_id": "j1", "status": "accepted"})
	})
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewSendTaskHandler(client)
	_, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "test"}))
	require.NoError(t, err)
	assert.Equal(t, "content", receivedField, "SendTask must send 'content' field, not 'message'")
}

func TestIntegration_SendTask_MissingMessage(t *testing.T) {
	h := NewSendTaskHandler(nil)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestIntegration_SendTask_ServerError(t *testing.T) {
	gw := newFakeGateway()
	gw.handle("POST", "/api/chat/send", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gateway overloaded", http.StatusServiceUnavailable)
	})
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewSendTaskHandler(client)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"message": "deploy"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Integration: AgentStatus (composite endpoint) ---

func TestIntegration_AgentStatus_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewAgentStatusHandler(client)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)
	assert.Equal(t, "ok", out["status"])
	assert.Equal(t, float64(1), out["active_jobs"], "one in_progress job should count as active")
	assert.Equal(t, float64(2), out["total_jobs"])
}

func TestIntegration_AgentStatus_HealthDown(t *testing.T) {
	gw := newFakeGateway()
	gw.handle("GET", "/api/health", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	})
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewAgentStatusHandler(client)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Integration: ListJobs ---

func TestIntegration_ListJobs_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewJobsHandler(client)
	res, err := h.HandleListJobs(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)
	jobs, ok := out["jobs"].([]any)
	require.True(t, ok)
	assert.Len(t, jobs, 2)
}

func TestIntegration_GetJob_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewJobsHandler(client)
	res, err := h.HandleGetJob(context.Background(), makeReq(map[string]any{"job_id": "j1"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)
	assert.Equal(t, "j1", out["id"])
	assert.Equal(t, "completed", out["state"])
}

func TestIntegration_GetJob_NotFound(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewJobsHandler(client)
	res, err := h.HandleGetJob(context.Background(), makeReq(map[string]any{"job_id": "nonexistent"}))
	require.NoError(t, err)
	assert.True(t, res.IsError, "should fail for unregistered job ID")
}

func TestIntegration_CancelJob_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewJobsHandler(client)
	res, err := h.HandleCancelJob(context.Background(), makeReq(map[string]any{"job_id": "j99"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

// --- Integration: SearchMemory ---

func TestIntegration_SearchMemory_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewMemoryHandler(client)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"query": "integration"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)
	results, ok := out["results"].([]any)
	require.True(t, ok)
	assert.Len(t, results, 1)
	first := results[0].(map[string]any)
	assert.Contains(t, first["content"], "integration")
}

func TestIntegration_SearchMemory_WithLimit(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewMemoryHandler(client)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{"query": "test", "limit": "3"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

// --- Integration: Routines ---

func TestIntegration_ListRoutines_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewRoutinesHandler(client)
	res, err := h.HandleListRoutines(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)
	routines, ok := out["routines"].([]any)
	require.True(t, ok)
	assert.Len(t, routines, 1)
}

func TestIntegration_DeleteRoutine_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewRoutinesHandler(client)
	res, err := h.HandleDeleteRoutine(context.Background(), makeReq(map[string]any{"routine_id": "r99"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

// --- Integration: ListTools ---

func TestIntegration_ListTools_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewToolsListHandler(client)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)
	tools, ok := out["tools"].([]any)
	require.True(t, ok)
	assert.Len(t, tools, 2)
}

// --- Integration: StackStatus ---

func TestIntegration_StackStatus_Full(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	router := setupRouter()
	routerTS := router.start()
	defer routerTS.Close()

	client := newIntegrationClient(ts.URL)
	h := &StackStatusHandler{client: client, routerURL: routerTS.URL}
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)

	routerData, ok := out["router"].(map[string]any)
	require.True(t, ok, "response should include router data")
	assert.Equal(t, true, routerData["ok"])
	assert.Equal(t, float64(2), routerData["healthy_nodes"])

	gwData, ok := out["gateway"].(map[string]any)
	require.True(t, ok, "response should include gateway data")
	assert.Equal(t, "ok", gwData["status"])
}

func TestIntegration_StackStatus_RouterDown(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := &StackStatusHandler{client: client, routerURL: "http://127.0.0.1:1"}
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError, "StackStatus should degrade gracefully when router is down")
	out := extractJSON(t, res)
	assert.Nil(t, out["router"], "router field should be nil when router is unreachable")
	assert.NotNil(t, out["gateway"], "gateway should still be present")
}

// --- Integration: SpawnAgent ---

func TestIntegration_SpawnAgent_OK(t *testing.T) {
	gw := setupGateway()
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewSpawnAgentHandler(client)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{
		"name":  "night-auditor",
		"model": "qwen3.5-27b",
		"tier":  "agent",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	out := extractJSON(t, res)
	assert.NotEmpty(t, out["job_id"])
	assert.Equal(t, "qwen3.5-27b", out["model"])
}

func TestIntegration_SpawnAgent_MissingName(t *testing.T) {
	h := NewSpawnAgentHandler(nil)
	res, err := h.Handle(context.Background(), makeReq(map[string]any{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

// --- Integration: Bearer Auth ---

func TestIntegration_BearerAuthHeader(t *testing.T) {
	var receivedAuth string
	gw := newFakeGateway()
	gw.handle("GET", "/api/health", func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		jsonResponse(w, ironclaw.HealthResponse{Status: "ok"})
	})
	ts := gw.start()
	defer ts.Close()

	client := ironclaw.NewClientWithHTTP(ts.URL, "my-secret-token", http.DefaultClient)
	h := NewHealthHandler(client)
	_, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.Equal(t, "Bearer my-secret-token", receivedAuth)
}

func TestIntegration_NoAuthWhenKeyEmpty(t *testing.T) {
	var receivedAuth string
	gw := newFakeGateway()
	gw.handle("GET", "/api/health", func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		jsonResponse(w, ironclaw.HealthResponse{Status: "ok"})
	})
	ts := gw.start()
	defer ts.Close()

	client := ironclaw.NewClientWithHTTP(ts.URL, "", http.DefaultClient)
	h := NewHealthHandler(client)
	_, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.Empty(t, receivedAuth)
}

// --- Integration: Response Size Limit ---

func TestIntegration_LargeResponseTruncated(t *testing.T) {
	gw := newFakeGateway()
	gw.handle("GET", "/api/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","padding":"`))
		padding := make([]byte, 11<<20)
		for i := range padding {
			padding[i] = 'x'
		}
		_, _ = w.Write(padding)
		_, _ = w.Write([]byte(`"}`))
	})
	ts := gw.start()
	defer ts.Close()

	client := newIntegrationClient(ts.URL)
	h := NewHealthHandler(client)
	res, err := h.Handle(context.Background(), makeReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError, "should fail when response exceeds MaxResponseBytes")
}
