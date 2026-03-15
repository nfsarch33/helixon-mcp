package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCLI is a CLIRunner that returns predefined output or error.
type mockCLI struct {
	out string
	err error
}

func (m *mockCLI) Run(ctx context.Context, args ...string) (string, error) {
	if m.err != nil {
		return m.out, m.err
	}
	return m.out, nil
}

func makeCEOReq(args map[string]interface{}) mcp.CallToolRequest {
	var req mcp.CallToolRequest
	req.Params.Arguments = args
	return req
}

func TestCRMBriefHandler_NilCLI(t *testing.T) {
	h := NewCRMBriefHandler(nil)
	res, err := h.Handle(context.Background(), makeCEOReq(map[string]interface{}{"contact_id": "jane"}))
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.True(t, res.IsError)
}

func TestCRMBriefHandler_Success(t *testing.T) {
	cli := &mockCLI{out: "# Meeting prep: Jane\n\n**Org:** Acme"}
	h := NewCRMBriefHandler(cli)
	res, err := h.Handle(context.Background(), makeCEOReq(map[string]interface{}{"contact_id": "jane"}))
	require.NoError(t, err)
	require.False(t, res.IsError)
	require.Len(t, res.Content, 1)
	text, ok := res.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Equal(t, cli.out, text.Text)
}

func TestMorningBriefHandler_NilCLI(t *testing.T) {
	h := NewMorningBriefHandler(nil)
	res, err := h.Handle(context.Background(), makeCEOReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestNightAuditHandler_Success(t *testing.T) {
	cli := &mockCLI{out: "## Night Audit -- 2026-03-15\n\nAll healthy."}
	h := NewNightAuditHandler(cli)
	res, err := h.Handle(context.Background(), makeCEOReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestSpawnPersonaHandler_MissingPersona(t *testing.T) {
	h := NewSpawnPersonaHandler(&mockCLI{})
	res, err := h.Handle(context.Background(), makeCEOReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestSpawnPersonaHandler_Success(t *testing.T) {
	cli := &mockCLI{out: "Spawned morning-coo"}
	h := NewSpawnPersonaHandler(cli)
	res, err := h.Handle(context.Background(), makeCEOReq(map[string]interface{}{"persona": "morning-coo"}))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestFleetStatusHandler_Success(t *testing.T) {
	cli := &mockCLI{out: "Fleet: 2 nodes healthy"}
	h := NewFleetStatusHandler(cli)
	res, err := h.Handle(context.Background(), makeCEOReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}

func TestLoadtestHandler_CLIError(t *testing.T) {
	cli := &mockCLI{err: errors.New("exec failed"), out: "mc-cli: not found"}
	h := NewLoadtestHandler(cli)
	res, err := h.Handle(context.Background(), makeCEOReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestCRMBriefHandler_MissingContactID(t *testing.T) {
	h := NewCRMBriefHandler(&mockCLI{})
	res, err := h.Handle(context.Background(), makeCEOReq(nil))
	require.NoError(t, err)
	assert.True(t, res.IsError)
}

func TestLoadtestHandler_Success(t *testing.T) {
	cli := &mockCLI{out: "Load test complete. 10 requests, 100% success."}
	h := NewLoadtestHandler(cli)
	res, err := h.Handle(context.Background(), makeCEOReq(nil))
	require.NoError(t, err)
	assert.False(t, res.IsError)
}
