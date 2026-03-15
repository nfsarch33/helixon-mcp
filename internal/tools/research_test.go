package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeRunner(output string, err error) commandRunner {
	return func(_ context.Context, _ string, _ string, _ string, _ ...string) (string, error) {
		return output, err
	}
}

func captureRunner(captured *[]string) commandRunner {
	return func(_ context.Context, _ string, _ string, name string, args ...string) (string, error) {
		*captured = append(*captured, name)
		*captured = append(*captured, args...)
		return `{"ok":true}`, nil
	}
}

func callTool(t *testing.T, args map[string]any) mcp.CallToolRequest {
	t.Helper()
	return mcp.CallToolRequest{
		Params: struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken `json:"progressToken,omitempty"`
			} `json:"_meta,omitempty"`
		}{
			Arguments: args,
		},
	}
}

func TestResearchHandler_ScrapeTool_Definition(t *testing.T) {
	h := NewResearchHandler()
	tool := h.ScrapeTool()
	assert.Equal(t, "ironclaw_research_scrape", tool.Name)
	assert.Contains(t, tool.Description, "Scrape")
}

func TestResearchHandler_HandleScrape_Success(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner(`{"title":"Test","content":"Hello"}`, nil),
		bin: "research-agent",
	}

	result, err := h.HandleScrape(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Test")
}

func TestResearchHandler_HandleScrape_MissingURL(t *testing.T) {
	h := NewResearchHandler()
	result, err := h.HandleScrape(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestResearchHandler_HandleScrape_WithOptions(t *testing.T) {
	var captured []string
	h := &ResearchHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandleScrape(context.Background(), callTool(t, map[string]any{
		"url":        "https://example.com",
		"selectors":  `{"title":"h1"}`,
		"dynamic":    "true",
		"rate_limit": "2s",
		"retries":    "3",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "--selectors")
	assert.Contains(t, captured, "--dynamic")
	assert.Contains(t, captured, "--rate-limit")
	assert.Contains(t, captured, "--retries")
}

func TestResearchHandler_HandleScrape_Error(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner("", fmt.Errorf("connection refused")),
		bin: "research-agent",
	}

	result, err := h.HandleScrape(context.Background(), callTool(t, map[string]any{
		"url": "https://unreachable.invalid",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "connection refused")
}

func TestResearchHandler_HandlePDF_Success(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner(`{"pages":3,"text":"extracted content"}`, nil),
		bin: "research-agent",
	}

	result, err := h.HandlePDF(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com/report.pdf",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "extracted content")
}

func TestResearchHandler_HandlePDF_LocalFile(t *testing.T) {
	var captured []string
	h := &ResearchHandler{run: captureRunner(&captured), bin: "research-agent"}

	_, err := h.HandlePDF(context.Background(), callTool(t, map[string]any{
		"file":   "/tmp/test.pdf",
		"output": "/tmp/out",
	}))
	require.NoError(t, err)
	assert.Contains(t, captured, "--file")
	assert.Contains(t, captured, "/tmp/test.pdf")
	assert.Contains(t, captured, "--output")
	assert.Contains(t, captured, "/tmp/out")
}

func TestResearchHandler_HandlePDF_NoArgs(t *testing.T) {
	h := NewResearchHandler()
	result, err := h.HandlePDF(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "required")
}

func TestResearchHandler_HandleSearch_Success(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner(`[{"title":"Report 1","score":0.92}]`, nil),
		bin: "research-agent",
	}

	result, err := h.HandleSearch(context.Background(), callTool(t, map[string]any{
		"query": "fitness equipment",
		"limit": "5",
		"tags":  "market,fitness",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Report 1")
}

func TestResearchHandler_HandleSearch_MissingQuery(t *testing.T) {
	h := NewResearchHandler()
	result, err := h.HandleSearch(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestResearchHandler_HandleStore_Success(t *testing.T) {
	var captured []string
	h := &ResearchHandler{run: captureRunner(&captured), bin: "research-agent"}

	result, err := h.HandleStore(context.Background(), callTool(t, map[string]any{
		"title":   "Q1 Market Report",
		"content": "Lorem ipsum dolor sit amet",
		"source":  "https://example.com/report",
		"tags":    "market,q1",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, captured, "--title")
	assert.Contains(t, captured, "--source")
	assert.Contains(t, captured, "--tags")
}

func TestResearchHandler_HandleStore_MissingTitle(t *testing.T) {
	h := NewResearchHandler()
	result, err := h.HandleStore(context.Background(), callTool(t, map[string]any{
		"content": "some text",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestResearchHandler_HandleStore_MissingContent(t *testing.T) {
	h := NewResearchHandler()
	result, err := h.HandleStore(context.Background(), callTool(t, map[string]any{
		"title": "Report",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestResearchHandler_HandlePipeline_Success(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner(`{"stages_completed":3}`, nil),
		bin: "research-agent",
	}

	result, err := h.HandlePipeline(context.Background(), callTool(t, map[string]any{
		"pipeline_file": "/path/to/pipeline.yml",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "completed")
}

func TestResearchHandler_HandlePipeline_MissingFile(t *testing.T) {
	h := NewResearchHandler()
	result, err := h.HandlePipeline(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestResearchHandler_HandlePipeline_Error(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner("", fmt.Errorf("pipeline stage failed")),
		bin: "research-agent",
	}

	result, err := h.HandlePipeline(context.Background(), callTool(t, map[string]any{
		"pipeline_file": "/path/to/pipeline.yml",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "pipeline stage failed")
}
