# Local Setup

Run one command from the repository root:

```bash
./scripts/start-local.sh
```

That command is the supported local entrypoint for both first-time and repeat runs.

## What It Does

On a first run, `start-local.sh` checks for required tools. If any are missing, it runs the one-time bootstrap internally and installs:

- system build tools
- Docker and Docker Compose support
- Go
- Bun
- Python tooling and `uv`
- Postgres client tools
- Codex CLI and the required Codex plugins

On every run, it then:

- starts Postgres with pgvector and NATS through Docker
- verifies the Local DB login before starting the API
- starts the Go API
- starts the AI worker
- starts the SvelteKit frontend
- prefixes logs as `[api]`, `[worker]`, and `[frontend]`

Open the app at:

```text
http://localhost:5173
```

## Notes

- First-time setup may ask for your sudo password.
- If Codex is not logged in, the script prints the exact `codex login` command to run.
- Press `Ctrl+C` in the `start-local.sh` terminal to stop API, worker, and frontend.
- Postgres is published on `localhost:55432` by default to avoid conflicts with existing local Postgres installs.

Advanced helper scripts live under `scripts/lib/` and are implementation details. Use them only when debugging the startup flow.
