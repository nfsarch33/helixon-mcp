package server

import (
	"context"
	"testing"

	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type mockClient struct{ mock.Mock }

func (m *mockClient) Health(ctx context.Context) (*ironclaw.HealthResponse, error) {
	args := m.Called(ctx); return args.Get(0).(*ironclaw.HealthResponse), args.Error(1)
}
func (m *mockClient) Chat(ctx context.Context, req ironclaw.ChatRequest) (*ironclaw.ChatResponse, error) {
	args := m.Called(ctx, req); return args.Get(0).(*ironclaw.ChatResponse), args.Error(1)
}
func (m *mockClient) ListJobs(ctx context.Context) (*ironclaw.JobsResponse, error) {
	args := m.Called(ctx); return args.Get(0).(*ironclaw.JobsResponse), args.Error(1)
}
func (m *mockClient) GetJob(ctx context.Context, id string) (*ironclaw.Job, error) {
	args := m.Called(ctx, id); return args.Get(0).(*ironclaw.Job), args.Error(1)
}
func (m *mockClient) CancelJob(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockClient) SearchMemory(ctx context.Context, req ironclaw.MemorySearchRequest) (*ironclaw.MemorySearchResponse, error) {
	args := m.Called(ctx, req); return args.Get(0).(*ironclaw.MemorySearchResponse), args.Error(1)
}
func (m *mockClient) ListRoutines(ctx context.Context) (*ironclaw.RoutinesResponse, error) {
	args := m.Called(ctx); return args.Get(0).(*ironclaw.RoutinesResponse), args.Error(1)
}
func (m *mockClient) CreateRoutine(ctx context.Context, req ironclaw.CreateRoutineRequest) (*ironclaw.Routine, error) {
	args := m.Called(ctx, req); return args.Get(0).(*ironclaw.Routine), args.Error(1)
}
func (m *mockClient) DeleteRoutine(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockClient) ListTools(ctx context.Context) (*ironclaw.ToolsResponse, error) {
	args := m.Called(ctx); return args.Get(0).(*ironclaw.ToolsResponse), args.Error(1)
}

func TestNew_RegistersAllTools(t *testing.T) {
	logger := zap.NewNop()
	srv := New(new(mockClient), logger, "0.1.0")
	count := srv.RegisteredToolCount()
	// health + chat + list_jobs + get_job + cancel_job + search_memory +
	// list_routines + create_routine + delete_routine + list_tools = 10
	assert.Equal(t, 10, count)
}

func TestRun_UnknownTransport(t *testing.T) {
	logger := zap.NewNop()
	srv := New(new(mockClient), logger, "0.1.0")
	err := srv.Run(context.Background(), "grpc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown transport")
}

func TestRun_SSENotImplemented(t *testing.T) {
	logger := zap.NewNop()
	srv := New(new(mockClient), logger, "0.1.0")
	err := srv.Run(context.Background(), "sse")
	assert.Error(t, err)
}
