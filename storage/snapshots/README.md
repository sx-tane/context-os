# Snapshot Storage

Stores historical context graph snapshots and comparable baseline outputs.

This folder currently contains only this README. Add snapshot artifacts here when graph, reasoning, or presentation outputs need stable comparison points.

## Responsibilities

- Capture stable point-in-time outputs for comparison and regression review.
- Support local analysis of graph evolution and reasoning changes.
- Keep snapshot naming and contents deterministic where possible.

## Maintenance Checklist

- Document snapshot naming or retention rules when they change.
- Keep snapshots free of volatile values when used for regression comparison.
- Align snapshot usage with harness and storage documentation.
