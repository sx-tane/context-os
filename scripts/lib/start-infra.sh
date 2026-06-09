#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DOCKER_CONFIG_DIR="$(mktemp -d)"
POSTGRES_PORT="${POSTGRES_PORT:-55432}"
DOCKER_CMD=(docker)

cleanup() {
  rm -rf "$DOCKER_CONFIG_DIR"
}

trap cleanup EXIT

export DOCKER_CONFIG="$DOCKER_CONFIG_DIR"

cd "$ROOT_DIR"

select_docker_cmd() {
  if docker ps >/dev/null 2>&1; then
    DOCKER_CMD=(docker)
    return 0
  fi
  if command -v sudo >/dev/null 2>&1 && sudo docker ps >/dev/null 2>&1; then
    DOCKER_CMD=(sudo docker)
    return 0
  fi
  echo "Docker is required and the current user cannot access the Docker daemon." >&2
  echo "Install/start Docker or run: sudo usermod -aG docker $USER && newgrp docker" >&2
  return 1
}

wait_for_health() {
  local container="$1"
  local timeout="${2:-60}"
  local start_ts
  start_ts="$(date +%s)"

  while true; do
    local status
    status="$("${DOCKER_CMD[@]}" inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || true)"
    if [[ "$status" == "healthy" ]] || [[ "$status" == "running" ]]; then
      return 0
    fi

    if (( $(date +%s) - start_ts >= timeout )); then
      echo "Timed out waiting for $container to become healthy (last status: ${status:-unknown})." >&2
      return 1
    fi

    sleep 2
  done
}

start_with_compose() {
  local compose_cmd=()
  if "${DOCKER_CMD[@]}" compose version >/dev/null 2>&1; then
    compose_cmd=("${DOCKER_CMD[@]}" compose)
  elif command -v docker-compose >/dev/null 2>&1; then
    compose_cmd=(docker-compose)
  else
    return 1
  fi

  local db_container nats_container
  POSTGRES_PORT="$POSTGRES_PORT" "${compose_cmd[@]}" up -d db nats
  db_container="$(POSTGRES_PORT="$POSTGRES_PORT" "${compose_cmd[@]}" ps -q db)"
  nats_container="$(POSTGRES_PORT="$POSTGRES_PORT" "${compose_cmd[@]}" ps -q nats)"
  wait_for_health "$db_container" 90
  wait_for_health "$nats_container" 30
  "${compose_cmd[@]}" ps
}

start_with_docker_run() {
  echo "[warn] Docker Compose is unavailable; using docker run fallback." >&2

  "${DOCKER_CMD[@]}" volume create db_data >/dev/null
  "${DOCKER_CMD[@]}" rm -f contextos-db contextos-nats >/dev/null 2>&1 || true

  "${DOCKER_CMD[@]}" run -d \
    --name contextos-db \
    --restart unless-stopped \
    -p "${POSTGRES_PORT}:5432" \
    -e POSTGRES_DB=contextos \
    -e POSTGRES_USER=contextos \
    -e POSTGRES_PASSWORD=contextos \
    -v db_data:/var/lib/postgresql/data \
    -v "$ROOT_DIR/migrations/0001_enable_pgvector.sql:/docker-entrypoint-initdb.d/0001_enable_pgvector.sql:ro" \
    -v "$ROOT_DIR/migrations/0002_workspace_schema.sql:/docker-entrypoint-initdb.d/0002_workspace_schema.sql:ro" \
    pgvector/pgvector:pg16

  "${DOCKER_CMD[@]}" run -d \
    --name contextos-nats \
    --restart unless-stopped \
    -p 4222:4222 \
    -p 8222:8222 \
    nats:2.10-alpine -js -m 8222

  wait_for_health contextos-db 90
  wait_for_health contextos-nats 30

  "${DOCKER_CMD[@]}" ps --filter "name=contextos-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
}

echo "Starting local infra: Postgres+pgvector and NATS"
select_docker_cmd
if ! start_with_compose; then
  start_with_docker_run
fi

echo
echo "Waiting for services to become healthy..."

echo
echo "Infra endpoints:"
echo "  Postgres: localhost:${POSTGRES_PORT} (db=contextos, user=contextos, password=contextos)"
echo "  NATS:     localhost:4222"
echo "  NATS UI:  http://localhost:8222"
