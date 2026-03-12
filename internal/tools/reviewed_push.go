package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

const defaultReviewPrompt = `Review this git diff using these categories in order: security, correctness, tests, performance, maintainability.

Security:
- hardcoded secrets or credentials
- missing validation
- unsafe paths or shell usage
- insecure server defaults

Correctness:
- broken logic
- missing edge-case handling
- missing error handling

Tests:
- missing tests for new behavior
- missing error-path coverage

Performance:
- unbounded work
- avoidable hot-path allocations
- queue or latency regressions

Maintainability:
- drift from local conventions
- dead code
- unclear naming

Return ONLY valid JSON in this shape:
{"verdict":"pass"|"fail","must_fix":[{"issue":"...","file":"...","line":0}],"should_fix":[{"issue":"...","file":"...","line":0}],"nits":[{"issue":"...","file":"...","line":0}]}`

type commandRunner func(ctx context.Context, workdir string, stdin string, name string, args ...string) (string, error)

type reviewFinding struct {
	Issue string `json:"issue"`
	File  string `json:"file,omitempty"`
	Line  int    `json:"line,omitempty"`
}

type reviewVerdict struct {
	Verdict   string          `json:"verdict"`
	MustFix   []reviewFinding `json:"must_fix"`
	ShouldFix []reviewFinding `json:"should_fix"`
	Nits      []reviewFinding `json:"nits"`
}

type reviewedPushResult struct {
	Allowed    bool          `json:"allowed"`
	Pushed     bool          `json:"pushed"`
	Remote     string        `json:"remote"`
	Branch     string        `json:"branch"`
	BaseRef    string        `json:"base_ref"`
	DiffBytes  int           `json:"diff_bytes"`
	Review     reviewVerdict `json:"review"`
	PushOutput string        `json:"push_output,omitempty"`
}

// ReviewedPushHandler runs Gemini review before an optional git push.
type ReviewedPushHandler struct {
	run commandRunner
}

// NewReviewedPushHandler creates a new handler with the default command runner.
func NewReviewedPushHandler() *ReviewedPushHandler {
	return &ReviewedPushHandler{run: runCommand}
}

func (h *ReviewedPushHandler) Tool() mcp.Tool {
	return mcp.NewTool(
		"ironclaw_reviewed_push",
		mcp.WithDescription("Run Gemini CLI review on a git diff, then optionally push only when no must-fix issues are found."),
		mcp.WithString("workdir",
			mcp.Required(),
			mcp.Description("Git repository directory to review and push from."),
		),
		mcp.WithString("remote",
			mcp.Description("Git remote name. Defaults to origin."),
		),
		mcp.WithString("branch",
			mcp.Description("Branch name to push. Defaults to the current branch."),
		),
		mcp.WithString("base_ref",
			mcp.Description("Optional base ref for the review diff. Defaults to <remote>/<branch>."),
		),
		mcp.WithString("review_only",
			mcp.Description("Optional true/false flag. When true, run review and skip the push step."),
		),
		mcp.WithString("model",
			mcp.Description("Optional Gemini model name. Falls back to GEMINI_REVIEW_MODEL if set."),
		),
	)
}

func (h *ReviewedPushHandler) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	workdir, err := requiredString(req, "workdir")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	remote := optionalString(req, "remote")
	if remote == "" {
		remote = "origin"
	}

	branch := optionalString(req, "branch")
	if branch == "" {
		branchOut, err := h.run(ctx, workdir, "", "git", "branch", "--show-current")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("resolving branch: %v", err)), nil
		}
		branch = strings.TrimSpace(branchOut)
		if branch == "" {
			return mcp.NewToolResultError("could not determine current branch"), nil
		}
	}

	baseRef := optionalString(req, "base_ref")
	if baseRef == "" {
		baseRef = fmt.Sprintf("%s/%s", remote, branch)
	}

	reviewOnly := optionalBool(req, "review_only")
	model := optionalString(req, "model")
	if model == "" {
		model = os.Getenv("GEMINI_REVIEW_MODEL")
	}

	diff, err := h.run(ctx, workdir, "", "git", "diff", "--no-ext-diff", fmt.Sprintf("%s..HEAD", baseRef))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("collecting diff against %s: %v", baseRef, err)), nil
	}
	if strings.TrimSpace(diff) == "" {
		return jsonResult(reviewedPushResult{
			Allowed:   true,
			Pushed:    false,
			Remote:    remote,
			Branch:    branch,
			BaseRef:   baseRef,
			DiffBytes: 0,
			Review: reviewVerdict{
				Verdict: "pass",
			},
		})
	}

	args := []string{"-p", defaultReviewPrompt}
	if model != "" {
		args = append(args, "-m", model)
	}
	reviewOut, err := h.run(ctx, workdir, diff, "gemini", args...)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("running Gemini review: %v", err)), nil
	}

	verdict, err := parseReviewVerdict(reviewOut)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("parsing Gemini review: %v", err)), nil
	}

	result := reviewedPushResult{
		Allowed:   len(verdict.MustFix) == 0,
		Pushed:    false,
		Remote:    remote,
		Branch:    branch,
		BaseRef:   baseRef,
		DiffBytes: len(diff),
		Review:    verdict,
	}

	if len(verdict.MustFix) > 0 || reviewOnly {
		return jsonResult(result)
	}

	pushOut, err := h.run(ctx, workdir, "", "git", "push", remote, fmt.Sprintf("HEAD:refs/heads/%s", branch))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("git push failed: %v", err)), nil
	}
	result.Pushed = true
	result.PushOutput = strings.TrimSpace(pushOut)
	return jsonResult(result)
}

func optionalBool(req mcp.CallToolRequest, key string) bool {
	v, ok := req.Params.Arguments[key]
	if !ok {
		return false
	}
	switch typed := v.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true") || typed == "1" || strings.EqualFold(typed, "yes")
	default:
		return false
	}
}

func parseReviewVerdict(raw string) (reviewVerdict, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return reviewVerdict{}, fmt.Errorf("empty review output")
	}

	var verdict reviewVerdict
	if err := json.Unmarshal([]byte(raw), &verdict); err == nil && verdict.Verdict != "" {
		return verdict, nil
	}

	var wrapped struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal([]byte(raw), &wrapped); err == nil && strings.TrimSpace(wrapped.Response) != "" {
		if err := json.Unmarshal([]byte(strings.TrimSpace(wrapped.Response)), &verdict); err == nil && verdict.Verdict != "" {
			return verdict, nil
		}
	}

	return reviewVerdict{}, fmt.Errorf("unsupported review output: %s", raw)
}

func runCommand(ctx context.Context, workdir string, stdin string, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workdir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
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
