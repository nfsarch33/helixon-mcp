package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var dualToolOpsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "ironclaw_dual_tool_ops_total",
		Help: "Total dual-ecosystem tool operations executed via MCP.",
	},
	[]string{"tool", "action", "status"},
)

// K8sOpsHandler exposes Kubernetes operations as an MCP tool, delegating to mc-cli k8s-ops.
type K8sOpsHandler struct {
	cli CLIRunner
}

func NewK8sOpsHandler(cli CLIRunner) *K8sOpsHandler {
	return &K8sOpsHandler{cli: cli}
}

func (h *K8sOpsHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_k8s_ops",
		mcp.WithDescription("Query Kubernetes cluster state: pods, services, logs, and node health."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: get_pods, get_services, get_logs, get_nodes, describe_pod")),
		mcp.WithString("namespace", mcp.Description("Kubernetes namespace (default: ironclaw)")),
		mcp.WithString("resource_name", mcp.Description("Name of the specific resource (pod, service)")),
		mcp.WithString("tail_lines", mcp.Description("Number of log lines to return (default: 50)")),
	)
}

func (h *K8sOpsHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	action, err := requiredString(req, "action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"k8s-ops", action}
	if ns := optionalString(req, "namespace"); ns != "" {
		args = append(args, "--namespace", ns)
	}
	if rn := optionalString(req, "resource_name"); rn != "" {
		args = append(args, "--name", rn)
	}
	if tl := optionalString(req, "tail_lines"); tl != "" {
		args = append(args, "--tail", tl)
	}

	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		dualToolOpsTotal.WithLabelValues("k8s_ops", action, "error").Inc()
		return mcp.NewToolResultError(fmt.Sprintf("k8s-ops %s: %v\n%s", action, err, out)), nil
	}
	dualToolOpsTotal.WithLabelValues("k8s_ops", action, "success").Inc()
	return mcp.NewToolResultText(out), nil
}

// TfOpsHandler exposes Terraform operations as an MCP tool, delegating to mc-cli tf-ops.
type TfOpsHandler struct {
	cli CLIRunner
}

func NewTfOpsHandler(cli CLIRunner) *TfOpsHandler {
	return &TfOpsHandler{cli: cli}
}

func (h *TfOpsHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_tf_ops",
		mcp.WithDescription("Run Terraform operations for infrastructure provisioning: plan, apply, output, state."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: plan, apply, output, state_list, destroy")),
		mcp.WithString("module", mcp.Required(), mcp.Description("Terraform module path (e.g. modules/inference, modules/monitoring)")),
		mcp.WithString("var_file", mcp.Description("Path to .tfvars file")),
		mcp.WithString("auto_approve", mcp.Description("Set to true to auto-approve apply/destroy (default: false)")),
	)
}

func (h *TfOpsHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	action, err := requiredString(req, "action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	module, err := requiredString(req, "module")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"tf-ops", action, "--module", module}
	if vf := optionalString(req, "var_file"); vf != "" {
		args = append(args, "--var-file", vf)
	}
	if aa := optionalString(req, "auto_approve"); aa == "true" || aa == "1" {
		args = append(args, "--auto-approve")
	}

	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		dualToolOpsTotal.WithLabelValues("tf_ops", action, "error").Inc()
		return mcp.NewToolResultError(fmt.Sprintf("tf-ops %s: %v\n%s", action, err, out)), nil
	}
	dualToolOpsTotal.WithLabelValues("tf_ops", action, "success").Inc()
	return mcp.NewToolResultText(out), nil
}

// FleetOpsHandler exposes GPU fleet operations as an MCP tool, delegating to mc-cli fleet-ops.
type FleetOpsHandler struct {
	cli CLIRunner
}

func NewFleetOpsHandler(cli CLIRunner) *FleetOpsHandler {
	return &FleetOpsHandler{cli: cli}
}

func (h *FleetOpsHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_fleet_ops",
		mcp.WithDescription("Manage GPU fleet: query GPU health, VRAM usage, schedule workloads, check OOM risk."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: list_gpus, check_oom, get_vram, assign_workload, list_nodes")),
		mcp.WithString("gpu_uuid", mcp.Description("Specific GPU UUID to query")),
		mcp.WithString("model_size", mcp.Description("Model size for scheduling (e.g. 9b, 27b)")),
		mcp.WithString("vram_threshold_mib", mcp.Description("VRAM threshold in MiB for OOM check")),
	)
}

func (h *FleetOpsHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	action, err := requiredString(req, "action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"fleet-ops", action}
	if uuid := optionalString(req, "gpu_uuid"); uuid != "" {
		args = append(args, "--gpu-uuid", uuid)
	}
	if ms := optionalString(req, "model_size"); ms != "" {
		args = append(args, "--model-size", ms)
	}
	if vt := optionalString(req, "vram_threshold_mib"); vt != "" {
		args = append(args, "--vram-threshold", vt)
	}

	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		dualToolOpsTotal.WithLabelValues("fleet_ops", action, "error").Inc()
		return mcp.NewToolResultError(fmt.Sprintf("fleet-ops %s: %v\n%s", action, err, out)), nil
	}
	dualToolOpsTotal.WithLabelValues("fleet_ops", action, "success").Inc()
	return mcp.NewToolResultText(out), nil
}

// GrafanaOpsHandler exposes Grafana provisioning as an MCP tool, delegating to mc-cli grafana-ops.
type GrafanaOpsHandler struct {
	cli CLIRunner
}

func NewGrafanaOpsHandler(cli CLIRunner) *GrafanaOpsHandler {
	return &GrafanaOpsHandler{cli: cli}
}

func (h *GrafanaOpsHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_grafana_provision",
		mcp.WithDescription("Provision or update Grafana dashboards dynamically for IronClaw agents and infrastructure."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: create_dashboard, list_dashboards, export")),
		mcp.WithString("dashboard_title", mcp.Description("Dashboard title")),
		mcp.WithString("persona", mcp.Description("Agent persona name for per-agent dashboards")),
		mcp.WithString("panels_json", mcp.Description("JSON array of panel definitions")),
	)
}

func (h *GrafanaOpsHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	action, err := requiredString(req, "action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"grafana-ops", action}
	if title := optionalString(req, "dashboard_title"); title != "" {
		args = append(args, "--title", title)
	}
	if persona := optionalString(req, "persona"); persona != "" {
		args = append(args, "--persona", persona)
	}
	if panels := optionalString(req, "panels_json"); panels != "" {
		args = append(args, "--panels", panels)
	}

	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		dualToolOpsTotal.WithLabelValues("grafana_provision", action, "error").Inc()
		return mcp.NewToolResultError(fmt.Sprintf("grafana-ops %s: %v\n%s", action, err, out)), nil
	}
	dualToolOpsTotal.WithLabelValues("grafana_provision", action, "success").Inc()
	return mcp.NewToolResultText(out), nil
}

// DualToolNames returns the tool names from the dual-ecosystem catalog for parity verification.
func DualToolNames() []string {
	return []string{
		"ironclaw_gws_run",
		"ironclaw_k8s_ops",
		"ironclaw_tf_ops",
		"ironclaw_fleet_ops",
		"ironclaw_grafana_provision",
	}
}

// AllDualToolNames returns a joined string for logging.
func AllDualToolNames() string {
	return strings.Join(DualToolNames(), ", ")
}
