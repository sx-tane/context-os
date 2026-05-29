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
- Regenerates Swagger docs (`swag init`) if `swag` is on the PATH
- Rebuilds the standalone HTML doc (`npx @redocly/cli build-docs`) if `npx` is available
- Starts the Go API (`go run ./apps/api`) in the background
- Starts the SvelteKit context UI dev server (`bun run dev`) in the background
- Shuts down both processes cleanly when you press `Ctrl+C`

The files under `apps/api/docs/` are generated local artifacts and are ignored by git. If `swag` is missing, generate them with `go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g apps/api/main.go -o apps/api/docs` before `go test ./...` or API startup.

```bash
chmod +x scripts/start-all.sh
./scripts/start-all.sh
```

Expected output when running:

```
uv not found; skipping AI worker environment sync
[warn] GITHUB_TOKEN is not set — the GitHub connector will only reach public repos.
       To authenticate: export GITHUB_TOKEN=ghp_... then re-run this script.
Regenerating Swagger docs...
HTML docs: apps/api/docs/api.html
Starting API on current terminal session...
Starting frontend dev server...
API PID:      <pid>
Worker PID:   skipped
Frontend PID: <pid>
Press Ctrl+C to stop all processes.
```

Set `GITHUB_TOKEN` before running to authenticate against private repositories:

```bash
export GITHUB_TOKEN=ghp_your_token_here
./scripts/start-all.sh
```

Once running:

- **http://localhost:5173** — ContextOS UI
- **http://localhost:8080/swagger/index.html** — interactive API docs
- **apps/api/docs/api.html** — standalone HTML docs (open in browser, no server needed)

### Optional: install `swag` for automatic doc generation

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Without it, `start-all.sh` skips doc generation and prints a warning.

---

## Order of use

```
1. ./scripts/setup-local.sh   # first time only
2. ./scripts/start-all.sh     # every time you want to run locally
```
