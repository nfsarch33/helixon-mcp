// Package agentrace emits NDJSON tool-call events to ~/logs/runx/agentrace-mcp.ndjson
// in the same schema as the sprintboard-mcp telemetry recorder so the
// SprintEval harness can consume both feeds without a per-server adapter.
//
// Schema: {"ts":"<RFC3339>","tool":"<name>","agent_id":"<id>","duration_ms":<int>,"success":<bool>,"error":"<msg?>"}
package agentrace

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// Recorder appends one NDJSON event per tool call. A nil Recorder is a
// valid no-op so callers can wire it unconditionally.
type Recorder struct {
	mu      sync.Mutex
	f       *os.File
	enabled bool
	agentID string
}

// Config controls Recorder construction. Zero values yield sensible
// defaults: log path resolves under $HOME/logs/runx/agentrace-mcp.ndjson,
// enabled tracks AGENTRACE_DISABLED!=1, agent_id falls back to
// `helixon-mcp` (preserving the post-rename naming).
type Config struct {
	LogPath string
	AgentID string
	Enabled bool
}

// DefaultConfig returns a config that writes to the canonical
// agentrace-mcp.ndjson path under the user's home directory.
func DefaultConfig() Config {
	return Config{
		LogPath: defaultLogPath(),
		AgentID: defaultAgentID(),
		Enabled: os.Getenv("AGENTRACE_DISABLED") != "1",
	}
}

func defaultLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "logs", "runx", "agentrace-mcp.ndjson")
}

func defaultAgentID() string {
	if id := os.Getenv("CURSOR_AGENT_ID"); id != "" {
		return id
	}
	return "helixon-mcp"
}

// New opens the configured log path for append. A disabled config returns
// a Recorder that silently skips Record.
func New(cfg Config) (*Recorder, error) {
	r := &Recorder{enabled: cfg.Enabled, agentID: cfg.AgentID}
	if !cfg.Enabled || cfg.LogPath == "" {
		return r, nil
	}
	if err := os.MkdirAll(filepath.Dir(cfg.LogPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(cfg.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	r.f = f
	return r, nil
}

// Close releases the file handle. Safe to call on a nil or disabled Recorder.
func (r *Recorder) Close() error {
	if r == nil || r.f == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	err := r.f.Close()
	r.f = nil
	return err
}

// Record emits a single NDJSON event. Safe to call on a nil Recorder.
func (r *Recorder) Record(tool string, duration time.Duration, err error) {
	if r == nil || !r.enabled || r.f == nil {
		return
	}
	event := map[string]any{
		"ts":          time.Now().Format(time.RFC3339),
		"tool":        tool,
		"agent_id":    r.agentID,
		"duration_ms": duration.Milliseconds(),
		"success":     err == nil,
	}
	if err != nil {
		event["error"] = err.Error()
	}
	data, marshalErr := json.Marshal(event)
	if marshalErr != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.f.Write(data)
	r.f.Write([]byte("\n"))
}

// Wrap returns a tool handler that times the inner handler and records the
// duration plus error status. The inner handler keeps full control of the
// MCP response; agentrace only observes.
func (r *Recorder) Wrap(toolName string, inner mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
	if r == nil || !r.enabled {
		return inner
	}
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		res, err := inner(ctx, req)
		recordedErr := err
		if recordedErr == nil && res != nil && res.IsError {
			recordedErr = errors.New("tool returned IsError")
		}
		r.Record(toolName, time.Since(start), recordedErr)
		return res, err
	}
}
