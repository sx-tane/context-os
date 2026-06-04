# graph

Graph view-model helpers for focused entity displays.

## Responsibilities

- Shape backend graph entities and relationships into UI-friendly link lists.
- Keep graph rendering focused on the selected entity and its direct relationships.
- Avoid backend fetches; network calls belong in `$lib/api`.

## Files

| File | Purpose |
| --- | --- |
| `viewModel.ts` | Builds graph links, labels, entity summaries, and relationship display rows. |

## Maintenance Notes

- Update `components/graph/README.md` when helper changes alter graph UI behavior.
- Run frontend tests after changing graph helper output.
