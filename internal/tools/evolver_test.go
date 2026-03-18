package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- StatusTool ---

func TestEvolverHandler_StatusTool_Definition(t *testing.T) {
	h := NewEvolverHandler()
	tool := h.StatusTool()
	assert.Equal(t, "ironclaw_evolver_status", tool.Name)
	assert.Contains(t, tool.Description, "capsule store status")
}

func TestEvolverHandler_HandleStatus_Success(t *testing.T) {
	h := &EvolverHandler{
		run: fakeRunner(`{"capsules":12,"pending_mutations":3,"recent_events":5}`, nil),
		bin: "research-agent",
	}
	result, err := h.HandleStatus(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(result), "capsules")
}

func TestEvolverHandler_HandleStatus_WithOptions(t *testing.T) {
	var captured []string
	h := &EvolverHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandleStatus(context.Background(), callTool(t, map[string]any{
		"data_dir": "/custom/gep",
		"format":   "markdown",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "evolver-status")
	assert.Contains(t, captured, "--data-dir")
	assert.Contains(t, captured, "/custom/gep")
	assert.Contains(t, captured, "--format")
	assert.Contains(t, captured, "markdown")
}

func TestEvolverHandler_HandleStatus_CLIError(t *testing.T) {
	h := &EvolverHandler{
		run: fakeRunner("", fmt.Errorf("capsule store not found")),
		bin: "research-agent",
	}
	result, err := h.HandleStatus(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(result), "evolver-status failed")
}

// --- ProposeTool ---

func TestEvolverHandler_ProposeTool_Definition(t *testing.T) {
	h := NewEvolverHandler()
	tool := h.ProposeTool()
	assert.Equal(t, "ironclaw_evolver_propose", tool.Name)
	assert.Contains(t, tool.Description, "mutation")
}

func TestEvolverHandler_HandlePropose_Success(t *testing.T) {
	h := &EvolverHandler{
		run: fakeRunner(`{"mutation_id":"mut-001","blast_radius":"low","strategy":"harden"}`, nil),
		bin: "research-agent",
	}
	result, err := h.HandlePropose(context.Background(), callTool(t, map[string]any{
		"signal_source": "auto-log",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(result), "mut-001")
}

func TestEvolverHandler_HandlePropose_MissingRequired(t *testing.T) {
	h := NewEvolverHandler()
	result, err := h.HandlePropose(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestEvolverHandler_HandlePropose_WithAllOptions(t *testing.T) {
	var captured []string
	h := &EvolverHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandlePropose(context.Background(), callTool(t, map[string]any{
		"signal_source": "manual",
		"description":   "selector repair for D2L navigation",
		"strategy":      "repair-only",
		"scope":         "tool",
		"data_dir":      "/custom/gep",
		"format":        "json",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "evolver-propose")
	assert.Contains(t, captured, "--signal-source")
	assert.Contains(t, captured, "manual")
	assert.Contains(t, captured, "--description")
	assert.Contains(t, captured, "selector repair for D2L navigation")
	assert.Contains(t, captured, "--strategy")
	assert.Contains(t, captured, "repair-only")
	assert.Contains(t, captured, "--scope")
	assert.Contains(t, captured, "tool")
}

// --- ValidateTool ---

func TestEvolverHandler_ValidateTool_Definition(t *testing.T) {
	h := NewEvolverHandler()
	tool := h.ValidateTool()
	assert.Equal(t, "ironclaw_evolver_validate", tool.Name)
	assert.Contains(t, tool.Description, "sandboxed validation")
}

func TestEvolverHandler_HandleValidate_Success(t *testing.T) {
	h := &EvolverHandler{
		run: fakeRunner(`{"mutation_id":"mut-001","passed":true,"duration_ms":2300}`, nil),
		bin: "research-agent",
	}
	result, err := h.HandleValidate(context.Background(), callTool(t, map[string]any{
		"mutation_id": "mut-001",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(result), "passed")
}

func TestEvolverHandler_HandleValidate_MissingRequired(t *testing.T) {
	h := NewEvolverHandler()
	result, err := h.HandleValidate(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestEvolverHandler_HandleValidate_WithOptions(t *testing.T) {
	var captured []string
	h := &EvolverHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandleValidate(context.Background(), callTool(t, map[string]any{
		"mutation_id": "mut-002",
		"timeout":     "300s",
		"data_dir":    "/custom/gep",
		"format":      "markdown",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "evolver-validate")
	assert.Contains(t, captured, "mut-002")
	assert.Contains(t, captured, "--timeout")
	assert.Contains(t, captured, "300s")
}

func TestEvolverHandler_HandleValidate_CLIError(t *testing.T) {
	h := &EvolverHandler{
		run: fakeRunner("", fmt.Errorf("sandbox creation failed")),
		bin: "research-agent",
	}
	result, err := h.HandleValidate(context.Background(), callTool(t, map[string]any{
		"mutation_id": "mut-003",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(result), "evolver-validate failed")
}

// --- PromoteTool ---

func TestEvolverHandler_PromoteTool_Definition(t *testing.T) {
	h := NewEvolverHandler()
	tool := h.PromoteTool()
	assert.Equal(t, "ironclaw_evolver_promote", tool.Name)
	assert.Contains(t, tool.Description, "Promote")
}

func TestEvolverHandler_HandlePromote_Success(t *testing.T) {
	h := &EvolverHandler{
		run: fakeRunner(`{"mutation_id":"mut-001","promoted":true,"capsule_id":"cap-007"}`, nil),
		bin: "research-agent",
	}
	result, err := h.HandlePromote(context.Background(), callTool(t, map[string]any{
		"mutation_id": "mut-001",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(result), "promoted")
}

func TestEvolverHandler_HandlePromote_MissingRequired(t *testing.T) {
	h := NewEvolverHandler()
	result, err := h.HandlePromote(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestEvolverHandler_HandlePromote_WithOptions(t *testing.T) {
	var captured []string
	h := &EvolverHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandlePromote(context.Background(), callTool(t, map[string]any{
		"mutation_id": "mut-004",
		"sync_mem0":   "false",
		"data_dir":    "/custom/gep",
		"format":      "json",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "evolver-promote")
	assert.Contains(t, captured, "mut-004")
	assert.Contains(t, captured, "--sync-mem0=false")
	assert.Contains(t, captured, "--data-dir")
	assert.Contains(t, captured, "/custom/gep")
}

func TestEvolverHandler_HandlePromote_CLIError(t *testing.T) {
	h := &EvolverHandler{
		run: fakeRunner("", fmt.Errorf("capsule store locked")),
		bin: "research-agent",
	}
	result, err := h.HandlePromote(context.Background(), callTool(t, map[string]any{
		"mutation_id": "mut-005",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(result), "evolver-promote failed")
}
