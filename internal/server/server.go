// Package server wires all MCP tools together and runs the MCP server.
package server

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/nfsarch33/ironclaw-mcp/internal/tools"
	"go.uber.org/zap"
)

// Server wraps the MCP server and its dependencies.
type Server struct {
	client    tools.IronclawClient
	cli       tools.CLIRunner
	logger    *zap.Logger
	version   string
	mcp       *server.MCPServer
	toolCount int
}

// New creates and configures a new MCP Server with all IronClaw tools registered.
// cli is optional; when set, CEO tools (crm_brief, morning_brief, night_audit, spawn_persona, fleet_status, loadtest) are registered and invoke mc-cli.
func New(client tools.IronclawClient, cli tools.CLIRunner, logger *zap.Logger, version string) *Server {
	s := &Server{
		client: client,
		cli:    cli,
		logger: logger,
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
	s.toolCount++

	chat := tools.NewChatHandler(s.client)
	srv.AddTool(chat.Tool(), chat.Handle)
	s.toolCount++

	jobs := tools.NewJobsHandler(s.client)
	srv.AddTool(jobs.ListJobsTool(), jobs.HandleListJobs)
	srv.AddTool(jobs.GetJobTool(), jobs.HandleGetJob)
	srv.AddTool(jobs.CancelJobTool(), jobs.HandleCancelJob)
	s.toolCount += 3

	mem := tools.NewMemoryHandler(s.client)
	srv.AddTool(mem.Tool(), mem.Handle)
	s.toolCount++

	routines := tools.NewRoutinesHandler(s.client)
	srv.AddTool(routines.ListRoutinesTool(), routines.HandleListRoutines)
	srv.AddTool(routines.DeleteRoutineTool(), routines.HandleDeleteRoutine)
	s.toolCount += 2

	toolsList := tools.NewToolsListHandler(s.client)
	srv.AddTool(toolsList.Tool(), toolsList.Handle)
	s.toolCount++

	stackStatus := tools.NewStackStatusHandler(s.client)
	srv.AddTool(stackStatus.Tool(), stackStatus.Handle)
	s.toolCount++

	spawnAgent := tools.NewSpawnAgentHandler(s.client)
	srv.AddTool(spawnAgent.Tool(), spawnAgent.Handle)
	s.toolCount++

	reviewedPush := tools.NewReviewedPushHandler()
	srv.AddTool(reviewedPush.Tool(), reviewedPush.Handle)
	s.toolCount++

	if s.cli != nil {
		srv.AddTool(tools.NewCRMBriefHandler(s.cli).Tool(), tools.NewCRMBriefHandler(s.cli).Handle)
		srv.AddTool(tools.NewMorningBriefHandler(s.cli).Tool(), tools.NewMorningBriefHandler(s.cli).Handle)
		srv.AddTool(tools.NewNightAuditHandler(s.cli).Tool(), tools.NewNightAuditHandler(s.cli).Handle)
		srv.AddTool(tools.NewSpawnPersonaHandler(s.cli).Tool(), tools.NewSpawnPersonaHandler(s.cli).Handle)
		srv.AddTool(tools.NewFleetStatusHandler(s.cli).Tool(), tools.NewFleetStatusHandler(s.cli).Handle)
		srv.AddTool(tools.NewLoadtestHandler(s.cli).Tool(), tools.NewLoadtestHandler(s.cli).Handle)
		s.toolCount += 6
	}

	return srv
}

// Run starts the MCP server using the configured transport.
func (s *Server) Run(ctx context.Context, transport string) error {
	s.logger.Info("MCP server ready", zap.String("transport", transport))
	switch transport {
	case "stdio":
		stdioSrv := server.NewStdioServer(s.mcp)
		return stdioSrv.Listen(ctx, os.Stdin, os.Stdout)
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
	return s.toolCount
}
