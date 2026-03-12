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
		assert.Equal(t, "/api/health", r.URL.Path)
		json.NewEncoder(w).Encode(HealthResponse{Status: "healthy", Channel: "gateway"}) //nolint:errcheck
	}))

	resp, err := client.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp.Status)
	assert.Equal(t, "gateway", resp.Channel)
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
	const threadID = "00000000-0000-0000-0000-000000000001"
	const messageID = "00000000-0000-0000-0000-000000000010"
	historyCalls := 0
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/threads":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"assistant_thread": map[string]any{"id": threadID},
				"threads":          []any{},
				"active_thread":    threadID,
			}))
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/history":
			assert.Equal(t, threadID, r.URL.Query().Get("thread_id"))
			historyCalls++
			if historyCalls == 1 {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"thread_id": threadID,
					"turns":     []any{},
					"has_more":  false,
				}))
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"thread_id": threadID,
				"turns": []any{
					map[string]any{
						"turn_number": 1,
						"user_input":  "hello",
						"response":    "hi there",
						"state":       "Completed",
						"started_at":  time.Now().UTC().Format(time.RFC3339),
						"tool_calls":  []any{},
					},
				},
				"has_more": false,
			}))
		case r.Method == http.MethodPost && r.URL.Path == "/api/chat/send":
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			var req map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "hello", req["content"])
			assert.Equal(t, threadID, req["thread_id"])
			w.WriteHeader(http.StatusAccepted)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"message_id": messageID,
				"status":     "accepted",
			}))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	resp, err := client.Chat(context.Background(), ChatRequest{Message: "hello"})
	require.NoError(t, err)
	assert.Equal(t, "hi there", resp.Response)
	assert.Equal(t, messageID, resp.MessageID)
	assert.Equal(t, threadID, resp.SessionID)
	assert.Equal(t, "completed", resp.Status)
}

func TestChat_BearerToken(t *testing.T) {
	historyCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mytoken", r.Header.Get("Authorization"))
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/threads":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"assistant_thread": map[string]any{"id": "00000000-0000-0000-0000-000000000001"},
				"threads":          []any{},
				"active_thread":    "00000000-0000-0000-0000-000000000001",
			}))
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/history":
			historyCalls++
			if historyCalls == 1 {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"thread_id": "00000000-0000-0000-0000-000000000001",
					"turns":     []any{},
					"has_more":  false,
				}))
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"thread_id": "00000000-0000-0000-0000-000000000001",
				"turns": []any{
					map[string]any{
						"turn_number": 1,
						"user_input":  "ping",
						"response":    "ok",
						"state":       "Completed",
						"started_at":  time.Now().UTC().Format(time.RFC3339),
						"tool_calls":  []any{},
					},
				},
				"has_more": false,
			}))
		case r.Method == http.MethodPost && r.URL.Path == "/api/chat/send":
			w.WriteHeader(http.StatusAccepted)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"message_id": "00000000-0000-0000-0000-000000000010",
				"status":     "accepted",
			}))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	client := NewClient(srv.URL, "mytoken", 5*time.Second)
	_, err := client.Chat(context.Background(), ChatRequest{Message: "ping"})
	require.NoError(t, err)
}

func TestChat_FailedTurnReturnsBackendError(t *testing.T) {
	const threadID = "00000000-0000-0000-0000-000000000001"

	historyCalls := 0
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/threads":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"assistant_thread": map[string]any{"id": threadID},
				"threads":          []any{},
				"active_thread":    threadID,
			}))
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/history":
			historyCalls++
			if historyCalls == 1 {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"thread_id": threadID,
					"turns":     []any{},
					"has_more":  false,
				}))
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"thread_id": threadID,
				"turns": []any{
					map[string]any{
						"turn_number": 1,
						"user_input":  "hello",
						"response":    nil,
						"state":       "Failed",
						"started_at":  time.Now().UTC().Format(time.RFC3339),
						"tool_calls": []any{
							map[string]any{
								"name":      "chat_completions",
								"has_error": true,
								"error":     "OpenAIToolParser requires token IDs and does not support text-based extraction.",
							},
						},
					},
				},
				"has_more": false,
			}))
		case r.Method == http.MethodPost && r.URL.Path == "/api/chat/send":
			w.WriteHeader(http.StatusAccepted)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"message_id": "00000000-0000-0000-0000-000000000010",
				"status":     "accepted",
			}))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	_, err := client.Chat(context.Background(), ChatRequest{Message: "hello"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OpenAIToolParser requires token IDs")
}

func TestChat_UsesProvidedSessionID(t *testing.T) {
	const threadID = "00000000-0000-0000-0000-000000000099"

	historyCalls := 0
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/threads":
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/history":
			assert.Equal(t, threadID, r.URL.Query().Get("thread_id"))
			historyCalls++
			if historyCalls == 1 {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"thread_id": threadID,
					"turns":     []any{},
					"has_more":  false,
				}))
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"thread_id": threadID,
				"turns": []any{
					map[string]any{
						"turn_number": 1,
						"user_input":  "resume this thread",
						"response":    "thread reused",
						"state":       "Completed",
						"started_at":  time.Now().UTC().Format(time.RFC3339),
						"tool_calls":  []any{},
					},
				},
				"has_more": false,
			}))
		case r.Method == http.MethodPost && r.URL.Path == "/api/chat/send":
			var req map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			assert.Equal(t, "resume this thread", req["content"])
			assert.Equal(t, threadID, req["thread_id"])
			w.WriteHeader(http.StatusAccepted)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"message_id": "00000000-0000-0000-0000-000000000010",
				"status":     "accepted",
			}))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	resp, err := client.Chat(context.Background(), ChatRequest{
		Message:   "resume this thread",
		SessionID: threadID,
	})
	require.NoError(t, err)
	assert.Equal(t, "thread reused", resp.Response)
	assert.Equal(t, threadID, resp.SessionID)
}

func TestChat_CancelledTurnReturnsTerminalError(t *testing.T) {
	const threadID = "00000000-0000-0000-0000-000000000001"

	historyCalls := 0
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/threads":
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"assistant_thread": map[string]any{"id": threadID},
				"threads":          []any{},
				"active_thread":    threadID,
			}))
		case r.Method == http.MethodGet && r.URL.Path == "/api/chat/history":
			historyCalls++
			if historyCalls == 1 {
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"thread_id": threadID,
					"turns":     []any{},
					"has_more":  false,
				}))
				return
			}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"thread_id": threadID,
				"turns": []any{
					map[string]any{
						"turn_number": 1,
						"user_input":  "hello",
						"response":    nil,
						"state":       "Cancelled",
						"started_at":  time.Now().UTC().Format(time.RFC3339),
						"tool_calls":  []any{},
					},
				},
				"has_more": false,
			}))
		case r.Method == http.MethodPost && r.URL.Path == "/api/chat/send":
			w.WriteHeader(http.StatusAccepted)
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
				"message_id": "00000000-0000-0000-0000-000000000010",
				"status":     "accepted",
			}))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	_, err := client.Chat(context.Background(), ChatRequest{Message: "hello"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `chat turn entered terminal state "Cancelled"`)
}

func TestListJobs_OK(t *testing.T) {
	jobs := JobsResponse{Jobs: []Job{{ID: "j1", State: "in_progress"}, {ID: "j2", State: "completed"}}}
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
		json.NewEncoder(w).Encode(Job{ID: "job-42", State: "completed"}) //nolint:errcheck
	}))
	job, err := client.GetJob(context.Background(), "job-42")
	require.NoError(t, err)
	assert.Equal(t, "completed", job.State)
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
			Results: []MemoryEntry{{Path: "notes/go.md", Content: "use interfaces"}},
		})
	}))
	resp, err := client.SearchMemory(context.Background(), MemorySearchRequest{Query: "golang tips", Limit: 5})
	require.NoError(t, err)
	assert.Len(t, resp.Results, 1)
	assert.Equal(t, "notes/go.md", resp.Results[0].Path)
}

func TestListRoutines_OK(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/routines", r.URL.Path)
		json.NewEncoder(w).Encode(RoutinesResponse{ //nolint:errcheck
			Routines: []Routine{{ID: "r1", Name: "daily-summary", Description: "Daily summary", Status: "active"}},
		})
	}))
	resp, err := client.ListRoutines(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp.Routines, 1)
	assert.Equal(t, "daily-summary", resp.Routines[0].Name)
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
		assert.Equal(t, "/api/extensions/tools", r.URL.Path)
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

func TestClient_ContextCancellation(t *testing.T) {
	client, _ := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.Health(ctx)
	require.Error(t, err)
}
