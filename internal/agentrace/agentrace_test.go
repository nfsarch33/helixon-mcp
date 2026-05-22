package agentrace

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// TestRecord_AppendsNDJSON exercises the happy-path event shape and
// asserts the schema matches the sprintboard-mcp telemetry feed.
func TestRecord_AppendsNDJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agentrace.ndjson")

	r, err := New(Config{LogPath: logPath, AgentID: "test-agent", Enabled: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close()

	r.Record("health", 12*time.Millisecond, nil)
	r.Record("chat", 5*time.Millisecond, errors.New("upstream busy"))
	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	body, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d: %q", len(lines), string(body))
	}

	var first, second map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("decode line 0: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("decode line 1: %v", err)
	}
	for _, key := range []string{"ts", "tool", "agent_id", "duration_ms", "success"} {
		if _, ok := first[key]; !ok {
			t.Errorf("missing field %q in line 0", key)
		}
	}
	if first["tool"] != "health" || first["agent_id"] != "test-agent" || first["success"] != true {
		t.Errorf("unexpected line 0: %v", first)
	}
	if second["success"] != false {
		t.Errorf("expected line 1 success=false, got %v", second["success"])
	}
	if second["error"] != "upstream busy" {
		t.Errorf("expected error=upstream busy, got %v", second["error"])
	}
}

// TestRecord_DisabledIsNoop confirms a disabled recorder writes nothing
// and does not create the log file.
func TestRecord_DisabledIsNoop(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agentrace.ndjson")

	r, err := New(Config{LogPath: logPath, Enabled: false})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r.Record("health", time.Millisecond, nil)
	r.Close()

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("expected log file to be absent, got err=%v", err)
	}
}

// TestRecord_NilRecorderIsNoop covers callers that wire a nil recorder.
func TestRecord_NilRecorderIsNoop(t *testing.T) {
	t.Parallel()
	var r *Recorder
	r.Record("health", time.Millisecond, nil)
	if err := r.Close(); err != nil {
		t.Errorf("nil Close: %v", err)
	}
}

// TestWrap_ObservesInnerHandler ensures Wrap forwards request, response,
// and error verbatim while emitting one NDJSON event per call.
func TestWrap_ObservesInnerHandler(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "agentrace.ndjson")

	r, err := New(Config{LogPath: logPath, AgentID: "wrap-test", Enabled: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer r.Close()

	wantErr := errors.New("inner failed")
	called := 0
	inner := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called++
		return nil, wantErr
	}
	wrapped := r.Wrap("inner_tool", inner)

	res, err := wrapped(context.Background(), mcp.CallToolRequest{})
	if err != wantErr {
		t.Errorf("Wrap should propagate inner error verbatim, got %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result, got %v", res)
	}
	if called != 1 {
		t.Errorf("expected inner called once, got %d", called)
	}

	r.Close()
	body, _ := os.ReadFile(logPath)
	if !strings.Contains(string(body), `"tool":"inner_tool"`) {
		t.Errorf("expected inner_tool event in log, got %q", string(body))
	}
	if !strings.Contains(string(body), `"success":false`) {
		t.Errorf("expected success=false event, got %q", string(body))
	}
}
