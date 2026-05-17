# Configuration

`helixon-mcp` is configured entirely with environment variables. The default
configuration is local-first: stdio transport, loopback Helixon gateway, and
only generic Helixon HTTP bridge tools.

## Core variables

| Variable | Default | Description |
|---|---|---|
| `HELIXON_BASE_URL` | `http://localhost:3000` | Base URL of the Helixon gateway. Must use `http` or `https`. Loopback-only by default. |
| `HELIXON_API_KEY` | empty | Optional bearer token sent to the Helixon gateway. Use it when the gateway requires authentication. |
| `HELIXON_TIMEOUT_SECONDS` | `30` | HTTP timeout for Helixon API calls. Valid range: `1` to `120`. |
| `HELIXON_ALLOW_NON_LOCALHOST` | `false` | Set to `true` to allow non-loopback `HELIXON_BASE_URL` hosts. |
| `MCP_TRANSPORT` | `stdio` | MCP transport: `stdio` or `sse`. |
| `MCP_SSE_ADDR` | `:8080` | Listen address used when `MCP_TRANSPORT=sse`. |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error`. Logs are written to stderr. |

## Optional tool surfaces

| Variable | Default | Description |
|---|---|---|
| `PROMETHEUS_URL` | empty | Enables `helixon_get_metrics` for Prometheus queries. |
No local shell helpers or deployment-specific workflow packs are registered by
this repository. Build those as separate MCP servers so `helixon-mcp` remains
a portable HTTP bridge.

## Cursor / Claude Code stdio example

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

## SSE transport

Stdio is recommended for local assistant integrations. Use SSE only when you
need an HTTP-accessible MCP endpoint:

```bash
export MCP_TRANSPORT=sse
export MCP_SSE_ADDR=127.0.0.1:8080
helixon-mcp
```

Keep SSE bound to loopback unless you have a separate authentication and network
boundary in front of it.

## Security notes

- Keep `HELIXON_ALLOW_NON_LOCALHOST=false` unless you intentionally connect to
  a remote gateway.
- Do not put tokens in shell history, repository files, or shared logs.
- The default tool surface only forwards requests to `HELIXON_BASE_URL`.
- Keep deployment-specific workflow automation in separate MCP servers.
