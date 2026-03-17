// Package server wires all MCP tools together and runs the MCP server.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/nfsarch33/ironclaw-mcp/internal/tools"
)

// Server wraps the MCP server and its dependencies.
type Server struct {
	client    tools.IronclawClient
	prom      tools.PrometheusQuerier
	cli       tools.CLIRunner
	gws       tools.CLIRunner
	logger    *slog.Logger
	version   string
	mcp       *mcpserver.MCPServer
	sse       *mcpserver.SSEServer
	toolCount int
}

// New creates and configures a new MCP Server with all IronClaw tools registered.
// prom, cli, and gws are optional; when set, the corresponding tools are registered.
func New(client tools.IronclawClient, prom tools.PrometheusQuerier, cli tools.CLIRunner, gws tools.CLIRunner, logger *slog.Logger, version string) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		client:  client,
		prom:    prom,
		cli:     cli,
		gws:     gws,
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
	addTool(research.CrawlTool(), research.HandleCrawl)
	addTool(research.DeakinTool(), research.HandleDeakin)
	addTool(research.AssessmentsTool(), research.HandleAssessments)

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

	if s.gws != nil {
		srv.AddTool(tools.NewGWSToolHandler(s.gws).Tool(), tools.NewGWSToolHandler(s.gws).Handle)
		s.toolCount += 1
	}

	if s.cli != nil {
		// Sprint 68: Core ops MCP tools
		addTool(tools.NewDoctorHandler(s.cli).Tool(), tools.NewDoctorHandler(s.cli).Handle)
		addTool(tools.NewStatusHandler(s.cli).Tool(), tools.NewStatusHandler(s.cli).Handle)
		addTool(tools.NewInstallHandler(s.cli).Tool(), tools.NewInstallHandler(s.cli).Handle)
		addTool(tools.NewDeployHandler(s.cli).Tool(), tools.NewDeployHandler(s.cli).Handle)
		addTool(tools.NewLogsHandler(s.cli).Tool(), tools.NewLogsHandler(s.cli).Handle)
		addTool(tools.NewSpawnFullHandler(s.cli).Tool(), tools.NewSpawnFullHandler(s.cli).Handle)
		addTool(tools.NewListAgentsHandler(s.cli).Tool(), tools.NewListAgentsHandler(s.cli).Handle)
		addTool(tools.NewStopAgentHandler(s.cli).Tool(), tools.NewStopAgentHandler(s.cli).Handle)
		addTool(tools.NewGPUStatusHandler(s.cli).Tool(), tools.NewGPUStatusHandler(s.cli).Handle)
		addTool(tools.NewCostSummaryHandler(s.cli).Tool(), tools.NewCostSummaryHandler(s.cli).Handle)
		addTool(tools.NewMemoryStatsHandler(s.cli).Tool(), tools.NewMemoryStatsHandler(s.cli).Handle)

		// Sprint 69: Extended ops MCP tools
		addTool(tools.NewFleetHandler(s.cli).Tool(), tools.NewFleetHandler(s.cli).Handle)
		addTool(tools.NewRoutineHandler(s.cli).Tool(), tools.NewRoutineHandler(s.cli).Handle)
		addTool(tools.NewA2AFullHandler(s.cli).Tool(), tools.NewA2AFullHandler(s.cli).Handle)
		addTool(tools.NewSnapshotHandler(s.cli).Tool(), tools.NewSnapshotHandler(s.cli).Handle)
		addTool(tools.NewRecoverHandler(s.cli).Tool(), tools.NewRecoverHandler(s.cli).Handle)
		addTool(tools.NewWorkspaceHandler(s.cli).Tool(), tools.NewWorkspaceHandler(s.cli).Handle)
		addTool(tools.NewCRMFullHandler(s.cli).Tool(), tools.NewCRMFullHandler(s.cli).Handle)
		addTool(tools.NewSkillsHandler(s.cli).Tool(), tools.NewSkillsHandler(s.cli).Handle)
		addTool(tools.NewCEOOrchestrateHandler(s.cli).Tool(), tools.NewCEOOrchestrateHandler(s.cli).Handle)
		addTool(tools.NewJobOpsHandler(s.cli).Tool(), tools.NewJobOpsHandler(s.cli).Handle)
		addTool(tools.NewExportDashboardsHandler(s.cli).Tool(), tools.NewExportDashboardsHandler(s.cli).Handle)

		k8sOps := tools.NewK8sOpsHandler(s.cli)
		addTool(k8sOps.Tool(), k8sOps.Handle)

		tfOps := tools.NewTfOpsHandler(s.cli)
		addTool(tfOps.Tool(), tfOps.Handle)

		fleetOps := tools.NewFleetOpsHandler(s.cli)
		addTool(fleetOps.Tool(), fleetOps.Handle)

		grafanaOps := tools.NewGrafanaOpsHandler(s.cli)
		addTool(grafanaOps.Tool(), grafanaOps.Handle)

		governanceOps := tools.NewGovernanceHandler(s.cli)
		addTool(governanceOps.Tool(), governanceOps.Handle)

		timelineOps := tools.NewTimelineHandler(s.cli)
		addTool(timelineOps.Tool(), timelineOps.Handle)

		llmRoute := tools.NewLLMRouteHandler(s.cli)
		addTool(llmRoute.Tool(), llmRoute.Handle)

		llmUsage := tools.NewLLMUsageHandler(s.cli)
		addTool(llmUsage.Tool(), llmUsage.Handle)

		llmBudget := tools.NewLLMBudgetHandler(s.cli)
		addTool(llmBudget.Tool(), llmBudget.Handle)
	}

	return srv
}

// Run starts the MCP server using the configured transport.
func (s *Server) Run(ctx context.Context, transport string) error {
	s.logger.Info("MCP server ready", "transport", transport)
	switch transport {
	case "stdio":
		stdioSrv := mcpserver.NewStdioServer(s.mcp)
		return stdioSrv.Listen(ctx, os.Stdin, os.Stdout)
	case "sse":
		return s.runSSE(ctx)
	default:
		return fmt.Errorf("unknown transport %q", transport)
	}
}

// SSEAddr is the default SSE listen address.
const SSEAddr = ":9090"

func (s *Server) runSSE(ctx context.Context) error {
	addr := SSEAddr
	if envAddr := os.Getenv("MCP_SSE_ADDR"); envAddr != "" {
		addr = envAddr
	}
	sseSrv := mcpserver.NewSSEServer(s.mcp,
		mcpserver.WithBaseURL(fmt.Sprintf("http://localhost%s", addr)),
		mcpserver.WithKeepAlive(true),
	)
	s.sse = sseSrv
	s.logger.Info("SSE server starting", "addr", addr)

	mux := http.NewServeMux()
	mux.Handle("/", sseSrv)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","transport":"sse","tools":%d}`, s.toolCount)
	})

	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		s.logger.Info("SSE server shutting down")
		sseSrv.Shutdown(context.Background())
		srv.Close()
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("SSE server: %w", err)
	}
	return nil
}

// MCPServer exposes the underlying MCP server (for testing).
func (s *Server) MCPServer() *mcpserver.MCPServer {
	return s.mcp
}

// RegisteredToolCount returns how many tools are registered (for testing).
func (s *Server) RegisteredToolCount() int {
	return s.toolCount
}
