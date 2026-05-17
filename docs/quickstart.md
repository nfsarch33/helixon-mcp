# Quickstart

This guide gets `helixon-mcp` running against a local Helixon gateway and
registered with an MCP-compatible assistant.

## 1. Install

```bash
go install github.com/nfsarch33/ironclaw-mcp/cmd/helixon-mcp@latest
```

Or build from source:

```bash
git clone https://github.com/nfsarch33/ironclaw-mcp.git
cd helixon-mcp
make build
./bin/helixon-mcp --version
```

## 2. Start Helixon

Start Helixon with its HTTP gateway enabled. The default local URL expected by
`helixon-mcp` is:

```text
http://localhost:3000
```

If the gateway requires a bearer token, keep that token in the environment:

```bash
export HELIXON_BASE_URL=http://localhost:3000
export HELIXON_API_KEY=your-gateway-token
```

## 3. Register with your MCP client

Example MCP configuration:

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

Restart or reload MCP servers in the client. The `helixon_*` tools should
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
SMOKE_STATEFUL_TOOL=helixon_chat make smoke
```

## Troubleshooting

- **Connection refused**: check that Helixon is running and that
  `HELIXON_BASE_URL` points to the gateway.
- **401 Unauthorized**: set `HELIXON_API_KEY` to the gateway token.
- **Non-loopback URL rejected**: set `HELIXON_ALLOW_NON_LOCALHOST=true` only
  when intentionally using a remote gateway.
- **No tools visible**: restart the MCP client after editing its config.
