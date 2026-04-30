# Quickstart

This guide gets `ironclaw-mcp` running against a local IronClaw gateway and
registered with an MCP-compatible assistant.

## 1. Install

```bash
go install github.com/nfsarch33/ironclaw-mcp/cmd/ironclaw-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/nfsarch33/ironclaw-mcp.git
cd ironclaw-mcp
make build
./bin/ironclaw-mcp --version
```

## 2. Start IronClaw

Start IronClaw with its HTTP gateway enabled. The default local URL expected by
`ironclaw-mcp` is:

```text
http://localhost:3000
```

If the gateway requires a bearer token, keep that token in the environment:

```bash
export IRONCLAW_BASE_URL=http://localhost:3000
export IRONCLAW_API_KEY=your-gateway-token
```

## 3. Register with your MCP client

Example MCP configuration:

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

Restart or reload MCP servers in the client. The `ironclaw_*` tools should
appear in the tool list.

## 4. Run a smoke test

From a source checkout:

```bash
make build
make smoke
```

For a structured report:

```bash
./scripts/smoke-test.sh --all --report
```

The smoke script uses deterministic payloads and skips destructive or slow tools
unless you explicitly configure them. To exercise chat, opt in:

```bash
SMOKE_STATEFUL_TOOL=ironclaw_chat make smoke
```

## Troubleshooting

- **Connection refused**: check that IronClaw is running and that
  `IRONCLAW_BASE_URL` points to the gateway.
- **401 Unauthorized**: set `IRONCLAW_API_KEY` to the gateway token.
- **Non-loopback URL rejected**: set `IRONCLAW_ALLOW_NON_LOCALHOST=true` only
  when intentionally using a remote gateway.
- **No tools visible**: restart the MCP client after editing its config.
