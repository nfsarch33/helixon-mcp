package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// CRMBriefHandler runs mc-cli crm prep-meeting for a contact.
type CRMBriefHandler struct {
	cli CLIRunner
}

// NewCRMBriefHandler creates a new CRMBriefHandler.
func NewCRMBriefHandler(cli CLIRunner) *CRMBriefHandler {
	return &CRMBriefHandler{cli: cli}
}

// Tool returns the ironclaw_crm_brief tool definition.
func (h *CRMBriefHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_crm_brief",
		mcp.WithDescription("Generate a meeting prep brief for a contact (Executive Hathaway). Calls mc-cli crm prep-meeting."),
		mcp.WithString("contact_id", mcp.Required(), mcp.Description("Contact ID (e.g. from crm list)")),
		mcp.WithString("objective", mcp.Description("Optional meeting objective")),
	)
}

// Handle runs mc-cli crm prep-meeting <id> [objective].
func (h *CRMBriefHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	id, err := requiredString(req, "contact_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	args := []string{"crm", "prep-meeting", id}
	if obj := optionalString(req, "objective"); obj != "" {
		args = append(args, obj)
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("crm prep-meeting: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// MorningBriefHandler runs mc-cli brief generate --live.
type MorningBriefHandler struct {
	cli CLIRunner
}

// NewMorningBriefHandler creates a new MorningBriefHandler.
func NewMorningBriefHandler(cli CLIRunner) *MorningBriefHandler {
	return &MorningBriefHandler{cli: cli}
}

// Tool returns the ironclaw_morning_brief tool definition.
func (h *MorningBriefHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_morning_brief",
		mcp.WithDescription("Generate the Morning COO brief (optionally with live GitHub/HN/repo data). Calls mc-cli brief generate [--live]."),
		mcp.WithString("date", mcp.Description("Date YYYY-MM-DD (default: today)")),
		mcp.WithString("live", mcp.Description("Set to true to include live feeds (GitHub, HN, repo diffs)")),
	)
}

// Handle runs mc-cli brief generate [--live] [--date YYYY-MM-DD].
func (h *MorningBriefHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	args := []string{"brief", "generate"}
	if live, _ := req.Params.Arguments["live"].(string); live == "true" || live == "1" {
		args = append(args, "--live")
	}
	if date, ok := req.Params.Arguments["date"].(string); ok && date != "" {
		args = append(args, "--date", date)
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("brief generate: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// NightAuditHandler triggers the Night Auditor routine.
type NightAuditHandler struct {
	cli CLIRunner
}

// NewNightAuditHandler creates a new NightAuditHandler.
func NewNightAuditHandler(cli CLIRunner) *NightAuditHandler {
	return &NightAuditHandler{cli: cli}
}

// Tool returns the ironclaw_run_night_audit tool definition.
func (h *NightAuditHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_run_night_audit",
		mcp.WithDescription("Run the Night Auditor pipeline (tests, infra check, report). Calls mc-cli audit run [--repos] [--file-incidents]."),
		mcp.WithString("repos", mcp.Description("Set to true to run testreporter on configured repos")),
		mcp.WithString("file_incidents", mcp.Description("Set to true to file incidents to global-kb when tests fail")),
	)
}

// Handle runs mc-cli audit run [--repos] [--file-incidents].
func (h *NightAuditHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	args := []string{"audit", "run"}
	if repos, _ := req.Params.Arguments["repos"].(string); repos == "true" || repos == "1" {
		args = append(args, "--repos")
	}
	if fileInc, _ := req.Params.Arguments["file_incidents"].(string); fileInc == "true" || fileInc == "1" {
		args = append(args, "--file-incidents")
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("audit run: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// SpawnPersonaHandler runs mc-cli spawn --persona <name>.
type SpawnPersonaHandler struct {
	cli CLIRunner
}

// NewSpawnPersonaHandler creates a new SpawnPersonaHandler.
func NewSpawnPersonaHandler(cli CLIRunner) *SpawnPersonaHandler {
	return &SpawnPersonaHandler{cli: cli}
}

// Tool returns the ironclaw_spawn_persona tool definition.
func (h *SpawnPersonaHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_spawn_persona",
		mcp.WithDescription("Spawn an agent with a persona (night-auditor, morning-coo, crm-assistant, executive-hathaway, commerce-orchestrator). Calls mc-cli spawn --persona."),
		mcp.WithString("persona", mcp.Required(), mcp.Description("Persona name")),
	)
}

// Handle runs mc-cli spawn --persona <name>.
func (h *SpawnPersonaHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	persona, err := requiredString(req, "persona")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out, err := h.cli.Run(ctx, "spawn", "--persona", persona)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("spawn --persona: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// FleetStatusHandler runs mc-cli fleet health.
type FleetStatusHandler struct {
	cli CLIRunner
}

// NewFleetStatusHandler creates a new FleetStatusHandler.
func NewFleetStatusHandler(cli CLIRunner) *FleetStatusHandler {
	return &FleetStatusHandler{cli: cli}
}

// Tool returns the ironclaw_fleet_status tool definition.
func (h *FleetStatusHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_fleet_status",
		mcp.WithDescription("Return aggregated fleet health (all registered IronClaw nodes). Calls mc-cli fleet health."),
	)
}

// Handle runs mc-cli fleet health.
func (h *FleetStatusHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	out, err := h.cli.Run(ctx, "fleet", "health")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("fleet health: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// LoadtestHandler runs mc-cli loadtest run.
type LoadtestHandler struct {
	cli CLIRunner
}

// NewLoadtestHandler creates a new LoadtestHandler.
func NewLoadtestHandler(cli CLIRunner) *LoadtestHandler {
	return &LoadtestHandler{cli: cli}
}

// Tool returns the ironclaw_loadtest tool definition.
func (h *LoadtestHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_loadtest",
		mcp.WithDescription("Run a load test against the IronClaw gateway (concurrent job submissions). Calls mc-cli loadtest run."),
		mcp.WithString("concurrency", mcp.Description("Number of concurrent requests (default 5)")),
		mcp.WithString("duration_secs", mcp.Description("Test duration in seconds (default 30)")),
	)
}

// Handle runs mc-cli loadtest run [--concurrency N] [--duration D].
func (h *LoadtestHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if h.cli == nil {
		return mcp.NewToolResultError("mc-cli not configured (CLIRunner nil)"), nil
	}
	args := []string{"loadtest", "run"}
	if c := optionalString(req, "concurrency"); c != "" {
		args = append(args, "--concurrency", c)
	}
	if d := optionalString(req, "duration_secs"); d != "" {
		args = append(args, "--duration", d+"s")
	}
	out, err := h.cli.Run(ctx, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("loadtest: %v\n%s", err, out)), nil
	}
	return mcp.NewToolResultText(out), nil
}
