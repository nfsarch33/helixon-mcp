package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// EvolverHandler provides MCP tools for the IronClaw evolver framework:
// capsule store status, mutation proposals, sandboxed validation, and promotion.
type EvolverHandler struct {
	run commandRunner
	bin string
}

// NewEvolverHandler creates a handler bridging to research-agent evolver subcommands.
func NewEvolverHandler() *EvolverHandler {
	return &EvolverHandler{run: runResearchCommand, bin: "research-agent"}
}

// StatusTool returns the MCP tool for capsule store summary.
func (h *EvolverHandler) StatusTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_evolver_status",
		mcp.WithDescription("Show evolver capsule store status: total capsules, pending mutations, recent evolution events, and health metrics. Provides an overview of the self-evolution system state."),
		mcp.WithString("data_dir",
			mcp.Description("Path to GEP assets directory. Default: uses global-kb/assets/gep/."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
	)
}

// HandleStatus executes evolver status via the research-agent CLI.
func (h *EvolverHandler) HandleStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := []string{"evolver-status"}

	if dd := optionalString(req, "data_dir"); dd != "" {
		args = append(args, "--data-dir", expandTilde(dd))
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("evolver-status failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// ProposeTool returns the MCP tool for proposing a mutation.
func (h *EvolverHandler) ProposeTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_evolver_propose",
		mcp.WithDescription("Propose a new evolution mutation based on execution signals. Analyzes recent error patterns, performance bottlenecks, or capability gaps and generates a structured mutation proposal with estimated blast radius."),
		mcp.WithString("signal_source",
			mcp.Required(),
			mcp.Description("Source of the evolution signal: 'auto-log' (scan recent logs), 'metric' (performance data), or 'manual' (explicit description)."),
		),
		mcp.WithString("description",
			mcp.Description("Manual description of the capability gap or error to address. Required when signal_source='manual'."),
		),
		mcp.WithString("strategy",
			mcp.Description("Evolution strategy: 'innovate', 'harden', 'repair-only', or 'balanced'. Default: 'balanced'."),
		),
		mcp.WithString("scope",
			mcp.Description("Mutation scope: 'prompt', 'tool', 'workflow', or 'config'. Default: 'prompt'."),
		),
		mcp.WithString("data_dir",
			mcp.Description("Path to GEP assets directory. Default: uses global-kb/assets/gep/."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
	)
}

// HandlePropose executes mutation proposal via the research-agent CLI.
func (h *EvolverHandler) HandlePropose(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	signalSource, err := requiredString(req, "signal_source")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"evolver-propose", "--signal-source", signalSource}

	if desc := optionalString(req, "description"); desc != "" {
		args = append(args, "--description", desc)
	}
	if strat := optionalString(req, "strategy"); strat != "" {
		args = append(args, "--strategy", strat)
	}
	if scope := optionalString(req, "scope"); scope != "" {
		args = append(args, "--scope", scope)
	}
	if dd := optionalString(req, "data_dir"); dd != "" {
		args = append(args, "--data-dir", expandTilde(dd))
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("evolver-propose failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// ValidateTool returns the MCP tool for running sandboxed validation.
func (h *EvolverHandler) ValidateTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_evolver_validate",
		mcp.WithDescription("Run sandboxed validation for a proposed mutation. Executes the mutation's validation commands in an isolated Docker container, verifying the patch works without affecting the host system."),
		mcp.WithString("mutation_id",
			mcp.Required(),
			mcp.Description("ID of the mutation to validate (from evolver-propose output)."),
		),
		mcp.WithString("timeout",
			mcp.Description("Validation timeout. Default: '180s'."),
		),
		mcp.WithString("data_dir",
			mcp.Description("Path to GEP assets directory. Default: uses global-kb/assets/gep/."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
	)
}

// HandleValidate executes sandboxed validation via the research-agent CLI.
func (h *EvolverHandler) HandleValidate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mutationID, err := requiredString(req, "mutation_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"evolver-validate", mutationID}

	if t := optionalString(req, "timeout"); t != "" {
		args = append(args, "--timeout", t)
	}
	if dd := optionalString(req, "data_dir"); dd != "" {
		args = append(args, "--data-dir", expandTilde(dd))
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("evolver-validate failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}

// PromoteTool returns the MCP tool for promoting a validated capsule.
func (h *EvolverHandler) PromoteTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_evolver_promote",
		mcp.WithDescription("Promote a validated mutation capsule to the global knowledge base. Updates capsules.json, appends an evolution event to events.jsonl, and optionally syncs to Mem0 for fleet-wide adoption."),
		mcp.WithString("mutation_id",
			mcp.Required(),
			mcp.Description("ID of the validated mutation to promote."),
		),
		mcp.WithString("sync_mem0",
			mcp.Description("Set to 'true' to sync the promoted capsule to Mem0 for fleet-wide access. Default: true."),
		),
		mcp.WithString("data_dir",
			mcp.Description("Path to GEP assets directory. Default: uses global-kb/assets/gep/."),
		),
		mcp.WithString("format",
			mcp.Description("Output format: 'json' (default) or 'markdown'."),
		),
	)
}

// HandlePromote executes capsule promotion via the research-agent CLI.
func (h *EvolverHandler) HandlePromote(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	mutationID, err := requiredString(req, "mutation_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	args := []string{"evolver-promote", mutationID}

	if sm := optionalString(req, "sync_mem0"); sm == "false" {
		args = append(args, "--sync-mem0=false")
	}
	if dd := optionalString(req, "data_dir"); dd != "" {
		args = append(args, "--data-dir", expandTilde(dd))
	}
	if f := optionalString(req, "format"); f != "" {
		args = append(args, "--format", f)
	}

	out, err := h.run(ctx, "", "", h.bin, args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("evolver-promote failed: %v", err)), nil
	}
	return mcp.NewToolResultText(out), nil
}
