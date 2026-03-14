# ironclaw-mcp

[![CI](https://github.com/nfsarch33/ironclaw-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/nfsarch33/ironclaw-mcp/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nfsarch33/ironclaw-mcp)](https://goreportcard.com/report/github.com/nfsarch33/ironclaw-mcp)

A production-ready **MCP (Model Context Protocol) server** written in Go that bridges [IronClaw](https://github.com/nearai/ironclaw) with any MCP-compatible AI coding assistant — Cursor, Claude Code, VS Code Copilot, and more.

## Tools Exposed

| Tool | Description |
|---|---|
| `ironclaw_health` | Check IronClaw gateway availability and channel |
| `ironclaw_chat` | Send a message and wait for the async IronClaw gateway response (optional `session_id` maps to gateway `thread_id`) |
| `ironclaw_list_jobs` | List all background jobs |
| `ironclaw_get_job` | Get details of a specific job |
| `ironclaw_cancel_job` | Cancel a running job |
| `ironclaw_search_memory` | Semantic search over IronClaw workspace memory |
| `ironclaw_list_routines` | List all scheduled routines |
| `ironclaw_delete_routine` | Delete a routine |
| `ironclaw_list_tools` | List all tools registered in IronClaw extensions |
| `ironclaw_stack_status` | Combined health of LLM router nodes, GPU availability, and gateway |
| `ironclaw_spawn_agent` | Spawn a new agent job with model and tier selection |
| `ironclaw_send_task` | Send a strategic task for background execution |
| `ironclaw_agent_status` | Agent thread states, active/total job counts, last heartbeat |
| `ironclaw_get_metrics` | Query Prometheus for agent metrics (requires `PROMETHEUS_URL`) |
| `ironclaw_reviewed_push` | Run Gemini diff review, then push only when no must-fix issues remain |

## Quick Start

### Prerequisites
- Go 1.23+
- A running [IronClaw](https://github.com/nearai/ironclaw) instance

### Run with `go run`

```bash
export IRONCLAW_BASE_URL=http://localhost:3000
export IRONCLAW_API_KEY=your-api-key   # required when IronClaw gateway auth is enabled
go run ./cmd/ironclaw-mcp
```

### Build and run

```bash
make build
./bin/ironclaw-mcp
```

### Docker

```bash
make docker-build
make docker-run
```

## Configuration

All configuration is via environment variables:

| Variable | Default | Description |
|---|---|---|
| `IRONCLAW_BASE_URL` | `http://localhost:3000` | IronClaw instance URL (must be http(s); loopback-only by default) |
| `IRONCLAW_API_KEY` | _(empty)_ | Bearer token for the IronClaw gateway (`GATEWAY_AUTH_TOKEN`) when auth is enabled |
| `IRONCLAW_ALLOW_NON_LOCALHOST` | `false` | Set to `true` to allow non-loopback hosts (e.g. remote IronClaw) |
| `IRONCLAW_TIMEOUT_SECONDS` | `30` | HTTP timeout in seconds (1–120) |
| `MCP_TRANSPORT` | `stdio` | `stdio` or `sse` |
| `MCP_SSE_ADDR` | `:8080` | Bind address when using SSE |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

**Secure local defaults:** By default, `IRONCLAW_BASE_URL` is restricted to loopback addresses (`localhost`, `127.0.0.1`, `[::1]`). Set `IRONCLAW_ALLOW_NON_LOCALHOST=true` only when you explicitly need to connect to a remote IronClaw instance.

## Cursor / Claude Code Integration

Add to your `~/.cursor/mcp.json` (or Claude Code config):

```json
{
  "mcpServers": {
    "ironclaw": {
      "command": "/path/to/bin/ironclaw-mcp",
      "env": {
        "IRONCLAW_BASE_URL": "http://localhost:3000",
        "IRONCLAW_API_KEY": "your-key"
      }
    }
  }
}
```

Or with Docker:

```json
{
  "mcpServers": {
    "ironclaw": {
      "command": "docker",
      "args": ["run", "--rm", "-i",
        "-e", "IRONCLAW_BASE_URL=http://host.docker.internal:3000",
        "ironclaw-mcp:latest"
      ]
    }
  }
}
```

## Development

```bash
# Full check (tidy + fmt + vet + lint + test)
make check

# Test with coverage report
make coverage

# Run linter only
make lint
```

## Project Structure

```
ironclaw-mcp/
├── cmd/ironclaw-mcp/     # Entry point
├── internal/
│   ├── config/           # Environment config + validation
│   ├── ironclaw/         # IronClaw HTTP API client
│   ├── tools/            # MCP tool handlers (one file per domain)
│   └── server/           # MCP server wiring
├── .github/workflows/    # CI: lint, test, build, security scan
├── .golangci.yml         # golangci-lint config
├── Dockerfile            # Multi-stage scratch image
└── Makefile              # Developer task runner
```

## Local Integration Path (Cursor + IronClaw)

The supported local setup for Cursor integration:

1. **IronClaw runtime** — bound to loopback (`GATEWAY_HOST=127.0.0.1`, default)
2. **ironclaw-mcp bridge** — stdio transport in Cursor, `IRONCLAW_BASE_URL=http://localhost:3000`
3. **Minimal config** — `~/.cursor/mcp.json` with `command` and `IRONCLAW_API_KEY` whenever the gateway is protected by `GATEWAY_AUTH_TOKEN`

### Smoke Test Workflow

1. Start IronClaw with loopback-first settings (gateway on `http://localhost:3000`)
2. Export `IRONCLAW_API_KEY` if the gateway is protected by `GATEWAY_AUTH_TOKEN`
3. Run `make smoke` for the gateway-only proof, or `SMOKE_STATEFUL_TOOL=ironclaw_chat make smoke` for the full local chat-path proof
4. The harness verifies:
   - IronClaw gateway `/api/health`
   - Router `/healthz` and `/v1/models` when the chat-path proof is requested
   - MCP `initialize`
   - MCP `tools/list`
   - `ironclaw_health`
   - One configurable stateful tool (`ironclaw_list_jobs` by default, or `SMOKE_STATEFUL_TOOL=ironclaw_chat`)
5. Add `ironclaw-mcp` to `~/.cursor/mcp.json` and reload MCP servers

#### `--all` and `--report` flags

```bash
# Test all 15 tools with deterministic payloads
make smoke SMOKE_ARGS="--all"

# Get JSON test results (pipe to jq, CI artifacts, etc.)
./scripts/smoke-test.sh --all --report

# JSON output includes per-tool pass/fail/skip status and timing
./scripts/smoke-test.sh --all --report | jq '.summary'
```

The `--all` flag exercises every tool with safe, deterministic payloads (destructive tools like `ironclaw_delete_routine` and slow tools like `ironclaw_chat` are skipped with documented reasons). The `--report` flag outputs a structured JSON report suitable for CI pipelines.

### Troubleshooting

- **Connection refused**: Ensure IronClaw is running and bound to localhost:3000
- **Router health check fails**: Ensure `llm-cluster-router` is listening on `SMOKE_ROUTER_URL` and the local upstreams have finished model load
- **Expected model missing from `/v1/models`**: Check the router config and make sure the primary `qwen3.5-27b` upstream is healthy before re-running chat smoke
- **401 Unauthorized**: Set `IRONCLAW_API_KEY` to the token printed by IronClaw at startup (or `GATEWAY_AUTH_TOKEN` if you set it)
- **Non-loopback URL rejected**: Use `IRONCLAW_ALLOW_NON_LOCALHOST=true` only when connecting to a remote IronClaw instance
- **Chat appears asynchronous**: `ironclaw-mcp` now polls IronClaw thread history after `/api/chat/send`; if the local model is overloaded or unavailable, the tool call can time out instead of returning a partial response
- **Need a full chat-path proof**: Run `SMOKE_STATEFUL_TOOL=ironclaw_chat make smoke` once the local router exposes `qwen3.5-27b`, or set `SMOKE_REQUIRE_ROUTER=true` to force the same router checks for a non-chat probe

## Architecture

```
Cursor/Claude ──stdio──► ironclaw-mcp (MCP server)
                              │
                    internal/tools/*
                              │
                    internal/ironclaw.Client
                              │
                         HTTP/REST
                              │
                        IronClaw :3000
```

## License

MIT
