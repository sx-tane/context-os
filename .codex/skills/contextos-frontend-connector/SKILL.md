---
name: contextos-frontend-connector
description: "Create a new ContextOS frontend connector component following the established Svelte pattern. Use when: adding a new <Name>Connector.svelte; wiring a new connector into +page.svelte; adding status and ingest behaviour to a connector card. Covers script structure, runConnectorIngest usage, AbortController + run-ID guard, checkStatus shape, ConnectorCard template, and +page.svelte registration."
argument-hint: "What is the connector name (e.g. Notion, GoogleDrive)?"
user-invocable: true
---

# ContextOS Frontend Connector Skill

## Outcome

Deliver a fully-wired connector component:

- `apps/frontend/src/lib/components/connectors/<Name>Connector.svelte` — the component
- `apps/frontend/src/routes/+page.svelte` — import + usage added
- Relevant frontend `README.md` files updated for component props, shared runners, and user-facing setup

---

## Decision Points

| Situation                                          | Action                                                                       |
| -------------------------------------------------- | ---------------------------------------------------------------------------- |
| Connector has a Codex plugin                       | Include `ModeToggle` with `"token"` and `"codex"` options; show `CodexBadge` |
| Connector has OAuth flow (like Slack)              | Add an OAuth connection button and handle `oauth_available` from status      |
| Connector has extra fields (email, base URL, etc.) | Add `FormField` for each; include in `runIngest` body                        |
| Connector is read-only / no token                  | Omit token `FormField`; provider always `"direct"`                           |

---

## Procedure

1. **Create the component** at `apps/frontend/src/lib/components/connectors/<Name>Connector.svelte`.
   Use the [connector skeleton](./assets/connector-skeleton.md) as the starting point.

2. **Implement `checkStatus()`**:
   - Call `getJSON<{...}>(/<name>/status")` — type the expected shape from the API.
   - Map response fields to local state (`connected`, etc.).

3. **Implement `runIngest()`**:
   - Use `runConnectorIngest` with an `AbortController` + run-ID guard (see skeleton).
   - Pass `connector: "<name>"`, all form fields, and `provider`.
   - All state setters use the canonical `setIfCurrent` pattern inside `runConnectorIngest`.

4. **Build the template**:
   - Wrap everything in `<ConnectorCard title="..." description="..." examples={[...]}>`.
   - Show a connected badge when `connected === true`.
   - Add `ModeToggle` if provider switching is needed.
   - Add `FormField` for each form input.
   - Add `CodexBadge` inside the codex provider branch (no reauth props needed).
   - Add `Button` for the ingest submit action; bind `disabled={loading}`.
   - Show `<LogPanel>` when `liveLog` is non-empty.
   - Show `<ErrorPanel>` when `errorMessage` is non-empty.
   - Show `<ResultPanel>` when `result` is non-null.

5. **Register in `+page.svelte`**:

   ```svelte
   import <Name>Connector from "$lib/components/connectors/<Name>Connector.svelte";
   ```

   Add the component in the `<main>` block, passing all four shared Codex props:

   ```svelte
   <<Name>Connector
     {codexLoggedIn}
     {codexAccount}
     {codexPlugins}
     {refreshCodexStatus}
   />
   ```

6. **Update documentation**:

- Update `apps/frontend/src/lib/components/connectors/README.md` when connector props, status fields, modes, or examples change.
- Update `apps/frontend/src/lib/README.md` when shared runner, API helper, config, or type behavior changes.
- Update `apps/frontend/README.md` when commands, setup, or environment expectations change.

---

## Component Structure Rules

### Script section order

1. Imports — svelte lifecycle, then `$lib/types`, then `$lib/api`, then `$lib/ingestRunner`, then child components.
2. Shared Codex `export let` props (4 props — always the same, always present).
3. `// Local state` block — `uri`, `provider`, `loading`, `errorMessage`, `result`, `liveLog`, `elapsed`, `ingestController`, `ingestRunID`.
4. Connector-specific status state (`connected`, etc.) — comment `// Status state`.
5. `onMount(checkStatus)`.
6. `onDestroy(() => { ingestController?.abort(); })`.
7. `checkStatus()` function.
8. `runIngest()` function.

### AbortController + run-ID guard pattern

Every async runner (ingest and reauth) must follow this guard:

```ts
functionController?.abort();
functionController = new AbortController();
const runID = ++functionRunID;
await runSomething({
  // ...
  signal: functionController.signal,
  isCurrent: () => runID === functionRunID,
  // ...
});
```

This prevents stale results from earlier aborted runs overwriting current state.

### `setLiveLog` / `setLog` setter pattern

Log setters must accept both a string (reset) and an updater function (append):

```ts
setLiveLog: (value) =>
  (liveLog = typeof value === "function" ? value(liveLog) : value),
```

### `setElapsed` setter pattern

```ts
setElapsed: (value) =>
  (elapsed = typeof value === "function" ? value(elapsed) : value),
```

---

## `runConnectorIngest` Options Reference

```ts
await runConnectorIngest({
  connector: "<name>",         // ConnectorKind
  uri,                          // string
  token,                        // string (optional)
  provider,                     // IngestProvider
  signal: ingestController.signal,
  isCurrent: () => ingestRunID === /* local copy */,
  setLoading:  (v) => (loading = v),
  setError:    (m) => (errorMessage = m),
  setResult:   (v) => (result = v),
  setLiveLog:  (v) => (liveLog  = typeof v === "function" ? v(liveLog)  : v),
  setElapsed:  (v) => (elapsed  = typeof v === "function" ? v(elapsed)  : v),
});
```

---

## `runCodexReauth` Options Reference

```ts
await runCodexReauth({
  plugin,
  refreshCodexStatus,
  signal: reauthController.signal,
  isCurrent: () => reauthRunID === /* local copy */,
  setPlugin:  (v) => (reauthPlugin  = v),
  setRunning: (v) => (reauthRunning = v),
  setLog:     (v) => (reauthLog = typeof v === "function" ? v(reauthLog) : v),
});
```

---

## References

- [Connector Skeleton](./assets/connector-skeleton.md) — copy-paste `.svelte` starting point
- [Checklist](./references/connector-checklist.md) — review before marking done
- Real examples:
  - [`apps/frontend/src/lib/components/connectors/GitHubConnector.svelte`](../../../../apps/frontend/src/lib/components/connectors/GitHubConnector.svelte)
  - [`apps/frontend/src/lib/components/connectors/JiraConnector.svelte`](../../../../apps/frontend/src/lib/components/connectors/JiraConnector.svelte)
  - [`apps/frontend/src/lib/components/connectors/SlackConnector.svelte`](../../../../apps/frontend/src/lib/components/connectors/SlackConnector.svelte)
  - [`apps/frontend/src/lib/ingestRunner.ts`](../../../../apps/frontend/src/lib/ingestRunner.ts)
  - [`apps/frontend/src/lib/reauthRunner.ts`](../../../../apps/frontend/src/lib/reauthRunner.ts)
  - [`apps/frontend/src/routes/+page.svelte`](../../../../apps/frontend/src/routes/+page.svelte)
