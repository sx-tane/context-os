#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR"

echo "Starting local infra: Postgres+pgvector and NATS"
docker compose up -d db nats

echo "\nWaiting for services to become healthy..."
docker compose ps

echo "\nInfra endpoints:"
echo "  Postgres: localhost:5432 (db=contextos, user=contextos, password=contextos)"
echo "  NATS:     localhost:4222"
echo "  NATS UI:  http://localhost:8222"
