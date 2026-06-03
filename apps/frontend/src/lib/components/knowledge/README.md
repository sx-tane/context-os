# knowledge components

Svelte components for knowledge and connector setup surfaces.

## Components

| File | Purpose |
| --- | --- |
| `KnowledgeInstall.svelte` | Presents workspace source setup, Codex plugin readiness, source discovery, manual URI entry, connected-source registration, and filesystem ingest state. |

## Visual Pattern

`KnowledgeInstall.svelte` is used inline in the main workspace page and as an overlay-capable setup panel. The current style is a source list, not stacked cards:

- Connector rows use separators and light hover color changes.
- Embedded and overlay setup panels use the same side inset so the header, close button, connector rows, and source options align with the route content.
- Inputs and action buttons inherit the mono font and use the same padded underline-fill treatment as the main route controls.
- Connector readiness badges stay inline with the connector name and use the same restrained mono badge treatment as other setup state labels.
- Connector rows, readiness badges, and nested source options stay transparent with separator lines so source links do not look like stacked boxes.
- SharePoint / OneDrive and Google Drive use stable text badges (`SP`, `GD`) instead of custom SVG marks.
- The header stays compact; source count and close control share the saved-source summary row.
- The panel shows sources already saved for the active workspace before the connector list. The save button count reacts to enabled external connector rows, newly checked discovery sources, and manually typed source URIs; already connected sources remain visible separately.
- External connector rows are directly clickable. Enabling a live connector is enough to save that plugin for chat; **Check plugin account** discovers repos, projects, channels, pages, folders, sites, or documents visible to the logged-in plugin account as optional scope filters. Manual URI entry stays collapsed as the second input path for pasted or known source references.
- External sources use connected-source language and show `connected` rather than ingest event counts until explicit analysis or ingest creates local artifacts.
- Filesystem remains the setup path that ingests local content into ContextOS storage immediately. Its row exposes browser file and folder upload controls first, with server-path ingest collapsed under a fallback details panel.
- `Reset all data` uses the shared destructive confirmation modal so it matches workspace removal instead of using a browser-native confirmation popup.

## Maintenance Notes

- Keep setup copy tied to real local workflow steps.
- Keep saved-source state scoped to `$project.workspacePath`; do not present plugin discovery results as already connected until `POST /workspace/source` marks the source ready for that workspace.
- Do not hide connector or Codex status failures behind generic success states.
- Update `apps/frontend/README.md` when setup commands or user-visible installation behavior changes.
