# Docker

Container and local infrastructure definitions for running ContextOS services in a reproducible environment.

## Files

- `Dockerfile.api`: Go API container image.
- `Dockerfile.frontend`: frontend development/runtime container image.
- `Dockerfile.worker`: AI worker container image.

## Usage

- Use these images for local containerized workflows and CI-compatible environment setup.
- Keep Dockerfile dependencies aligned with the local setup scripts.
- Update this README when new service images or shared base-image expectations are added.

## Local Compose Services

`docker-compose.yml` now includes local-first infrastructure services:

- `db` — PostgreSQL 16 with pgvector extension enabled on first boot
- `nats` — NATS with JetStream and monitoring enabled
- `api`, `worker`, and `frontend` services wired to shared network aliases

Quick start:

```bash
docker compose up --build
```

Infra-only start:

```bash
./scripts/start-infra.sh
```

## Maintenance Checklist

- Keep build steps deterministic and minimal.
- Document new runtime ports, environment variables, or mounted volumes.
- Recheck local startup docs when Docker workflows change.
