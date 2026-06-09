# scripts/lib

Internal shell helpers for local developer startup.

## Responsibilities

- Keep setup, infra, status, and reusable startup functions out of top-level script entrypoints.
- Report local API, worker, and frontend port health without mutating services.
- Keep shell helpers POSIX-adjacent Bash functions that can be sourced by `scripts/*.sh`.

## Files

| File | Purpose |
| --- | --- |
| `local-stack.sh` | Port owner detection, health probes, reusable-stack detection, and status summary output for local ContextOS services. |
| `setup-local.sh` | Internal first-run Linux tool bootstrap used by `scripts/start-local.sh`. |
| `start-infra.sh` | Internal Postgres/pgvector and NATS startup helper used by `scripts/start-local.sh`. |
| `status-local.sh` | Internal status helper for debugging local service ports. |

## Maintenance Notes

- Keep `scripts/start-local.sh` as the only normal user-facing startup command.
- Keep health checks short so `scripts/lib/status-local.sh` remains responsive when a service is offline.
- Run `bash -n scripts/lib/*.sh scripts/*.sh` after changing these helpers.
