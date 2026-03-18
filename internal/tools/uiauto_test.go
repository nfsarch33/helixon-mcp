package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- NavigateTool ---

func TestUIAutoHandler_NavigateTool_Definition(t *testing.T) {
	h := NewUIAutoHandler()
	tool := h.NavigateTool()
	assert.Equal(t, "ironclaw_ui_navigate", tool.Name)
	assert.Contains(t, tool.Description, "Navigate")
	assert.Contains(t, tool.Description, "PageWaiter")
}

func TestUIAutoHandler_HandleNavigate_Success(t *testing.T) {
	h := &UIAutoHandler{
		run: fakeRunner(`{"title":"Example","url":"https://example.com","load_ms":450}`, nil),
		bin: "research-agent",
	}
	result, err := h.HandleNavigate(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(result), "Example")
}

func TestUIAutoHandler_HandleNavigate_MissingURL(t *testing.T) {
	h := NewUIAutoHandler()
	result, err := h.HandleNavigate(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestUIAutoHandler_HandleNavigate_WithOptions(t *testing.T) {
	var captured []string
	h := &UIAutoHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandleNavigate(context.Background(), callTool(t, map[string]any{
		"url":              "https://example.com",
		"wait_selector":    ".main-content",
		"timeout":          "60s",
		"chrome_debug_url": "localhost:9222",
		"extract":          "true",
		"screenshot":       "true",
		"format":           "markdown",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "ui-navigate")
	assert.Contains(t, captured, "--wait-selector")
	assert.Contains(t, captured, ".main-content")
	assert.Contains(t, captured, "--timeout")
	assert.Contains(t, captured, "60s")
	assert.Contains(t, captured, "--chrome-debug-url")
	assert.Contains(t, captured, "--extract")
	assert.Contains(t, captured, "--screenshot")
	assert.Contains(t, captured, "--format")
	assert.Contains(t, captured, "markdown")
}

func TestUIAutoHandler_HandleNavigate_CLIError(t *testing.T) {
	h := &UIAutoHandler{
		run: fakeRunner("", fmt.Errorf("connection refused")),
		bin: "research-agent",
	}
	result, err := h.HandleNavigate(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, resultText(result), "ui-navigate failed")
}

// --- DiscoverTool ---

func TestUIAutoHandler_DiscoverTool_Definition(t *testing.T) {
	h := NewUIAutoHandler()
	tool := h.DiscoverTool()
	assert.Equal(t, "ironclaw_ui_discover", tool.Name)
	assert.Contains(t, tool.Description, "Discover DOM patterns")
}

func TestUIAutoHandler_HandleDiscover_Success(t *testing.T) {
	h := &UIAutoHandler{
		run: fakeRunner(`{"patterns":3,"landmarks":["nav","main","footer"]}`, nil),
		bin: "research-agent",
	}
	result, err := h.HandleDiscover(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(result), "patterns")
}

func TestUIAutoHandler_HandleDiscover_MissingURL(t *testing.T) {
	h := NewUIAutoHandler()
	result, err := h.HandleDiscover(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestUIAutoHandler_HandleDiscover_WithOptions(t *testing.T) {
	var captured []string
	h := &UIAutoHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandleDiscover(context.Background(), callTool(t, map[string]any{
		"url":              "https://deakin.edu.au",
		"chrome_debug_url": "localhost:9222",
		"store":            "false",
		"page_id":          "deakin-home",
		"format":           "json",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "ui-discover")
	assert.Contains(t, captured, "--chrome-debug-url")
	assert.Contains(t, captured, "--store=false")
	assert.Contains(t, captured, "--page-id")
	assert.Contains(t, captured, "deakin-home")
}

// --- HealTool ---

func TestUIAutoHandler_HealTool_Definition(t *testing.T) {
	h := NewUIAutoHandler()
	tool := h.HealTool()
	assert.Equal(t, "ironclaw_ui_heal", tool.Name)
	assert.Contains(t, tool.Description, "repair a broken CSS selector")
}

func TestUIAutoHandler_HandleHeal_Success(t *testing.T) {
	h := &UIAutoHandler{
		run: fakeRunner(`{"repaired":true,"best_candidate":{"selector":"a.d2l-link","strategy":"aria","confidence":0.84}}`, nil),
		bin: "research-agent",
	}
	result, err := h.HandleHeal(context.Background(), callTool(t, map[string]any{
		"url":          "https://d2l.deakin.edu.au",
		"selector":     "div.old-links",
		"element_type": "content_links",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(result), "repaired")
}

func TestUIAutoHandler_HandleHeal_MissingRequired(t *testing.T) {
	h := NewUIAutoHandler()

	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing url", map[string]any{"selector": "div", "element_type": "button"}},
		{"missing selector", map[string]any{"url": "https://example.com", "element_type": "button"}},
		{"missing element_type", map[string]any{"url": "https://example.com", "selector": "div"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := h.HandleHeal(context.Background(), callTool(t, tt.args))
			require.NoError(t, err)
			assert.True(t, result.IsError)
		})
	}
}

func TestUIAutoHandler_HandleHeal_WithOptions(t *testing.T) {
	var captured []string
	h := &UIAutoHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandleHeal(context.Background(), callTool(t, map[string]any{
		"url":              "https://d2l.deakin.edu.au",
		"selector":         "div.old-links",
		"element_type":     "content_links",
		"chrome_debug_url": "localhost:9222",
		"format":           "markdown",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "ui-heal")
	assert.Contains(t, captured, "--selector")
	assert.Contains(t, captured, "div.old-links")
	assert.Contains(t, captured, "--element-type")
	assert.Contains(t, captured, "content_links")
	assert.Contains(t, captured, "--chrome-debug-url")
}

// --- VerifyTool ---

func TestUIAutoHandler_VerifyTool_Definition(t *testing.T) {
	h := NewUIAutoHandler()
	tool := h.VerifyTool()
	assert.Equal(t, "ironclaw_ui_verify", tool.Name)
	assert.Contains(t, tool.Description, "Vision-Language Model")
}

func TestUIAutoHandler_HandleVerify_Success(t *testing.T) {
	h := &UIAutoHandler{
		run: fakeRunner(`{"verified":true,"confidence":0.92,"explanation":"Login button visible in top-right corner"}`, nil),
		bin: "research-agent",
	}
	result, err := h.HandleVerify(context.Background(), callTool(t, map[string]any{
		"url":         "https://example.com",
		"description": "login button is visible",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, resultText(result), "verified")
}

func TestUIAutoHandler_HandleVerify_MissingRequired(t *testing.T) {
	h := NewUIAutoHandler()

	tests := []struct {
		name string
		args map[string]any
	}{
		{"missing url", map[string]any{"description": "check button"}},
		{"missing description", map[string]any{"url": "https://example.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := h.HandleVerify(context.Background(), callTool(t, tt.args))
			require.NoError(t, err)
			assert.True(t, result.IsError)
		})
	}
}

func TestUIAutoHandler_HandleVerify_WithOptions(t *testing.T) {
	var captured []string
	h := &UIAutoHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandleVerify(context.Background(), callTool(t, map[string]any{
		"url":              "https://example.com",
		"description":      "content list has 5 modules",
		"chrome_debug_url": "localhost:9222",
		"model":            "qwen3-vl-72b",
		"format":           "json",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "ui-verify")
	assert.Contains(t, captured, "--description")
	assert.Contains(t, captured, "content list has 5 modules")
	assert.Contains(t, captured, "--model")
	assert.Contains(t, captured, "qwen3-vl-72b")
}

// helper to extract text from tool result
func resultText(r *mcp.CallToolResult) string {
	if len(r.Content) == 0 {
		return ""
	}
	if tc, ok := r.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}
