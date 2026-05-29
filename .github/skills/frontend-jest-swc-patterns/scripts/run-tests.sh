#!/usr/bin/env bash
# run-tests.sh — convenience wrapper for the ContextOS frontend Jest suite.
# Usage:
#   ./scripts/run-tests.sh              run all tests
#   ./scripts/run-tests.sh --coverage   run with coverage report
#   ./scripts/run-tests.sh <pattern>    run tests matching a name pattern
#
# Requires: bun installed, deps installed (bun install in apps/frontend/)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
FRONTEND_DIR="$REPO_ROOT/apps/frontend"

cd "$FRONTEND_DIR"

if [[ "${1:-}" == "--coverage" ]]; then
  bun run test:coverage
elif [[ -n "${1:-}" ]]; then
  bun run test -- --testNamePattern="$1"
else
  bun run test
fi
