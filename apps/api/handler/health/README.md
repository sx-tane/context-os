# handler/health

Liveness handler for `GET /health`.

## Handlers

| Function | Route     | Method |
| -------- | --------- | ------ |
| `Health` | `/health` | GET    |

Returns `{"status":"ok","service":"context-os-api"}` when the API process is running.
