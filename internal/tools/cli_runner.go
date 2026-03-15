package tools

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// CLIRunner runs mc-cli (or equivalent) and returns combined stdout+stderr.
// When nil, CEO tools that depend on it return a "not configured" message.
type CLIRunner interface {
	Run(ctx context.Context, args ...string) (output string, err error)
}

// ExecCLIRunner runs the given binary with args.
type ExecCLIRunner struct {
	Bin     string
	Timeout time.Duration
}

// NewExecCLIRunner creates a runner for mc-cli. bin defaults to "mc-cli" if empty.
func NewExecCLIRunner(bin string) *ExecCLIRunner {
	if bin == "" {
		bin = "mc-cli"
	}
	return &ExecCLIRunner{Bin: bin, Timeout: 60 * time.Second}
}

// Run executes the binary with args and returns combined output.
func (e *ExecCLIRunner) Run(ctx context.Context, args ...string) (string, error) {
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, e.Bin, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
