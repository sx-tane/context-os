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

if ! command -v bun >/dev/null 2>&1 && ! command -v npm >/dev/null 2>&1; then
  echo "bun or npm is required to start the frontend" >&2
  exit 1
fi

prefix_logs() {
  local label="$1"
  awk -v label="$label" '{ print "[" label "] " $0; fflush(); }'
}

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

install_codex_plugin() {
  local name="$1" registry_name="$2"
  local plugin_id="${registry_name%%@*}"
  local plugin_list="${CODEX_PLUGIN_LIST:-}"

  if [[ -z "$plugin_list" ]]; then
    echo "  [warn] Could not read Codex plugin list; run: codex plugin list"
    return 0
  fi

  # codex plugin list can display registry slugs and friendly names. Prefer an
  # exact registry-name match, then fall back to the slug or friendly name.
  if grep -Fqi "$registry_name" <<<"$plugin_list" || \
     grep -Fqi "$plugin_id" <<<"$plugin_list" || \
     grep -Fqi "$name" <<<"$plugin_list"; then
    echo "  [ok] ${name} plugin already installed"
    return 0
  fi

  if [[ "${INSTALL_CODEX_PLUGINS:-0}" != "1" ]]; then
    echo "  [missing] ${name} plugin. To install: codex plugin add ${registry_name}"
    return 0
  fi

  codex plugin add "${registry_name}" >/dev/null 2>&1 && \
    echo "  [ok] ${name} plugin installed" || \
    echo "  [warn] Could not install ${name} Codex plugin. Run: codex plugin add ${registry_name}"
}

if command -v codex >/dev/null 2>&1; then
  echo "Codex CLI: $(codex --version)"
  echo "Codex home: ${CODEX_HOME:-$HOME/.codex}"
  CODEX_PLUGIN_LIST="$(codex plugin list 2>&1 || true)"
  if [[ "${INSTALL_CODEX_PLUGINS:-0}" == "1" ]]; then
    echo "Ensuring all required Codex plugins are installed..."
  else
    echo "Checking required Codex plugins without installing. Set INSTALL_CODEX_PLUGINS=1 to install missing plugins."
  fi
  install_codex_plugin "GitHub"          "github@openai-curated"
  install_codex_plugin "Atlassian Rovo"  "atlassian-rovo@openai-curated"
  install_codex_plugin "Slack"           "slack@openai-curated"
  install_codex_plugin "Google Drive"    "google-drive@openai-curated"
  install_codex_plugin "Notion"          "notion@openai-curated"
  install_codex_plugin "SharePoint"      "sharepoint@openai-curated"
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
    python3.12 health.py 2>&1 | prefix_logs "worker"
  ) &
  WORKER_PID=$!
else
  echo "uv not found; skipping AI worker environment sync" >&2
fi

if [[ -z "${GITHUB_TOKEN:-}" ]]; then
  echo "[warn] GITHUB_TOKEN is not set — the GitHub connector will only reach public repos."
  echo "       To authenticate: export GITHUB_TOKEN=ghp_... then re-run this script."
fi

if [[ -z "${GOOGLE_DRIVE_FOLDER_ID:-}" ]] && \
   [[ -z "${GOOGLE_DRIVE_OAUTH_CREDENTIALS_PATH:-}" ]] && \
   [[ -z "${GOOGLE_DRIVE_SERVICE_ACCOUNT_PATH:-}" ]]; then
  echo "[info] Google Drive env vars are not set — the Google Drive connector will require credentials per request."
  echo "       To configure: set GOOGLE_DRIVE_OAUTH_CREDENTIALS_PATH, GOOGLE_DRIVE_SERVICE_ACCOUNT_PATH,"
  echo "       or use the Codex Google Drive plugin (no credentials needed once codex login is done)."
fi

if command -v swag >/dev/null 2>&1; then
  echo "Regenerating OpenAPI docs..."
  (
    cd "$ROOT_DIR"
    swag init -g apps/api/main.go -o apps/api/docs --quiet 2>/dev/null || \
      swag init -g apps/api/main.go -o apps/api/docs
  )
  if command -v npx >/dev/null 2>&1; then
    npx --yes @redocly/cli build-docs \
      "$ROOT_DIR/apps/api/docs/swagger.yaml" \
      --output "$ROOT_DIR/apps/api/docs/api.html" \
      --title "ContextOS API" 2>/dev/null || true
  fi
else
  echo "[info] swag not found; skipping OpenAPI doc generation."
  echo "       Install with: go install github.com/swaggo/swag/cmd/swag@v1.16.4"
fi

echo "Regenerating frontend TypeScript types from swagger..."
(
  cd "$ROOT_DIR/apps/frontend"
  if command -v bun >/dev/null 2>&1; then
    bun run codegen 2>/dev/null || \
      echo "[warn] Frontend codegen failed; types may be stale. Run: cd apps/frontend && bun run codegen"
  else
    npm run codegen 2>/dev/null || \
      echo "[warn] Frontend codegen failed; types may be stale. Run: cd apps/frontend && npm run codegen"
  fi
)

echo "Starting API. Backend logs will be prefixed with [api]."
(
  cd "$ROOT_DIR"
  go run ./apps/api 2>&1 | prefix_logs "api"
) &
API_PID=$!

echo "Starting frontend dev server. Frontend logs will be prefixed with [frontend]."
(
  cd "$ROOT_DIR/apps/frontend"
  if command -v bun >/dev/null 2>&1; then
    bun install
    bun run dev
  else
    npm install
    npm run dev
  fi
) 2>&1 | prefix_logs "frontend" &
FRONTEND_PID=$!

echo "API PID:      $API_PID"
echo "Worker PID:   ${WORKER_PID:-skipped}"
echo "Frontend PID: $FRONTEND_PID"
echo "Press Ctrl+C to stop all processes."

# Keep running until the frontend exits or Ctrl+C is pressed.
wait "$FRONTEND_PID"
