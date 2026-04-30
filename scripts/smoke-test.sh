#!/usr/bin/env bash
# Deterministic smoke test for Cursor -> ironclaw-mcp -> IronClaw.
#
# Usage:
#   ./scripts/smoke-test.sh           # default: health + one stateful tool
#   ./scripts/smoke-test.sh --all     # test all default tools with deterministic payloads
#   ./scripts/smoke-test.sh --report  # output JSON results (combinable with --all)

set -euo pipefail

SMOKE_ALL=0
SMOKE_REPORT=0
for arg in "$@"; do
  case "$arg" in
    --all)    SMOKE_ALL=1 ;;
    --report) SMOKE_REPORT=1 ;;
    --help|-h)
      echo "Usage: $0 [--all] [--report]"
      echo "  --all     Test all default MCP tools with deterministic payloads"
      echo "  --report  Output JSON test results to stdout"
      exit 0
      ;;
    *) echo "Unknown argument: $arg"; exit 1 ;;
  esac
done

BASE_URL="${IRONCLAW_BASE_URL:-http://localhost:3000}"
API_KEY="${IRONCLAW_API_KEY:-${GATEWAY_AUTH_TOKEN:-}}"
BINARY="${BINARY:-./bin/ironclaw-mcp}"
STATEFUL_TOOL="${SMOKE_STATEFUL_TOOL:-ironclaw_list_jobs}"
CHAT_MESSAGE="${SMOKE_CHAT_MESSAGE:-Smoke test: reply with a short acknowledgement.}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-45}"
ROUTER_URL="${SMOKE_ROUTER_URL:-http://127.0.0.1:8080}"
EXPECT_MODEL="${SMOKE_EXPECT_MODEL:-}"
REQUIRE_ROUTER="${SMOKE_REQUIRE_ROUTER:-auto}"

health_url="${BASE_URL%/}/api/health"
router_health_url="${ROUTER_URL%/}/healthz"
router_models_url="${ROUTER_URL%/}/v1/models"
curl_args=(-sf --retry 5 --retry-delay 1)
if [[ -n "$API_KEY" ]]; then
  curl_args+=(-H "Authorization: Bearer $API_KEY")
fi

require_router=0
if [[ "$REQUIRE_ROUTER" == "true" ]]; then
  require_router=1
elif [[ "$REQUIRE_ROUTER" == "auto" && "$STATEFUL_TOOL" == "ironclaw_chat" ]]; then
  require_router=1
fi

smoke_mode="gateway-only"
total_steps=4
if [[ "$SMOKE_ALL" -eq 1 ]]; then
  smoke_mode="all-tools"
  total_steps=4
  if [[ "$require_router" -eq 1 ]]; then
    total_steps=5
  fi
elif [[ "$require_router" -eq 1 ]]; then
  smoke_mode="full chat-path"
  total_steps=5
fi

echo "=== IronClaw MCP Smoke Test ==="
echo "IRONCLAW_BASE_URL=$BASE_URL"
echo "MCP binary=$BINARY"
if [[ "$SMOKE_ALL" -eq 1 ]]; then
  echo "Mode=--all (testing all tools)"
else
  echo "Stateful tool=$STATEFUL_TOOL"
fi
echo "Smoke mode=$smoke_mode"
if [[ -n "$API_KEY" ]]; then
  echo "Gateway auth=configured"
else
  echo "Gateway auth=not set"
fi
if [[ "$require_router" -eq 1 ]]; then
  echo "Router URL=$ROUTER_URL"
  echo "Expected model=$EXPECT_MODEL"
fi
echo

echo "[1/$total_steps] Checking IronClaw gateway health at $health_url ..."
if ! curl "${curl_args[@]}" "$health_url" >/tmp/ironclaw-smoke-health.json; then
  echo "FAIL: IronClaw not reachable at $health_url."
  echo "Start IronClaw first, or export IRONCLAW_BASE_URL / IRONCLAW_API_KEY for your local gateway."
  exit 1
fi
echo "OK: IronClaw gateway is reachable"
echo

if [[ "$require_router" -eq 1 ]]; then
  echo "[2/$total_steps] Checking local router health at $router_health_url ..."
  if ! curl -sf "$router_health_url" >/tmp/ironclaw-smoke-router-health.json; then
    echo "FAIL: Router not reachable at $router_health_url."
    echo "Start llm-cluster-router and the local model upstreams first."
    exit 1
  fi
  if ! curl -sf "$router_models_url" >/tmp/ironclaw-smoke-router-models.json; then
    echo "FAIL: Router model list not reachable at $router_models_url."
    exit 1
  fi
  if [[ -n "$EXPECT_MODEL" ]] && ! python3 - "$EXPECT_MODEL" /tmp/ironclaw-smoke-router-models.json <<'PY'
import json
import sys

expected, path = sys.argv[1:]
with open(path, "r", encoding="utf-8") as fh:
    payload = json.load(fh)
models = {item.get("id") for item in payload.get("data", []) if isinstance(item, dict)}
if expected not in models:
    raise SystemExit(1)
PY
  then
    echo "FAIL: Router is healthy but expected model '$EXPECT_MODEL' is not listed."
    echo "Verify the configured model upstream is registered and healthy."
    exit 1
  fi
  echo "OK: Router health and model inventory look good"
  echo
  binary_step=3
  handshake_step=4
  final_step=5
else
  binary_step=2
  handshake_step=3
  final_step=4
fi

echo "[$binary_step/$total_steps] Checking ironclaw-mcp binary ..."
if [[ ! -x "$BINARY" ]]; then
  echo "FAIL: Binary not found at $BINARY. Run: make build"
  exit 2
fi
echo "OK: Binary found"
echo

echo "[$handshake_step/$total_steps] Running stdio MCP handshake and tool checks ..."
python3 - "$BINARY" "$BASE_URL" "$API_KEY" "$STATEFUL_TOOL" "$CHAT_MESSAGE" "$TIMEOUT_SECONDS" "$SMOKE_ALL" "$SMOKE_REPORT" <<'PY'
import json
import os
import subprocess
import sys
import threading
import time


binary, base_url, api_key, stateful_tool, chat_message, timeout_seconds, smoke_all, smoke_report = sys.argv[1:]
timeout_seconds = int(timeout_seconds)
smoke_all = smoke_all == "1"
smoke_report = smoke_report == "1"

env = os.environ.copy()
env["IRONCLAW_BASE_URL"] = base_url
if api_key:
    env["IRONCLAW_API_KEY"] = api_key

proc = subprocess.Popen(
    [binary],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    env=env,
)

stderr_lines = []

def _read_stderr():
    assert proc.stderr is not None
    for line in proc.stderr:
        stderr_lines.append(line.decode("utf-8", errors="replace"))

stderr_thread = threading.Thread(target=_read_stderr, daemon=True)
stderr_thread.start()

report_results = []

def fail(message: str, code: int) -> None:
    try:
        proc.terminate()
        proc.wait(timeout=5)
    except Exception:
        proc.kill()
    if smoke_report:
        report_results.append({"tool": "FATAL", "status": "fail", "error": message})
        json.dump({"results": report_results, "status": "fail"}, sys.stdout, indent=2)
        print()
    if stderr_lines:
        sys.stderr.write("bridge stderr:\n" + "".join(stderr_lines[-20:]) + "\n")
    sys.stderr.write(message + "\n")
    sys.exit(code)


def send(payload: dict) -> None:
    data = (json.dumps(payload) + "\n").encode("utf-8")
    assert proc.stdin is not None
    proc.stdin.write(data)
    proc.stdin.flush()


def read_message() -> dict:
    assert proc.stdout is not None
    line = proc.stdout.readline()
    if not line:
        fail("FAIL: bridge stdout closed unexpectedly", 4)
    return json.loads(line.decode("utf-8"))


next_id = 0
def request(method: str, params: dict | None = None) -> dict:
    global next_id
    next_id += 1
    rid = next_id
    send({
        "jsonrpc": "2.0",
        "id": rid,
        "method": method,
        "params": params or {},
    })
    while True:
        msg = read_message()
        if msg.get("id") == rid:
            if "error" in msg:
                fail(f"FAIL: {method} returned error: {json.dumps(msg['error'])}", 4)
            return msg["result"]


def call_tool(name: str, arguments: dict) -> dict:
    return request("tools/call", {"name": name, "arguments": arguments})


def test_tool(name: str, arguments: dict, expect_error: bool = False) -> dict:
    """Call a tool and record the result. Returns the raw MCP result."""
    start = time.time()
    try:
        result = call_tool(name, arguments)
        elapsed_ms = int((time.time() - start) * 1000)
        is_error = result.get("isError", False)

        entry = {
            "tool": name,
            "elapsed_ms": elapsed_ms,
        }

        if expect_error:
            entry["status"] = "pass" if is_error else "fail"
            if not is_error:
                entry["error"] = "expected error but got success"
        else:
            entry["status"] = "pass" if not is_error else "fail"
            if is_error:
                content = result.get("content", [{}])
                entry["error"] = content[0].get("text", "unknown") if content else "empty"

        report_results.append(entry)
        if not smoke_report:
            status = "ok" if entry["status"] == "pass" else "FAIL"
            print(f"  {name}: {status} ({elapsed_ms}ms)")
        return result
    except Exception as exc:
        elapsed_ms = int((time.time() - start) * 1000)
        report_results.append({
            "tool": name,
            "status": "fail",
            "elapsed_ms": elapsed_ms,
            "error": str(exc),
        })
        if not smoke_report:
            print(f"  {name}: FAIL ({elapsed_ms}ms) - {exc}")
        return {}


init_result = request(
    "initialize",
    {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "ironclaw-mcp-smoke", "version": "0.1.0"},
    },
)

send({"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}})

tools_result = request("tools/list")
tools = {tool["name"] for tool in tools_result.get("tools", [])}

all_expected = {
    "ironclaw_health",
    "ironclaw_chat",
    "ironclaw_list_jobs",
    "ironclaw_get_job",
    "ironclaw_cancel_job",
    "ironclaw_search_memory",
    "ironclaw_list_routines",
    "ironclaw_delete_routine",
    "ironclaw_list_tools",
    "ironclaw_stack_status",
    "ironclaw_spawn_agent",
    "ironclaw_send_task",
    "ironclaw_agent_status",
}
# ironclaw_get_metrics is conditional (only when PROMETHEUS_URL is set)

required = all_expected if smoke_all else {
    "ironclaw_health",
    "ironclaw_chat",
    "ironclaw_list_jobs",
    "ironclaw_search_memory",
    "ironclaw_list_routines",
    "ironclaw_delete_routine",
    "ironclaw_list_tools",
}
missing = sorted(required - tools)
if missing:
    fail(f"FAIL: tools/list missing expected tools: {missing}", 3)

if smoke_all:
    if not smoke_report:
        print(f"Testing all {len(all_expected)} default tools (+ optional adjuncts if available):")

    test_tool("ironclaw_health", {})
    test_tool("ironclaw_list_jobs", {})
    test_tool("ironclaw_get_job", {"job_id": "smoke-nonexistent"}, expect_error=True)
    test_tool("ironclaw_cancel_job", {"job_id": "smoke-nonexistent"}, expect_error=True)
    test_tool("ironclaw_search_memory", {"query": "smoke-test", "limit": "3"})
    test_tool("ironclaw_list_routines", {})
    test_tool("ironclaw_list_tools", {})
    test_tool("ironclaw_stack_status", {})
    test_tool("ironclaw_agent_status", {})
    test_tool("ironclaw_send_task", {"message": "smoke-test: acknowledge this task"})
    test_tool("ironclaw_spawn_agent", {"name": "smoke-test-agent", "model": "example-model", "tier": "fast"})

    # These tools need special handling:
    # - ironclaw_chat requires LLM round-trip (slow, may timeout)
    # - ironclaw_delete_routine is destructive
    if not smoke_report:
        print("  ironclaw_chat: skipped (requires LLM round-trip)")
        print("  ironclaw_delete_routine: skipped (destructive)")
    report_results.extend([
        {"tool": "ironclaw_chat", "status": "skipped", "reason": "requires LLM round-trip"},
        {"tool": "ironclaw_delete_routine", "status": "skipped", "reason": "destructive"},
    ])

    if "ironclaw_get_metrics" in tools:
        test_tool("ironclaw_get_metrics", {})
    else:
        report_results.append({
            "tool": "ironclaw_get_metrics",
            "status": "skipped",
            "reason": "PROMETHEUS_URL not configured",
        })
        if not smoke_report:
            print("  ironclaw_get_metrics: skipped (PROMETHEUS_URL not set)")

else:
    health_result = call_tool("ironclaw_health", {})
    if health_result.get("isError"):
        fail(f"FAIL: ironclaw_health returned an error: {json.dumps(health_result)}", 4)

    stateful_args = {}
    if stateful_tool == "ironclaw_chat":
        stateful_args = {"message": chat_message}
    elif stateful_tool == "ironclaw_search_memory":
        stateful_args = {"query": "smoke", "limit": "3"}
    elif stateful_tool == "ironclaw_delete_routine":
        fail("FAIL: ironclaw_delete_routine is destructive and cannot be used as the smoke stateful tool", 4)
    elif stateful_tool != "ironclaw_list_jobs":
        fail(f"FAIL: unsupported SMOKE_STATEFUL_TOOL={stateful_tool}", 4)

    stateful_result = call_tool(stateful_tool, stateful_args)
    if stateful_result.get("isError"):
        fail(f"FAIL: {stateful_tool} returned an error: {json.dumps(stateful_result)}", 4)

passed = sum(1 for r in report_results if r.get("status") == "pass")
failed = sum(1 for r in report_results if r.get("status") == "fail")
skipped = sum(1 for r in report_results if r.get("status") == "skipped")

if smoke_report:
    json.dump({
        "results": report_results,
        "summary": {"passed": passed, "failed": failed, "skipped": skipped},
        "tools_registered": len(tools),
        "server_info": init_result.get("serverInfo", {}),
        "status": "fail" if failed > 0 else "pass",
    }, sys.stdout, indent=2)
    print()
else:
    if smoke_all:
        print(f"\nResults: {passed} passed, {failed} failed, {skipped} skipped")
    else:
        print("initialize server:", init_result.get("serverInfo", {}))
        print("tools/list count:", len(tools))
        print("ironclaw_health ok")
        print(f"{stateful_tool} ok")

try:
    proc.terminate()
    proc.wait(timeout=5)
except Exception:
    proc.kill()

if failed > 0:
    sys.exit(1)
PY

if [[ "$SMOKE_REPORT" -eq 1 ]]; then
  exit 0
fi

echo "OK: MCP handshake and tool calls succeeded"
echo

echo "[$final_step/$total_steps] Smoke test completed successfully"
echo
echo "Verified:"
echo "  - /api/health on the IronClaw gateway"
if [[ "$require_router" -eq 1 ]]; then
  echo "  - /healthz on the local router"
  echo "  - /v1/models includes $EXPECT_MODEL"
fi
echo "  - MCP initialize"
echo "  - MCP tools/list"
if [[ "$SMOKE_ALL" -eq 1 ]]; then
  echo "  - All default MCP tools (with deterministic payloads)"
else
  echo "  - ironclaw_health"
  echo "  - $STATEFUL_TOOL"
fi
echo
echo "Tips:"
echo "  - Use --all to test all default tools with deterministic payloads"
echo "  - Use --report to get JSON output (combinable with --all)"
echo "  - Use SMOKE_STATEFUL_TOOL=ironclaw_chat for the full local LLM round-trip"
echo "  - Set SMOKE_REQUIRE_ROUTER=true to force router checks for non-chat probes"
echo "  - Override SMOKE_ROUTER_URL / SMOKE_EXPECT_MODEL when testing alternate router layouts"
echo "  - Export IRONCLAW_API_KEY (or GATEWAY_AUTH_TOKEN) when gateway auth is enabled"
