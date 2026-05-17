package tools

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/nfsarch33/helixon-mcp/internal/helixon"
)

// MockIronclawClient is a testify mock for IronclawClient.
type MockIronclawClient struct {
	mock.Mock
}

func (m *MockIronclawClient) Health(ctx context.Context) (*helixon.HealthResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.HealthResponse), args.Error(1)
}

func (m *MockIronclawClient) Chat(ctx context.Context, req helixon.ChatRequest) (*helixon.ChatResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.ChatResponse), args.Error(1)
}

func (m *MockIronclawClient) ListJobs(ctx context.Context) (*helixon.JobsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.JobsResponse), args.Error(1)
}

func (m *MockIronclawClient) GetJob(ctx context.Context, jobID string) (*helixon.Job, error) {
	args := m.Called(ctx, jobID)
	return args.Get(0).(*helixon.Job), args.Error(1)
}

func (m *MockIronclawClient) CancelJob(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockIronclawClient) SearchMemory(ctx context.Context, req helixon.MemorySearchRequest) (*helixon.MemorySearchResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.MemorySearchResponse), args.Error(1)
}

func (m *MockIronclawClient) WriteMemory(ctx context.Context, req helixon.MemoryWriteRequest) (*helixon.MemoryWriteResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.MemoryWriteResponse), args.Error(1)
}

func (m *MockIronclawClient) ReadMemory(ctx context.Context, req helixon.MemoryReadRequest) (*helixon.MemoryReadResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.MemoryReadResponse), args.Error(1)
}

func (m *MockIronclawClient) TreeMemory(ctx context.Context, req helixon.MemoryTreeRequest) (*helixon.MemoryTreeResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.MemoryTreeResponse), args.Error(1)
}

func (m *MockIronclawClient) ListRoutines(ctx context.Context) (*helixon.RoutinesResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.RoutinesResponse), args.Error(1)
}

func (m *MockIronclawClient) DeleteRoutine(ctx context.Context, routineID string) error {
	args := m.Called(ctx, routineID)
	return args.Error(0)
}

func (m *MockIronclawClient) ListTools(ctx context.Context) (*helixon.ToolsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.ToolsResponse), args.Error(1)
}

func (m *MockIronclawClient) StackStatus(ctx context.Context, routerURL string) (*helixon.StackStatusResponse, error) {
	args := m.Called(ctx, routerURL)
	return args.Get(0).(*helixon.StackStatusResponse), args.Error(1)
}

func (m *MockIronclawClient) SpawnAgent(ctx context.Context, req helixon.SpawnAgentRequest) (*helixon.SpawnAgentResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.SpawnAgentResponse), args.Error(1)
}

func (m *MockIronclawClient) SendTask(ctx context.Context, req helixon.SendTaskRequest) (*helixon.SendTaskResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*helixon.SendTaskResponse), args.Error(1)
}

func (m *MockIronclawClient) AgentStatus(ctx context.Context) (*helixon.AgentStatusResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*helixon.AgentStatusResponse), args.Error(1)
}

// MockPrometheusQuerier is a testify mock for PrometheusQuerier.
type MockPrometheusQuerier struct {
	mock.Mock
}

func (m *MockPrometheusQuerier) Query(ctx context.Context, query string) (string, error) {
	args := m.Called(ctx, query)
	return args.String(0), args.Error(1)
}
