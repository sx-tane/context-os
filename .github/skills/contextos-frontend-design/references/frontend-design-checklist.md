# Frontend Design Skill — Completion Checklist

Run before marking frontend UI design work complete.

## Visual Fit

- [ ] Surface uses the existing warm neutral palette and mono typography.
- [ ] Rows, panels, and controls have clear left/right padding.
- [ ] Flat rows and separators are used instead of unnecessary boxed cards.
- [ ] Buttons follow the padded underline-fill treatment, including close/skip/save/danger actions.
- [ ] Inputs and selects use underline fields and match nearby controls.

## Readability

- [ ] Important labels are visible without hover.
- [ ] Dense metadata is reduced or moved to a detail panel.
- [ ] Text truncates or wraps intentionally and does not overlap.
- [ ] Local scroll areas do not create distracting visible nested scrollbars unless intentionally needed.

## Domain Surfaces

- [ ] Source setup remains a source list with separators, not stacked cards.
- [ ] Graph views focus selected entity relationships and do not draw every link at once.
- [ ] Chat and insight panes keep scroll behavior local and readable.

## Final

- [ ] `npm run check` passes from `apps/frontend`.
- [ ] `npm run test` passes from `apps/frontend` when behavior or helpers changed.
- [ ] Nearest frontend README updated when layout, behavior, or component rules changed.
- [ ] README sync guard passes for the working diff.
