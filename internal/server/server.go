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
	logger    *slog.Logger
	version   string
	mcp       *mcpserver.MCPServer
	sse       *mcpserver.SSEServer
	toolCount int
}

// New creates and configures a new MCP Server with the generic IronClaw gateway
// tools plus optional Prometheus metrics support.
func New(client tools.IronclawClient, prom tools.PrometheusQuerier, logger *slog.Logger, version string) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		client:  client,
		prom:    prom,
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

	sendTask := tools.NewSendTaskHandler(s.client)
	addTool(sendTask.Tool(), sendTask.Handle)

	agentStatus := tools.NewAgentStatusHandler(s.client)
	addTool(agentStatus.Tool(), agentStatus.Handle)

	if s.prom != nil {
		getMetrics := tools.NewGetMetricsHandler(s.prom)
		addTool(getMetrics.Tool(), getMetrics.Handle)
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
		if err := sseSrv.Shutdown(context.Background()); err != nil {
			s.logger.Debug("SSE MCP shutdown returned error", "error", err)
		}
		if err := srv.Close(); err != nil {
			s.logger.Debug("SSE HTTP server close returned error", "error", err)
		}
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
