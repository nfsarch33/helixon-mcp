package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// ResearchHandler provides MCP tools for the research-agent CLI.
type ResearchHandler struct {
	run commandRunner
	bin string
}

// NewResearchHandler creates a handler that invokes the research-agent binary.
// Uses a custom runner that strips DATABASE_URL from child process environment
// to prevent IronClaw's libsql config from leaking into the research-agent's
// getStore logic, which would incorrectly try a postgres connection.
func NewResearchHandler() *ResearchHandler {
	return &ResearchHandler{run: runResearchCommand, bin: "research-agent"}
}

// runResearchCommand wraps runCommand but strips DATABASE_URL and DATABASE_BACKEND
// from the child process environment so the research-agent uses its local SQLite store.
func runResearchCommand(ctx context.Context, workdir string, stdin string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workdir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	env := os.Environ()
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, "DATABASE_URL=") && !strings.HasPrefix(e, "DATABASE_BACKEND=") {
			filtered = append(filtered, e)
		}
	}
	cmd.Env = filtered

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errText := strings.TrimSpace(stderr.String())
		if errText == "" {
			errText = err.Error()
		}
		return "", fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), errText)
	}

	if text := strings.TrimSpace(stdout.String()); text != "" {
		return text, nil
	}
	return strings.TrimSpace(stderr.String()), nil
}

// ScrapeTool returns the MCP tool for web scraping.
func (h *ResearchHandler) ScrapeTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_scrape",
		mcp.WithDescription("Scrape a web page using Colly (static HTML) or chromedp (dynamic JS). Returns title, content, links, and extracted fields. Optionally extracts article content and converts to clean markdown."),
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
		mcp.WithString("extract",
			mcp.Description("Set to 'true' to extract article content, strip noise (ads/nav), and convert to markdown. Default: false."),
		),
		mcp.WithString("max_tokens",
			mcp.Description("Max token budget for extracted content. Only used when extract=true. '0' = unlimited. Example: '4000'."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default), 'markdown', or 'text'."),
		),
	)
}

// HandleScrape executes a web scrape via the research-agent CLI.
func (h *ResearchHandler) HandleScrape(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := requiredString(req, "url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"scrape", url, "--format", "json"}

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
	if optionalBool(req, "extract") {
		args = append(args, "--extract")
	}
	if mt := optionalString(req, "max_tokens"); mt != "" {
		args = append(args, "--max-tokens", mt)
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
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
// CLI signature: research-agent pdf <url|path> [--output-dir DIR] [--extract]
func (h *ResearchHandler) HandlePDF(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := optionalString(req, "url")
	file := optionalString(req, "file")
	if url == "" && file == "" {
		return mcp.NewToolResultError("either 'url' or 'file' argument is required"), nil
	}

	target := url
	if target == "" {
		target = expandTilde(file)
	}
	args := []string{"pdf", target}

	output := optionalString(req, "output")
	if output == "" {
		output = "/tmp/research-pdf"
	}
	args = append(args, "--output-dir", output)

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
		mcp.WithString("data_dir",
			mcp.Description("Local data store directory. Default: '.research-data'."),
		),
	)
}

// HandleSearch executes a semantic search via the research-agent CLI.
func (h *ResearchHandler) HandleSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := requiredString(req, "query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"search", query}

	if limit := optionalString(req, "limit"); limit != "" {
		args = append(args, "--limit", limit)
	}
	dataDir := optionalString(req, "data_dir")
	if dataDir != "" {
		args = append(args, "--data-dir", expandTilde(dataDir))
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
// CLI signature: research-agent store <file> [--title T] [--tags T] [--source-type T] [--data-dir D]
func (h *ResearchHandler) HandleStore(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, err := requiredString(req, "title")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	content, err := requiredString(req, "content")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	tmpFile, err := os.CreateTemp("", "research-store-*.md")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("create temp file: %v", err)), nil
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return mcp.NewToolResultError(fmt.Sprintf("write temp file: %v", err)), nil
	}
	tmpFile.Close()

	args := []string{"store", tmpFile.Name(), "--title", title, "--source-type", "file"}

	if source := optionalString(req, "source"); source != "" {
		args = append(args, "--source-type", source)
	}
	if tags := optionalString(req, "tags"); tags != "" {
		args = append(args, "--tags", tags)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("store failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
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

// HandlePipeline is currently unsupported -- the CLI has no 'pipeline' subcommand yet.
func (h *ResearchHandler) HandlePipeline(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError("pipeline subcommand is not yet implemented in the research-agent CLI; planned for Sprint R20"), nil
}

// TranscriptTool returns the MCP tool for media download and transcription.
func (h *ResearchHandler) TranscriptTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_transcript",
		mcp.WithDescription("Download media from a URL (YouTube, podcast, lecture), transcribe audio using faster-whisper, and optionally summarize. Returns structured transcript with timestamps."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("Media URL to download and transcribe (YouTube, podcast RSS, direct media link)."),
		),
		mcp.WithString("language",
			mcp.Description("Transcription language code. Default: 'en'."),
		),
		mcp.WithString("model",
			mcp.Description("Whisper model size: tiny, base, small, medium, large. Default: 'base'."),
		),
		mcp.WithString("summarize",
			mcp.Description("Set to 'true' to generate a summary after transcription. Default: false."),
		),
		mcp.WithString("audio_only",
			mcp.Description("Set to 'false' to download video as well. Default: true (audio only)."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default), 'markdown', or 'text'."),
		),
	)
}

// HandleTranscript executes media download and transcription via the research-agent CLI.
func (h *ResearchHandler) HandleTranscript(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := requiredString(req, "url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"transcript", url}

	if lang := optionalString(req, "language"); lang != "" {
		args = append(args, "--language", lang)
	}
	if model := optionalString(req, "model"); model != "" {
		args = append(args, "--model", model)
	}
	if optionalBool(req, "summarize") {
		args = append(args, "--summarize")
	}
	if ao := optionalString(req, "audio_only"); ao == "false" {
		args = append(args, "--audio-only=false")
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("transcript failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// ExtractTool returns the MCP tool for content extraction and cleaning.
func (h *ResearchHandler) ExtractTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_extract",
		mcp.WithDescription("Extract article content from a URL or HTML file. Strips noise (ads, navigation, banners), converts to clean markdown, and optionally compresses to a token budget."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL or local HTML file path to extract content from."),
		),
		mcp.WithString("max_tokens",
			mcp.Description("Max token budget for output. '0' = unlimited. Example: '4000'."),
		),
		mcp.WithString("compress",
			mcp.Description("Compression level: 'none', 'light', 'medium' (default), 'aggressive'."),
		),
		mcp.WithString("keep_links",
			mcp.Description("Preserve links in markdown output. Default: 'true'."),
		),
		mcp.WithString("keep_images",
			mcp.Description("Preserve images in markdown output. Default: 'false'."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default), 'markdown', or 'text'."),
		),
	)
}

// HandleExtract executes content extraction via the research-agent CLI.
func (h *ResearchHandler) HandleExtract(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := requiredString(req, "url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"extract", url}

	if mt := optionalString(req, "max_tokens"); mt != "" {
		args = append(args, "--max-tokens", mt)
	}
	if c := optionalString(req, "compress"); c != "" {
		args = append(args, "--compress", c)
	}
	if kl := optionalString(req, "keep_links"); kl == "false" {
		args = append(args, "--keep-links=false")
	}
	if ki := optionalString(req, "keep_images"); ki == "true" {
		args = append(args, "--keep-images")
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("extract failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// CrawlTool returns the MCP tool for multi-page web crawling.
func (h *ResearchHandler) CrawlTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_crawl",
		mcp.WithDescription("Crawl multiple pages starting from a URL using BFS traversal. Supports domain scoping, depth/page limits, URL filtering, authenticated sessions, and content extraction. Ideal for LMS course capture and multi-page research."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("Starting URL for the crawl."),
		),
		mcp.WithString("depth",
			mcp.Description("Maximum link-following depth. Default: '3'."),
		),
		mcp.WithString("max_pages",
			mcp.Description("Maximum number of pages to crawl. Default: '50'."),
		),
		mcp.WithString("domain",
			mcp.Description("Comma-separated allowed domains for link following. Default: same as start URL."),
		),
		mcp.WithString("url_filter",
			mcp.Description("Regex pattern — only crawl URLs matching this pattern."),
		),
		mcp.WithString("url_exclude",
			mcp.Description("Regex pattern — skip URLs matching this pattern."),
		),
		mcp.WithString("dynamic",
			mcp.Description("Set to 'true' to use chromedp for JS-rendered pages. Default: false."),
		),
		mcp.WithString("extract",
			mcp.Description("Set to 'true' to extract article content from each page. Default: false."),
		),
		mcp.WithString("max_tokens",
			mcp.Description("Max token budget per extracted page. Only used when extract=true. Example: '4000'."),
		),
		mcp.WithString("page_delay",
			mcp.Description("Delay between page fetches. Example: '2s'. Default: '1s'."),
		),
		mcp.WithString("chrome_debug_url",
			mcp.Description("WebSocket URL for Chrome DevTools (e.g. ws://localhost:9222) for authenticated sessions."),
		),
		mcp.WithString("cookie_file",
			mcp.Description("Path to JSON cookie jar for authenticated sessions."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
		mcp.WithString("output",
			mcp.Description("File path to write output. If omitted, returns to stdout."),
		),
	)
}

// HandleCrawl executes a multi-page crawl via the research-agent CLI.
func (h *ResearchHandler) HandleCrawl(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := requiredString(req, "url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"crawl", url}

	if d := optionalString(req, "depth"); d != "" {
		args = append(args, "--depth", d)
	}
	if mp := optionalString(req, "max_pages"); mp != "" {
		args = append(args, "--max-pages", mp)
	}
	if dom := optionalString(req, "domain"); dom != "" {
		for _, d := range strings.Split(dom, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				args = append(args, "--domain", d)
			}
		}
	}
	if uf := optionalString(req, "url_filter"); uf != "" {
		args = append(args, "--url-filter", uf)
	}
	if ue := optionalString(req, "url_exclude"); ue != "" {
		args = append(args, "--url-exclude", ue)
	}
	if optionalBool(req, "dynamic") {
		args = append(args, "--dynamic")
	}
	if optionalBool(req, "extract") {
		args = append(args, "--extract")
	}
	if mt := optionalString(req, "max_tokens"); mt != "" {
		args = append(args, "--max-tokens", mt)
	}
	if pd := optionalString(req, "page_delay"); pd != "" {
		args = append(args, "--page-delay", pd)
	}
	if cdu := optionalString(req, "chrome_debug_url"); cdu != "" {
		args = append(args, "--chrome-debug-url", cdu)
	}
	if cf := optionalString(req, "cookie_file"); cf != "" {
		args = append(args, "--cookie-file", cf)
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}
	if o := optionalString(req, "output"); o != "" {
		args = append(args, "--output", o)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("crawl failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// DeakinTool returns the MCP tool for scraping Deakin D2L course content.
func (h *ResearchHandler) DeakinTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_deakin",
		mcp.WithDescription("Scrape Deakin University D2L course content for all 4 active units (HPS203, HSH206, MMH250, MMM240). Extracts weekly content, assessments with due dates, and unit guides. Requires an authenticated Chrome debug session."),
		mcp.WithString("chrome_debug_url",
			mcp.Required(),
			mcp.Description("Chrome DevTools debug URL (e.g. 'localhost:9222' or full ws:// URL) with an authenticated D2L session."),
		),
		mcp.WithString("output_dir",
			mcp.Description("Output directory for scraped content. Default: ~/ai-agent-business-stack/data/deakin-courses/."),
		),
		mcp.WithString("unit",
			mcp.Description("Filter to a single unit code (e.g. 'HPS203') or unit ID. Default: all 4 units."),
		),
		mcp.WithString("max_pages",
			mcp.Description("Maximum pages to crawl per unit. Default: '200'."),
		),
		mcp.WithString("depth",
			mcp.Description("Maximum link-following depth. Default: '1'."),
		),
		mcp.WithString("extract",
			mcp.Description("Set to 'true' to extract article content from each page. Default: true."),
		),
		mcp.WithString("download_pdfs",
			mcp.Description("Set to 'true' to download and extract PDFs linked from External Resource pages. Default: false."),
		),
		mcp.WithString("download_docs",
			mcp.Description("Set to 'true' to download and extract Word/Office documents (.docx, .doc, .xlsx, .pptx) linked from External Resource pages. Default: false."),
		),
	)
}

// HandleDeakin executes the Deakin D2L course scrape via the research-agent CLI.
func (h *ResearchHandler) HandleDeakin(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	chromeURL, err := requiredString(req, "chrome_debug_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"--chrome-debug-url", chromeURL, "deakin"}

	if od := optionalString(req, "output_dir"); od != "" {
		args = append(args, "--output-dir", expandTilde(od))
	}
	if u := optionalString(req, "unit"); u != "" {
		args = append(args, "--unit", u)
	}
	if mp := optionalString(req, "max_pages"); mp != "" {
		args = append(args, "--max-pages", mp)
	}
	if d := optionalString(req, "depth"); d != "" {
		args = append(args, "--depth", d)
	}
	if ext := optionalString(req, "extract"); ext != "false" {
		args = append(args, "--extract")
	}
	if optionalBool(req, "download_pdfs") {
		args = append(args, "--download-pdfs")
	}
	if optionalBool(req, "download_docs") {
		args = append(args, "--download-docs")
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("deakin scrape failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// AssessmentsTool returns the MCP tool for retrieving Deakin assessment due dates.
func (h *ResearchHandler) AssessmentsTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_research_assessments",
		mcp.WithDescription("Retrieve Deakin assessment due dates from a previous scrape. Returns all assessments sorted by due date across all units, including assignment names, types, due/end dates, and submission status. Can also trigger a fresh assessment-only extraction."),
		mcp.WithString("data_dir",
			mcp.Description("Directory containing scrape output. Default: ~/ai-agent-business-stack/data/deakin-courses/."),
		),
		mcp.WithString("unit",
			mcp.Description("Filter to a single unit code (e.g. 'HPS203'). Default: all units."),
		),
		mcp.WithString("refresh",
			mcp.Description("Set to 'true' to trigger a fresh assessment extraction (requires chrome_debug_url). Default: false."),
		),
		mcp.WithString("chrome_debug_url",
			mcp.Description("Chrome DevTools debug URL for fresh extraction. Only required when refresh=true."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
	)
}

// HandleAssessments retrieves or refreshes Deakin assessment due dates.
func (h *ResearchHandler) HandleAssessments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dataDir := optionalString(req, "data_dir")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = home + "/ai-agent-business-stack/data/deakin-courses"
	} else {
		dataDir = expandTilde(dataDir)
	}

	if optionalBool(req, "refresh") {
		chromeURL := optionalString(req, "chrome_debug_url")
		if chromeURL == "" {
			return mcp.NewToolResultError("chrome_debug_url is required when refresh=true"), nil
		}
		args := []string{"--chrome-debug-url", chromeURL, "deakin", "--output-dir", dataDir, "--extract"}
		if u := optionalString(req, "unit"); u != "" {
			args = append(args, "--unit", u)
		}
		_, err := h.run(ctx, "", "", h.bin, args...)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("assessment refresh failed: %v", err)), nil
		}
	}

	format := optionalString(req, "format")
	unit := optionalString(req, "unit")

	var targetFile string
	if unit != "" {
		if format == "markdown" {
			targetFile = dataDir + "/" + strings.ToUpper(unit) + "/assessments.md"
		} else {
			targetFile = dataDir + "/" + strings.ToUpper(unit) + "/assessments.json"
		}
	} else {
		if format == "markdown" {
			targetFile = dataDir + "/all-assessments.md"
		} else {
			targetFile = dataDir + "/all-assessments.json"
		}
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("read assessments from %s: %v", targetFile, err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}
