---
description: "Use when editing or reviewing Svelte frontend pages and components. Enforces current ContextOS design patterns for spacing, buttons, panels, graph views, source setup, chat panes, and readable operational UI."
applyTo: "apps/frontend/src/**/*.svelte"
---

# Frontend Design Instructions

## Skill

For design workflow, examples, and completion checks, apply the **contextos-frontend-design** skill.

## Key Rules

- Read the nearest frontend README before changing layout, spacing, controls, or visual state.
- Keep operational screens restrained: warm background, mono typography, separators, and flat rows.
- Use the padded underline-fill button treatment for primary, secondary, close, skip, save, and danger actions.
- Give rows, panels, and controls explicit left/right padding.
- Prefer local scroll areas with hidden scrollbars when a pane needs scrolling; avoid whole-page or accidental nested visible scrollbars.
- For source setup, use source-list rows with separators, not stacked cards.
- For graph views, keep names visible and focus selected-entity relationships instead of drawing all links.

## Verify

- Run `npm run check` from `apps/frontend` after Svelte UI changes.
- Run `npm run test` from `apps/frontend` when behavior or shared helpers changed.

## Documentation

- Update the nearest frontend README when the UI pattern, component behavior, or layout contract changes.
