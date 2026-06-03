#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
[[ -d /usr/local/go/bin ]] && export PATH="/usr/local/go/bin:$PATH"
[[ -d "$HOME/go/bin" ]] && export PATH="$HOME/go/bin:$PATH"
API_PID=""
FRONTEND_PID=""
WORKER_PID=""
API_LOG_PID=""
FRONTEND_LOG_PID=""
WORKER_LOG_PID=""
API_ADDR="${API_ADDR:-:8080}"
WORKER_PORT="${WORKER_PORT:-8081}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"
cleanup() {
  local exit_code=$?

  trap - EXIT INT TERM

  for pid in "$FRONTEND_LOG_PID" "$WORKER_LOG_PID" "$API_LOG_PID"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
    fi
  done

  for pid in "$FRONTEND_PID" "$WORKER_PID" "$API_PID"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill -- "-$pid" 2>/dev/null || kill "$pid" 2>/dev/null || true
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

port_from_addr() {
  local addr="$1"
  printf '%s\n' "${addr##*:}"
}

port_owner() {
  local port="$1"
  ss -ltnp 2>/dev/null | awk -v port=":$port" '$4 ~ port "$" { print }'
}

require_port_available() {
  local name="$1" port="$2" owner=""

  owner="$(port_owner "$port")"
  if [[ -z "$owner" ]]; then
    return 0
  fi

  echo "[error] ${name} port ${port} is already in use:"
  printf '%s\n' "$owner" | sed 's/^/        /'
  return 1
}

check_required_ports() {
  local failed=0

  require_port_available "API" "$(port_from_addr "$API_ADDR")" || failed=1
  require_port_available "AI worker" "$WORKER_PORT" || failed=1
  require_port_available "Frontend" "$FRONTEND_PORT" || failed=1

  if [[ "$failed" == "1" ]]; then
    echo
    echo "Another ContextOS stack appears to be running. Stop it first, then rerun this script."
    echo "Useful check: ss -ltnp | grep -E ':($(port_from_addr "$API_ADDR")|${WORKER_PORT}|${FRONTEND_PORT})\\b'"
    exit 1
  fi
}

start_logged_service() {
  local pid_var="$1" log_pid_var="$2" label="$3" cwd="$4" log_file=""
  shift 4

  log_file="$(mktemp -t "contextos-${label}.XXXXXX.log")"
  : >"$log_file"
  tail -n +1 -F "$log_file" 2>/dev/null | prefix_logs "$label" &
  printf -v "$log_pid_var" '%s' "$!"

  setsid bash -c 'cd "$1"; shift; exec "$@"' bash "$cwd" "$@" \
    >"$log_file" 2>&1 &
  printf -v "$pid_var" '%s' "$!"
}

ensure_swag() {
  local swag_bin=""

  if command -v swag >/dev/null 2>&1; then
    command -v swag
    return 0
  fi

  echo "[info] swag not found; installing github.com/swaggo/swag/cmd/swag@v1.16.4..." >&2
  if go install github.com/swaggo/swag/cmd/swag@v1.16.4 >/dev/null 2>&1; then
    swag_bin="$(go env GOBIN)"
    if [[ -z "$swag_bin" ]]; then
      swag_bin="$(go env GOPATH)/bin"
    fi
    swag_bin="${swag_bin%/}/swag"
    if [[ -x "$swag_bin" ]]; then
      printf '%s\n' "$swag_bin"
      return 0
    fi
  fi

  return 1
}

check_required_ports

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
  start_logged_service WORKER_PID WORKER_LOG_PID "worker" "$ROOT_DIR/apps/ai-worker" env WORKER_PORT="$WORKER_PORT" uv run python health.py
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

swagger_json="$ROOT_DIR/apps/api/docs/swagger.json"
frontend_types="$ROOT_DIR/apps/frontend/src/lib/generated/api.d.ts"

if [[ "${UPDATE_API_DOCS:-0}" == "1" || ! -f "$swagger_json" ]]; then
  SWAG_BIN=""
  if SWAG_BIN="$(ensure_swag)"; then
    echo "Regenerating OpenAPI docs..."
    (
      cd "$ROOT_DIR"
      "$SWAG_BIN" init -g apps/api/main.go -o apps/api/docs --quiet 2>/dev/null || \
        "$SWAG_BIN" init -g apps/api/main.go -o apps/api/docs
    )
    if command -v npx >/dev/null 2>&1; then
      npx --yes @redocly/cli build-docs \
        "$ROOT_DIR/apps/api/docs/swagger.yaml" \
        --output "$ROOT_DIR/apps/api/docs/api.html" \
        --title "ContextOS API" 2>/dev/null || true
    fi
  else
    echo "[warn] swag could not be installed; skipping OpenAPI doc generation."
    echo "       Install with: go install github.com/swaggo/swag/cmd/swag@v1.16.4"
  fi
else
  echo "OpenAPI docs already generated. Set UPDATE_API_DOCS=1 to regenerate."
fi

if [[ "${UPDATE_API_TYPES:-0}" == "1" || ! -f "$frontend_types" || "$swagger_json" -nt "$frontend_types" ]]; then
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
else
  echo "Frontend TypeScript types are up to date. Set UPDATE_API_TYPES=1 to regenerate."
fi

echo "Starting API. Backend logs will be prefixed with [api]."
start_logged_service API_PID API_LOG_PID "api" "$ROOT_DIR" env API_ADDR="$API_ADDR" go run ./apps/api

echo "Starting frontend dev server. Frontend logs will be prefixed with [frontend]."
if command -v bun >/dev/null 2>&1; then
  start_logged_service FRONTEND_PID FRONTEND_LOG_PID "frontend" "$ROOT_DIR/apps/frontend" bash -c 'bun install && bun run dev -- --port "$1" --strictPort' bash "$FRONTEND_PORT"
else
  start_logged_service FRONTEND_PID FRONTEND_LOG_PID "frontend" "$ROOT_DIR/apps/frontend" bash -c 'npm install && npm run dev -- --port "$1" --strictPort' bash "$FRONTEND_PORT"
fi

echo "API PID:      $API_PID"
echo "Worker PID:   ${WORKER_PID:-skipped}"
echo "Frontend PID: $FRONTEND_PID"
echo "Press Ctrl+C to stop all processes."

# Keep running until the frontend exits or Ctrl+C is pressed.
wait "$FRONTEND_PID"
