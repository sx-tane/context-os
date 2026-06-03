# knowledge components

Svelte components for knowledge and connector setup surfaces.

## Components

| File | Purpose |
| --- | --- |
| `KnowledgeInstall.svelte` | Presents workspace source setup, Codex plugin readiness, source discovery, manual URI entry, and source ingest state. |

## Visual Pattern

`KnowledgeInstall.svelte` is used inline in the main workspace page and as an overlay-capable setup panel. The current style is a source list, not stacked cards:

- Connector rows use separators and light hover color changes.
- Embedded and overlay setup panels use the same side inset so the header, close button, connector rows, and source options align with the route content.
- Inputs and action buttons inherit the mono font and use the same padded underline-fill treatment as the main route controls.
- Connector readiness badges stay inline with the connector name and use the same restrained mono badge treatment as other setup state labels.
- SharePoint / OneDrive and Google Drive use stable text badges (`SP`, `GD`) instead of custom SVG marks.
- The header shows the active workspace name and path so first-time setup is tied to the right workspace.

## Maintenance Notes

- Keep setup copy tied to real local workflow steps.
- Do not hide connector or Codex status failures behind generic success states.
- Update `apps/frontend/README.md` when setup commands or user-visible installation behavior changes.
