package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nfsarch33/ironclaw-mcp/internal/helixon"
)

func TestSearch_GoldenRRFOrder(t *testing.T) {
	t.Parallel()

	client := new(MockIronclawClient)
	client.On("SearchMemory", context.Background(), helixon.MemorySearchRequest{Query: "coaching content", Limit: 2}).
		Return(&helixon.MemorySearchResponse{Results: []helixon.MemoryEntry{
			{Path: "coaching/session.md", Content: "session plan", Score: 0.91},
			{Path: "content/ai.md", Content: "article seed", Score: 0.82},
		}}, nil)

	handler := NewWorkspaceMemoryHandler(client)
	res, err := handler.HandleSearch(context.Background(), makeReq(map[string]any{"query": "coaching content", "limit": "2"}))
	if err != nil {
		t.Fatalf("HandleSearch() error = %v", err)
	}
	if res.IsError {
		t.Fatalf("HandleSearch() returned MCP error: %#v", res.Content)
	}

	var out helixon.MemorySearchResponse
	if err := json.Unmarshal([]byte(res.Content[0].(mcp.TextContent).Text), &out); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if got := out.Results[0].Path; got != "coaching/session.md" {
		t.Fatalf("first result path = %q, want coaching/session.md", got)
	}
}

func TestWorkspaceMemory_WriteReadTree(t *testing.T) {
	t.Parallel()

	client := new(MockIronclawClient)
	client.On("WriteMemory", context.Background(), helixon.MemoryWriteRequest{
		Path:    "content/v313.md",
		Content: "seed",
	}).Return(&helixon.MemoryWriteResponse{Path: "content/v313.md", Version: 1}, nil)
	client.On("ReadMemory", context.Background(), helixon.MemoryReadRequest{Path: "content/v313.md"}).
		Return(&helixon.MemoryReadResponse{Path: "content/v313.md", Content: "seed", Version: 1}, nil)
	client.On("TreeMemory", context.Background(), helixon.MemoryTreeRequest{Prefix: "content"}).
		Return(&helixon.MemoryTreeResponse{Entries: []helixon.MemoryTreeEntry{{Path: "content/v313.md", Type: "file"}}}, nil)

	handler := NewWorkspaceMemoryHandler(client)
	for name, call := range map[string]func() (*mcp.CallToolResult, error){
		"write": func() (*mcp.CallToolResult, error) {
			return handler.HandleWrite(context.Background(), makeReq(map[string]any{"path": "content/v313.md", "content": "seed"}))
		},
		"read": func() (*mcp.CallToolResult, error) {
			return handler.HandleRead(context.Background(), makeReq(map[string]any{"path": "content/v313.md"}))
		},
		"tree": func() (*mcp.CallToolResult, error) {
			return handler.HandleTree(context.Background(), makeReq(map[string]any{"prefix": "content"}))
		},
	} {
		t.Run(name, func(t *testing.T) {
			res, err := call()
			if err != nil {
				t.Fatalf("%s error = %v", name, err)
			}
			if res.IsError {
				t.Fatalf("%s returned MCP error: %#v", name, res.Content)
			}
		})
	}
}
