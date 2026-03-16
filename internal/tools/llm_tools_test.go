package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMRouteHandler_Tool(t *testing.T) {
	h := NewLLMRouteHandler(nil)
	tool := h.Tool()
	assert.Equal(t, "ironclaw_llm_route", tool.Name)
}

func TestLLMRouteHandler_NilCLI(t *testing.T) {
	h := NewLLMRouteHandler(nil)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestLLMRouteHandler_DefaultRoute(t *testing.T) {
	cli := &dualMockCLI{output: `{"provider":"local-qwen-9b","reason":"simple task"}`}
	h := NewLLMRouteHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"llm-route"}, cli.lastArgs)
}

func TestLLMRouteHandler_ComplexRoute(t *testing.T) {
	cli := &dualMockCLI{output: `{"provider":"deepseek-v3.2","reason":"complex task"}`}
	h := NewLLMRouteHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"complexity":            "complex",
		"context_len":           "50000",
		"require_large_context": "true",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"llm-route", "--complexity", "complex", "--context-len", "50000", "--gpus"}, cli.lastArgs)
}

func TestLLMRouteHandler_CLIError(t *testing.T) {
	cli := &dualMockCLI{err: fmt.Errorf("timeout")}
	h := NewLLMRouteHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestLLMUsageHandler_Tool(t *testing.T) {
	h := NewLLMUsageHandler(nil)
	tool := h.Tool()
	assert.Equal(t, "ironclaw_llm_usage", tool.Name)
}

func TestLLMUsageHandler_NilCLI(t *testing.T) {
	h := NewLLMUsageHandler(nil)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestLLMUsageHandler_Summary(t *testing.T) {
	cli := &dualMockCLI{output: `{"local-qwen-9b":{"input_tokens":1000}}`}
	h := NewLLMUsageHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"llm-usage"}, cli.lastArgs)
}

func TestLLMBudgetHandler_Tool(t *testing.T) {
	h := NewLLMBudgetHandler(nil)
	tool := h.Tool()
	assert.Equal(t, "ironclaw_llm_budget", tool.Name)
}

func TestLLMBudgetHandler_NilCLI(t *testing.T) {
	h := NewLLMBudgetHandler(nil)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestLLMBudgetHandler_Check(t *testing.T) {
	cli := &dualMockCLI{output: "Daily LLM Budget:\n  Budget:  $5.00\n  Spent:   $0.000000\n  Remaining: $5.000000\n  Exhausted: false"}
	h := NewLLMBudgetHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"llm-budget"}, cli.lastArgs)
}
