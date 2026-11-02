#!/usr/bin/env bash
# Shared helper for gmid video scenes.
set -u

GMID=${GMID_BIN:-/home/akh/dev/gmid/bin/gmid}
MCP=${GMID_MCP:-/home/akh/dev/gmid/bin/mcpcheck}
PORT=${1:-8800}
URL="http://127.0.0.1:${PORT}/mcp"
PID=""

# Keep video scenes deterministic and away from the user's preferences. The
# config starts fresh for every scene, so no library preset or stale panel
# state leaks in from an earlier take.
export GMID_CONFIG=/tmp/gmid-video-prefs.json
rm -f "$GMID_CONFIG"

cleanup() {
  if [ -n "$PID" ]; then
    kill "$PID" 2>/dev/null || true
    wait "$PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

start_gmid() {
  # Extra args after the port go to gmid.
  "$GMID" -scene=shell -width 1920 -height 1080 -mcp-addr "127.0.0.1:${PORT}" >"/tmp/gmid-video-${PORT}.log" 2>&1 &
  PID=$!
  for _ in $(seq 1 120); do
    if "$MCP" -url "$URL" -tool gmid_status >/dev/null 2>&1; then
      sleep 2
      return 0
    fi
    sleep 0.25
  done
  echo "gmid did not start; see /tmp/gmid-video-${PORT}.log" >&2
  return 1
}

drive() {
  out=$("$MCP" -url "$URL" -tool "$1" -args "$2" 2>&1) || true
  if [ -n "${GMID_DEBUG:-}" ]; then
    echo ">> $1 $2" >&2
    echo "$out" | head -c 300 >&2
    echo >&2
  fi
  case "$out" in
  *"{"*) ;;
  *) echo "FAILED $1: $out" >>"/tmp/gmid-video-${PORT}.drive.log" ;;
  esac
}

# hold <seconds>
hold() { sleep "$1"; }
