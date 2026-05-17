# Changelog

All notable changes to `helixon-mcp` are recorded in this file. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned

- v1.0.0 release: dedicated CLI binary, not just an MCP server.
  - `helixon-mcp serve` (current behaviour, default)
  - `helixon-mcp doctor` (config + Helixon connectivity probes)
  - `helixon-mcp tools list|describe|invoke`
  - `helixon-mcp smoke` (replaces `scripts/smoke-test.sh`)
- Push `internal/tools` and `internal/helixon` test coverage to >= 80 %
  using the existing `httptest.NewServer` mock harness.
- Add `cmd/helixon-mcp` integration tests covering CLI flag parsing,
  graceful shutdown, and SSE transport setup.

## [0.5.1] - 2026-05-01

### Changed

- Added public project documentation for configuration, quickstart, and
  architecture.
- Added MIT license, security policy, contributing guide, and code of conduct.
- Removed non-gateway helper and legacy workflow surfaces from this repository.
  The tool catalog now contains only generic Helixon HTTP gateway tools plus
  the optional Prometheus metrics query tool.
- Added `--help` and `--version` entrypoint coverage.
- Updated README install, Docker, troubleshooting, and release sections for
  public open-source users.

## [0.5.0] - 2026-04-30

### Changed

- **Generic-by-default tool surface.** The bridge defaults to the generic
  Helixon HTTP-gateway tools plus `helixon_get_metrics` when
  `PROMETHEUS_URL` is set.
- Deployment-specific helper surfaces were no longer enabled by default.
- Tool descriptions were rewritten in capability-based language.
- Source-comment markers from earlier internal planning were replaced with
  capability labels in `internal/server/server.go` and
  `internal/server/server_test.go`.

### Migration

Deployment-specific workflows should move to a separate MCP server.

### Rationale

`helixon-mcp` is the canonical **generic** Helixon HTTP bridge. Deployment-
specific orchestration (fleet operations, persona spawn, Workspace) is
better served by a dedicated sibling MCP that can evolve its tool catalog
independently. This release flips the default surface so a fresh install
matches that intent without breaking existing operators.

## [0.4.0] - 2026-04-30

### Removed (BREAKING)

- All `helixon_research_*` MCP tools. These wrapped an external
  domain-specific research CLI and are not part of the Helixon HTTP bridge
  surface.
- All `helixon_ui_*` MCP tools (`navigate`, `discover`, `heal`, `verify`).
- All `helixon_evolver_*` MCP tools (`status`, `propose`, `validate`,
  `promote`).
- Internal source files: `internal/tools/research.go`,
  `internal/tools/research_test.go`, `internal/tools/uiauto.go`,
  `internal/tools/uiauto_test.go`, `internal/tools/evolver.go`,
  `internal/tools/evolver_test.go`. Approx. 2 629 lines of Go removed.

### Migration

Consumers that relied on the removed surfaces should call their preferred
research, browser-automation, or evolution tooling directly. `helixon-mcp`
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
  they are not generic Helixon HTTP gateway operations.
- Prometheus metrics endpoint and `helixon_get_metrics` tool.
