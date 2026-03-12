#!/usr/bin/env bash
# Deterministic smoke test for Cursor -> ironclaw-mcp -> IronClaw.

set -euo pipefail

BASE_URL="${IRONCLAW_BASE_URL:-http://localhost:3000}"
API_KEY="${IRONCLAW_API_KEY:-${GATEWAY_AUTH_TOKEN:-}}"
BINARY="${BINARY:-./bin/ironclaw-mcp}"
STATEFUL_TOOL="${SMOKE_STATEFUL_TOOL:-ironclaw_list_jobs}"
CHAT_MESSAGE="${SMOKE_CHAT_MESSAGE:-Smoke test: reply with a short acknowledgement.}"
TIMEOUT_SECONDS="${SMOKE_TIMEOUT_SECONDS:-45}"
ROUTER_URL="${SMOKE_ROUTER_URL:-http://127.0.0.1:8080}"
EXPECT_MODEL="${SMOKE_EXPECT_MODEL:-qwen3.5-27b}"
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
if [[ "$require_router" -eq 1 ]]; then
  smoke_mode="full chat-path"
  total_steps=5
fi

echo "=== IronClaw MCP Smoke Test ==="
echo "IRONCLAW_BASE_URL=$BASE_URL"
echo "MCP binary=$BINARY"
echo "Stateful tool=$STATEFUL_TOOL"
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
  if ! python3 - "$EXPECT_MODEL" /tmp/ironclaw-smoke-router-models.json <<'PY'
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
    echo "Verify the local 27B upstream is registered and healthy."
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
python3 - "$BINARY" "$BASE_URL" "$API_KEY" "$STATEFUL_TOOL" "$CHAT_MESSAGE" "$TIMEOUT_SECONDS" <<'PY'
import json
import os
import subprocess
import sys
import threading


binary, base_url, api_key, stateful_tool, chat_message, timeout_seconds = sys.argv[1:]
timeout_seconds = int(timeout_seconds)

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


def fail(message: str, code: int) -> None:
    try:
        proc.terminate()
        proc.wait(timeout=5)
    except Exception:
        proc.kill()
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


def request(method: str, params: dict | None = None, request_id: int = 1) -> dict:
    send({
        "jsonrpc": "2.0",
        "id": request_id,
        "method": method,
        "params": params or {},
    })
    while True:
        msg = read_message()
        if msg.get("id") == request_id:
            if "error" in msg:
                fail(f"FAIL: {method} returned error: {json.dumps(msg['error'])}", 4)
            return msg["result"]


init_result = request(
    "initialize",
    {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {"name": "ironclaw-mcp-smoke", "version": "0.1.0"},
    },
    request_id=1,
)

send({"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}})

tools_result = request("tools/list", {}, request_id=2)
tools = {tool["name"] for tool in tools_result.get("tools", [])}
required = {
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

health_result = request("tools/call", {"name": "ironclaw_health", "arguments": {}}, request_id=3)
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

stateful_result = request(
    "tools/call",
    {"name": stateful_tool, "arguments": stateful_args},
    request_id=4,
)
if stateful_result.get("isError"):
    fail(f"FAIL: {stateful_tool} returned an error: {json.dumps(stateful_result)}", 4)

print("initialize server:", init_result.get("serverInfo", {}))
print("tools/list count:", len(tools))
print("ironclaw_health ok")
print(f"{stateful_tool} ok")

try:
    proc.terminate()
    proc.wait(timeout=5)
except Exception:
    proc.kill()
PY
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
echo "  - ironclaw_health"
echo "  - $STATEFUL_TOOL"
echo
echo "Tips:"
echo "  - Use SMOKE_STATEFUL_TOOL=ironclaw_chat for the full local LLM round-trip"
echo "  - Set SMOKE_REQUIRE_ROUTER=true to force router checks for non-chat probes"
echo "  - Override SMOKE_ROUTER_URL / SMOKE_EXPECT_MODEL when testing alternate router layouts"
echo "  - Export IRONCLAW_API_KEY (or GATEWAY_AUTH_TOKEN) when gateway auth is enabled"
