package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// UIAutoHandler provides MCP tools for UI automation, pattern discovery,
// self-healing, and VLM visual verification via the research-agent CLI.
type UIAutoHandler struct {
	run commandRunner
	bin string
}

// NewUIAutoHandler creates a handler bridging to research-agent uiauto subcommands.
func NewUIAutoHandler() *UIAutoHandler {
	return &UIAutoHandler{run: runResearchCommand, bin: "research-agent"}
}

// NavigateTool returns the MCP tool for navigating to a URL with full page load detection.
func (h *UIAutoHandler) NavigateTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_ui_navigate",
		mcp.WithDescription("Navigate to a URL using chromedp with PageWaiter for full page load detection (network idle, DOM stable, element visible). Returns page title, URL, load metrics, and optionally extracted content."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL to navigate to."),
		),
		mcp.WithString("wait_selector",
			mcp.Description("CSS selector to wait for before considering page loaded."),
		),
		mcp.WithString("timeout",
			mcp.Description("Navigation timeout. Default: '30s'."),
		),
		mcp.WithString("chrome_debug_url",
			mcp.Description("Chrome DevTools debug URL for authenticated sessions."),
		),
		mcp.WithString("extract",
			mcp.Description("Set to 'true' to extract page content as markdown. Default: false."),
		),
		mcp.WithString("screenshot",
			mcp.Description("Set to 'true' to capture a screenshot. Default: false."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
	)
}

// HandleNavigate executes page navigation via the research-agent CLI.
func (h *UIAutoHandler) HandleNavigate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := requiredString(req, "url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"ui-navigate", url}

	if ws := optionalString(req, "wait_selector"); ws != "" {
		args = append(args, "--wait-selector", ws)
	}
	if t := optionalString(req, "timeout"); t != "" {
		args = append(args, "--timeout", t)
	}
	if cdu := optionalString(req, "chrome_debug_url"); cdu != "" {
		args = append(args, "--chrome-debug-url", cdu)
	}
	if optionalBool(req, "extract") {
		args = append(args, "--extract")
	}
	if optionalBool(req, "screenshot") {
		args = append(args, "--screenshot")
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ui-navigate failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// DiscoverTool returns the MCP tool for DOM pattern discovery on a URL.
func (h *UIAutoHandler) DiscoverTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_ui_discover",
		mcp.WithDescription("Discover DOM patterns on a URL: fingerprints, ARIA landmarks, structural layout, key interactive elements. Stores discovered patterns for future self-healing reference."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL to discover patterns on."),
		),
		mcp.WithString("chrome_debug_url",
			mcp.Description("Chrome DevTools debug URL for authenticated sessions."),
		),
		mcp.WithString("store",
			mcp.Description("Set to 'true' to persist discovered patterns to the pattern store. Default: true."),
		),
		mcp.WithString("page_id",
			mcp.Description("Identifier for this page in the pattern store. Default: derived from URL."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
	)
}

// HandleDiscover executes DOM pattern discovery via the research-agent CLI.
func (h *UIAutoHandler) HandleDiscover(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := requiredString(req, "url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"ui-discover", url}

	if cdu := optionalString(req, "chrome_debug_url"); cdu != "" {
		args = append(args, "--chrome-debug-url", cdu)
	}
	if store := optionalString(req, "store"); store == "false" {
		args = append(args, "--store=false")
	}
	if pid := optionalString(req, "page_id"); pid != "" {
		args = append(args, "--page-id", pid)
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ui-discover failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// HealTool returns the MCP tool for self-healing a broken CSS selector.
func (h *UIAutoHandler) HealTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_ui_heal",
		mcp.WithDescription("Attempt to repair a broken CSS selector using the multi-strategy fallback chain (CSS variants, XPath, ARIA, text content, VLM). Returns the best replacement selector with confidence score and strategy used."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL of the page where the selector is broken."),
		),
		mcp.WithString("selector",
			mcp.Required(),
			mcp.Description("The broken CSS selector to repair."),
		),
		mcp.WithString("element_type",
			mcp.Required(),
			mcp.Description("Semantic element type (e.g. 'navigation', 'content_links', 'login_button')."),
		),
		mcp.WithString("chrome_debug_url",
			mcp.Description("Chrome DevTools debug URL for authenticated sessions."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
	)
}

// HandleHeal executes selector self-healing via the research-agent CLI.
func (h *UIAutoHandler) HandleHeal(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := requiredString(req, "url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	selector, err := requiredString(req, "selector")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	elementType, err := requiredString(req, "element_type")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"ui-heal", url, "--selector", selector, "--element-type", elementType}

	if cdu := optionalString(req, "chrome_debug_url"); cdu != "" {
		args = append(args, "--chrome-debug-url", cdu)
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ui-heal failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// VerifyTool returns the MCP tool for VLM visual verification.
func (h *UIAutoHandler) VerifyTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_ui_verify",
		mcp.WithDescription("Visually verify a UI element or page state using a Vision-Language Model (VLM). Takes a screenshot and asks the VLM to confirm whether the expected element or state is present. Useful as a final verification when DOM-based checks are inconclusive."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL of the page to verify."),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Natural language description of what to verify (e.g. 'login button is visible', 'content list has 5 modules')."),
		),
		mcp.WithString("chrome_debug_url",
			mcp.Description("Chrome DevTools debug URL for authenticated sessions."),
		),
		mcp.WithString("model",
			mcp.Description("VLM model to use. Default: 'qwen3-vl' via llm-cluster-router."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
	)
}

// HandleVerify executes VLM visual verification via the research-agent CLI.
func (h *UIAutoHandler) HandleVerify(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url, err := requiredString(req, "url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	desc, err := requiredString(req, "description")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"ui-verify", url, "--description", desc}

	if cdu := optionalString(req, "chrome_debug_url"); cdu != "" {
		args = append(args, "--chrome-debug-url", cdu)
	}
	if m := optionalString(req, "model"); m != "" {
		args = append(args, "--model", m)
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ui-verify failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}
