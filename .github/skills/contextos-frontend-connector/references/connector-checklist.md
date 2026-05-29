# Frontend Connector Checklist

Use before merging a new `<Name>Connector.svelte` or a `+page.svelte` update.

---

## Script Section

- [ ] Imports are in the correct order: svelte lifecycle → `$lib/types` → `$lib/api` → `$lib/ingestRunner` → `$lib/reauthRunner` → child components.
- [ ] Exactly four shared Codex `export let` props: `codexLoggedIn`, `codexAccount`, `codexPlugins`, `refreshCodexStatus`.
- [ ] All local state variables are declared in the `// Local state` block in the canonical order.
- [ ] Connector-specific status vars are in a `// Status state` block after local state.
- [ ] Reauth state vars (`reauthPlugin`, `reauthLog`, `reauthRunning`) are present only when the connector has a Codex plugin.
- [ ] `onMount(checkStatus)` is present.
- [ ] `onDestroy` aborts both `ingestController` and `reauthController`.

## `checkStatus`

- [ ] Calls `getJSON<{...}>("/<name>/status")` with the correct path.
- [ ] Return type is typed inline with all expected fields as optional (`field?: type`).
- [ ] All fields use `?? defaultValue` fallback when mapping to local state.

## `runIngest`

- [ ] Aborts previous `ingestController` before creating a new one.
- [ ] Increments `ingestRunID` and captures `const runID = ++ingestRunID`.
- [ ] Passes `isCurrent: () => runID === ingestRunID`.
- [ ] `setLiveLog` uses the updater-function pattern: `typeof value === "function" ? value(liveLog) : value`.
- [ ] `setElapsed` uses the updater-function pattern: `typeof value === "function" ? value(elapsed) : value`.
- [ ] `connector` prop matches the API route name exactly (lowercase, same as Go package).

## `runReauth` (only when connector has a Codex plugin)

- [ ] Aborts previous `reauthController` before creating a new one.
- [ ] Increments `reauthRunID` and captures `const runID = ++reauthRunID`.
- [ ] Passes `isCurrent: () => runID === reauthRunID`.
- [ ] `setLog` uses the updater-function pattern.

## Template

- [ ] Root element is `<ConnectorCard title="..." description="..." examples={[...]}>`.
- [ ] Connected badge is shown conditionally when `connected === true`.
- [ ] `ModeToggle` is present only when the connector supports multiple providers.
- [ ] Every `FormField` is bound with `bind:value={...}`.
- [ ] `Button` has `disabled={loading}` and shows elapsed time while loading.
- [ ] `LogPanel` is wrapped in `{#if liveLog}`.
- [ ] `ErrorPanel` is wrapped in `{#if errorMessage}`.
- [ ] `ResultPanel` is wrapped in `{#if result}`.

## `+page.svelte` Registration

- [ ] Import added at the top of the `<script>` block.
- [ ] Component added to `<main>` with all four Codex props passed.
- [ ] No other page-level state modified or broken.

## Type Safety

- [ ] `bun run check` passes with 0 errors after the change.
- [ ] No `any` casts introduced without a comment explaining why.

## Documentation

- [ ] `apps/frontend/src/lib/components/connectors/README.md` updated for connector props, status fields, modes, and examples.
- [ ] `apps/frontend/src/lib/README.md` updated when shared runner, API helper, config, or type behavior changes.
- [ ] `apps/frontend/README.md` updated when commands, setup, or environment expectations change.
