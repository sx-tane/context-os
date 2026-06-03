#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

compose_cmd=()
if docker compose version >/dev/null 2>&1; then
  compose_cmd=(docker compose)
elif command -v docker-compose >/dev/null 2>&1; then
  compose_cmd=(docker-compose)
else
  echo "Docker Compose is required. Install the Docker Compose plugin or make docker-compose available in PATH." >&2
  exit 1
fi

echo "Starting local infra: Postgres+pgvector and NATS"
"${compose_cmd[@]}" up -d db nats

echo "\nWaiting for services to become healthy..."
"${compose_cmd[@]}" ps

echo "\nInfra endpoints:"
echo "  Postgres: localhost:5432 (db=contextos, user=contextos, password=contextos)"
echo "  NATS:     localhost:4222"
echo "  NATS UI:  http://localhost:8222"
