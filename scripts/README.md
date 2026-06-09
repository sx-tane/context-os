# Scripts

Local developer scripts for ContextOS.

## Start Everything

Use one command from the repository root:

```bash
./scripts/start-local.sh
```

This is the supported entrypoint for first-time and repeat local runs.

On a first run, `start-local.sh` detects missing tools and runs the bootstrap path internally. On repeat runs, it skips setup and starts the stack directly.

What it manages:

- required local tooling bootstrap on Linux
- Docker-backed Postgres with pgvector and NATS
- Local DB login verification before API startup
- Go API on `http://localhost:8080`
- AI worker on `http://localhost:8081`
- SvelteKit frontend on `http://localhost:5173`
- Codex CLI/plugin readiness checks
- OpenAPI/frontend type reuse or regeneration when needed
- clean shutdown of API, worker, and frontend with `Ctrl+C`

Expected successful startup includes:

```text
[api] ... db: connected and migrations applied
[worker] context-os ai-worker health listening on :8081
[frontend] VITE ... ready
```

Open:

- Frontend: `http://localhost:5173`
- API health: `http://localhost:8080/health`
- Swagger: `http://localhost:8080/swagger/`

## First-Time Behavior

If tools are missing, `start-local.sh` calls the internal setup helper with repository validation skipped so the stack can start quickly. It may ask for `sudo` to install system packages.

The bootstrap installs:

- base build utilities
- Docker / Docker Compose support
- Go and goimports
- Bun
- Python tooling and `uv`
- Postgres client tools
- Codex CLI and required plugins

If Docker was installed during that same run and your user is not yet in the Docker group, the infra helper falls back to `sudo docker` for the current run.

## Repeat Runs

Run the same command:

```bash
./scripts/start-local.sh
```

If a fully healthy ContextOS stack is already running, the script prints the existing URLs and exits. The reuse check requires both `/health` and DB-backed workspace routes, so an API that started without Local DB is not treated as healthy.

## Common Options

```bash
INSTALL_CODEX_PLUGINS=1 ./scripts/start-local.sh
UPDATE_API_DOCS=1 UPDATE_API_TYPES=1 ./scripts/start-local.sh
CONTEXTOS_API_REQUEST_LOGS=1 ./scripts/start-local.sh
CONTEXTOS_PROXY_LOGS=1 ./scripts/start-local.sh
VITE_CONTEXTOS_DEBUG_LOGS=1 ./scripts/start-local.sh
POSTGRES_PORT=55433 ./scripts/start-local.sh
```

Set `GITHUB_TOKEN` before startup if you need private repository access:

```bash
export GITHUB_TOKEN=ghp_your_token_here
./scripts/start-local.sh
```

## Internal Helpers

These scripts exist for debugging and maintenance. Normal users should not need them.

| Helper | Purpose |
| --- | --- |
| `scripts/lib/setup-local.sh` | Installs Linux local tooling. Called automatically by `start-local.sh` when tools are missing. |
| `scripts/lib/start-infra.sh` | Starts only Postgres/pgvector and NATS. Called automatically by `start-local.sh`. |
| `scripts/lib/status-local.sh` | Prints API, worker, and frontend port status without starting or stopping services. |

Local infra endpoints:

- Postgres: `localhost:55432` (`contextos/contextos/contextos`)
- NATS: `localhost:4222`
- NATS UI: `http://localhost:8222`

Logs for the current run are written to `.tmp/contextos/logs/`.
