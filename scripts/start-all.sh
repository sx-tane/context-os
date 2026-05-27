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

if ! command -v codex >/dev/null 2>&1; then
  if command -v npm >/dev/null 2>&1; then
    echo "Codex CLI not found; installing @openai/codex globally..."
    npm install -g @openai/codex
  else
    echo "[warn] Codex CLI is not installed and npm is unavailable."
    echo "       Codex plugin ingestion will be disabled until codex is on PATH."
  fi
fi

# Returns 0 (true) when running inside a headless, remote, or SSH environment.
is_headless() {
  [[ -n "${CODESPACES:-}" ]]                        && return 0
  [[ -n "${VSCODE_REMOTE_CONTAINERS_SESSION:-}" ]]  && return 0
  [[ -n "${SSH_TTY:-}" ]]                            && return 0
  [[ -n "${SSH_CONNECTION:-}" ]]                     && return 0
  [[ -z "${DISPLAY:-}" ]]                            && return 0
  return 1
}

if command -v codex >/dev/null 2>&1; then
  echo "Codex CLI: $(codex --version)"
  echo "Ensuring Codex GitHub and Slack plugins are installed..."
  codex plugin add github@openai-curated >/dev/null 2>&1 || \
    echo "[warn] Could not install GitHub Codex plugin."
  codex plugin add slack@openai-curated >/dev/null 2>&1 || \
    echo "[warn] Could not install Slack Codex plugin."
  if ! codex login status >/dev/null 2>&1; then
    if is_headless; then
      echo
      echo "╔══════════════════════════════════════════════════════════════╗"
      echo "║  Codex CLI login required (remote / headless environment)   ║"
      echo "║  Run this command in your terminal, then restart:           ║"
      echo "║                                                              ║"
      echo "║    codex login --device-auth                                ║"
      echo "║                                                              ║"
      echo "║  You will be shown a URL + code to approve in your browser. ║"
      echo "╚══════════════════════════════════════════════════════════════╝"
      echo
    else
      echo
      echo "╔══════════════════════════════════════════════════════════════╗"
      echo "║  Codex CLI login required                                   ║"
      echo "║  Run this command in your terminal, then restart:           ║"
      echo "║                                                              ║"
      echo "║    codex login                                              ║"
      echo "║                                                              ║"
      echo "║  Your browser will open the Codex authorization page.       ║"
      echo "╚══════════════════════════════════════════════════════════════╝"
      echo
    fi
  fi
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
  if command -v swag >/dev/null 2>&1; then
    echo "Regenerating Swagger docs..."
    swag init -g apps/api/main.go -o apps/api/docs --quiet
    if command -v npx >/dev/null 2>&1; then
      npx --yes @redocly/cli build-docs apps/api/docs/swagger.json --output apps/api/docs/api.html 2>/dev/null \
        && echo "HTML docs: apps/api/docs/api.html"
    fi
  else
    echo "[warn] swag not found; skipping doc generation. Install with: go install github.com/swaggo/swag/cmd/swag@latest"
  fi
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