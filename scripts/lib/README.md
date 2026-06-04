# scripts/lib

Shared shell helpers for local developer scripts.

## Responsibilities

- Keep reusable startup/status functions out of top-level script entrypoints.
- Report local API, worker, and frontend port health without mutating services.
- Keep shell helpers POSIX-adjacent Bash functions that can be sourced by `scripts/*.sh`.

## Files

| File | Purpose |
| --- | --- |
| `local-stack.sh` | Port owner detection, health probes, reusable-stack detection, and status summary output for local ContextOS services. |

## Maintenance Notes

- Do not start or stop processes from this folder; top-level scripts own lifecycle side effects.
- Keep health checks short so `scripts/status-local.sh` remains responsive when a service is offline.
- Run `bash -n scripts/lib/*.sh scripts/*.sh` after changing these helpers.
