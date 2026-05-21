# Scripts

Local developer and maintenance scripts for ContextOS.

---

## setup-local.sh

Installs all required tools on Ubuntu/Linux and validates the repository. Run this once when setting up a new machine.

What it does:

1. Installs system packages (`curl`, `wget`, `build-essential`, etc.)
2. Installs Go 1.24.13
3. Installs Bun (UI runtime)
4. Installs Python 3.12 and pip
5. Installs `uv` (Python environment manager)
6. Verifies all tool versions
7. Runs `go mod tidy`, `go test ./...`, UI check, and `uv sync`

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
- Starts the Go API (`go run ./apps/api`) in the background
- Starts the SvelteKit context UI dev server (`bun run dev`) in the background
- Shuts down both processes cleanly when you press `Ctrl+C`

```bash
chmod +x scripts/start-all.sh
./scripts/start-all.sh
```

Expected output when running:

```
Preparing AI worker environment...
Starting API on current terminal session...
Starting context UI dev server...
API PID: <pid>
Context UI PID: <pid>
Press Ctrl+C to stop both processes.
Note: API process has already exited (scaffold stub). Context UI is still running.
```

The `API process has already exited` note is expected while the API is a scaffold stub that prints one log line and exits.
The context UI keeps running until you press Ctrl+C.
The `SIGTERM` message on context UI exit is normal — it means the process was stopped cleanly.

---

## Order of use

```
1. ./scripts/setup-local.sh   # first time only
2. ./scripts/start-all.sh     # every time you want to run locally
```
