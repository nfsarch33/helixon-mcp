// Package server wires all MCP tools together and runs the MCP server.
package server

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nfsarch33/ironclaw-mcp/internal/tools"
	"go.uber.org/zap"
)

// Server wraps the MCP server and its dependencies.
type Server struct {
	client  tools.IronclawClient
	logger  *zap.Logger
	version string
	mcp     *server.MCPServer
}

// New creates and configures a new MCP Server with all IronClaw tools registered.
func New(client tools.IronclawClient, logger *zap.Logger, version string) *Server {
	s := &Server{
		client:  client,
		logger:  logger,
		version: version,
	}
	s.mcp = s.buildMCPServer()
	return s
}

func (s *Server) buildMCPServer() *server.MCPServer {
	srv := server.NewMCPServer(
		"ironclaw-mcp",
		s.version,
		server.WithToolCapabilities(true),
	)

	health := tools.NewHealthHandler(s.client)
	srv.AddTool(health.Tool(), health.Handle)

	chat := tools.NewChatHandler(s.client)
	srv.AddTool(chat.Tool(), chat.Handle)

	jobs := tools.NewJobsHandler(s.client)
	srv.AddTool(jobs.ListJobsTool(), jobs.HandleListJobs)
	srv.AddTool(jobs.GetJobTool(), jobs.HandleGetJob)
	srv.AddTool(jobs.CancelJobTool(), jobs.HandleCancelJob)

	mem := tools.NewMemoryHandler(s.client)
	srv.AddTool(mem.Tool(), mem.Handle)

	routines := tools.NewRoutinesHandler(s.client)
	srv.AddTool(routines.ListRoutinesTool(), routines.HandleListRoutines)
	srv.AddTool(routines.CreateRoutineTool(), routines.HandleCreateRoutine)
	srv.AddTool(routines.DeleteRoutineTool(), routines.HandleDeleteRoutine)

	toolsList := tools.NewToolsListHandler(s.client)
	srv.AddTool(toolsList.Tool(), toolsList.Handle)

	return srv
}

// Run starts the MCP server using the configured transport.
func (s *Server) Run(ctx context.Context, transport string) error {
	s.logger.Info("MCP server ready", zap.String("transport", transport))
	switch transport {
	case "stdio":
		stdioSrv := server.NewStdioServer(s.mcp)
		return stdioSrv.Listen(ctx, nil, nil)
	case "sse":
		return fmt.Errorf("SSE transport not yet implemented; use stdio")
	default:
		return fmt.Errorf("unknown transport %q", transport)
	}
}

// MCPServer exposes the underlying MCP server (for testing).
func (s *Server) MCPServer() *server.MCPServer {
	return s.mcp
}

// RegisteredToolCount returns how many tools are registered (for testing).
func (s *Server) RegisteredToolCount() int {
	tools, _ := s.mcp.ListTools(context.Background(), mcp.ListToolsRequest{})
	return len(tools.Tools)
}
