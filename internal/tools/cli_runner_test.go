package tools

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecCLIRunner_EmptyBinDefaultsToMcCli(t *testing.T) {
	r := NewExecCLIRunner("")
	require.NotNil(t, r)
	assert.Equal(t, "mc-cli", r.Bin)
	assert.Greater(t, r.Timeout, time.Duration(0))
}

func TestNewExecCLIRunner_CustomBin(t *testing.T) {
	r := NewExecCLIRunner("/usr/bin/mc-cli")
	require.NotNil(t, r)
	assert.Equal(t, "/usr/bin/mc-cli", r.Bin)
}

func TestExecCLIRunner_Run_Echo(t *testing.T) {
	// Use a command that exists on Unix to verify Run() executes and returns output.
	r := NewExecCLIRunner("sh")
	r.Timeout = 0 // no timeout for test
	out, err := r.Run(context.Background(), "-c", "echo hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", out)
}

func TestExecCLIRunner_Run_ExitNonZero(t *testing.T) {
	r := NewExecCLIRunner("sh")
	r.Timeout = 0
	_, err := r.Run(context.Background(), "-c", "exit 1")
	assert.Error(t, err)
}
