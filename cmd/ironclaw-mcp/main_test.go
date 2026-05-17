package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_VersionFlag(t *testing.T) {
	out := captureStdout(t, func() {
		oldArgs := os.Args
		t.Cleanup(func() { os.Args = oldArgs })
		os.Args = []string{"helixon-mcp", "--version"}

		err := run()
		require.NoError(t, err)
	})

	assert.Equal(t, "helixon-mcp "+version+"\n", out)
}

func TestRun_HelpFlag(t *testing.T) {
	out := captureStdout(t, func() {
		oldArgs := os.Args
		t.Cleanup(func() { os.Args = oldArgs })
		os.Args = []string{"helixon-mcp", "--help"}

		err := run()
		require.NoError(t, err)
	})

	assert.Contains(t, out, "Usage:")
	assert.Contains(t, out, "HELIXON_BASE_URL")
	assert.Contains(t, out, "PROMETHEUS_URL")
	assert.Contains(t, out, "Source: https://github.com/nfsarch33/helixon-mcp")
}

func TestBuildLogger_AcceptsKnownLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "warning", "error", "unknown"} {
		t.Run(level, func(t *testing.T) {
			assert.NotNil(t, buildLogger(level))
		})
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = writer
	t.Cleanup(func() { os.Stdout = oldStdout })

	fn()

	require.NoError(t, writer.Close())
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	require.NoError(t, err)
	return buf.String()
}
