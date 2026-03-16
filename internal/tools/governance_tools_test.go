package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGovernanceHandler_Tool(t *testing.T) {
	h := NewGovernanceHandler(nil)
	tool := h.Tool()
	assert.Equal(t, "ironclaw_governance", tool.Name)
}

func TestGovernanceHandler_NilCLI(t *testing.T) {
	h := NewGovernanceHandler(nil)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{"action": "list-pending"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestGovernanceHandler_MissingAction(t *testing.T) {
	h := NewGovernanceHandler(&dualMockCLI{output: ""})
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestGovernanceHandler_ListPending(t *testing.T) {
	cli := &dualMockCLI{output: `[{"id":"appr-1","status":"pending"}]`}
	h := NewGovernanceHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{"action": "list-pending"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"governance", "list-pending"}, cli.lastArgs)
}

func TestGovernanceHandler_Approve(t *testing.T) {
	cli := &dualMockCLI{output: "approved"}
	h := NewGovernanceHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action":      "approve",
		"request_id":  "appr-1",
		"approved_by": "jason",
		"reason":      "looks good",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, cli.lastArgs, "--id")
	assert.Contains(t, cli.lastArgs, "appr-1")
	assert.Contains(t, cli.lastArgs, "--by")
	assert.Contains(t, cli.lastArgs, "jason")
}

func TestGovernanceHandler_RiskClassify(t *testing.T) {
	cli := &dualMockCLI{output: "Tool: tf_ops\nAction: destroy\nRisk: critical"}
	h := NewGovernanceHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action":      "risk-classify",
		"tool":        "ironclaw_tf_ops",
		"tool_action": "destroy",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Contains(t, cli.lastArgs, "--tool")
	assert.Contains(t, cli.lastArgs, "--action")
}

func TestGovernanceHandler_CLIError(t *testing.T) {
	cli := &dualMockCLI{err: fmt.Errorf("connection refused")}
	h := NewGovernanceHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{"action": "list-pending"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}
