#!/usr/bin/env bash
# Checks that explanatory responses are required to use Mermaid diagrams.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
copilot_instructions="$repo_root/AGENTS.md"
github_readme="$repo_root/.codex/README.md"
authoring_skill="$repo_root/.codex/skills/contextos-authoring/SKILL.md"

score=0
findings=()

if grep -qi 'Mermaid diagram' "$copilot_instructions"; then
  score=$((score + 40))
else
    findings+=("missing-agents-mermaid-rule")
fi

for term in architecture workflows pipeline "skill routing"; do
  if grep -qi "$term" "$copilot_instructions"; then
    score=$((score + 5))
  else
    findings+=("missing-scope-$term")
  fi
done

if grep -q '^## Mermaid Explanation Policy' "$github_readme"; then
  score=$((score + 20))
else
  findings+=("missing-readme-mermaid-policy")
fi

if grep -q 'Mermaid' "$authoring_skill"; then
  score=$((score + 20))
else
  findings+=("missing-authoring-mermaid-reference")
fi

printf "Mermaid policy score: %s/100\n" "$score"

if [[ ${#findings[@]} -gt 0 ]]; then
  printf "Findings: %s\n" "${findings[*]}"
  exit 1
fi

echo "Mermaid policy benchmark passed."
