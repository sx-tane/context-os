#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$ROOT_DIR/scripts/lib/local-stack.sh"

API_ADDR="${API_ADDR:-:8080}"
WORKER_PORT="${WORKER_PORT:-8081}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"
CONTEXTOS_LOG_DIR="${CONTEXTOS_LOG_DIR:-$ROOT_DIR/.tmp/contextos/logs}"

echo "ContextOS local stack"
while IFS='|' read -r name status url owner; do
  case "$status" in
    healthy)
      echo "[ok]      ${name}: ${url}"
      ;;
    free)
      echo "[free]    ${name}: port is available"
      ;;
    occupied)
      echo "[blocked] ${name}: port is occupied but health check failed (${url})"
      printf '%s\n' "$owner" | sed 's/^/          /'
      ;;
    *)
      echo "[unknown] ${name}: ${status}"
      ;;
  esac
done <<<"$(contextos_print_stack_status)"

echo
echo "Frontend URL: http://localhost:${FRONTEND_PORT}/"
echo "API health:   http://localhost:$(contextos_port_from_addr "$API_ADDR")/health"
echo "Worker health: http://localhost:${WORKER_PORT}/health"
echo "Log dir:      ${CONTEXTOS_LOG_DIR}"

if contextos_stack_reusable; then
  echo
  contextos_print_reuse_summary
fi
