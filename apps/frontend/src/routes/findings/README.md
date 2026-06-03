# findings route

Frontend route for role-based findings and mismatch analysis.

## Files

- `+page.svelte` renders the findings workflow and calls the presentation API for role-specific summaries and mismatch evidence.

## Maintenance Notes

- Preserve evidence, confidence, impact, severity, and recommended action fields when displaying findings.
- Keep findings refresh behavior aligned with `/presentation/findings`.
- Update `apps/api/handler/presentation/README.md` when API response fields or execution metadata change.
