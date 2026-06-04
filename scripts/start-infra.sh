#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCKER_CONFIG_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$DOCKER_CONFIG_DIR"
}

trap cleanup EXIT

export DOCKER_CONFIG="$DOCKER_CONFIG_DIR"

cd "$ROOT_DIR"

wait_for_health() {
  local container="$1"
  local timeout="${2:-60}"
  local start_ts
  start_ts="$(date +%s)"

  while true; do
    local status
    status="$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$container" 2>/dev/null || true)"
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
  if docker compose version >/dev/null 2>&1; then
    compose_cmd=(docker compose)
  elif command -v docker-compose >/dev/null 2>&1; then
    compose_cmd=(docker-compose)
  else
    return 1
  fi

  "${compose_cmd[@]}" up -d db nats
  "${compose_cmd[@]}" ps
}

start_with_docker_run() {
  echo "[warn] Docker Compose is unavailable; using docker run fallback." >&2

  docker volume create db_data >/dev/null
  docker rm -f contextos-db contextos-nats >/dev/null 2>&1 || true

  docker run -d \
    --name contextos-db \
    --restart unless-stopped \
    -p 5432:5432 \
    -e POSTGRES_DB=contextos \
    -e POSTGRES_USER=contextos \
    -e POSTGRES_PASSWORD=contextos \
    -v db_data:/var/lib/postgresql/data \
    -v "$ROOT_DIR/migrations/0001_enable_pgvector.sql:/docker-entrypoint-initdb.d/0001_enable_pgvector.sql:ro" \
    -v "$ROOT_DIR/migrations/0002_workspace_schema.sql:/docker-entrypoint-initdb.d/0002_workspace_schema.sql:ro" \
    pgvector/pgvector:pg16

  docker run -d \
    --name contextos-nats \
    --restart unless-stopped \
    -p 4222:4222 \
    -p 8222:8222 \
    nats:2.10-alpine -js -m 8222

  wait_for_health contextos-db 90
  wait_for_health contextos-nats 30

  docker ps --filter "name=contextos-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
}

echo "Starting local infra: Postgres+pgvector and NATS"
if ! start_with_compose; then
  start_with_docker_run
fi

echo
echo "Waiting for services to become healthy..."

echo
echo "Infra endpoints:"
echo "  Postgres: localhost:5432 (db=contextos, user=contextos, password=contextos)"
echo "  NATS:     localhost:4222"
echo "  NATS UI:  http://localhost:8222"
