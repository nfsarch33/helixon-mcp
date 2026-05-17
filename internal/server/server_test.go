package server

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/nfsarch33/helixon-mcp/internal/helixon"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockClient struct{ mock.Mock }

func (m *mockClient) Health(ctx context.Context) (*helixon.HealthResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.HealthResponse), args.Error(1)
}
func (m *mockClient) Chat(ctx context.Context, req helixon.ChatRequest) (*helixon.ChatResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.ChatResponse), args.Error(1)
}
func (m *mockClient) ListJobs(ctx context.Context) (*helixon.JobsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.JobsResponse), args.Error(1)
}
func (m *mockClient) GetJob(ctx context.Context, id string) (*helixon.Job, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*helixon.Job), args.Error(1)
}
func (m *mockClient) CancelJob(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockClient) SearchMemory(ctx context.Context, req helixon.MemorySearchRequest) (*helixon.MemorySearchResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.MemorySearchResponse), args.Error(1)
}
func (m *mockClient) WriteMemory(ctx context.Context, req helixon.MemoryWriteRequest) (*helixon.MemoryWriteResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.MemoryWriteResponse), args.Error(1)
}
func (m *mockClient) ReadMemory(ctx context.Context, req helixon.MemoryReadRequest) (*helixon.MemoryReadResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.MemoryReadResponse), args.Error(1)
}
func (m *mockClient) TreeMemory(ctx context.Context, req helixon.MemoryTreeRequest) (*helixon.MemoryTreeResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.MemoryTreeResponse), args.Error(1)
}
func (m *mockClient) ListRoutines(ctx context.Context) (*helixon.RoutinesResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.RoutinesResponse), args.Error(1)
}
func (m *mockClient) DeleteRoutine(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockClient) ListTools(ctx context.Context) (*helixon.ToolsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.ToolsResponse), args.Error(1)
}
func (m *mockClient) StackStatus(ctx context.Context, routerURL string) (*helixon.StackStatusResponse, error) {
	args := m.Called(ctx, routerURL)
	return args.Get(0).(*helixon.StackStatusResponse), args.Error(1)
}
func (m *mockClient) SpawnAgent(ctx context.Context, req helixon.SpawnAgentRequest) (*helixon.SpawnAgentResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.SpawnAgentResponse), args.Error(1)
}
func (m *mockClient) SendTask(ctx context.Context, req helixon.SendTaskRequest) (*helixon.SendTaskResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.SendTaskResponse), args.Error(1)
}
func (m *mockClient) AgentStatus(ctx context.Context) (*helixon.AgentStatusResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.AgentStatusResponse), args.Error(1)
}

type mockProm struct{ mock.Mock }

func (m *mockProm) Query(ctx context.Context, query string) (string, error) {
	args := m.Called(ctx, query)
	return args.String(0), args.Error(1)
}

func TestNew_RegistersBaseTools(t *testing.T) {
	srv := New(new(mockClient), nil, discardLogger(), "0.1.0")
	count := srv.RegisteredToolCount()
	// Generic Helixon HTTP-bridge baseline.
	assert.Equal(t, 17, count)
}

func TestNew_WithPrometheus_RegistersMetricsTool(t *testing.T) {
	srv := New(new(mockClient), new(mockProm), discardLogger(), "0.1.0")
	count := srv.RegisteredToolCount()
	// Generic baseline + helixon_get_metrics when PROMETHEUS_URL is configured.
	assert.Equal(t, 18, count)
}

func TestMCPServer_ReturnsConfiguredServer(t *testing.T) {
	srv := New(new(mockClient), nil, discardLogger(), "0.1.0")
	assert.NotNil(t, srv.MCPServer())
}

func TestRun_UnknownTransport(t *testing.T) {
	srv := New(new(mockClient), nil, discardLogger(), "0.1.0")
	err := srv.Run(context.Background(), "grpc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown transport")
}

func TestRun_SSE_StartsAndStops(t *testing.T) {
	srv := New(new(mockClient), nil, discardLogger(), "0.1.0")

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
