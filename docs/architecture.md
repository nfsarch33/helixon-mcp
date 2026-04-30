# Architecture

`ironclaw-mcp` is a small bridge between MCP clients and the IronClaw HTTP
gateway. The default surface is deployment-agnostic: handlers translate MCP tool
calls into documented IronClaw HTTP requests and return structured tool results.

```text
MCP client
  Cursor / Claude Code / Copilot / Zed
        |
        | stdio or SSE
        v
ironclaw-mcp
        |
        | internal/server
        v
internal/tools handlers
        |
        | internal/ironclaw.Client
        v
IronClaw HTTP gateway
```

## Packages

| Package | Responsibility |
|---|---|
| `cmd/ironclaw-mcp` | Entrypoint, env loading, transport startup, basic `--help` and `--version`. |
| `internal/config` | Environment parsing and validation. |
| `internal/server` | MCP server construction and tool registration. |
| `internal/tools` | MCP tool schemas and handlers. |
| `internal/ironclaw` | HTTP client for IronClaw gateway endpoints. |

## Default tool registration

The default server registers only generic IronClaw gateway tools:

- health
- chat
- jobs
- memory search
- routines
- tool listing
- stack status
- agent spawn
- task send
- agent status

Optional adjuncts are gated by environment variables:

- `PROMETHEUS_URL` adds Prometheus metric queries.

Deployment-specific workflows, local shell helpers, and fleet operations belong
in separate MCP servers. They are intentionally not registered here.

## Transport

Stdio is the primary transport for local assistant integrations. SSE is provided
for setups that need an HTTP-accessible MCP endpoint, but should remain
loopback-bound unless you add your own network controls.

## Logging

The MCP protocol uses stdout, so application logs are written to stderr with
Go's `slog` JSON handler.

## Testing strategy

- Tool handlers are tested with mock clients or mock command runners.
- HTTP client behaviour is tested with `httptest.NewServer`.
- Server registration counts are pinned in `internal/server/server_test.go`.
- CI runs lint, race-enabled tests, coverage gate, build, Docker build, and
  `govulncheck`.
