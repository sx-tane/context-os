#!/usr/bin/env bash

# Shared local-stack helpers for startup and status scripts.

contextos_port_from_addr() {
  local addr="$1"
  printf '%s\n' "${addr##*:}"
}

contextos_port_owner() {
  local port="$1"
  ss -ltnp 2>/dev/null | awk -v port=":$port" '$4 ~ port "$" { print }'
}

contextos_probe_url() {
  local url="$1"
  command -v curl >/dev/null 2>&1 || return 1
  curl -fsS --max-time 2 "$url" >/dev/null 2>&1
}

contextos_api_health_url() {
  local port="$1"
  printf 'http://127.0.0.1:%s/health\n' "$port"
}

contextos_api_workspace_url() {
  local port="$1"
  printf 'http://127.0.0.1:%s/workspace/status?path=contextos-default\n' "$port"
}

contextos_worker_health_url() {
  local port="$1"
  printf 'http://127.0.0.1:%s/health\n' "$port"
}

contextos_frontend_url() {
  local port="$1"
  printf 'http://127.0.0.1:%s/\n' "$port"
}

contextos_service_status() {
  local name="$1" port="$2" url="$3" owner=""

  owner="$(contextos_port_owner "$port")"
  if [[ -z "$owner" ]]; then
    printf '%s|free||\n' "$name"
    return 0
  fi

  if contextos_probe_url "$url"; then
    printf '%s|healthy|%s|%s\n' "$name" "$url" "$owner"
    return 0
  fi

  printf '%s|occupied|%s|%s\n' "$name" "$url" "$owner"
}

contextos_print_stack_status() {
  local api_addr="${API_ADDR:-:8080}"
  local api_port="${API_PORT:-$(contextos_port_from_addr "$api_addr")}"
  local worker_port="${WORKER_PORT:-8081}"
  local frontend_port="${FRONTEND_PORT:-5173}"
  local api_url worker_url frontend_url

  api_url="$(contextos_api_health_url "$api_port")"
  worker_url="$(contextos_worker_health_url "$worker_port")"
  frontend_url="$(contextos_frontend_url "$frontend_port")"

  contextos_service_status "API" "$api_port" "$api_url"
  contextos_service_status "Worker" "$worker_port" "$worker_url"
  contextos_service_status "Frontend" "$frontend_port" "$frontend_url"
}

contextos_stack_reusable() {
  local statuses line status api_addr api_port

  statuses="$(contextos_print_stack_status)"
  while IFS='|' read -r _name status _url _owner; do
    [[ "$status" == "healthy" ]] || return 1
  done <<<"$statuses"

  api_addr="${API_ADDR:-:8080}"
  api_port="${API_PORT:-$(contextos_port_from_addr "$api_addr")}"
  contextos_probe_url "$(contextos_api_workspace_url "$api_port")" || return 1
  return 0
}

contextos_print_reuse_summary() {
  local api_addr="${API_ADDR:-:8080}"
  local api_port="${API_PORT:-$(contextos_port_from_addr "$api_addr")}"
  local worker_port="${WORKER_PORT:-8081}"
  local frontend_port="${FRONTEND_PORT:-5173}"
  local log_dir="${CONTEXTOS_LOG_DIR:-}"

  echo "Existing ContextOS stack is already running."
  echo "Frontend: http://localhost:${frontend_port}/"
  echo "API:      http://localhost:${api_port}/health"
  echo "Worker:   http://localhost:${worker_port}/health"
  if [[ -n "$log_dir" ]]; then
    echo "Logs:     ${log_dir}"
    echo "Tail:     tail -F ${log_dir}/api.log ${log_dir}/worker.log ${log_dir}/frontend.log"
  fi
}
