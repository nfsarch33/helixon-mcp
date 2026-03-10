package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// JobsHandler handles job-related MCP tools.
type JobsHandler struct {
	client IronclawClient
}

// NewJobsHandler creates a new JobsHandler.
func NewJobsHandler(client IronclawClient) *JobsHandler {
	return &JobsHandler{client: client}
}

// ListJobsTool returns the ironclaw_list_jobs tool definition.
func (h *JobsHandler) ListJobsTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_list_jobs",
		mcp.WithDescription("List all background jobs in IronClaw, including their status (running, done, failed)."),
	)
}

// GetJobTool returns the ironclaw_get_job tool definition.
func (h *JobsHandler) GetJobTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_get_job",
		mcp.WithDescription("Get details of a specific IronClaw background job by ID."),
		mcp.WithString("job_id",
			mcp.Required(),
			mcp.Description("The job ID to retrieve."),
		),
	)
}

// CancelJobTool returns the ironclaw_cancel_job tool definition.
func (h *JobsHandler) CancelJobTool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_cancel_job",
		mcp.WithDescription("Cancel a running IronClaw background job."),
		mcp.WithString("job_id",
			mcp.Required(),
			mcp.Description("The job ID to cancel."),
		),
	)
}

// HandleListJobs handles the list jobs tool call.
func (h *JobsHandler) HandleListJobs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := h.client.ListJobs(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("listing jobs: %v", err)), nil
	}
	return jsonResult(resp)
}

// HandleGetJob handles the get job tool call.
func (h *JobsHandler) HandleGetJob(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID, err := requiredString(req, "job_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	job, err := h.client.GetJob(ctx, jobID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("getting job %q: %v", jobID, err)), nil
	}
	return jsonResult(job)
}

// HandleCancelJob handles the cancel job tool call.
func (h *JobsHandler) HandleCancelJob(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID, err := requiredString(req, "job_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := h.client.CancelJob(ctx, jobID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cancelling job %q: %v", jobID, err)), nil
	}
	return jsonResult(map[string]string{"status": "cancelled", "job_id": jobID})
}
