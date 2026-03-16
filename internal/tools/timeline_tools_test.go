package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimelineHandler_Tool(t *testing.T) {
	h := NewTimelineHandler(nil)
	tool := h.Tool()
	assert.Equal(t, "ironclaw_timeline", tool.Name)
}

func TestTimelineHandler_NilCLI(t *testing.T) {
	h := NewTimelineHandler(nil)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{"action": "list"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestTimelineHandler_MissingAction(t *testing.T) {
	h := NewTimelineHandler(&dualMockCLI{output: ""})
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestTimelineHandler_List(t *testing.T) {
	cli := &dualMockCLI{output: `[{"id":"evt-1","type":"spawn"}]`}
	h := NewTimelineHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action": "list",
		"type":   "spawn",
		"actor":  "ceo",
		"limit":  "10",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"timeline", "list", "--type", "spawn", "--actor", "ceo", "--limit", "10"}, cli.lastArgs)
}

func TestTimelineHandler_Export(t *testing.T) {
	cli := &dualMockCLI{output: "exported 42 events"}
	h := NewTimelineHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action": "export",
		"file":   "/tmp/timeline.json",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"timeline", "export", "--file", "/tmp/timeline.json"}, cli.lastArgs)
}

func TestTimelineHandler_Import(t *testing.T) {
	cli := &dualMockCLI{output: "imported 42 events"}
	h := NewTimelineHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{
		"action": "import",
		"file":   "/tmp/timeline.json",
	}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
	assert.Equal(t, []string{"timeline", "import", "--file", "/tmp/timeline.json"}, cli.lastArgs)
}

func TestTimelineHandler_CLIError(t *testing.T) {
	cli := &dualMockCLI{err: fmt.Errorf("io error")}
	h := NewTimelineHandler(cli)
	res, err := h.Handle(context.Background(), dualCallReq(map[string]interface{}{"action": "list"}))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}
