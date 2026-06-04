# Scripts

Local developer and maintenance scripts for ContextOS.

---

## setup-local.sh

Installs all required tools on Ubuntu/Linux and validates the repository. Run this once when setting up a new machine.

What it does:

1. Installs system packages (`curl`, `wget`, `build-essential`, `nodejs`, etc.)
2. Installs Go 1.24.13 and goimports
3. Installs Bun (UI runtime)
4. Installs Python tooling for the worker environment
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

## start-all.sh / start-local.sh

Starts all local services in a single terminal session. Run this after setup is complete.

What it does:

- Runs `uv sync` in `apps/ai-worker` if `uv` is available
- Reuses committed OpenAPI docs and frontend TypeScript types during normal startup
- Regenerates OpenAPI docs only when `apps/api/docs/swagger.json` is missing or `UPDATE_API_DOCS=1` is set
- Regenerates frontend TypeScript types only when missing, older than `swagger.json`, or `UPDATE_API_TYPES=1` is set
- Reuses an already-running healthy ContextOS stack on API `8080`, worker `8081`, and frontend `5173` by printing the existing URLs and log location instead of starting a second stack
- Fails fast when a required port is occupied by something that does not pass the expected ContextOS health check and prints the owning process
- Checks all required Codex plugins. Missing plugins are reported without prompting; to install them during startup, run `INSTALL_CODEX_PLUGINS=1 ./scripts/start-all.sh`:
  - `github@openai-curated`
  - `atlassian-rovo@openai-curated`
  - `slack@openai-curated`
  - `google-drive@openai-curated`
  - `notion@openai-curated`
  - `sharepoint@openai-curated`
- Prints the Codex home being checked and reads `codex plugin list` once so plugin status is consistent for the whole startup run.
- Starts the Go API (`go run ./apps/api`) in the background with `[api]` log prefixes
- Starts the SvelteKit context UI dev server (`bun run dev` or `npm run dev`) on a strict `5173` port with `[frontend]` log prefixes
- Shuts down API, worker, and frontend process groups cleanly when you press `Ctrl+C`

```bash
chmod +x scripts/start-local.sh
./scripts/start-local.sh
```

Or, if you prefer the original entrypoint:

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

If the stack is already running, startup prints the existing frontend/API/worker URLs and exits successfully. To inspect current local service health at any time:

```bash
./scripts/status-local.sh
```

`start-all.sh` writes service logs to `.tmp/contextos/logs/` for the current run. When an existing healthy stack is reused, tail those files from another terminal:

```bash
tail -F .tmp/contextos/logs/api.log .tmp/contextos/logs/worker.log .tmp/contextos/logs/frontend.log
```

If startup reports that a port is occupied but the stack is not healthy, stop or move the owning process first. To inspect owners:

```bash
ss -ltnp | grep -E ':(8080|8081|5173)\b'
```

Set `GITHUB_TOKEN` before running to authenticate against private repositories:

```bash
export GITHUB_TOKEN=ghp_your_token_here
./scripts/start-all.sh
```

Optional local debug logging is quiet by default. Enable only the layer you are investigating:

```bash
CONTEXTOS_API_REQUEST_LOGS=1 ./scripts/start-all.sh   # Go API request start/done lines
CONTEXTOS_PROXY_LOGS=1 ./scripts/start-all.sh         # Vite proxy request/response lines
VITE_CONTEXTOS_DEBUG_LOGS=1 ./scripts/start-all.sh    # Browser API request logs in dev console
```

Once running:

- **http://localhost:5173** — ContextOS chat-first UI (homepage is now the chat interface)
- **http://localhost:5173/connectors** — Connector debug surface (individual ingest + reauth)
- **http://localhost:5173/findings** — Role-based findings and PMO summary UI
- **http://localhost:8080/health** — API health endpoint
- **http://localhost:8080/swagger/** — Interactive Swagger UI
- **http://localhost:8080/swagger/doc.json** — Raw OpenAPI spec (Postman/Insomnia)
- **apps/api/docs/api.html** — Standalone Redoc HTML (open directly in browser after docs are generated)

Generated docs under `apps/api/docs/` are committed OpenAPI artifacts and the source for frontend type generation. Frontend types at `apps/frontend/src/lib/generated/api.d.ts` are committed to the repository. Normal startup reuses both to keep launch fast; run `UPDATE_API_DOCS=1 UPDATE_API_TYPES=1 ./scripts/start-all.sh` after API shape changes. The rendered docs page is `apps/api/docs/api.html`. The AI worker uses `uv run python`, so it does not depend on a system `python3.12` binary being installed.

---

## start-infra.sh

Starts local infrastructure only (Postgres + pgvector and NATS). Uses Docker Compose when available and falls back to plain `docker run` containers otherwise.

```bash
chmod +x scripts/start-infra.sh
./scripts/start-infra.sh
```

Infra endpoints:

- **localhost:5432** — PostgreSQL (`contextos/contextos/contextos`)
- **localhost:4222** — NATS client port
- **http://localhost:8222** — NATS monitoring endpoint

---

## status-local.sh

Prints local API, worker, and frontend port status without starting or stopping anything. Use it when the browser says `API Offline`, `Worker Ready`, or `Codex Unavailable` and you need to separate a real backend failure from a stale or already-running terminal session.

```bash
./scripts/status-local.sh
```

Statuses:

- `[ok]` — the expected ContextOS health check or frontend page responded.
- `[free]` — no process is listening on that service port.
- `[blocked]` — a process owns the port, but it did not respond like ContextOS.

---

## Order of use

```
1. ./scripts/setup-local.sh   # first time only
2. ./scripts/start-local.sh    # every time you want to run locally; reuses a healthy existing stack
3. ./scripts/status-local.sh   # optional status check when a page or terminal looks stale
```
