# Security Policy

`ironclaw-mcp` is a stdio/SSE MCP server that bridges AI coding assistants to
an [IronClaw](https://github.com/nearai/ironclaw) HTTP gateway. It runs on the
operator's workstation alongside the assistant, with credentials supplied via
environment variables.

## Supported versions

The latest minor release on `main` is the only branch that receives security
fixes. Older `v0.x` releases are not maintained.

| Version | Supported |
|---------|-----------|
| `v0.5.x` (latest minor) | ✅ |
| `< v0.5.0` | ❌ |

## Reporting a vulnerability

Please **do not open a public GitHub issue** for security reports.

Use GitHub's private vulnerability reporting flow instead:

1. Go to the project's **Security** tab on GitHub.
2. Click **"Report a vulnerability"**.
3. Describe the issue, attach a minimal reproducer if possible, and indicate
   whether the issue is exploitable in the default loopback-only configuration
   or only with `IRONCLAW_ALLOW_NON_LOCALHOST=true`.

We aim to acknowledge new reports within **72 hours** and to provide an
initial triage decision (accept / decline / need more information) within
**7 days**. Coordinated disclosure timelines are agreed case-by-case.

## Threat model summary

`ironclaw-mcp` is intended for **single-operator, local-trust** deployments:

- The MCP transport is `stdio` by default; the assistant is the only client.
- The IronClaw gateway base URL is **loopback-only by default**
  (`localhost`, `127.0.0.1`, `[::1]`). Non-loopback hosts are rejected unless
  `IRONCLAW_ALLOW_NON_LOCALHOST=true` is set explicitly.
- The bearer token (`IRONCLAW_API_KEY`) is read from the environment, never
  logged, and only forwarded to the configured `IRONCLAW_BASE_URL`.
- The server does not persist data, listen on the network in stdio mode,
  expose unauthenticated endpoints, or escalate filesystem privileges.

Out of scope:

- Hardening against a hostile MCP host (the AI assistant is trusted).
- Hardening against a hostile IronClaw gateway (a compromised gateway can
  return malicious tool responses; treat returned content as untrusted input
  in the LLM, not in the bridge).
- Multi-tenant or shared-host deployment of the bridge itself.

## Hardening recommendations for operators

- Keep `IRONCLAW_ALLOW_NON_LOCALHOST=false` (the default).
- Bind any SSE transport (`MCP_TRANSPORT=sse`) to loopback addresses only.
- Set `IRONCLAW_API_KEY` to the value emitted by IronClaw at startup; rotate
  whenever the gateway emits a new token.
- Run the binary as a non-root user; the Docker image runs from `scratch`
  with no shell.

Thanks for helping keep `ironclaw-mcp` and its operators safe.
