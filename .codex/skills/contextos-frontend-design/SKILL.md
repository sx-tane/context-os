---
name: contextos-frontend-design
description: "Implement or review ContextOS frontend UI design following the current app pattern. Use when: editing Svelte pages or components; changing layout, spacing, buttons, panels, lists, source setup, graph views, chat UI, or visual states; making frontend UI match existing design. Covers restrained mono styling, separator-based containers, hidden scrollbars, readable graph/entity layouts, and README-aligned verification."
argument-hint: "Which frontend surface or component is changing?"
user-invocable: true
---

# ContextOS Frontend Design Skill

## Outcome

Deliver frontend UI changes that feel native to the current ContextOS app:

- Existing warm, restrained mono visual language is preserved.
- Components use the local spacing, separator, button, and scroll patterns.
- Dense operational surfaces stay readable without extra boxed/card noise.
- Nearest frontend README is updated when behavior, layout, or component rules change.

---

## Decision Points

| Situation | Action |
| --- | --- |
| Editing `apps/frontend/src/routes/+page.svelte` | Follow the route layout rules in `apps/frontend/src/routes/README.md`. |
| Editing source setup UI | Follow `apps/frontend/src/lib/components/knowledge/README.md`; use source-list rows, not stacked cards. |
| Editing graph UI | Follow `apps/frontend/src/lib/components/graph/README.md`; draw selected-entity relationships, not every edge. |
| Adding a connector component | Also apply `contextos-frontend-connector`. |
| Adding or changing frontend tests | Also apply `frontend-jest-swc-patterns`. |

---

## Procedure

1. **Read local context first** — inspect the target Svelte file, its nearest README, and any sibling component with the same role.

2. **Apply the visual pattern**:
   - Use warm neutral backgrounds already present in the app.
   - Prefer full-width sections, separators, and flat rows over nested cards.
   - Use cards only for repeated item surfaces, modals, and genuinely framed tools.
   - Preserve readable labels; do not require hover to identify important entities or actions.
   - Keep text from touching container edges; use explicit left/right padding on rows, panels, and controls.
   - Use hidden scrollbars only when scroll remains local and discoverable by wheel/trackpad.

3. **Use the current control style**:
   - Buttons use the padded underline-fill treatment from the main route.
   - Inputs/selects use underline fields with transparent or warm backgrounds.
   - Dangerous actions keep the same control shape with a danger accent, not a separate button style.
   - Icon-only buttons need an accessible label/title.

4. **Keep operational density under control**:
   - Show only the metadata required for the current task.
   - Move secondary details into detail panels or expanded states.
   - For graph/entity views, make the selected entity and its direct relationships the primary view.

5. **Use the skeleton and checklist**:
   - For style changes, adapt the [style skeleton](./assets/frontend-style-skeleton.md).
   - Before finishing, run the [design checklist](./references/frontend-design-checklist.md).

6. **Verify and document**:
   - Run `npm run check` from `apps/frontend`.
   - Run `npm run test` from `apps/frontend` when behavior or helpers changed.
   - Run README sync guard after README or UI behavior changes.

---

## Do Not

- Do not add decorative gradients, floating blobs, or marketing-style hero/card layouts to operational screens.
- Do not put cards inside cards or turn page sections into floating decorative cards.
- Do not add visible scrollbars to nested panes unless the scrollbar is the intended primary navigation.
- Do not introduce hardcoded entity type assumptions such as `person`, `org`, or `feature`.
- Do not make one-off button styles for `Close`, `Skip`, `Save`, or destructive actions.

## References

- [Frontend Style Skeleton](./assets/frontend-style-skeleton.md) — reusable CSS snippets for rows, buttons, and local scroll panes.
- [Frontend Design Checklist](./references/frontend-design-checklist.md) — completion checklist for UI visual fit, readability, and validation.
