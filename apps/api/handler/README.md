# apps/api/handler

HTTP handlers for the ContextOS API, organized as per-domain Go packages.
Source connector HTTP handlers live together in `connectors/`; shared plumbing lives in `shared/`.

---

## Package layout

| Package                               | Route prefix    | Exports                                                   |
| ------------------------------------- | --------------- | --------------------------------------------------------- |
| [`health/`](health/README.md)         | `/health`       | `Health`                                                  |
| [`connectors/codex/`](connectors/codex/README.md) | `/codex/*` | `Status`, `Login`, `PluginReauth` |
| [`connectors/github/`](connectors/github/README.md) | `/github/*` | `Status`, `Ingest`, `IngestStream` |
| [`connectors/googledrive/`](connectors/googledrive/README.md) | `/googledrive/*` | `Status`, `Ingest` |
| [`connectors/jira/`](connectors/jira/README.md) | `/jira/*` | `Status`, `Ingest`, `IngestStream` |
| [`connectors/notion/`](connectors/notion/README.md) | `/notion/*` | `Status`, `Ingest`, `IngestStream` |
| [`connectors/sharepoint/`](connectors/sharepoint/README.md) | `/sharepoint/*` | `Status`, `Ingest`, `IngestStream` |
| [`connectors/slack/`](connectors/slack/README.md) | `/slack/*` | `Status`, `Connect`, `Callback`, `Ingest`, `IngestStream` |
| [`connectors/filesystem/`](connectors/filesystem/README.md) | `/filesystem/*` | `Ingest`, `Upload` |
| [`shared/`](shared/README.md)         | —               | Ingest helpers, SSE infrastructure                        |

Each domain package has its own README with handler table and design notes.

---

## Handler pattern

Every handler follows the same four-step structure:

```go
func Action(w http.ResponseWriter, r *http.Request) {
    // 1. Method guard
    if r.Method != http.MethodPost {
        response.WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "POST required")
        return
    }

    // 2. Decode request body
    var req request.DomainAction
    if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&req); err != nil {
        response.WriteError(w, http.StatusBadRequest, "invalid_json", err.Error())
        return
    }

    // 3. Build connector + metadata, delegate ingest
    shared.WriteSourceIngest(w, r, connector, shared.SourceIngestInput{...})

    // 4. WriteSourceIngest writes the JSON response — no further code here
}
```

Rules:

- No business logic in handlers. Decode → validate shape → delegate to `internal/`.
- All connectors are created fresh per request (`connector.NewConnector()`).
- Errors always use `response.WriteError` or `response.WriteConnectorError`.

---

## SSE / streaming pattern

Streaming handlers call `shared.StreamCodexIngest`:

```go
func IngestStream(w http.ResponseWriter, r *http.Request) {
    shared.StreamCodexIngest(
        w, r,
        codexsource.PluginName,          // Codex plugin identifier
        []string{"capability"},          // reported capability list
        func(dec *json.Decoder) (request.T, error) { /* decode */ },
        func(req request.T) string { return req.URI },
        func(req request.T) string { return req.Provider },
        func(req request.T) string { return req.Token },
    )
}
```

The SSE stream emits:

- `event: log` — one per Codex CLI stdout/stderr line while running
- `event: status` — every 2 s while waiting, so the browser connection stays alive
- `event: result` — one final event with the full `response.Ingest` JSON payload
- `event: error` — if the process fails or the request is invalid

---

## shared/ — shared plumbing

See [`shared/README.md`](shared/README.md) for the full export reference.
Key helpers consumed by domain packages:

| Helper                     | Purpose                                                                  |
| -------------------------- | ------------------------------------------------------------------------ |
| `shared.RunSourceIngest`   | Method guard + JSON decode + call `WriteSourceIngest`                    |
| `shared.WriteSourceIngest` | Validates URI/content, calls `connector.Ingest`, writes response         |
| `shared.NewIngestResponse` | Builds `response.Ingest` from connector + events (also used by SSE path) |
| `shared.CapabilityStrings` | Converts `[]contracts.Capability` → `[]string`                           |
| `shared.CloneStringMap`    | Copies a metadata map, dropping empty/whitespace values                  |
| `shared.SetMetadata`       | Conditionally sets a key in a metadata map (skips empty values)          |
| `shared.Preview`           | Truncates content to first 500 runes for display                         |

---

## Adding a new connector handler

1. Create `apps/api/handler/connectors/<domain>/<domain>.go`.
2. Add a `Status` handler (GET) that reads env vars and reports connection state.
3. Add an `Ingest` handler (POST) using `runSourceIngest` or `writeSourceIngest`.
4. Optionally add an `IngestStream` handler using `streamCodexIngest` for Codex CLI providers.
5. Add `apps/api/handler/connectors/<domain>/<domain>_test.go` with at least: method-not-allowed, invalid JSON, and key env-var behavior tests.
6. Register the routes in `apps/api/main.go`.
7. Add Swagger annotations to each handler and regenerate: `swag init -g main.go -o docs`.

---

## Running tests

```bash
go test context-os/apps/api/handler -v
```
