# feedback

Svelte components that surface the results, logs, errors, and system health of an ingest run.

## Components

### StatusSection

Renders the system health panel at the top of the main page. Shows API and AI Worker status (online / offline / checking) and the Codex installation / login state.

| Prop                | Type            | Purpose                                              |
| ------------------- | --------------- | ---------------------------------------------------- |
| `apiStatus`         | `ServiceStatus` | Result of the `/api/health` probe.                   |
| `workerStatus`      | `ServiceStatus` | Result of the AI Worker `/health` probe.             |
| `codexInstalled`    | `boolean`       | Whether the Codex CLI is installed.                  |
| `codexVersion`      | `string`        | Reported Codex version string.                       |
| `codexLoggedIn`     | `boolean`       | Whether Codex has an active login.                   |
| `codexAccount`      | `string`        | Logged-in account name/email.                        |
| `codexLoginLog`     | `string`        | SSE log output from an in-progress login stream.     |
| `codexLoginRunning` | `boolean`       | Shows a spinner and log panel while login is active. |
| `onLoginClick`      | `() => void`    | Callback wired to the "Log in" button.               |

---

### LogPanel

Scrollable `<pre>` block that displays live SSE log output during a Codex ingest or re-auth stream.

| Prop          | Type      | Default                 | Purpose                                                                                           |
| ------------- | --------- | ----------------------- | ------------------------------------------------------------------------------------------------- |
| `log`         | `string`  | `""`                    | Full log text accumulated so far.                                                                 |
| `loading`     | `boolean` | `false`                 | Shows `placeholder` text when `log` is empty.                                                     |
| `visible`     | `boolean` | `true`                  | When `false` the element is not rendered at all (used to hide the panel for non-Codex providers). |
| `placeholder` | `string`  | `"Waiting for output…"` | Text shown while loading and log is empty.                                                        |

---

### IngestResult

Renders the structured output of a completed ingest. Iterates over the `events` array inside an `IngestResult` and displays each event's type, URI, preview text, and metadata key/value pairs.

| Prop     | Type                   | Purpose                                                |
| -------- | ---------------------- | ------------------------------------------------------ |
| `result` | `IngestResult \| null` | The ingest response body; renders nothing when `null`. |

Each event card shows:

- event type badge
- source URI
- `preview` field when present
- metadata key/value table

---

### ErrorPanel

Minimal inline error display. Shows a styled error message when `message` is non-empty.

| Prop      | Type     | Purpose                                                   |
| --------- | -------- | --------------------------------------------------------- |
| `message` | `string` | Error text; component renders nothing when this is empty. |
