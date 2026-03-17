package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

type DoctorHandler struct{ cli CLIRunner }

func NewDoctorHandler(cli CLIRunner) *DoctorHandler { return &DoctorHandler{cli: cli} }

func (h *DoctorHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_doctor",
		mcp.WithDescription("Run Mission Control health checks (mc-cli doctor). Returns JSON results."),
		mcp.WithString("suite", mcp.Description("Optional suite filter: all, toolchain, services, stack")),
	)
}

func (h *DoctorHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	args := []string{"doctor", "--json"}
	if s := optionalString(req, "suite"); s != "" {
		args = append(args, "--suite", s)
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("doctor: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type StatusHandler struct{ cli CLIRunner }

func NewStatusHandler(cli CLIRunner) *StatusHandler { return &StatusHandler{cli: cli} }

func (h *StatusHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_status",
		mcp.WithDescription("Get Mission Control Prometheus metrics summary (mc-cli status --json)."),
	)
}

func (h *StatusHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	out, err := h.cli.Run(ctx, "status", "--json")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("status: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type InstallHandler struct{ cli CLIRunner }

func NewInstallHandler(cli CLIRunner) *InstallHandler { return &InstallHandler{cli: cli} }

func (h *InstallHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_install",
		mcp.WithDescription("Check or fix Mission Control dependencies (mc-cli install)."),
		mcp.WithBoolean("fix", mcp.Description("Auto-install missing dependencies")),
	)
}

func (h *InstallHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	args := []string{"install", "--json"}
	if fix, _ := req.Params.Arguments["fix"].(bool); fix {
		args = append(args, "--fix")
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("install: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type DeployHandler struct{ cli CLIRunner }

func NewDeployHandler(cli CLIRunner) *DeployHandler { return &DeployHandler{cli: cli} }

func (h *DeployHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_deploy",
		mcp.WithDescription("Manage Mission Control stack deployment (mc-cli deploy)."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: init, up, down, k8s")),
		mcp.WithString("method", mcp.Description("Deploy method for k8s: terraform, helm, kustomize")),
		mcp.WithBoolean("dry_run", mcp.Description("Dry run mode")),
	)
}

func (h *DeployHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	action, err := requiredString(req, "action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := []string{"deploy", action}
	if m := optionalString(req, "method"); m != "" {
		args = append(args, "--method", m)
	}
	if dr, _ := req.Params.Arguments["dry_run"].(bool); dr {
		args = append(args, "--dry-run")
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("deploy %s: %v\n%s", action, err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type LogsHandler struct{ cli CLIRunner }

func NewLogsHandler(cli CLIRunner) *LogsHandler { return &LogsHandler{cli: cli} }

func (h *LogsHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_logs",
		mcp.WithDescription("View IronClaw container logs (mc-cli logs)."),
		mcp.WithNumber("tail", mcp.Description("Number of lines to tail (default 50)")),
	)
}

func (h *LogsHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	args := []string{"logs"}
	if n, ok := req.Params.Arguments["tail"].(float64); ok && n > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", int(n)))
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("logs: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type SpawnFullHandler struct{ cli CLIRunner }

func NewSpawnFullHandler(cli CLIRunner) *SpawnFullHandler { return &SpawnFullHandler{cli: cli} }

func (h *SpawnFullHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_spawn_full",
		mcp.WithDescription("Spawn a new IronClaw agent instance with full config (mc-cli spawn)."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Agent instance name")),
		mcp.WithString("model", mcp.Description("LLM model (e.g. qwen3.5-27b, qwen3.5-9b)")),
		mcp.WithString("gpu", mcp.Description("GPU device ID")),
		mcp.WithString("persona", mcp.Description("Agent persona template")),
	)
}

func (h *SpawnFullHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	name, err := requiredString(req, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := []string{"spawn", "--name", name}
	if m := optionalString(req, "model"); m != "" {
		args = append(args, "--model", m)
	}
	if g := optionalString(req, "gpu"); g != "" {
		args = append(args, "--gpu", g)
	}
	if p := optionalString(req, "persona"); p != "" {
		args = append(args, "--persona", p)
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("spawn: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type ListAgentsHandler struct{ cli CLIRunner }

func NewListAgentsHandler(cli CLIRunner) *ListAgentsHandler { return &ListAgentsHandler{cli: cli} }

func (h *ListAgentsHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_list_agents",
		mcp.WithDescription("List managed agent instances (mc-cli list --json)."),
	)
}

func (h *ListAgentsHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	out, err := h.cli.Run(ctx, "list", "--json")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("list: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type StopAgentHandler struct{ cli CLIRunner }

func NewStopAgentHandler(cli CLIRunner) *StopAgentHandler { return &StopAgentHandler{cli: cli} }

func (h *StopAgentHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_stop_agent",
		mcp.WithDescription("Stop a managed agent instance (mc-cli stop)."),
		mcp.WithString("name", mcp.Required(), mcp.Description("Agent instance name to stop")),
	)
}

func (h *StopAgentHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	name, err := requiredString(req, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out, err := h.cli.Run(ctx, "stop", name)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("stop %s: %v\n%s", name, err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type GPUStatusHandler struct{ cli CLIRunner }

func NewGPUStatusHandler(cli CLIRunner) *GPUStatusHandler { return &GPUStatusHandler{cli: cli} }

func (h *GPUStatusHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_gpu_status",
		mcp.WithDescription("Get GPU VRAM, temperature, and utilization (mc-cli gpu status --json)."),
	)
}

func (h *GPUStatusHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	out, err := h.cli.Run(ctx, "gpu", "status", "--json")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("gpu status: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type CostSummaryHandler struct{ cli CLIRunner }

func NewCostSummaryHandler(cli CLIRunner) *CostSummaryHandler {
	return &CostSummaryHandler{cli: cli}
}

func (h *CostSummaryHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_cost_summary",
		mcp.WithDescription("Get LLM token cost summary (mc-cli cost summary --json)."),
	)
}

func (h *CostSummaryHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	out, err := h.cli.Run(ctx, "cost", "summary", "--json")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cost summary: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

type MemoryStatsHandler struct{ cli CLIRunner }

func NewMemoryStatsHandler(cli CLIRunner) *MemoryStatsHandler {
	return &MemoryStatsHandler{cli: cli}
}

func (h *MemoryStatsHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_memory_stats",
		mcp.WithDescription("Get Mem0 memory statistics (mc-cli memory stats --json)."),
	)
}

func (h *MemoryStatsHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	out, err := h.cli.Run(ctx, "memory", "stats", "--json")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("memory stats: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}
