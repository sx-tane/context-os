# Homepage Redesign Notes

## Product Model

Use normal-user language on the homepage:

- Profile: Codex account or identity.
- Project: saved work context under a profile.
- Sources: selected repos, channels, docs, folders, or projects.

Keep workspace path, DB rows, sync counts, and audit trail as internal diagnostics.

## First-Time Flow

1. Connect sources.
2. Select multiple repos/channels from real connected accounts.
3. Save selected sources.
4. Run analysis across all selected sources.
5. Ask chat questions against the selected sources.

Manual URI entry should remain as an advanced fallback, not the primary path.

## Homepage Layout

Top bar:

- `profile · project · source count`
- compact source management button
- advanced diagnostics toggle

Main split view:

- Left: chat.
- Right: tabs for `Findings`, `Graph`, and `Activity`.

First-run right panel:

- source checklist preview for GitHub and Slack.
- disabled run/chat state until sources are connected.

## Hide By Default

- workspace path list.
- `Check DB`.
- `Fresh start`.
- DB truth checklist.
- local events, DB entities, DB edges, audit rows.
- stage checklist.
- separate status button.

These belong in an advanced diagnostics panel.

## Typography

Use a developer-tool style without making long text hard to read:

- Mono: top bar, buttons, tabs, status labels, source chips, graph labels.
- Sans: chat messages, findings text, activity feed content.

Preferred pairing:

- Body: Inter or system sans.
- Developer labels: IBM Plex Mono.
