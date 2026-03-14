// Package ironclaw provides an HTTP client for the IronClaw REST API.
package ironclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the IronClaw web gateway API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient HTTPDoer
}

// HTTPDoer is the interface for making HTTP requests (allows mocking in tests).
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewClient constructs a Client with the given base URL and optional API key.
func NewClient(baseURL, apiKey string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// NewClientWithHTTP constructs a Client with a custom HTTPDoer (for testing).
func NewClientWithHTTP(baseURL, apiKey string, doer HTTPDoer) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: doer,
	}
}

// --- Request / Response types -----------------------------------------------

// ChatRequest is the payload for POST /api/chat.
type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

// ChatResponse is returned by POST /api/chat.
type ChatResponse struct {
	Response  string `json:"response"`
	JobID     string `json:"job_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// Job represents a background job in IronClaw.
type Job struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	Error     string `json:"error,omitempty"`
}

// JobsResponse is returned by GET /api/jobs.
type JobsResponse struct {
	Jobs []Job `json:"jobs"`
}

// MemorySearchRequest is the payload for POST /api/memory/search.
type MemorySearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// MemoryEntry is a single memory/workspace entry.
type MemoryEntry struct {
	Path    string  `json:"path"`
	Content string  `json:"content"`
	Score   float64 `json:"score,omitempty"`
}

// MemorySearchResponse is returned by POST /api/memory/search.
type MemorySearchResponse struct {
	Entries []MemoryEntry `json:"entries"`
}

// Routine represents an IronClaw scheduled routine.
type Routine struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Enabled  bool   `json:"enabled"`
	LastRun  string `json:"last_run,omitempty"`
}

// RoutinesResponse is returned by GET /api/routines.
type RoutinesResponse struct {
	Routines []Routine `json:"routines"`
}

// CreateRoutineRequest is the payload for POST /api/routines.
type CreateRoutineRequest struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Prompt   string `json:"prompt"`
}

// ToolInfo represents a registered IronClaw tool.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
}

// ToolsResponse is returned by GET /api/tools.
type ToolsResponse struct {
	Tools []ToolInfo `json:"tools"`
}

// SendTaskRequest is the payload for POST /api/chat/send.
type SendTaskRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

// SendTaskResponse is returned by POST /api/chat/send.
type SendTaskResponse struct {
	JobID     string `json:"job_id"`
	SessionID string `json:"session_id,omitempty"`
	Status    string `json:"status"`
}

// AgentStatusResponse is returned by GET /api/status.
type AgentStatusResponse struct {
	Status        string         `json:"status"`
	ActiveJobs    int            `json:"active_jobs"`
	TotalJobs     int            `json:"total_jobs"`
	Threads       []ThreadStatus `json:"threads,omitempty"`
	LastHeartbeat string         `json:"last_heartbeat,omitempty"`
}

// ThreadStatus represents a single agent thread.
type ThreadStatus struct {
	ID    string `json:"id"`
	State string `json:"state"`
	JobID string `json:"job_id,omitempty"`
}

// HealthResponse is returned by GET /health.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// --- API methods -------------------------------------------------------------

// Health checks whether IronClaw is reachable and healthy.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	if err := c.get(ctx, "/health", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Chat sends a message to IronClaw and returns the response.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	var resp ChatResponse
	if err := c.post(ctx, "/api/chat", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListJobs returns all jobs from IronClaw.
func (c *Client) ListJobs(ctx context.Context) (*JobsResponse, error) {
	var resp JobsResponse
	if err := c.get(ctx, "/api/jobs", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetJob returns a specific job by ID.
func (c *Client) GetJob(ctx context.Context, jobID string) (*Job, error) {
	var resp Job
	if err := c.get(ctx, fmt.Sprintf("/api/jobs/%s", jobID), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelJob cancels a running job.
func (c *Client) CancelJob(ctx context.Context, jobID string) error {
	return c.post(ctx, fmt.Sprintf("/api/jobs/%s/cancel", jobID), nil, nil)
}

// SearchMemory searches the IronClaw workspace memory.
func (c *Client) SearchMemory(ctx context.Context, req MemorySearchRequest) (*MemorySearchResponse, error) {
	var resp MemorySearchResponse
	if err := c.post(ctx, "/api/memory/search", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListRoutines returns all scheduled routines.
func (c *Client) ListRoutines(ctx context.Context) (*RoutinesResponse, error) {
	var resp RoutinesResponse
	if err := c.get(ctx, "/api/routines", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CreateRoutine creates a new scheduled routine.
func (c *Client) CreateRoutine(ctx context.Context, req CreateRoutineRequest) (*Routine, error) {
	var resp Routine
	if err := c.post(ctx, "/api/routines", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteRoutine deletes a routine by ID.
func (c *Client) DeleteRoutine(ctx context.Context, routineID string) error {
	return c.delete(ctx, fmt.Sprintf("/api/routines/%s", routineID))
}

// ListTools returns all registered tools.
func (c *Client) ListTools(ctx context.Context) (*ToolsResponse, error) {
	var resp ToolsResponse
	if err := c.get(ctx, "/api/tools", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SendTask sends a strategic task to IronClaw for background execution.
func (c *Client) SendTask(ctx context.Context, req SendTaskRequest) (*SendTaskResponse, error) {
	var resp SendTaskResponse
	if err := c.post(ctx, "/api/chat/send", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// AgentStatus returns the current agent thread states and job counts.
func (c *Client) AgentStatus(ctx context.Context) (*AgentStatusResponse, error) {
	var resp AgentStatusResponse
	if err := c.get(ctx, "/api/status", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// --- HTTP helpers -----------------------------------------------------------

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building GET request: %w", err)
	}
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, in, out interface{}) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshalling request: %w", err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("building POST request: %w", err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(req, out)
}

func (c *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building DELETE request: %w", err)
	}
	return c.do(req, nil)
}

func (c *Client) do(req *http.Request, out interface{}) error {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request %s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ironclaw API error %d: %s", resp.StatusCode, string(body))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}
