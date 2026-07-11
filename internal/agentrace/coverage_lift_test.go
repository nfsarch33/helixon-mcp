package agentrace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestDefaultConfig_ResolvesCanonicalLogPath ensures DefaultConfig builds
// a path under $HOME/logs/runx/agentrace-mcp.ndjson (or "" when HOME is
// unset).
func TestDefaultConfig_ResolvesCanonicalLogPath(t *testing.T) {
	t.Parallel()
	home, homeErr := os.UserHomeDir()
	cfg := DefaultConfig()
	if homeErr != nil {
		// When HOME is unreadable, defaultLogPath returns "".
		if cfg.LogPath != "" {
			t.Fatalf("expected empty LogPath when HOME unreadable, got %q", cfg.LogPath)
		}
		return
	}
	want := filepath.Join(home, "logs", "runx", "agentrace-mcp.ndjson")
	if cfg.LogPath != want {
		t.Fatalf("expected LogPath=%q, got %q", want, cfg.LogPath)
	}
	if cfg.AgentID == "" {
		t.Errorf("expected non-empty AgentID")
	}
}

// TestDefaultConfig_DisabledByEnv confirms AGENTRACE_DISABLED=1 disables.
func TestDefaultConfig_DisabledByEnv(t *testing.T) {
	t.Setenv("AGENTRACE_DISABLED", "1")
	cfg := DefaultConfig()
	if cfg.Enabled {
		t.Errorf("expected Enabled=false when AGENTRACE_DISABLED=1, got true")
	}
}

// TestDefaultConfig_DefaultEnabled confirms Enabled is true when the env
// is unset or set to anything other than "1".
func TestDefaultConfig_DefaultEnabled(t *testing.T) {
	t.Setenv("AGENTRACE_DISABLED", "")
	cfg := DefaultConfig()
	if !cfg.Enabled {
		t.Errorf("expected Enabled=true when AGENTRACE_DISABLED unset, got false")
	}
}

// TestDefaultAgentID_OverrideFromEnv confirms CURSOR_AGENT_ID takes
// precedence over the hardcoded fallback.
func TestDefaultAgentID_OverrideFromEnv(t *testing.T) {
	t.Setenv("CURSOR_AGENT_ID", "agent-123")
	if got := defaultAgentID(); got != "agent-123" {
		t.Fatalf("expected defaultAgentID to use CURSOR_AGENT_ID, got %q", got)
	}
}

// TestDefaultAgentID_FallbackWhenUnset confirms the default agent id.
func TestDefaultAgentID_FallbackWhenUnset(t *testing.T) {
	t.Setenv("CURSOR_AGENT_ID", "")
	if got := defaultAgentID(); got != "helixon-mcp" {
		t.Fatalf("expected default agent id 'helixon-mcp', got %q", got)
	}
}

// TestNew_DisabledSkipsOpen confirms a disabled recorder does not open
// the log file even when LogPath is set.
func TestNew_DisabledSkipsOpen(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agentrace.ndjson")
	r, err := New(Config{LogPath: logPath, Enabled: false})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	if r.f != nil {
		t.Errorf("expected nil file handle for disabled recorder")
	}
	if _, statErr := os.Stat(logPath); !os.IsNotExist(statErr) {
		t.Errorf("expected log file absent, got stat err=%v", statErr)
	}
}

// TestNew_EmptyLogPathSkipsOpen covers the LogPath=="" short-circuit
// (returns a no-op recorder without creating directories).
func TestNew_EmptyLogPathSkipsOpen(t *testing.T) {
	t.Parallel()
	r, err := New(Config{Enabled: true, LogPath: ""})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	if r.f != nil {
		t.Errorf("expected nil file handle when LogPath empty")
	}
}

// TestNew_CreatesParentDir confirms MkdirAll is invoked so callers can
// point at a fresh ~/logs/runx path.
func TestNew_CreatesParentDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "deep", "nested", "agentrace.ndjson")

	r, err := New(Config{LogPath: logPath, Enabled: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	if r.f == nil {
		t.Fatalf("expected file handle to be opened")
	}
	r.Record("init", 0, nil)
	r.Close()
	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(body), `"tool":"init"`) {
		t.Errorf("expected init event, got %q", string(body))
	}
}

// TestRecord_NilRecorderDoesNotPanic guards callers that wire nil.
func TestRecord_NilRecorderDoesNotPanic(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic, got %v", r)
		}
	}()
	var r *Recorder
	r.Record("noop", 0, nil)
}

// TestRecord_DisabledDoesNotWrite ensures disabled recorders are a true
// no-op, including not even producing a newline when enabled=false.
func TestRecord_DisabledDoesNotWrite(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agentrace.ndjson")
	r, err := New(Config{LogPath: logPath, Enabled: false})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	r.Record("tool-a", 0, nil)
	r.Record("tool-b", 0, nil)
	if _, statErr := os.Stat(logPath); !os.IsNotExist(statErr) {
		t.Errorf("expected no log file for disabled recorder, got err=%v", statErr)
	}
}

// TestRecord_NoFileSkipsWrite confirms Record on a recorder whose file
// was never opened (e.g. construction failed but caller still uses it)
// does not panic.
func TestRecord_NoFileSkipsWrite(t *testing.T) {
	t.Parallel()
	r := &Recorder{enabled: true, agentID: "test"}
	r.Record("tool", 0, nil) // should not panic; r.f is nil
}

// TestWrap_NilOrDisabledReturnsInner confirms Wrap is identity for
// nil or disabled recorders.
func TestWrap_NilOrDisabledReturnsInner(t *testing.T) {
	t.Parallel()
	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{}, nil
	}
	// Wrap with nil recorder returns the inner handler.
	var nilRec *Recorder
	wrapped := nilRec.Wrap("name", inner)
	if wrapped == nil {
		t.Fatalf("expected non-nil wrapped for nil recorder")
	}
	res, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Errorf("nil-recorder wrap should not return error, got %v", err)
	}
	if res == nil {
		t.Errorf("expected non-nil result from inner handler")
	}

	// Disabled recorder also returns the inner handler.
	disabledRec := &Recorder{enabled: false}
	wrapped2 := disabledRec.Wrap("name", inner)
	if wrapped2 == nil {
		t.Fatalf("expected non-nil wrapped for disabled recorder")
	}
	res2, err := wrapped2(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Errorf("disabled-recorder wrap should not return error, got %v", err)
	}
	if res2 == nil {
		t.Errorf("expected non-nil result from inner handler via disabled wrap")
	}
}

// TestWrap_IsErrorResultRecordedAsFailure covers the Wrap branch where
// the inner handler returns (res with IsError=true, err=nil) — recorded
// as a failure event.
func TestWrap_IsErrorResultRecordedAsFailure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agentrace.ndjson")
	r, err := New(Config{LogPath: logPath, AgentID: "iserror-test", Enabled: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return &mcp.CallToolResult{IsError: true}, nil
	}
	wrapped := r.Wrap("iserror_tool", inner)
	res, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("inner err not propagated: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected IsError result, got %+v", res)
	}
	r.Close()

	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(body), `"tool":"iserror_tool"`) {
		t.Errorf("expected iserror_tool event, got %q", string(body))
	}
	if !strings.Contains(string(body), `"success":false`) {
		t.Errorf("expected success=false when IsError=true, got %q", string(body))
	}
	if !strings.Contains(string(body), "tool returned IsError") {
		t.Errorf("expected 'tool returned IsError' error, got %q", string(body))
	}
}

// TestWrap_HappyPathRecordsSuccess covers Wrap with (res, nil) — emits
// a success event without an error field.
func TestWrap_HappyPathRecordsSuccess(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agentrace.ndjson")
	r, err := New(Config{LogPath: logPath, AgentID: "happy-test", Enabled: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		time.Sleep(time.Millisecond)
		return &mcp.CallToolResult{}, nil
	}
	wrapped := r.Wrap("happy_tool", inner)
	res, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res == nil {
		t.Fatalf("expected non-nil result")
	}
	r.Close()

	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(body), `"tool":"happy_tool"`) {
		t.Errorf("expected happy_tool event, got %q", string(body))
	}
	if !strings.Contains(string(body), `"success":true`) {
		t.Errorf("expected success=true, got %q", string(body))
	}
	if strings.Contains(string(body), `"error"`) {
		t.Errorf("expected no error field on success, got %q", string(body))
	}
}

// TestClose_AfterRecordReleasesFile ensures Close is idempotent enough
// to not panic on a Recorder whose file has already been closed.
func TestClose_AfterRecordReleasesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agentrace.ndjson")
	r, err := New(Config{LogPath: logPath, Enabled: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second Close should be a no-op (r.f is nil).
	if err := r.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

// TestRecord_ErrorMessageFormat checks that the recorded error message
// is included verbatim in the NDJSON event.
func TestRecord_ErrorMessageFormat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agentrace.ndjson")
	r, err := New(Config{LogPath: logPath, AgentID: "err-fmt", Enabled: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	r.Record("fail_tool", 100*time.Millisecond, errors.New("downstream timeout: xyz"))
	r.Close()

	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(body), "downstream timeout: xyz") {
		t.Errorf("expected verbatim error message, got %q", string(body))
	}
	if !strings.Contains(string(body), `"duration_ms":100`) {
		t.Errorf("expected duration_ms:100, got %q", string(body))
	}
}
