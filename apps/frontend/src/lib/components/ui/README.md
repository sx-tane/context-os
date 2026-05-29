# ui

Primitive building-block components shared across the ContextOS frontend. These have no domain knowledge — they only handle visual structure and interaction.

## Components

### Button

Styled `<button>` element.

| Prop       | Type                              | Default     | Purpose                                                     |
| ---------- | --------------------------------- | ----------- | ----------------------------------------------------------- |
| `type`     | `"button" \| "submit" \| "reset"` | `"button"`  | HTML button type.                                           |
| `disabled` | `boolean`                         | `false`     | Disables click interaction and applies muted styling.       |
| `variant`  | `"primary" \| "secondary"`        | `"primary"` | Visual style; secondary is used for lower-emphasis actions. |

Forwards all native button events. Use inside connector forms and status panels.

---

### FormField

Labelled input wrapper. Renders a `<label>` and an `<input>` (or `<textarea>`) in a consistent vertical stack.

| Prop          | Type                            | Default  | Purpose                                      |
| ------------- | ------------------------------- | -------- | -------------------------------------------- |
| `label`       | `string`                        | —        | Visible label text.                          |
| `value`       | `string`                        | `""`     | Bound input value (use `bind:value`).        |
| `placeholder` | `string`                        | `""`     | Input placeholder.                           |
| `type`        | `"text" \| "password" \| "url"` | `"text"` | Input type; use `"password"` for tokens.     |
| `multiline`   | `boolean`                       | `false`  | Renders a `<textarea>` instead of `<input>`. |
| `disabled`    | `boolean`                       | `false`  | Passes through to the underlying element.    |

---

### ModeToggle

Two-option toggle that switches between `"token"` and `"codex"` authentication modes inside connector cards.

| Prop         | Type             | Purpose                                                                   |
| ------------ | ---------------- | ------------------------------------------------------------------------- |
| `value`      | `IngestProvider` | Currently selected mode.                                                  |
| `codexLabel` | `string`         | Label shown for the Codex option (e.g. `"Codex + GitHub Plugin"`).        |
| `tokenLabel` | `string`         | Label shown for the direct-token option (e.g. `"Personal Access Token"`). |
| `disabled`   | `boolean`        | Disables both options when `true`.                                        |

Emits a `change` event with the new `IngestProvider` value when the selection changes.
