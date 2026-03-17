package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCLIRunner struct {
	output string
	err    error
	called [][]string
}

func (m *mockCLIRunner) Run(_ context.Context, args ...string) (string, error) {
	m.called = append(m.called, args)
	return m.output, m.err
}

func invokeHandler(t *testing.T, tool interface {
	Handle(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}, args map[string]interface{}) *mcp.CallToolResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	res, err := tool.Handle(context.Background(), req)
	require.NoError(t, err)
	return res
}

func TestDoctorHandler(t *testing.T) {
	t.Run("nil-cli", func(t *testing.T) {
		h := NewDoctorHandler(nil)
		res := invokeHandler(t, h, nil)
		assert.True(t, res.IsError)
	})

	t.Run("success", func(t *testing.T) {
		mock := &mockCLIRunner{output: `{"ok": true}`}
		h := NewDoctorHandler(mock)
		res := invokeHandler(t, h, nil)
		assert.False(t, res.IsError)
		assert.Len(t, mock.called, 1)
		assert.Equal(t, []string{"doctor", "--json"}, mock.called[0])
	})

	t.Run("with-suite", func(t *testing.T) {
		mock := &mockCLIRunner{output: `{"ok": true}`}
		h := NewDoctorHandler(mock)
		_ = invokeHandler(t, h, map[string]interface{}{"suite": "services"})
		assert.Equal(t, []string{"doctor", "--json", "--suite", "services"}, mock.called[0])
	})

	t.Run("tool-definition", func(t *testing.T) {
		h := NewDoctorHandler(nil)
		tool := h.Tool()
		assert.Equal(t, "ironclaw_doctor", tool.Name)
	})
}

func TestStatusHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `{"metrics": {}}`}
	h := NewStatusHandler(mock)
	res := invokeHandler(t, h, nil)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"status", "--json"}, mock.called[0])
}

func TestInstallHandler(t *testing.T) {
	t.Run("check-only", func(t *testing.T) {
		mock := &mockCLIRunner{output: `{"deps": []}`}
		h := NewInstallHandler(mock)
		_ = invokeHandler(t, h, nil)
		assert.Equal(t, []string{"install", "--json"}, mock.called[0])
	})

	t.Run("with-fix", func(t *testing.T) {
		mock := &mockCLIRunner{output: `{"deps": []}`}
		h := NewInstallHandler(mock)
		_ = invokeHandler(t, h, map[string]interface{}{"fix": true})
		assert.Equal(t, []string{"install", "--json", "--fix"}, mock.called[0])
	})
}

func TestDeployHandler(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		mock := &mockCLIRunner{output: "deployed"}
		h := NewDeployHandler(mock)
		_ = invokeHandler(t, h, map[string]interface{}{"action": "up"})
		assert.Equal(t, []string{"deploy", "up"}, mock.called[0])
	})

	t.Run("k8s-with-method", func(t *testing.T) {
		mock := &mockCLIRunner{output: "deployed"}
		h := NewDeployHandler(mock)
		_ = invokeHandler(t, h, map[string]interface{}{"action": "k8s", "method": "terraform", "dry_run": true})
		assert.Equal(t, []string{"deploy", "k8s", "--method", "terraform", "--dry-run"}, mock.called[0])
	})

	t.Run("missing-action", func(t *testing.T) {
		h := NewDeployHandler(&mockCLIRunner{})
		res := invokeHandler(t, h, nil)
		assert.True(t, res.IsError)
	})
}

func TestLogsHandler(t *testing.T) {
	mock := &mockCLIRunner{output: "log output"}
	h := NewLogsHandler(mock)
	_ = invokeHandler(t, h, map[string]interface{}{"tail": float64(100)})
	assert.Equal(t, []string{"logs", "--tail", "100"}, mock.called[0])
}

func TestSpawnFullHandler(t *testing.T) {
	t.Run("full-config", func(t *testing.T) {
		mock := &mockCLIRunner{output: "spawned"}
		h := NewSpawnFullHandler(mock)
		_ = invokeHandler(t, h, map[string]interface{}{
			"name":    "worker-1",
			"model":   "qwen3.5-27b",
			"gpu":     "GPU-abc",
			"persona": "night-auditor",
		})
		assert.Equal(t, []string{"spawn", "--name", "worker-1", "--model", "qwen3.5-27b", "--gpu", "GPU-abc", "--persona", "night-auditor"}, mock.called[0])
	})

	t.Run("missing-name", func(t *testing.T) {
		h := NewSpawnFullHandler(&mockCLIRunner{})
		res := invokeHandler(t, h, nil)
		assert.True(t, res.IsError)
	})
}

func TestListAgentsHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `[{"name":"agent1"}]`}
	h := NewListAgentsHandler(mock)
	res := invokeHandler(t, h, nil)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"list", "--json"}, mock.called[0])
}

func TestStopAgentHandler(t *testing.T) {
	mock := &mockCLIRunner{output: "stopped"}
	h := NewStopAgentHandler(mock)
	_ = invokeHandler(t, h, map[string]interface{}{"name": "worker-1"})
	assert.Equal(t, []string{"stop", "worker-1"}, mock.called[0])
}

func TestGPUStatusHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `{"gpus": []}`}
	h := NewGPUStatusHandler(mock)
	_ = invokeHandler(t, h, nil)
	assert.Equal(t, []string{"gpu", "status", "--json"}, mock.called[0])
}

func TestCostSummaryHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `{"daily_spend": 0.5}`}
	h := NewCostSummaryHandler(mock)
	_ = invokeHandler(t, h, nil)
	assert.Equal(t, []string{"cost", "summary", "--json"}, mock.called[0])
}

func TestMemoryStatsHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `{"entries": 42}`}
	h := NewMemoryStatsHandler(mock)
	_ = invokeHandler(t, h, nil)
	assert.Equal(t, []string{"memory", "stats", "--json"}, mock.called[0])
}
