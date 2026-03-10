package ironclaw

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewClient(srv.URL, "", 5*time.Second)
	return client, srv
}

func TestHealth_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/health", r.URL.Path)
		json.NewEncoder(w).Encode(HealthResponse{Status: "ok", Version: "1.0.0"}) //nolint:errcheck
	}))

	resp, err := client.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
	assert.Equal(t, "1.0.0", resp.Version)
}

func TestHealth_ServerError(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	_, err := client.Health(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestChat_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var req ChatRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "hello", req.Message)
		json.NewEncoder(w).Encode(ChatResponse{Response: "hi there", JobID: "job-1"}) //nolint:errcheck
	}))

	resp, err := client.Chat(context.Background(), ChatRequest{Message: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "hi there", resp.Response)
	assert.Equal(t, "job-1", resp.JobID)
}

func TestChat_BearerToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))
		json.NewEncoder(w).Encode(ChatResponse{Response: "ok"}) //nolint:errcheck
	}))
	defer srv.Close()
	client := NewClient(srv.URL, "mytoken", 5*time.Second)
	_, err := client.Chat(context.Background(), ChatRequest{Message: "ping"})
	require.NoError(t, err)
}

func TestListJobs_OK(t *testing.T) {
	jobs := JobsResponse{Jobs: []Job{{ID: "j1", Status: "running"}, {ID: "j2", Status: "done"}}}
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/jobs", r.URL.Path)
		json.NewEncoder(w).Encode(jobs) //nolint:errcheck
	}))
	resp, err := client.ListJobs(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp.Jobs, 2)
	assert.Equal(t, "j1", resp.Jobs[0].ID)
}

func TestGetJob_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/jobs/job-42", r.URL.Path)
		json.NewEncoder(w).Encode(Job{ID: "job-42", Status: "done"}) //nolint:errcheck
	}))
	job, err := client.GetJob(context.Background(), "job-42")
	require.NoError(t, err)
	assert.Equal(t, "done", job.Status)
}

func TestCancelJob_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/jobs/job-42/cancel", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.CancelJob(context.Background(), "job-42")
	require.NoError(t, err)
}

func TestSearchMemory_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/memory/search", r.URL.Path)
		var req MemorySearchRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "golang tips", req.Query)
		json.NewEncoder(w).Encode(MemorySearchResponse{ //nolint:errcheck
			Entries: []MemoryEntry{{Path: "notes/go.md", Content: "use interfaces"}},
		})
	}))
	resp, err := client.SearchMemory(context.Background(), MemorySearchRequest{Query: "golang tips", Limit: 5})
	require.NoError(t, err)
	assert.Len(t, resp.Entries, 1)
	assert.Equal(t, "notes/go.md", resp.Entries[0].Path)
}

func TestListRoutines_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/routines", r.URL.Path)
		json.NewEncoder(w).Encode(RoutinesResponse{ //nolint:errcheck
			Routines: []Routine{{ID: "r1", Name: "daily-summary", Schedule: "0 9 * * *"}},
		})
	}))
	resp, err := client.ListRoutines(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp.Routines, 1)
	assert.Equal(t, "daily-summary", resp.Routines[0].Name)
}

func TestCreateRoutine_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/routines", r.URL.Path)
		json.NewEncoder(w).Encode(Routine{ID: "new-r", Name: "my-routine"}) //nolint:errcheck
	}))
	routine, err := client.CreateRoutine(context.Background(), CreateRoutineRequest{
		Name:     "my-routine",
		Schedule: "0 8 * * *",
		Prompt:   "summarise news",
	})
	require.NoError(t, err)
	assert.Equal(t, "new-r", routine.ID)
}

func TestDeleteRoutine_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/api/routines/r99", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	err := client.DeleteRoutine(context.Background(), "r99")
	require.NoError(t, err)
}

func TestListTools_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/tools", r.URL.Path)
		json.NewEncoder(w).Encode(ToolsResponse{ //nolint:errcheck
			Tools: []ToolInfo{{Name: "web_search", Description: "search the web"}},
		})
	}))
	resp, err := client.ListTools(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp.Tools, 1)
	assert.Equal(t, "web_search", resp.Tools[0].Name)
}

func TestClient_NotFound(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	_, err := client.Health(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}
