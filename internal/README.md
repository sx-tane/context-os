# Internal Pipeline Stages

This folder contains ContextOS stage implementations and runtime orchestration internals.

## Stage Responsibilities

- Implement stage logic behind stable domain contracts.
- Preserve traceability and provenance across stage outputs.
- Keep stages decoupled to protect pipeline boundaries.

## Stage Flow

```mermaid
flowchart LR
	ING[ingestion] --> NORM[normalization]
	NORM --> CLS[classification]
	CLS --> EXT[extraction]
	EXT --> ID[identity]
	ID --> REL[relationship]
	REL --> G[graph]
	G --> REAS[reasoning]
	REAS --> EXEC[execution]
	EXEC --> PRES[presentation]
```

## Maintenance Checklist

- Update stage READMEs when behavior or contracts change.
- Add tests for new stage paths, including replay/duplicate handling.
- Avoid cross-importing between unrelated `internal/` stages.
