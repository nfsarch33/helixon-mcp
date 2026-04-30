# Configuration

`ironclaw-mcp` is configured entirely with environment variables. The default
configuration is local-first: stdio transport, loopback IronClaw gateway, and
only generic IronClaw HTTP bridge tools.

## Core variables

| Variable | Default | Description |
|---|---|---|
| `IRONCLAW_BASE_URL` | `http://localhost:3000` | Base URL of the IronClaw gateway. Must use `http` or `https`. Loopback-only by default. |
| `IRONCLAW_API_KEY` | empty | Optional bearer token sent to the IronClaw gateway. Use it when the gateway requires authentication. |
| `IRONCLAW_TIMEOUT_SECONDS` | `30` | HTTP timeout for IronClaw API calls. Valid range: `1` to `120`. |
| `IRONCLAW_ALLOW_NON_LOCALHOST` | `false` | Set to `true` to allow non-loopback `IRONCLAW_BASE_URL` hosts. |
| `MCP_TRANSPORT` | `stdio` | MCP transport: `stdio` or `sse`. |
| `MCP_SSE_ADDR` | `:8080` | Listen address used when `MCP_TRANSPORT=sse`. |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, or `error`. Logs are written to stderr. |

## Optional tool surfaces

| Variable | Default | Description |
|---|---|---|
| `PROMETHEUS_URL` | empty | Enables `ironclaw_get_metrics` for Prometheus queries. |
No local shell helpers or deployment-specific workflow packs are registered by
this repository. Build those as separate MCP servers so `ironclaw-mcp` remains
a portable HTTP bridge.

## Cursor / Claude Code stdio example

```json
{
  "mcpServers": {
    "ironclaw": {
      "command": "ironclaw-mcp",
      "env": {
        "IRONCLAW_BASE_URL": "http://localhost:3000",
        "IRONCLAW_API_KEY": "your-gateway-token"
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
ironclaw-mcp
```

Keep SSE bound to loopback unless you have a separate authentication and network
boundary in front of it.

## Security notes

- Keep `IRONCLAW_ALLOW_NON_LOCALHOST=false` unless you intentionally connect to
  a remote gateway.
- Do not put tokens in shell history, repository files, or shared logs.
- The default tool surface only forwards requests to `IRONCLAW_BASE_URL`.
- Keep deployment-specific workflow automation in separate MCP servers.
