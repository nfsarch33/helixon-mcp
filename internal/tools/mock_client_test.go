package tools

import (
	"context"

	"github.com/nfsarch33/ironclaw-mcp/internal/ironclaw"
	"github.com/stretchr/testify/mock"
)

// MockIronclawClient is a testify mock for IronclawClient.
type MockIronclawClient struct {
	mock.Mock
}

func (m *MockIronclawClient) Health(ctx context.Context) (*ironclaw.HealthResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ironclaw.HealthResponse), args.Error(1)
}

func (m *MockIronclawClient) Chat(ctx context.Context, req ironclaw.ChatRequest) (*ironclaw.ChatResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ironclaw.ChatResponse), args.Error(1)
}

func (m *MockIronclawClient) ListJobs(ctx context.Context) (*ironclaw.JobsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ironclaw.JobsResponse), args.Error(1)
}

func (m *MockIronclawClient) GetJob(ctx context.Context, jobID string) (*ironclaw.Job, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).(*ironclaw.Job), args.Error(1)
}

func (m *MockIronclawClient) CancelJob(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockIronclawClient) SearchMemory(ctx context.Context, req ironclaw.MemorySearchRequest) (*ironclaw.MemorySearchResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ironclaw.MemorySearchResponse), args.Error(1)
}

func (m *MockIronclawClient) ListRoutines(ctx context.Context) (*ironclaw.RoutinesResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ironclaw.RoutinesResponse), args.Error(1)
}

func (m *MockIronclawClient) CreateRoutine(ctx context.Context, req ironclaw.CreateRoutineRequest) (*ironclaw.Routine, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*ironclaw.Routine), args.Error(1)
}

func (m *MockIronclawClient) DeleteRoutine(ctx context.Context, routineID string) error {
	args := m.Called(ctx, routineID)
	return args.Error(0)
}

func (m *MockIronclawClient) ListTools(ctx context.Context) (*ironclaw.ToolsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*ironclaw.ToolsResponse), args.Error(1)
}
