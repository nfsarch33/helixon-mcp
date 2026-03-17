package tools

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestGenericCLIHandler(t *testing.T) {
	t.Run("nil-cli", func(t *testing.T) {
		h := NewGenericCLIHandler(nil, "test_tool", "test", "test")
		res := invokeHandler(t, h, map[string]interface{}{"action": "list"})
		assert.True(t, res.IsError)
	})

	t.Run("missing-action", func(t *testing.T) {
		h := NewGenericCLIHandler(&mockCLIRunner{}, "test_tool", "test", "test")
		res := invokeHandler(t, h, nil)
		assert.True(t, res.IsError)
	})

	t.Run("with-args", func(t *testing.T) {
		mock := &mockCLIRunner{output: "ok"}
		h := NewGenericCLIHandler(mock, "test_tool", "test", "fleet")
		_ = invokeHandler(t, h, map[string]interface{}{"action": "register", "args": "--name node1"})
		assert.Equal(t, []string{"fleet", "register", "--name", "node1"}, mock.called[0])
	})
}

func TestFleetHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `[{"name":"node1"}]`}
	h := NewFleetHandler(mock)
	assert.Equal(t, "ironclaw_fleet_ops_full", h.Tool().Name)
	_ = invokeHandler(t, h, map[string]interface{}{"action": "list"})
	assert.Equal(t, []string{"fleet", "list"}, mock.called[0])
}

func TestRoutineHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `[]`}
	h := NewRoutineHandler(mock)
	assert.Equal(t, "ironclaw_routine_ops", h.Tool().Name)
	_ = invokeHandler(t, h, map[string]interface{}{"action": "list"})
	assert.Equal(t, []string{"routine", "list"}, mock.called[0])
}

func TestA2AFullHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `{}`}
	h := NewA2AFullHandler(mock)
	assert.Equal(t, "ironclaw_a2a_ops", h.Tool().Name)
	_ = invokeHandler(t, h, map[string]interface{}{"action": "list"})
	assert.Equal(t, []string{"a2a", "list"}, mock.called[0])
}

func TestSnapshotHandler(t *testing.T) {
	mock := &mockCLIRunner{output: "taken"}
	h := NewSnapshotHandler(mock)
	_ = invokeHandler(t, h, map[string]interface{}{"action": "take"})
	assert.Equal(t, []string{"snapshot", "take"}, mock.called[0])
}

func TestRecoverHandler(t *testing.T) {
	t.Run("export-state", func(t *testing.T) {
		mock := &mockCLIRunner{output: "exported"}
		h := NewRecoverHandler(mock)
		_ = invokeHandler(t, h, map[string]interface{}{"action": "export-state"})
		assert.Equal(t, []string{"export-state"}, mock.called[0])
	})

	t.Run("restore-with-from-and-dry-run", func(t *testing.T) {
		mock := &mockCLIRunner{output: "restored"}
		h := NewRecoverHandler(mock)
		_ = invokeHandler(t, h, map[string]interface{}{
			"action":  "restore-state",
			"from":    "/path/to/snapshot",
			"dry_run": true,
		})
		assert.Equal(t, []string{"restore-state", "--from", "/path/to/snapshot", "--dry-run"}, mock.called[0])
	})

	t.Run("nil-cli", func(t *testing.T) {
		h := NewRecoverHandler(nil)
		res := invokeHandler(t, h, map[string]interface{}{"action": "recover"})
		assert.True(t, res.IsError)
	})
}

func TestWorkspaceHandler(t *testing.T) {
	mock := &mockCLIRunner{output: "files"}
	h := NewWorkspaceHandler(mock)
	_ = invokeHandler(t, h, map[string]interface{}{"action": "tree"})
	assert.Equal(t, []string{"workspace", "tree"}, mock.called[0])
}

func TestCRMFullHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `[{"name":"contact1"}]`}
	h := NewCRMFullHandler(mock)
	_ = invokeHandler(t, h, map[string]interface{}{"action": "list"})
	assert.Equal(t, []string{"crm", "list"}, mock.called[0])
}

func TestSkillsHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `{"top": []}`}
	h := NewSkillsHandler(mock)
	_ = invokeHandler(t, h, map[string]interface{}{"action": "stats"})
	assert.Equal(t, []string{"skills", "stats"}, mock.called[0])
}

func TestCEOOrchestrateHandler(t *testing.T) {
	t.Run("full-config", func(t *testing.T) {
		mock := &mockCLIRunner{output: "orchestrated"}
		h := NewCEOOrchestrateHandler(mock)
		_ = invokeHandler(t, h, map[string]interface{}{
			"workers": float64(3),
			"persona": "night-auditor",
			"task":    "review infra",
		})
		assert.Equal(t, []string{"ceo", "orchestrate", "--workers", "3", "--persona", "night-auditor", "--task", "review infra"}, mock.called[0])
	})

	t.Run("nil-cli", func(t *testing.T) {
		h := NewCEOOrchestrateHandler(nil)
		res := invokeHandler(t, h, map[string]interface{}{})
		assert.True(t, res.IsError)
	})
}

func TestJobOpsHandler(t *testing.T) {
	mock := &mockCLIRunner{output: `[]`}
	h := NewJobOpsHandler(mock)
	_ = invokeHandler(t, h, map[string]interface{}{"action": "list"})
	assert.Equal(t, []string{"job", "list"}, mock.called[0])
}

func TestExportDashboardsHandler(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		mock := &mockCLIRunner{output: "exported 4 dashboards"}
		h := NewExportDashboardsHandler(mock)
		_ = invokeHandler(t, h, nil)
		assert.Equal(t, []string{"export-dashboards"}, mock.called[0])
	})

	t.Run("with-output-dir", func(t *testing.T) {
		mock := &mockCLIRunner{output: "exported"}
		h := NewExportDashboardsHandler(mock)
		_ = invokeHandler(t, h, map[string]interface{}{"output_dir": "/tmp/dashboards"})
		assert.Equal(t, []string{"export-dashboards", "--output", "/tmp/dashboards"}, mock.called[0])
	})
}

func TestToolRegistrationCount(t *testing.T) {
	mock := &mockCLIRunner{}

	allTools := []interface{ Tool() mcp.Tool }{
		NewDoctorHandler(mock),
		NewStatusHandler(mock),
		NewInstallHandler(mock),
		NewDeployHandler(mock),
		NewLogsHandler(mock),
		NewSpawnFullHandler(mock),
		NewListAgentsHandler(mock),
		NewStopAgentHandler(mock),
		NewGPUStatusHandler(mock),
		NewCostSummaryHandler(mock),
		NewMemoryStatsHandler(mock),
		NewFleetHandler(mock),
		NewRoutineHandler(mock),
		NewA2AFullHandler(mock),
		NewSnapshotHandler(mock),
		NewRecoverHandler(mock),
		NewWorkspaceHandler(mock),
		NewCRMFullHandler(mock),
		NewSkillsHandler(mock),
		NewCEOOrchestrateHandler(mock),
		NewJobOpsHandler(mock),
		NewExportDashboardsHandler(mock),
	}

	names := make(map[string]bool)
	for _, h := range allTools {
		name := h.Tool().Name
		if names[name] {
			t.Errorf("duplicate tool name: %s", name)
		}
		names[name] = true
	}

	if len(names) < 22 {
		t.Errorf("expected >= 22 new tools, got %d", len(names))
	}
	t.Logf("total new MCP tools: %d", len(names))
}
