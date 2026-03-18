package server

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockClient struct{ mock.Mock }

func (m *mockClient) Health(ctx context.Context) (*ironclaw.HealthResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ironclaw.HealthResponse), args.Error(1)
}
func (m *mockClient) Chat(ctx context.Context, req ironclaw.ChatRequest) (*ironclaw.ChatResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ironclaw.ChatResponse), args.Error(1)
}
func (m *mockClient) ListJobs(ctx context.Context) (*ironclaw.JobsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ironclaw.JobsResponse), args.Error(1)
}
func (m *mockClient) GetJob(ctx context.Context, id string) (*ironclaw.Job, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*ironclaw.Job), args.Error(1)
}
func (m *mockClient) CancelJob(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockClient) SearchMemory(ctx context.Context, req ironclaw.MemorySearchRequest) (*ironclaw.MemorySearchResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ironclaw.MemorySearchResponse), args.Error(1)
}
func (m *mockClient) ListRoutines(ctx context.Context) (*ironclaw.RoutinesResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ironclaw.RoutinesResponse), args.Error(1)
}
func (m *mockClient) DeleteRoutine(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockClient) ListTools(ctx context.Context) (*ironclaw.ToolsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ironclaw.ToolsResponse), args.Error(1)
}
func (m *mockClient) StackStatus(ctx context.Context, routerURL string) (*ironclaw.StackStatusResponse, error) {
	args := m.Called(ctx, routerURL)
	return args.Get(0).(*ironclaw.StackStatusResponse), args.Error(1)
}
func (m *mockClient) SpawnAgent(ctx context.Context, req ironclaw.SpawnAgentRequest) (*ironclaw.SpawnAgentResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ironclaw.SpawnAgentResponse), args.Error(1)
}
func (m *mockClient) SendTask(ctx context.Context, req ironclaw.SendTaskRequest) (*ironclaw.SendTaskResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ironclaw.SendTaskResponse), args.Error(1)
}
func (m *mockClient) AgentStatus(ctx context.Context) (*ironclaw.AgentStatusResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ironclaw.AgentStatusResponse), args.Error(1)
}

type mockProm struct{ mock.Mock }

func (m *mockProm) Query(ctx context.Context, query string) (string, error) {
	args := m.Called(ctx, query)
	return args.String(0), args.Error(1)
}

type mockCLI struct{}

func (m *mockCLI) Run(ctx context.Context, args ...string) (string, error) {
	return "", nil
}

func TestNew_RegistersBaseTools(t *testing.T) {
	srv := New(new(mockClient), nil, nil, nil, discardLogger(), "0.1.0")
	count := srv.RegisteredToolCount()
	// 14 core + 10 research + 4 uiauto + 4 evolver = 32 (no prometheus = no get_metrics)
	assert.Equal(t, 32, count)
}

func TestNew_WithPrometheus_RegistersMetricsTool(t *testing.T) {
	srv := New(new(mockClient), new(mockProm), nil, nil, discardLogger(), "0.1.0")
	count := srv.RegisteredToolCount()
	// 32 base + get_metrics = 33
	assert.Equal(t, 33, count)
}

func TestNew_WithCLI_RegistersCEOTools(t *testing.T) {
	srv := New(new(mockClient), nil, &mockCLI{}, nil, discardLogger(), "0.1.0")
	count := srv.RegisteredToolCount()
	// 32 base + 6 CEO + 9 dual-ops + 11 ops + 11 extended = 69
	assert.Equal(t, 69, count)
}

func TestNew_WithAll_RegistersAllTools(t *testing.T) {
	srv := New(new(mockClient), new(mockProm), &mockCLI{}, &mockCLI{}, discardLogger(), "0.1.0")
	count := srv.RegisteredToolCount()
	// 32 base + get_metrics + 6 CEO + 1 GWS + 9 dual-ops + 22 new ops/extended = 71
	assert.Equal(t, 71, count)
}

func TestRun_UnknownTransport(t *testing.T) {
	srv := New(new(mockClient), nil, nil, nil, discardLogger(), "0.1.0")
	err := srv.Run(context.Background(), "grpc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown transport")
}

func TestRun_SSE_StartsAndStops(t *testing.T) {
	srv := New(new(mockClient), nil, nil, nil, discardLogger(), "0.1.0")

	ctx, cancel := context.WithCancel(context.Background())
	t.Setenv("MCP_SSE_ADDR", ":0")

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx, "sse")
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("SSE server did not stop within timeout")
	}
}
