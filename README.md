# ironclaw-mcp

[![CI](https://github.com/nfsarch33/ironclaw-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/nfsarch33/ironclaw-mcp/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nfsarch33/ironclaw-mcp)](https://goreportcard.com/report/github.com/nfsarch33/ironclaw-mcp)

A production-ready **MCP (Model Context Protocol) server** written in Go that bridges [IronClaw](https://github.com/nearai/ironclaw) with any MCP-compatible AI coding assistant — Cursor, Claude Code, VS Code Copilot, and more.

## Tools Exposed

| Tool | Description |
|---|---|
| `ironclaw_health` | Check IronClaw availability and version |
| `ironclaw_chat` | Send a message and get a response (with optional session context) |
| `ironclaw_list_jobs` | List all background jobs |
| `ironclaw_get_job` | Get details of a specific job |
| `ironclaw_cancel_job` | Cancel a running job |
| `ironclaw_search_memory` | Semantic search over IronClaw workspace memory |
| `ironclaw_list_routines` | List all scheduled routines |
| `ironclaw_create_routine` | Create a new scheduled routine |
| `ironclaw_delete_routine` | Delete a routine |
| `ironclaw_list_tools` | List all tools registered in IronClaw |

## Quick Start

### Prerequisites
- Go 1.23+
- A running [IronClaw](https://github.com/nearai/ironclaw) instance

### Run with `go run`

```bash
export IRONCLAW_BASE_URL=http://localhost:3000
export IRONCLAW_API_KEY=your-api-key   # optional
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
| `IRONCLAW_BASE_URL` | `http://localhost:3000` | IronClaw instance URL |
| `IRONCLAW_API_KEY` | _(empty)_ | Optional bearer token |
| `IRONCLAW_TIMEOUT_SECONDS` | `30` | HTTP timeout in seconds |
| `MCP_TRANSPORT` | `stdio` | `stdio` or `sse` |
| `MCP_SSE_ADDR` | `:8080` | Bind address when using SSE |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |

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
