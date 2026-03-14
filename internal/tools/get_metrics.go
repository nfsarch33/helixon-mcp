package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// Default PromQL queries for key agent metrics.
var defaultMetricQueries = map[string]string{
	"llm_calls_total":     `sum(ironclaw_llm_calls_total)`,
	"llm_tokens_total":    `sum(ironclaw_llm_tokens_total)`,
	"gpu_vram_used_bytes": `DCGM_FI_DEV_FB_USED`,
	"gpu_vram_free_bytes": `DCGM_FI_DEV_FB_FREE`,
	"router_health":       `up{job="llm-cluster-router"}`,
	"active_jobs":         `ironclaw_active_jobs`,
}

// GetMetricsHandler handles the ironclaw_get_metrics MCP tool.
type GetMetricsHandler struct {
	prom PrometheusQuerier
}

// NewGetMetricsHandler creates a new GetMetricsHandler.
func NewGetMetricsHandler(prom PrometheusQuerier) *GetMetricsHandler {
	return &GetMetricsHandler{prom: prom}
}

// Tool returns the ironclaw_get_metrics MCP tool definition.
func (h *GetMetricsHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_get_metrics",
		mcp.WithDescription("Query Prometheus for key IronClaw agent metrics: LLM calls, token usage, GPU VRAM, router health, and active jobs. Optionally pass a custom PromQL query."),
		mcp.WithString("query",
			mcp.Description("Optional custom PromQL query. If omitted, returns all default agent metrics."),
		),
	)
}

// Handle executes the get metrics tool.
func (h *GetMetricsHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	customQuery := optionalString(req, "query")

	if customQuery != "" {
		result, err := h.prom.Query(ctx, customQuery)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("prometheus query failed: %v", err)), nil
		}
		return jsonResult(map[string]string{"query": customQuery, "result": result})
	}

	results := make(map[string]string, len(defaultMetricQueries))
	var errs []string
	for name, query := range defaultMetricQueries {
		result, err := h.prom.Query(ctx, query)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		results[name] = result
	}

	out := map[string]any{"metrics": results}
	if len(errs) > 0 {
		out["errors"] = strings.Join(errs, "; ")
	}
	return jsonResult(out)
}
