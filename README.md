# helixon-mcp

[![CI](https://github.com/nfsarch33/ironclaw-mcp/actions/workflows/ci.yml/badge.svg)](https://github.com/nfsarch33/ironclaw-mcp/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nfsarch33/ironclaw-mcp)](https://goreportcard.com/report/github.com/nfsarch33/ironclaw-mcp)
[![Go Reference](https://pkg.go.dev/badge/github.com/nfsarch33/ironclaw-mcp.svg)](https://pkg.go.dev/github.com/nfsarch33/ironclaw-mcp)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A general-purpose **MCP (Model Context Protocol) server** in Go that bridges
[Helixon](https://github.com/nearai/helixon) — an open-source autonomous AI
assistant runtime — with any MCP-compatible client (Cursor, Claude Code,
VS Code Copilot, Continue, Zed, …).

The default tool surface speaks **only the documented Helixon HTTP gateway
API** and ships no deployment-specific assumptions, so a fresh install works
against any Helixon instance you point it at.

## Install in 2 minutes

```bash
# 1. Install (or build from source — see below)
go install github.com/nfsarch33/ironclaw-mcp/cmd/helixon-mcp@latest

# 2. Point at your Helixon gateway
export HELIXON_BASE_URL=http://localhost:3000
export HELIXON_API_KEY=your-gateway-token   # if GATEWAY_AUTH_TOKEN is set

# 3. Sanity check
helixon-mcp --version
```

Then add it to your MCP client. For Cursor / Claude Code / VS Code Copilot:

```json
{
  "mcpServers": {
    "helixon": {
      "command": "helixon-mcp",
      "env": {
        "HELIXON_BASE_URL": "http://localhost:3000",
        "HELIXON_API_KEY": "your-gateway-token"
      }
    }
  }
}
```

Or via Docker (no Go toolchain required):

```json
{
  "mcpServers": {
    "helixon": {
      "command": "docker",
      "args": [
        "run", "--rm", "-i",
        "-e", "HELIXON_BASE_URL=http://host.docker.internal:3000",
        "-e", "HELIXON_API_KEY",
        "ghcr.io/nfsarch33/helixon-mcp:latest"
      ]
    }
  }
}
```

Reload MCP servers in your client and the `helixon_*` tools will appear.

> Found a bug or want a feature? Please open an issue at
> <https://github.com/nfsarch33/ironclaw-mcp/issues>.

## Default Tool Surface (13 tools)

These tools are always registered when the Helixon gateway is reachable.
They speak only the documented Helixon HTTP API and are deployment-agnostic.

| Tool | Description |
|---|---|
| `helixon_health` | Check Helixon gateway availability and channel |
| `helixon_chat` | Send a message and wait for the async Helixon gateway response (optional `session_id` maps to gateway `thread_id`) |
| `helixon_list_jobs` | List all background jobs |
| `helixon_get_job` | Get details of a specific job |
| `helixon_cancel_job` | Cancel a running job |
| `helixon_search_memory` | Semantic search over Helixon workspace memory |
| `helixon_list_routines` | List all scheduled routines |
| `helixon_delete_routine` | Delete a routine |
| `helixon_list_tools` | List all tools registered in Helixon extensions |
| `helixon_stack_status` | Combined health of LLM router nodes, GPU availability, and gateway |
| `helixon_spawn_agent` | Spawn a new agent job with model and tier selection |
| `helixon_send_task` | Enqueue a background task/message via the Helixon gateway |
| `helixon_agent_status` | Agent thread states, active/total job counts, last heartbeat |

### Optional adjuncts

| Env var / flag | Tool(s) added | Notes |
|---|---|---|
| `PROMETHEUS_URL=...` | `helixon_get_metrics` | Prometheus query bridge for Helixon metrics; +1 tool |

No local shell helpers or deployment-specific tool packs are registered in this
repository. Fleet, workspace, and review workflows belong in separate MCP
servers so the default bridge stays portable.

## Configuration

All configuration is via environment variables.

| Variable | Default | Description |
|---|---|---|
| `HELIXON_BASE_URL` | `http://localhost:3000` | Helixon instance URL (must be http(s); loopback-only by default) |
| `HELIXON_API_KEY` | _(empty)_ | Bearer token for the Helixon gateway (`GATEWAY_AUTH_TOKEN`) when auth is enabled |
| `HELIXON_ALLOW_NON_LOCALHOST` | `false` | Set to `true` to allow non-loopback hosts (e.g. remote Helixon) |
| `HELIXON_TIMEOUT_SECONDS` | `30` | HTTP timeout in seconds (1–120) |
| `MCP_TRANSPORT` | `stdio` | `stdio` or `sse` |
| `MCP_SSE_ADDR` | `:8080` | Bind address when using SSE |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `PROMETHEUS_URL` | _(empty)_ | If set, registers the `helixon_get_metrics` tool |

**Secure local defaults:** By default, `HELIXON_BASE_URL` is restricted to
loopback addresses (`localhost`, `127.0.0.1`, `[::1]`). Set
`HELIXON_ALLOW_NON_LOCALHOST=true` only when you explicitly need to connect
to a remote Helixon instance.

See [`docs/configuration.md`](docs/configuration.md) for the full reference,
including transport-specific notes and Cursor integration tips.

## Build from source

```bash
git clone https://github.com/nfsarch33/ironclaw-mcp.git
cd helixon-mcp
make build
./bin/helixon-mcp --version
```

Or with Docker:

```bash
make docker-build
make docker-run
```

Prerequisites: **Go 1.24+** and a running [Helixon](https://github.com/nearai/helixon)
instance.

## Smoke test

A `make smoke` target exercises the local install end-to-end against an
already-running Helixon gateway:

```bash
make smoke                                          # gateway-only proof
SMOKE_STATEFUL_TOOL=helixon_chat make smoke         # full local chat path
make smoke SMOKE_ARGS="--all"                        # all 13 default tools
./scripts/smoke-test.sh --all --report | jq '.summary'  # JSON for CI
```

The `--all` flag exercises every tool with safe, deterministic payloads
(destructive tools are skipped with documented reasons). The `--report`
flag emits a structured JSON report suitable for CI pipelines.

See [`docs/quickstart.md`](docs/quickstart.md) for the full smoke
checklist.

## Development

```bash
make check        # tidy + fmt + vet + lint + test
make coverage     # generates coverage.out + HTML report
make lint         # golangci-lint only
```

Test coverage gate in CI is **70 %** total; current total is ~84 %.
See [`CONTRIBUTING.md`](CONTRIBUTING.md) for branch / commit conventions.

## Architecture

```
Cursor/Claude/Copilot ──stdio──► helixon-mcp (MCP server)
                                          │
                                  internal/tools/*
                                          │
                                  internal/helixon.Client
                                          │
                                       HTTP/REST
                                          │
                                    Helixon :3000
```

See [`docs/architecture.md`](docs/architecture.md) for the package-level
breakdown.

## Project Structure

```
helixon-mcp/
├── cmd/helixon-mcp/     # Entry point
├── internal/
│   ├── config/           # Environment config + validation
│   ├── helixon/         # Helixon HTTP API client
│   ├── tools/            # MCP tool handlers (one file per domain)
│   └── server/           # MCP server wiring
├── docs/                 # Quickstart, architecture, configuration
├── .github/              # CI workflows, issue/PR templates
├── .golangci.yml         # golangci-lint config
├── Dockerfile            # Multi-stage minimal image
└── Makefile              # Developer task runner
```

## Troubleshooting

- **Connection refused** — ensure Helixon is running and bound to
  `localhost:3000` (or set `HELIXON_BASE_URL`).
- **401 Unauthorized** — set `HELIXON_API_KEY` to the token Helixon prints
  at startup, or to whatever you set `GATEWAY_AUTH_TOKEN` to.
- **Non-loopback URL rejected** — set `HELIXON_ALLOW_NON_LOCALHOST=true`
  only when intentionally connecting to a remote Helixon instance.
- **Chat appears asynchronous** — `helixon-mcp` polls Helixon thread
  history after `/api/chat/send`; if the underlying model is overloaded the
  tool can time out instead of returning a partial response.

## Recent releases

- **v0.5.1 — public polish:** Added public documentation and governance
  files, removed non-gateway helper surfaces, and scrubbed release notes for
  public consumption.
- **v0.5.0 — generic-by-default:** Tool surface flipped to ship only the
  generic Helixon HTTP-gateway bridge by default.
- **v0.4.0 — scraper cleanup:** Removed all `helixon_research_*`,
  `helixon_ui_*`, and `helixon_evolver_*` tools that wrapped the
  external domain-specific research CLI.

See [`CHANGELOG.md`](CHANGELOG.md) for the full release history.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md) for development workflow,
branch / commit conventions, and the test coverage policy. Security issues:
see [`SECURITY.md`](SECURITY.md).

## License

[MIT](LICENSE) © 2026 helixon-mcp authors.
