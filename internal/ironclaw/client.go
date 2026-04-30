// Package ironclaw provides an HTTP client for the IronClaw REST API.
package ironclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxResponseBytes limits response body size to prevent memory exhaustion.
const MaxResponseBytes = 10 << 20 // 10 MiB

const chatPollInterval = 250 * time.Millisecond

// Client is an HTTP client for the IronClaw web gateway API.
type Client struct {
	baseURL    string
	apiKey     string
	timeout    time.Duration
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
		timeout: timeout,
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
		timeout:    30 * time.Second,
		httpClient: doer,
	}
}

// --- Request / Response types -----------------------------------------------

// ChatRequest is the payload for the bridge-level chat operation.
type ChatRequest struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

// ChatResponse is returned once the async IronClaw chat turn has completed.
type ChatResponse struct {
	Response  string `json:"response"`
	MessageID string `json:"message_id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Status    string `json:"status,omitempty"`
}

// Job represents a background job in IronClaw.
type Job struct {
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	State     string `json:"state"`
	UserID    string `json:"user_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	StartedAt string `json:"started_at,omitempty"`
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
	Results []MemoryEntry `json:"results"`
}

// Routine represents an IronClaw scheduled routine.
type Routine struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Description         string `json:"description,omitempty"`
	Enabled             bool   `json:"enabled"`
	TriggerType         string `json:"trigger_type,omitempty"`
	TriggerSummary      string `json:"trigger_summary,omitempty"`
	ActionType          string `json:"action_type,omitempty"`
	Status              string `json:"status,omitempty"`
	LastRunAt           string `json:"last_run_at,omitempty"`
	NextFireAt          string `json:"next_fire_at,omitempty"`
	RunCount            uint64 `json:"run_count,omitempty"`
	ConsecutiveFailures uint32 `json:"consecutive_failures,omitempty"`
}

// RoutinesResponse is returned by GET /api/routines.
type RoutinesResponse struct {
	Routines []Routine `json:"routines"`
}

// ToolInfo represents a registered IronClaw tool.
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ToolsResponse is returned by GET /api/tools.
type ToolsResponse struct {
	Tools []ToolInfo `json:"tools"`
}

// SendTaskRequest is the payload for POST /api/chat/send.
type SendTaskRequest struct {
	Message   string `json:"content"`
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

// HealthResponse is returned by GET /api/health.
type HealthResponse struct {
	Status  string `json:"status"`
	Channel string `json:"channel,omitempty"`
}

// RouterHealthResponse is returned by GET /healthz on the llm-cluster-router.
type RouterHealthResponse struct {
	OK             bool         `json:"ok"`
	HealthyNodes   int          `json:"healthy_nodes"`
	TotalNodes     int          `json:"total_nodes"`
	QueueDepth     int          `json:"queue_depth"`
	Inflight       int          `json:"inflight_requests"`
	MaxConcurrency int          `json:"max_concurrency"`
	MaxQueueDepth  int          `json:"max_queue_depth"`
	Nodes          []RouterNode `json:"nodes"`
}

// RouterNode is a single upstream node in the router health response.
type RouterNode struct {
	Name    string   `json:"name"`
	Tier    string   `json:"tier"`
	URL     string   `json:"url"`
	Models  []string `json:"models"`
	Healthy bool     `json:"healthy"`
}

// GatewayStatusResponse is returned by GET /api/gateway/status.
type GatewayStatusResponse struct {
	Status      string `json:"status"`
	Uptime      string `json:"uptime,omitempty"`
	Connections int    `json:"connections,omitempty"`
}

// StackStatusResponse combines router and gateway health for a unified view.
type StackStatusResponse struct {
	Router  *RouterHealthResponse  `json:"router,omitempty"`
	Gateway *GatewayStatusResponse `json:"gateway,omitempty"`
}

// SpawnAgentRequest is the payload for POST /api/jobs (spawn a new agent job).
type SpawnAgentRequest struct {
	Name  string `json:"name"`
	Model string `json:"model,omitempty"`
	Tier  string `json:"tier,omitempty"`
}

// SpawnAgentResponse is returned by the spawn operation.
type SpawnAgentResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
	Model  string `json:"model,omitempty"`
}

type chatSendRequest struct {
	Content  string `json:"content"`
	ThreadID string `json:"thread_id,omitempty"`
}

type chatSendAcceptedResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
}

type threadInfo struct {
	ID string `json:"id"`
}

type threadListResponse struct {
	AssistantThread *threadInfo  `json:"assistant_thread"`
	Threads         []threadInfo `json:"threads"`
	ActiveThread    string       `json:"active_thread"`
}

type historyTurn struct {
	UserInput string            `json:"user_input"`
	Response  *string           `json:"response"`
	State     string            `json:"state"`
	ToolCalls []historyToolCall `json:"tool_calls"`
}

type historyResponse struct {
	ThreadID string        `json:"thread_id"`
	Turns    []historyTurn `json:"turns"`
}

type historyToolCall struct {
	Error string `json:"error"`
}

// --- API methods -------------------------------------------------------------

// Health checks whether IronClaw is reachable and healthy.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	var resp HealthResponse
	if err := c.get(ctx, "/api/health", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// Chat sends a message to IronClaw, then polls history until a response is available.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	threadID, err := c.resolveThreadID(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("resolving thread: %w", err)
	}

	initialHistory, err := c.getHistory(ctx, threadID, 100)
	if err != nil {
		return nil, fmt.Errorf("loading chat history: %w", err)
	}

	var accepted chatSendAcceptedResponse
	if err := c.post(ctx, "/api/chat/send", chatSendRequest{
		Content:  req.Message,
		ThreadID: threadID,
	}, &accepted); err != nil {
		return nil, err
	}

	response, err := c.waitForChatResponse(ctx, threadID, len(initialHistory.Turns), req.Message)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{
		Response:  response,
		MessageID: accepted.MessageID,
		SessionID: threadID,
		Status:    "completed",
	}, nil
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

// DeleteRoutine deletes a routine by ID.
func (c *Client) DeleteRoutine(ctx context.Context, routineID string) error {
	return c.delete(ctx, fmt.Sprintf("/api/routines/%s", routineID))
}

// ListTools returns all registered tools.
func (c *Client) ListTools(ctx context.Context) (*ToolsResponse, error) {
	var resp ToolsResponse
	if err := c.get(ctx, "/api/extensions/tools", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StackStatus returns combined router and gateway health.
func (c *Client) StackStatus(ctx context.Context, routerURL string) (*StackStatusResponse, error) {
	result := &StackStatusResponse{}

	if routerURL != "" {
		routerClient := &Client{
			baseURL:    routerURL,
			httpClient: c.httpClient,
		}
		var router RouterHealthResponse
		if err := routerClient.get(ctx, "/healthz", &router); err == nil {
			result.Router = &router
		}
	}

	var gateway GatewayStatusResponse
	if err := c.get(ctx, "/api/gateway/status", &gateway); err == nil {
		result.Gateway = &gateway
	}

	return result, nil
}

// SpawnAgent creates a new agent job in IronClaw.
func (c *Client) SpawnAgent(ctx context.Context, req SpawnAgentRequest) (*SpawnAgentResponse, error) {
	chatReq := chatSendRequest{
		Content: fmt.Sprintf("[spawn-agent] name=%s model=%s tier=%s", req.Name, req.Model, req.Tier),
	}
	var accepted chatSendAcceptedResponse
	if err := c.post(ctx, "/api/chat/send", chatReq, &accepted); err != nil {
		return nil, fmt.Errorf("spawning agent: %w", err)
	}
	return &SpawnAgentResponse{
		JobID:  accepted.MessageID,
		Status: accepted.Status,
		Model:  req.Model,
	}, nil
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
// Combines /api/health and /api/gateway/status since /api/status is not implemented.
func (c *Client) AgentStatus(ctx context.Context) (*AgentStatusResponse, error) {
	resp := &AgentStatusResponse{}

	var health HealthResponse
	if err := c.get(ctx, "/api/health", &health); err != nil {
		return nil, err
	}
	resp.Status = health.Status

	var gateway GatewayStatusResponse
	if err := c.get(ctx, "/api/gateway/status", &gateway); err == nil {
		resp.Status = gateway.Status
	}

	var jobs JobsResponse
	if err := c.get(ctx, "/api/jobs", &jobs); err == nil {
		for _, j := range jobs.Jobs {
			if j.State == "running" || j.State == "in_progress" {
				resp.ActiveJobs++
			}
		}
		resp.TotalJobs = len(jobs.Jobs)
	}

	return resp, nil
}

// --- HTTP helpers -----------------------------------------------------------

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building GET request: %w", err)
	}
	return c.do(req, out)
}

func (c *Client) getWithQuery(ctx context.Context, path string, query url.Values, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building GET request: %w", err)
	}
	req.URL.RawQuery = query.Encode()
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
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, MaxResponseBytes)

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(limited)
		return fmt.Errorf("ironclaw API error %d: %s", resp.StatusCode, string(body))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(limited).Decode(out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func (c *Client) resolveThreadID(ctx context.Context, sessionID string) (string, error) {
	if sessionID != "" {
		return sessionID, nil
	}

	var threads threadListResponse
	if err := c.get(ctx, "/api/chat/threads", &threads); err == nil {
		switch {
		case threads.AssistantThread != nil && threads.AssistantThread.ID != "":
			return threads.AssistantThread.ID, nil
		case threads.ActiveThread != "":
			return threads.ActiveThread, nil
		case len(threads.Threads) > 0 && threads.Threads[0].ID != "":
			return threads.Threads[0].ID, nil
		}
	}

	var thread threadInfo
	if err := c.post(ctx, "/api/chat/thread/new", nil, &thread); err != nil {
		return "", fmt.Errorf("creating chat thread: %w", err)
	}
	if thread.ID == "" {
		return "", fmt.Errorf("creating chat thread: missing thread id")
	}
	return thread.ID, nil
}

func (c *Client) getHistory(ctx context.Context, threadID string, limit int) (*historyResponse, error) {
	var history historyResponse
	query := url.Values{}
	query.Set("thread_id", threadID)
	if limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", limit))
	}
	if err := c.getWithQuery(ctx, "/api/chat/history", query, &history); err != nil {
		return nil, err
	}
	return &history, nil
}

func (c *Client) waitForChatResponse(ctx context.Context, threadID string, initialTurnCount int, message string) (string, error) {
	pollCtx := ctx
	cancel := func() {}
	if _, ok := ctx.Deadline(); !ok && c.timeout > 0 {
		pollCtx, cancel = context.WithTimeout(ctx, c.timeout)
	}
	defer cancel()

	ticker := time.NewTicker(chatPollInterval)
	defer ticker.Stop()

	for {
		history, err := c.getHistory(pollCtx, threadID, initialTurnCount+10)
		if err != nil {
			return "", fmt.Errorf("loading chat history: %w", err)
		}

		for i := len(history.Turns) - 1; i >= initialTurnCount && i >= 0; i-- {
			turn := history.Turns[i]
			if turn.UserInput != message {
				continue
			}
			if turn.Response != nil && *turn.Response != "" {
				return *turn.Response, nil
			}
			if err := terminalTurnError(turn); err != "" {
				return "", fmt.Errorf("backend turn failed: %s", err)
			}
		}

		select {
		case <-pollCtx.Done():
			return "", fmt.Errorf("waiting for chat response: %w", pollCtx.Err())
		case <-ticker.C:
		}
	}
}

func terminalTurnError(turn historyTurn) string {
	state := strings.ToLower(turn.State)
	if state != "failed" && state != "cancelled" {
		return ""
	}
	for _, toolCall := range turn.ToolCalls {
		if toolCall.Error != "" {
			return toolCall.Error
		}
	}
	if turn.Response != nil && *turn.Response != "" {
		return *turn.Response
	}
	return fmt.Sprintf("chat turn entered terminal state %q without a response", turn.State)
}
