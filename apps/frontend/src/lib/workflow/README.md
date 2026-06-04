# workflow

Frontend workflow view models and local workflow state helpers.

## Files

| File | Purpose |
| --- | --- |
| `types.ts` | Narrow workflow-only aliases for evidence basket and finding-action state. Public route/component imports may also use the canonical exports from `$lib/types`. |
| `viewModel.ts` | Pure helpers for analysis previews, Activity filters, source health, evidence pinning, finding actions, and workspace snapshot text. |

## Maintenance Notes

- Keep helpers deterministic and UI-independent; components should receive already-built models and callbacks from the route.
- Basket presence narrows analysis to basket concrete sources only. Preview keeps other concrete sources visible as available but not selected.
- Ask-from-evidence helpers only build composer text. They must not auto-submit chat or start analysis.
