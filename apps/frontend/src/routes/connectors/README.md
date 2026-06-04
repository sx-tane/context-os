# connectors route

Frontend route for connector debugging, Codex login status, and source ingest controls.

## Files

- `+page.svelte` renders API and worker status, Codex login controls, curated connector components, and generic source connector cards from `sourceConnectorConfigs`.

## Route Behavior

The route probes the API and worker services on mount, checks `/codex/status`, supports streamed Codex login, and passes the shared Codex props into each connector component.

## Maintenance Notes

- Keep connector cards wired with `codexLoggedIn`, `codexAccount`, `codexPlugins`, and `refreshCodexStatus`.
- Update `apps/frontend/src/lib/components/connectors/README.md` when adding or changing connector component contracts.
- Update API handler docs when route behavior depends on new backend endpoints.
