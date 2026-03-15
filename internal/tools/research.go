package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// ResearchHandler provides MCP tools for the research-agent CLI.
type ResearchHandler struct {
	run commandRunner
	bin string
}

// NewResearchHandler creates a handler that invokes the research-agent binary.
func NewResearchHandler() *ResearchHandler {
	return &ResearchHandler{run: runCommand, bin: "research-agent"}
}

// ScrapeTool returns the MCP tool for web scraping.
func (h *ResearchHandler) ScrapeTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_scrape",
		mcp.WithDescription("Scrape a web page using Colly (static HTML) or chromedp (dynamic JS). Returns title, content, links, and extracted fields."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL to scrape."),
		),
		mcp.WithString("selectors",
			mcp.Description(`JSON object mapping field names to CSS selectors. Example: {"titles":"h2","prices":".price"}`),
		),
		mcp.WithString("dynamic",
			mcp.Description("Set to 'true' to use chromedp for JS-rendered pages. Default: false (Colly)."),
		),
		mcp.WithString("rate_limit",
			mcp.Description("Minimum delay between requests. Example: '2s'. Default: '1s'."),
		),
		mcp.WithString("retries",
			mcp.Description("Number of retry attempts on failure. Default: '1'."),
		),
	)
}

// HandleScrape executes a web scrape via the research-agent CLI.
func (h *ResearchHandler) HandleScrape(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := requiredString(req, "url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"scrape", "--url", url, "--json"}

	if sel := optionalString(req, "selectors"); sel != "" {
		args = append(args, "--selectors", sel)
	}
	if optionalBool(req, "dynamic") {
		args = append(args, "--dynamic")
	}
	if rl := optionalString(req, "rate_limit"); rl != "" {
		args = append(args, "--rate-limit", rl)
	}
	if retries := optionalString(req, "retries"); retries != "" {
		args = append(args, "--retries", retries)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("scrape failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// PDFTool returns the MCP tool for PDF download and extraction.
func (h *ResearchHandler) PDFTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_pdf",
		mcp.WithDescription("Download a PDF from a URL and extract its text content. Returns extracted text and metadata."),
		mcp.WithString("url",
			mcp.Description("URL of the PDF to download and extract. Provide either url or file."),
		),
		mcp.WithString("file",
			mcp.Description("Local file path of the PDF to extract. Provide either url or file."),
		),
		mcp.WithString("output",
			mcp.Description("Output directory for extracted content. Default: /tmp/research-pdf."),
		),
	)
}

// HandlePDF executes PDF extraction via the research-agent CLI.
func (h *ResearchHandler) HandlePDF(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := optionalString(req, "url")
	file := optionalString(req, "file")
	if url == "" && file == "" {
		return mcp.NewToolResultError("either 'url' or 'file' argument is required"), nil
	}

	args := []string{"pdf"}
	if url != "" {
		args = append(args, "--url", url)
	}
	if file != "" {
		args = append(args, "--file", file)
	}

	output := optionalString(req, "output")
	if output == "" {
		output = "/tmp/research-pdf"
	}
	args = append(args, "--output", output)

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pdf extraction failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// SearchTool returns the MCP tool for semantic search across stored research.
func (h *ResearchHandler) SearchTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_search",
		mcp.WithDescription("Search stored research documents using semantic similarity. Returns matching documents with scores."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Natural language search query."),
		),
		mcp.WithString("limit",
			mcp.Description("Maximum number of results. Default: '10'."),
		),
		mcp.WithString("tags",
			mcp.Description("Comma-separated tag filter. Example: 'market,fitness'."),
		),
	)
}

// HandleSearch executes a semantic search via the research-agent CLI.
func (h *ResearchHandler) HandleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := requiredString(req, "query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"search", "--query", query, "--json"}

	if limit := optionalString(req, "limit"); limit != "" {
		args = append(args, "--limit", limit)
	}
	if tags := optionalString(req, "tags"); tags != "" {
		args = append(args, "--tags", tags)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// StoreTool returns the MCP tool for storing research documents.
func (h *ResearchHandler) StoreTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_store",
		mcp.WithDescription("Store a research document in the datastore with embeddings for future semantic search."),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Title of the research document."),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Text content to store."),
		),
		mcp.WithString("source",
			mcp.Description("Source URL or identifier."),
		),
		mcp.WithString("tags",
			mcp.Description("Comma-separated tags. Example: 'seo,competitor,fitness'."),
		),
	)
}

// HandleStore stores a research document via the research-agent CLI.
func (h *ResearchHandler) HandleStore(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := requiredString(req, "title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	content, err := requiredString(req, "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"store", "--title", title}

	if source := optionalString(req, "source"); source != "" {
		args = append(args, "--source", source)
	}
	if tags := optionalString(req, "tags"); tags != "" {
		args = append(args, "--tags", tags)
	}

	out, err := h.run(ctx, "", "", h.bin, append(args, "--content", content)...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("store failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type pipelineRunResult struct {
	Pipeline string `json:"pipeline"`
	Status   string `json:"status"`
	Output   string `json:"output"`
}

// PipelineTool returns the MCP tool for running research pipelines.
func (h *ResearchHandler) PipelineTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_pipeline",
		mcp.WithDescription("Run a research pipeline defined in YAML. Pipelines execute concurrent stages with dependency ordering."),
		mcp.WithString("pipeline_file",
			mcp.Required(),
			mcp.Description("Path to the pipeline YAML file."),
		),
	)
}

// HandlePipeline executes a research pipeline.
func (h *ResearchHandler) HandlePipeline(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	pipelineFile, err := requiredString(req, "pipeline_file")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	out, err := h.run(ctx, "", "", h.bin, "pipeline", "--file", pipelineFile, "--json")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("pipeline failed: %v", err)), nil
	}

	result := pipelineRunResult{
		Pipeline: pipelineFile,
		Status:   "completed",
		Output:   strings.TrimSpace(out),
	}
	b, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}
