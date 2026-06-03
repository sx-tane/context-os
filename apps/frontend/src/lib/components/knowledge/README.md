# knowledge components

Svelte components for knowledge and connector setup surfaces.

## Components

| File | Purpose |
| --- | --- |
| `KnowledgeInstall.svelte` | Presents workspace source setup, Codex plugin readiness, source discovery, manual URI entry, and source ingest state. |

## Visual Pattern

`KnowledgeInstall.svelte` is used inline in the main workspace page and as an overlay-capable setup panel. The current style is a source list, not stacked cards:

- Connector rows use separators and light hover color changes.
- Embedded setup panels rely on the route container for horizontal padding; overlay panels provide their own inner padding.
- Inputs and action buttons inherit the mono font and use the same padded underline-fill treatment as the main route controls.
- SharePoint / OneDrive and Google Drive use stable text badges (`SP`, `GD`) instead of custom SVG marks.
- The header shows the active workspace name and path so first-time setup is tied to the right workspace.

## Maintenance Notes

- Keep setup copy tied to real local workflow steps.
- Do not hide connector or Codex status failures behind generic success states.
- Update `apps/frontend/README.md` when setup commands or user-visible installation behavior changes.
