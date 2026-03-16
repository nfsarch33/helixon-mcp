package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dualMockCLI struct {
	lastArgs []string
	output   string
	err      error
}

func (m *dualMockCLI) Run(_ context.Context, args ...string) (string, error) {
	m.lastArgs = args
	return m.output, m.err
}

func dualCallReq(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: args,
		},
	}
}

func TestK8sOpsHandler_Tool(t *testing.T) {
	h := NewK8sOpsHandler(nil)
	tool := h.Tool()
	assert.Equal(t, "ironclaw_k8s_ops", tool.Name)
}

func TestK8sOpsHandler_Handle_NilCLI(t *testing.T) {
	h := NewK8sOpsHandler(nil)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{"action": "get_pods"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestK8sOpsHandler_Handle_GetPods(t *testing.T) {
	mock := &dualMockCLI{output: `[{"name":"pod1"}]`}
	h := NewK8sOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action":    "get_pods",
		"namespace": "default",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"k8s-ops", "get_pods", "--namespace", "default"}, mock.lastArgs)
}

func TestK8sOpsHandler_Handle_GetLogs(t *testing.T) {
	mock := &dualMockCLI{output: "log line 1\nlog line 2"}
	h := NewK8sOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action":        "get_logs",
		"resource_name": "my-pod",
		"tail_lines":    "100",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"k8s-ops", "get_logs", "--name", "my-pod", "--tail", "100"}, mock.lastArgs)
}

func TestK8sOpsHandler_Handle_MissingAction(t *testing.T) {
	mock := &dualMockCLI{}
	h := NewK8sOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestK8sOpsHandler_Handle_CLIError(t *testing.T) {
	mock := &dualMockCLI{err: fmt.Errorf("connection refused"), output: "timeout"}
	h := NewK8sOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{"action": "get_nodes"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestTfOpsHandler_Tool(t *testing.T) {
	h := NewTfOpsHandler(nil)
	tool := h.Tool()
	assert.Equal(t, "ironclaw_tf_ops", tool.Name)
}

func TestTfOpsHandler_Handle_Plan(t *testing.T) {
	mock := &dualMockCLI{output: "Plan: 3 to add"}
	h := NewTfOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action": "plan",
		"module": "modules/monitoring",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"tf-ops", "plan", "--module", "modules/monitoring"}, mock.lastArgs)
}

func TestTfOpsHandler_Handle_ApplyWithApproval(t *testing.T) {
	mock := &dualMockCLI{output: "Apply complete!"}
	h := NewTfOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action":       "apply",
		"module":       "modules/core",
		"auto_approve": "true",
		"var_file":     "prod.tfvars",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"tf-ops", "apply", "--module", "modules/core", "--var-file", "prod.tfvars", "--auto-approve"}, mock.lastArgs)
}

func TestTfOpsHandler_Handle_MissingModule(t *testing.T) {
	mock := &dualMockCLI{}
	h := NewTfOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action": "plan",
	}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestTfOpsHandler_Handle_NilCLI(t *testing.T) {
	h := NewTfOpsHandler(nil)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action": "plan",
		"module": "modules/core",
	}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestFleetOpsHandler_Tool(t *testing.T) {
	h := NewFleetOpsHandler(nil)
	tool := h.Tool()
	assert.Equal(t, "ironclaw_fleet_ops", tool.Name)
}

func TestFleetOpsHandler_Handle_ListGPUs(t *testing.T) {
	mock := &dualMockCLI{output: `[{"uuid":"gpu-0","name":"RTX 3090"}]`}
	h := NewFleetOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action": "list_gpus",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"fleet-ops", "list_gpus"}, mock.lastArgs)
}

func TestFleetOpsHandler_Handle_CheckOOM(t *testing.T) {
	mock := &dualMockCLI{output: `{"at_risk":0}`}
	h := NewFleetOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action":             "check_oom",
		"vram_threshold_mib": "20480",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"fleet-ops", "check_oom", "--vram-threshold", "20480"}, mock.lastArgs)
}

func TestFleetOpsHandler_Handle_AssignWorkload(t *testing.T) {
	mock := &dualMockCLI{output: `{"gpu":"RTX 3090"}`}
	h := NewFleetOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action":     "assign_workload",
		"model_size": "27b",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"fleet-ops", "assign_workload", "--model-size", "27b"}, mock.lastArgs)
}

func TestGrafanaOpsHandler_Tool(t *testing.T) {
	h := NewGrafanaOpsHandler(nil)
	tool := h.Tool()
	assert.Equal(t, "ironclaw_grafana_provision", tool.Name)
}

func TestGrafanaOpsHandler_Handle_CreateDashboard(t *testing.T) {
	mock := &dualMockCLI{output: `{"uid":"abc123"}`}
	h := NewGrafanaOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action":          "create_dashboard",
		"dashboard_title": "CEO Agent",
		"persona":         "executive-hathaway",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"grafana-ops", "create_dashboard", "--title", "CEO Agent", "--persona", "executive-hathaway"}, mock.lastArgs)
}

func TestGrafanaOpsHandler_Handle_ListDashboards(t *testing.T) {
	mock := &dualMockCLI{output: `[{"title":"CEO Agent"}]`}
	h := NewGrafanaOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action": "list_dashboards",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"grafana-ops", "list_dashboards"}, mock.lastArgs)
}

func TestGrafanaOpsHandler_Handle_Export(t *testing.T) {
	mock := &dualMockCLI{output: `{"panels":[]}`}
	h := NewGrafanaOpsHandler(mock)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action":          "export",
		"dashboard_title": "CEO Agent",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"grafana-ops", "export", "--title", "CEO Agent"}, mock.lastArgs)
}

func TestDualToolNames_Completeness(t *testing.T) {
	names := DualToolNames()
	assert.Len(t, names, 5)
	expected := map[string]bool{
		"ironclaw_gws_run":           true,
		"ironclaw_k8s_ops":           true,
		"ironclaw_tf_ops":            true,
		"ironclaw_fleet_ops":         true,
		"ironclaw_grafana_provision": true,
	}
	for _, n := range names {
		assert.True(t, expected[n], "unexpected tool name: %s", n)
	}
}
