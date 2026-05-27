#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_PID=""
FRONTEND_PID=""
WORKER_PID=""
cleanup() {
  local exit_code=$?

  trap - EXIT INT TERM

  for pid in "$FRONTEND_PID" "$WORKER_PID" "$API_PID"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
    fi
  done

  wait || true
  exit "$exit_code"
}

trap cleanup EXIT INT TERM

if ! command -v go >/dev/null 2>&1; then
  echo "go is required to start the API" >&2
  exit 1
fi

if ! command -v bun >/dev/null 2>&1; then
  echo "bun is required to start the frontend" >&2
  exit 1
fi

if command -v uv >/dev/null 2>&1; then
  echo "Preparing AI worker environment..."
  (
    cd "$ROOT_DIR/apps/ai-worker"
    uv sync
  )

  echo "Starting AI worker health server..."
  (
    cd "$ROOT_DIR/apps/ai-worker"
    python3.12 health.py
  ) &
  WORKER_PID=$!
else
  echo "uv not found; skipping AI worker environment sync" >&2
fi

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "[warn] GITHUB_TOKEN is not set — the GitHub connector will only reach public repos."
  echo "       To authenticate: export GITHUB_TOKEN=ghp_... then re-run this script."
fi

echo "Starting API on current terminal session..."
(
  cd "$ROOT_DIR"
  go run ./apps/api
) &
API_PID=$!

echo "Starting frontend dev server..."
(
  cd "$ROOT_DIR/apps/frontend"
  bun install
  bun run dev
) &
FRONTEND_PID=$!

echo "API PID:      $API_PID"
echo "Worker PID:   ${WORKER_PID:-skipped}"
echo "Frontend PID: $FRONTEND_PID"
echo "Press Ctrl+C to stop all processes."

# Keep running until the frontend exits or Ctrl+C is pressed.
wait "$FRONTEND_PID"