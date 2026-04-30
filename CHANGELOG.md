# Changelog

All notable changes to `ironclaw-mcp` are recorded in this file. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned

- v1.0.0 release: dedicated CLI binary, not just an MCP server.
  - `ironclaw-mcp serve` (current behaviour, default)
  - `ironclaw-mcp doctor` (config + IronClaw connectivity probes)
  - `ironclaw-mcp tools list|describe|invoke`
  - `ironclaw-mcp smoke` (replaces `scripts/smoke-test.sh`)
- Push `internal/tools` and `internal/ironclaw` test coverage to >= 80 %
  using the existing `httptest.NewServer` mock harness.
- Add `cmd/ironclaw-mcp` integration tests covering CLI flag parsing,
  graceful shutdown, and SSE transport setup.

## [0.5.1] - 2026-05-01

### Changed

- Added public project documentation for configuration, quickstart, and
  architecture.
- Added MIT license, security policy, contributing guide, and code of conduct.
- Removed non-gateway helper and legacy workflow surfaces from this repository.
  The tool catalog now contains only generic IronClaw HTTP gateway tools plus
  the optional Prometheus metrics query tool.
- Added `--help` and `--version` entrypoint coverage.
- Updated README install, Docker, troubleshooting, and release sections for
  public open-source users.

## [0.5.0] - 2026-04-30

### Changed

- **Generic-by-default tool surface.** The bridge defaults to the generic
  IronClaw HTTP-gateway tools plus `ironclaw_get_metrics` when
  `PROMETHEUS_URL` is set.
- Deployment-specific helper surfaces were no longer enabled by default.
- Tool descriptions were rewritten in capability-based language.
- Source-comment markers from earlier internal planning were replaced with
  capability labels in `internal/server/server.go` and
  `internal/server/server_test.go`.

### Migration

Deployment-specific workflows should move to a separate MCP server.

### Rationale

`ironclaw-mcp` is the canonical **generic** IronClaw HTTP bridge. Deployment-
specific orchestration (fleet operations, persona spawn, Workspace) is
better served by a dedicated sibling MCP that can evolve its tool catalog
independently. This release flips the default surface so a fresh install
matches that intent without breaking existing operators.

## [0.4.0] - 2026-04-30

### Removed (BREAKING)

- All `ironclaw_research_*` MCP tools. These wrapped an external
  domain-specific research CLI and are not part of the IronClaw HTTP bridge
  surface.
- All `ironclaw_ui_*` MCP tools (`navigate`, `discover`, `heal`, `verify`).
- All `ironclaw_evolver_*` MCP tools (`status`, `propose`, `validate`,
  `promote`).
- Internal source files: `internal/tools/research.go`,
  `internal/tools/research_test.go`, `internal/tools/uiauto.go`,
  `internal/tools/uiauto_test.go`, `internal/tools/evolver.go`,
  `internal/tools/evolver_test.go`. Approx. 2 629 lines of Go removed.

### Migration

Consumers that relied on the removed surfaces should call their preferred
research, browser-automation, or evolution tooling directly. `ironclaw-mcp`
no longer wraps those domain-specific workflows.

### Changed

- `internal/server/server.go` no longer registers research, UI-auto, or
  evolver handlers. At the time, base tool count became **14** (15 with
  `PROMETHEUS_URL`).
- `internal/server/server_test.go` updated with new tool-count
  expectations and the v1.0 baseline comment.
- `README.md` documents the cleanup, the v1.0 direction, and the new
  per-config tool counts.

## [0.3.0] - 2026-04-21

### Added

- Optional local review and ops helper surfaces. Removed in v0.5.1 because
  they are not generic IronClaw HTTP gateway operations.
- Prometheus metrics endpoint and `ironclaw_get_metrics` tool.
