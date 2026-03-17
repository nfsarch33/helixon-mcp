package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func cliHandler(name, desc string, args []string) (mcp.Tool, func(CLIRunner) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool(name, mcp.WithDescription(desc))
	handler := func(cli CLIRunner) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			if cli == nil {
				return mcp.NewToolResultError("mc-cli not configured"), nil
			}
			out, err := cli.Run(ctx, args...)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("%s: %v\n%s", name, err, out)), nil
			}
			return mcp.NewToolResultText(out), nil
		}
	}
	return tool, handler
}

// GenericCLIHandler wraps a mc-cli subcommand as an MCP tool with a single required action parameter.
type GenericCLIHandler struct {
	cli      CLIRunner
	toolName string
	desc     string
	subcmd   string
}

func NewGenericCLIHandler(cli CLIRunner, toolName, desc, subcmd string) *GenericCLIHandler {
	return &GenericCLIHandler{cli: cli, toolName: toolName, desc: desc, subcmd: subcmd}
}

func (h *GenericCLIHandler) Tool() mcp.Tool {
	return mcp.NewTool(h.toolName,
		mcp.WithDescription(h.desc),
		mcp.WithString("action", mcp.Required(), mcp.Description("Sub-action to perform")),
		mcp.WithString("args", mcp.Description("Additional arguments (space-separated)")),
	)
}

func (h *GenericCLIHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	action, err := requiredString(req, "action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := []string{h.subcmd, action}
	if extra := optionalString(req, "args"); extra != "" {
		args = append(args, strings.Fields(extra)...)
	}
	out, cliErr := h.cli.Run(ctx, args...)
	if cliErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("%s %s: %v\n%s", h.subcmd, action, cliErr, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// FleetHandler wraps fleet operations: register, list, remove, health, drain.
type FleetHandler struct{ *GenericCLIHandler }

func NewFleetHandler(cli CLIRunner) *FleetHandler {
	return &FleetHandler{NewGenericCLIHandler(cli, "ironclaw_fleet_ops_full",
		"Fleet node management (mc-cli fleet <action>). Actions: register, list, remove, health, drain, undrain.",
		"fleet")}
}

// RoutineHandler wraps routine operations: list, load, trigger, validate.
type RoutineHandler struct{ *GenericCLIHandler }

func NewRoutineHandler(cli CLIRunner) *RoutineHandler {
	return &RoutineHandler{NewGenericCLIHandler(cli, "ironclaw_routine_ops",
		"Routine management (mc-cli routine <action>). Actions: list, load, trigger, validate, sync, diff.",
		"routine")}
}

// A2AFullHandler wraps a2a operations: list, delegate, status.
type A2AFullHandler struct{ *GenericCLIHandler }

func NewA2AFullHandler(cli CLIRunner) *A2AFullHandler {
	return &A2AFullHandler{NewGenericCLIHandler(cli, "ironclaw_a2a_ops",
		"A2A agent communication (mc-cli a2a <action>). Actions: list, delegate, status.",
		"a2a")}
}

// SnapshotHandler wraps snapshot operations: take, list.
type SnapshotHandler struct{ *GenericCLIHandler }

func NewSnapshotHandler(cli CLIRunner) *SnapshotHandler {
	return &SnapshotHandler{NewGenericCLIHandler(cli, "ironclaw_snapshot_ops",
		"System snapshot management (mc-cli snapshot <action>). Actions: take, list.",
		"snapshot")}
}

// RecoverHandler wraps recovery: export-state, restore-state, recover.
type RecoverHandler struct{ cli CLIRunner }

func NewRecoverHandler(cli CLIRunner) *RecoverHandler { return &RecoverHandler{cli: cli} }

func (h *RecoverHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_recover",
		mcp.WithDescription("Crash recovery (mc-cli recover or export-state/restore-state)."),
		mcp.WithString("action", mcp.Required(), mcp.Description("Action: export-state, restore-state, recover")),
		mcp.WithString("from", mcp.Description("Snapshot path for restore")),
		mcp.WithBoolean("dry_run", mcp.Description("Dry run mode")),
	)
}

func (h *RecoverHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	action, err := requiredString(req, "action")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := []string{action}
	if f := optionalString(req, "from"); f != "" {
		args = append(args, "--from", f)
	}
	if dr, _ := req.Params.Arguments["dry_run"].(bool); dr {
		args = append(args, "--dry-run")
	}
	out, cliErr := h.cli.Run(ctx, args...)
	if cliErr != nil {
		return mcp.NewToolResultError(fmt.Sprintf("%s: %v\n%s", action, cliErr, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// WorkspaceHandler wraps workspace memory operations.
type WorkspaceHandler struct{ *GenericCLIHandler }

func NewWorkspaceHandler(cli CLIRunner) *WorkspaceHandler {
	return &WorkspaceHandler{NewGenericCLIHandler(cli, "ironclaw_workspace",
		"Workspace memory operations (mc-cli workspace <action>). Actions: search, read, write, list, tree.",
		"workspace")}
}

// CRMFullHandler wraps CRM operations beyond prep-meeting.
type CRMFullHandler struct{ *GenericCLIHandler }

func NewCRMFullHandler(cli CLIRunner) *CRMFullHandler {
	return &CRMFullHandler{NewGenericCLIHandler(cli, "ironclaw_crm_ops",
		"CRM contact management (mc-cli crm <action>). Actions: search, get, list, note, update, schedule.",
		"crm")}
}

// SkillsHandler wraps skill activation stats.
type SkillsHandler struct{ *GenericCLIHandler }

func NewSkillsHandler(cli CLIRunner) *SkillsHandler {
	return &SkillsHandler{NewGenericCLIHandler(cli, "ironclaw_skills",
		"Skill activation metrics (mc-cli skills <action>). Actions: stats, top, missed.",
		"skills")}
}

// CEOOrchestrateHandler wraps the CEO orchestration command.
type CEOOrchestrateHandler struct{ cli CLIRunner }

func NewCEOOrchestrateHandler(cli CLIRunner) *CEOOrchestrateHandler {
	return &CEOOrchestrateHandler{cli: cli}
}

func (h *CEOOrchestrateHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_ceo_orchestrate",
		mcp.WithDescription("Run CEO multi-worker orchestration (mc-cli ceo orchestrate)."),
		mcp.WithNumber("workers", mcp.Description("Number of worker agents")),
		mcp.WithString("persona", mcp.Description("Worker persona template")),
		mcp.WithString("task", mcp.Description("Task description for workers")),
	)
}

func (h *CEOOrchestrateHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	args := []string{"ceo", "orchestrate"}
	if w, ok := req.Params.Arguments["workers"].(float64); ok && w > 0 {
		args = append(args, "--workers", fmt.Sprintf("%d", int(w)))
	}
	if p := optionalString(req, "persona"); p != "" {
		args = append(args, "--persona", p)
	}
	if t := optionalString(req, "task"); t != "" {
		args = append(args, "--task", t)
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("ceo orchestrate: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// JobOpsHandler wraps job lifecycle operations.
type JobOpsHandler struct{ *GenericCLIHandler }

func NewJobOpsHandler(cli CLIRunner) *JobOpsHandler {
	return &JobOpsHandler{NewGenericCLIHandler(cli, "ironclaw_job_ops",
		"Job lifecycle operations (mc-cli job <action>). Actions: list, summary, status, cancel, restart, follow, watch.",
		"job")}
}

// ExportDashboardsHandler wraps Grafana dashboard export.
type ExportDashboardsHandler struct{ cli CLIRunner }

func NewExportDashboardsHandler(cli CLIRunner) *ExportDashboardsHandler {
	return &ExportDashboardsHandler{cli: cli}
}

func (h *ExportDashboardsHandler) Tool() mcp.Tool {
	return mcp.NewTool("ironclaw_export_dashboards",
		mcp.WithDescription("Export Grafana dashboards as JSON (mc-cli export-dashboards)."),
		mcp.WithString("output_dir", mcp.Description("Directory to write dashboard JSON files")),
	)
}

func (h *ExportDashboardsHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured"), nil
	}
	args := []string{"export-dashboards"}
	if d := optionalString(req, "output_dir"); d != "" {
		args = append(args, "--output", d)
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("export-dashboards: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}
