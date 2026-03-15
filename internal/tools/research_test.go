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

// --- Transcript tool tests ---

func TestResearchHandler_TranscriptTool_Definition(t *testing.T) {
	h := NewResearchHandler()
	tool := h.TranscriptTool()
	assert.Equal(t, "ironclaw_research_transcript", tool.Name)
	assert.Contains(t, tool.Description, "transcribe")
}

func TestResearchHandler_HandleTranscript_Success(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner(`{"download":{"title":"Test Video"},"transcript":{"full_text":"Hello world"}}`, nil),
		bin: "research-agent",
	}

	result, err := h.HandleTranscript(context.Background(), callTool(t, map[string]any{
		"url": "https://youtube.com/watch?v=test",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Hello world")
}

func TestResearchHandler_HandleTranscript_WithOptions(t *testing.T) {
	var capturedArgs []string
	h := &ResearchHandler{
		run: func(_ context.Context, _, _ string, _ string, args ...string) (string, error) {
			capturedArgs = args
			return "{}", nil
		},
		bin: "research-agent",
	}

	_, err := h.HandleTranscript(context.Background(), callTool(t, map[string]any{
		"url":        "https://example.com/video",
		"language":   "zh",
		"model":      "large",
		"summarize":  "true",
		"audio_only": "false",
		"format":     "markdown",
	}))
	require.NoError(t, err)
	assert.Contains(t, capturedArgs, "--language")
	assert.Contains(t, capturedArgs, "zh")
	assert.Contains(t, capturedArgs, "--model")
	assert.Contains(t, capturedArgs, "large")
	assert.Contains(t, capturedArgs, "--summarize")
	assert.Contains(t, capturedArgs, "--audio-only=false")
	assert.Contains(t, capturedArgs, "--format")
	assert.Contains(t, capturedArgs, "markdown")
}

func TestResearchHandler_HandleTranscript_MissingURL(t *testing.T) {
	h := NewResearchHandler()
	result, err := h.HandleTranscript(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestResearchHandler_HandleTranscript_Error(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner("", fmt.Errorf("yt-dlp not found")),
		bin: "research-agent",
	}

	result, err := h.HandleTranscript(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "yt-dlp not found")
}

// --- Extract tool tests ---

func TestResearchHandler_ExtractTool_Definition(t *testing.T) {
	h := NewResearchHandler()
	tool := h.ExtractTool()
	assert.Equal(t, "ironclaw_research_extract", tool.Name)
	assert.Contains(t, tool.Description, "Extract")
}

func TestResearchHandler_HandleExtract_Success(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner(`{"markdown":"# Article\n\nContent here","tokens_saved":500}`, nil),
		bin: "research-agent",
	}

	result, err := h.HandleExtract(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com/article",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "Article")
}

func TestResearchHandler_HandleExtract_WithOptions(t *testing.T) {
	var capturedArgs []string
	h := &ResearchHandler{
		run: func(_ context.Context, _, _ string, _ string, args ...string) (string, error) {
			capturedArgs = args
			return "{}", nil
		},
		bin: "research-agent",
	}

	_, err := h.HandleExtract(context.Background(), callTool(t, map[string]any{
		"url":         "https://example.com",
		"max_tokens":  "4000",
		"compress":    "aggressive",
		"keep_links":  "false",
		"keep_images": "true",
		"format":      "text",
	}))
	require.NoError(t, err)
	assert.Contains(t, capturedArgs, "--max-tokens")
	assert.Contains(t, capturedArgs, "4000")
	assert.Contains(t, capturedArgs, "--compress")
	assert.Contains(t, capturedArgs, "aggressive")
	assert.Contains(t, capturedArgs, "--keep-links=false")
	assert.Contains(t, capturedArgs, "--keep-images")
	assert.Contains(t, capturedArgs, "--format")
	assert.Contains(t, capturedArgs, "text")
}

func TestResearchHandler_HandleExtract_MissingURL(t *testing.T) {
	h := NewResearchHandler()
	result, err := h.HandleExtract(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestResearchHandler_HandleExtract_Error(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner("", fmt.Errorf("extraction failed")),
		bin: "research-agent",
	}

	result, err := h.HandleExtract(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "extraction failed")
}

// --- Scrape with extract/max_tokens tests ---

func TestResearchHandler_HandleScrape_WithExtractOptions(t *testing.T) {
	var capturedArgs []string
	h := &ResearchHandler{
		run: func(_ context.Context, _, _ string, _ string, args ...string) (string, error) {
			capturedArgs = args
			return "{}", nil
		},
		bin: "research-agent",
	}

	_, err := h.HandleScrape(context.Background(), callTool(t, map[string]any{
		"url":        "https://example.com",
		"extract":    "true",
		"max_tokens": "2000",
		"format":     "markdown",
	}))
	require.NoError(t, err)
	assert.Contains(t, capturedArgs, "--extract")
	assert.Contains(t, capturedArgs, "--max-tokens")
	assert.Contains(t, capturedArgs, "2000")
	assert.Contains(t, capturedArgs, "--format")
	assert.Contains(t, capturedArgs, "markdown")
}

// --- Crawl tool tests ---

func TestResearchHandler_CrawlTool_Definition(t *testing.T) {
	h := NewResearchHandler()
	tool := h.CrawlTool()
	assert.Equal(t, "ironclaw_research_crawl", tool.Name)
	assert.Contains(t, tool.Description, "Crawl")
	assert.Contains(t, tool.Description, "BFS")
}

func TestResearchHandler_HandleCrawl_Success(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner(`{"total_pages":5,"pages":[{"url":"https://example.com","title":"Home"}]}`, nil),
		bin: "research-agent",
	}

	result, err := h.HandleCrawl(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com",
	}))
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "total_pages")
}

func TestResearchHandler_HandleCrawl_MissingURL(t *testing.T) {
	h := NewResearchHandler()
	result, err := h.HandleCrawl(context.Background(), callTool(t, map[string]any{}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
}

func TestResearchHandler_HandleCrawl_WithAllOptions(t *testing.T) {
	var capturedArgs []string
	h := &ResearchHandler{
		run: func(_ context.Context, _, _ string, _ string, args ...string) (string, error) {
			capturedArgs = args
			return "{}", nil
		},
		bin: "research-agent",
	}

	_, err := h.HandleCrawl(context.Background(), callTool(t, map[string]any{
		"url":              "https://lms.example.edu/d2l/le/content/12345/Home",
		"depth":            "4",
		"max_pages":        "100",
		"domain":           "lms.example.edu, cdn.example.edu",
		"url_filter":       "/d2l/le/content/",
		"url_exclude":      "\\.(css|js)$",
		"dynamic":          "true",
		"extract":          "true",
		"max_tokens":       "4000",
		"page_delay":       "3s",
		"chrome_debug_url": "ws://localhost:9222",
		"cookie_file":      "/tmp/cookies.json",
		"format":           "markdown",
		"output":           "/tmp/crawl-output.md",
	}))
	require.NoError(t, err)

	assert.Contains(t, capturedArgs, "crawl")
	assert.Contains(t, capturedArgs, "https://lms.example.edu/d2l/le/content/12345/Home")
	assert.Contains(t, capturedArgs, "--depth")
	assert.Contains(t, capturedArgs, "4")
	assert.Contains(t, capturedArgs, "--max-pages")
	assert.Contains(t, capturedArgs, "100")
	assert.Contains(t, capturedArgs, "--domain")
	assert.Contains(t, capturedArgs, "lms.example.edu")
	assert.Contains(t, capturedArgs, "cdn.example.edu")
	assert.Contains(t, capturedArgs, "--url-filter")
	assert.Contains(t, capturedArgs, "/d2l/le/content/")
	assert.Contains(t, capturedArgs, "--url-exclude")
	assert.Contains(t, capturedArgs, "--dynamic")
	assert.Contains(t, capturedArgs, "--extract")
	assert.Contains(t, capturedArgs, "--max-tokens")
	assert.Contains(t, capturedArgs, "4000")
	assert.Contains(t, capturedArgs, "--page-delay")
	assert.Contains(t, capturedArgs, "3s")
	assert.Contains(t, capturedArgs, "--chrome-debug-url")
	assert.Contains(t, capturedArgs, "ws://localhost:9222")
	assert.Contains(t, capturedArgs, "--cookie-file")
	assert.Contains(t, capturedArgs, "/tmp/cookies.json")
	assert.Contains(t, capturedArgs, "--format")
	assert.Contains(t, capturedArgs, "markdown")
	assert.Contains(t, capturedArgs, "--output")
	assert.Contains(t, capturedArgs, "/tmp/crawl-output.md")
}

func TestResearchHandler_HandleCrawl_Error(t *testing.T) {
	h := &ResearchHandler{
		run: fakeRunner("", fmt.Errorf("crawl timed out")),
		bin: "research-agent",
	}

	result, err := h.HandleCrawl(context.Background(), callTool(t, map[string]any{
		"url": "https://example.com",
	}))
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].(mcp.TextContent).Text, "crawl timed out")
}

func TestResearchHandler_HandleCrawl_MinimalOptions(t *testing.T) {
	var capturedArgs []string
	h := &ResearchHandler{
		run: func(_ context.Context, _, _ string, _ string, args ...string) (string, error) {
			capturedArgs = args
			return "{}", nil
		},
		bin: "research-agent",
	}

	_, err := h.HandleCrawl(context.Background(), callTool(t, map[string]any{
		"url":   "https://example.com",
		"depth": "2",
	}))
	require.NoError(t, err)
	assert.Equal(t, "crawl", capturedArgs[0])
	assert.Equal(t, "https://example.com", capturedArgs[1])
	assert.Contains(t, capturedArgs, "--depth")
	assert.Contains(t, capturedArgs, "2")
	assert.NotContains(t, capturedArgs, "--dynamic")
	assert.NotContains(t, capturedArgs, "--extract")
	assert.NotContains(t, capturedArgs, "--chrome-debug-url")
}
