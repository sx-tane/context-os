# Scripts

Local developer and maintenance scripts for ContextOS.

---

## setup-local.sh

Installs all required tools on Ubuntu/Linux and validates the repository. Run this once when setting up a new machine.

What it does:

1. Installs system packages (`curl`, `wget`, `build-essential`, `nodejs`, `npm`, etc.)
2. Installs Go 1.24.13 and goimports
3. Installs Bun (UI runtime)
4. Installs Python 3.12 and pip
5. Installs `uv` (Python environment manager)
6. Installs Codex CLI and GitHub, Atlassian Rovo, and Slack plugins
7. Performs Codex CLI authentication (with device auth support)
8. Verifies all tool versions
9. Runs `go mod tidy`, `go test ./...`, SvelteKit UI check, and `uv sync`

```bash
chmod +x scripts/setup-local.sh
./scripts/setup-local.sh
```

After it finishes, restart your shell or run `source ~/.bashrc` to reload the updated PATH.

---

## start-all.sh

Starts all local services in a single terminal session. Run this after setup is complete.

What it does:

- Runs `uv sync` in `apps/ai-worker` if `uv` is available
- Regenerates OpenAPI docs (`swag init` → `apps/api/_docs/`) and Redoc HTML if `swag` is installed
- Regenerates frontend TypeScript types from the OpenAPI spec (`bun run codegen` or `npm run codegen` in `apps/frontend`)
- Checks all required Codex plugins. Missing plugins are reported without prompting; to install them during startup, run `INSTALL_CODEX_PLUGINS=1 ./scripts/start-all.sh`:
  - `github@openai-curated`
  - `atlassian-rovo@openai-curated`
  - `slack@openai-curated`
  - `google-drive@openai-curated`
  - `notion@openai-curated`
  - `sharepoint@openai-curated`
- Prints the Codex home being checked and reads `codex plugin list` once so plugin status is consistent for the whole startup run.
- Starts the Go API (`go run ./apps/api`) in the background with `[api]` log prefixes
- Starts the SvelteKit context UI dev server (`bun run dev` or `npm run dev`) in the background with `[frontend]` log prefixes
- Shuts down both processes cleanly when you press `Ctrl+C`

```bash
chmod +x scripts/start-all.sh
./scripts/start-all.sh
```

Expected output when running:

```
Codex CLI: codex-cli 0.136.0
Codex home: /home/user/.codex
uv not found; skipping AI worker environment sync
[missing] Slack plugin. To install: codex plugin add slack@openai-curated
[warn] GITHUB_TOKEN is not set — the GitHub connector will only reach public repos.
       To authenticate: export GITHUB_TOKEN=ghp_... then re-run this script.
Starting API. Backend logs will be prefixed with [api].
Starting frontend dev server. Frontend logs will be prefixed with [frontend].
API PID:      <pid>
Worker PID:   skipped
Frontend PID: <pid>
Press Ctrl+C to stop all processes.
[api] <backend log line>
[frontend] <frontend log line>
```

Set `GITHUB_TOKEN` before running to authenticate against private repositories:

```bash
export GITHUB_TOKEN=ghp_your_token_here
./scripts/start-all.sh
```

Once running:

- **http://localhost:5173** — ContextOS chat-first UI (homepage is now the chat interface)
- **http://localhost:5173/connectors** — Connector debug surface (individual ingest + reauth)
- **http://localhost:5173/findings** — Role-based findings and PMO summary UI
- **http://localhost:8080/health** — API health endpoint
- **http://localhost:8080/swagger/** — Interactive Swagger UI
- **http://localhost:8080/swagger/doc.json** — Raw OpenAPI spec (Postman/Insomnia)
- **apps/api/\_docs/api.html** — Standalone Redoc HTML (open directly in browser after docs are generated)

Generated docs under `apps/api/_docs/` are local artifacts (gitignored) and are not required for the API to start. Frontend types at `apps/frontend/src/lib/generated/api.d.ts` are committed to the repository and are regenerated automatically on each startup.

---

## start-infra.sh

Starts local infrastructure only (Postgres + pgvector and NATS) using Docker Compose.

```bash
chmod +x scripts/start-infra.sh
./scripts/start-infra.sh
```

Infra endpoints:

- **localhost:5432** — PostgreSQL (`contextos/contextos/contextos`)
- **localhost:4222** — NATS client port
- **http://localhost:8222** — NATS monitoring endpoint

---

## Order of use

```
1. ./scripts/setup-local.sh   # first time only
2. ./scripts/start-all.sh     # every time you want to run locally
```
