// Package server wires all MCP tools together and runs the MCP server.
package server

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/nfsarch33/ironclaw-mcp/internal/tools"
	"go.uber.org/zap"
)

// Server wraps the MCP server and its dependencies.
type Server struct {
	client    tools.IronclawClient
	prom      tools.PrometheusQuerier
	cli       tools.CLIRunner
	logger    *zap.Logger
	version   string
	mcp       *mcpserver.MCPServer
	toolCount int
}

// New creates and configures a new MCP Server with all IronClaw tools registered.
// prom and cli are optional; when set, the corresponding tools are registered.
func New(client tools.IronclawClient, prom tools.PrometheusQuerier, cli tools.CLIRunner, logger *zap.Logger, version string) *Server {
	s := &Server{
		client:  client,
		prom:    prom,
		cli:     cli,
		logger:  logger,
		version: version,
	}
	s.mcp = s.buildMCPServer()
	return s
}

func (s *Server) buildMCPServer() *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer(
		"ironclaw-mcp",
		s.version,
		mcpserver.WithToolCapabilities(true),
	)

	addTool := func(t mcp.Tool, h mcpserver.ToolHandlerFunc) {
		srv.AddTool(t, h)
		s.toolCount++
	}

	health := tools.NewHealthHandler(s.client)
	addTool(health.Tool(), health.Handle)

	chat := tools.NewChatHandler(s.client)
	addTool(chat.Tool(), chat.Handle)

	jobs := tools.NewJobsHandler(s.client)
	addTool(jobs.ListJobsTool(), jobs.HandleListJobs)
	addTool(jobs.GetJobTool(), jobs.HandleGetJob)
	addTool(jobs.CancelJobTool(), jobs.HandleCancelJob)

	mem := tools.NewMemoryHandler(s.client)
	addTool(mem.Tool(), mem.Handle)

	routines := tools.NewRoutinesHandler(s.client)
	addTool(routines.ListRoutinesTool(), routines.HandleListRoutines)
	addTool(routines.DeleteRoutineTool(), routines.HandleDeleteRoutine)

	toolsList := tools.NewToolsListHandler(s.client)
	addTool(toolsList.Tool(), toolsList.Handle)

	stackStatus := tools.NewStackStatusHandler(s.client)
	addTool(stackStatus.Tool(), stackStatus.Handle)

	spawnAgent := tools.NewSpawnAgentHandler(s.client)
	addTool(spawnAgent.Tool(), spawnAgent.Handle)

	reviewedPush := tools.NewReviewedPushHandler()
	addTool(reviewedPush.Tool(), reviewedPush.Handle)

	sendTask := tools.NewSendTaskHandler(s.client)
	addTool(sendTask.Tool(), sendTask.Handle)

	agentStatus := tools.NewAgentStatusHandler(s.client)
	addTool(agentStatus.Tool(), agentStatus.Handle)

	research := tools.NewResearchHandler()
	addTool(research.ScrapeTool(), research.HandleScrape)
	addTool(research.PDFTool(), research.HandlePDF)
	addTool(research.SearchTool(), research.HandleSearch)
	addTool(research.StoreTool(), research.HandleStore)
	addTool(research.PipelineTool(), research.HandlePipeline)
	addTool(research.TranscriptTool(), research.HandleTranscript)
	addTool(research.ExtractTool(), research.HandleExtract)

	if s.prom != nil {
		getMetrics := tools.NewGetMetricsHandler(s.prom)
		addTool(getMetrics.Tool(), getMetrics.Handle)
	}

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
		stdioSrv := mcpserver.NewStdioServer(s.mcp)
		return stdioSrv.Listen(ctx, os.Stdin, os.Stdout)
	case "sse":
		return fmt.Errorf("SSE transport not yet implemented; use stdio")
	default:
		return fmt.Errorf("unknown transport %q", transport)
	}
}

// MCPServer exposes the underlying MCP server (for testing).
func (s *Server) MCPServer() *mcpserver.MCPServer {
	return s.mcp
}

// RegisteredToolCount returns how many tools are registered (for testing).
func (s *Server) RegisteredToolCount() int {
	return s.toolCount
}
