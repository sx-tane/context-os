#!/usr/bin/env bash
# Checks that explanatory responses are required to use Mermaid diagrams.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
copilot_instructions="$repo_root/.github/copilot-instructions.md"
github_readme="$repo_root/.github/README.md"
authoring_skill="$repo_root/.github/skills/contextos-authoring/SKILL.md"

score=0
findings=()
notes=()

add_finding() {
  local file="$1"
  local check="$2"
  findings+=("$file: $check")
}

if grep -qi 'Mermaid diagram' "$copilot_instructions"; then
  score=$((score + 40))
else
  add_finding "$copilot_instructions" "missing Mermaid diagram rule"
fi

for term in architecture workflows pipeline "skill routing"; do
  if grep -qi "$term" "$copilot_instructions"; then
    score=$((score + 5))
  else
    add_finding "$copilot_instructions" "missing scope term: $term"
  fi
done

if [[ -f "$github_readme" ]]; then
  if grep -q '^## Mermaid Explanation Policy' "$github_readme"; then
    score=$((score + 20))
  else
    add_finding "$github_readme" "missing ## Mermaid Explanation Policy"
  fi
else
  score=$((score + 20))
  notes+=("$github_readme: optional README not present; top-level GitHub README policy check skipped")
fi

if grep -q 'Mermaid' "$authoring_skill"; then
  score=$((score + 20))
else
  add_finding "$authoring_skill" "missing Mermaid reference"
fi

printf "Mermaid policy score: %s/100\n" "$score"

if [[ ${#notes[@]} -gt 0 ]]; then
  printf "Notes:\n"
  printf " - %s\n" "${notes[@]}"
fi

if [[ ${#findings[@]} -gt 0 ]]; then
  printf "Findings:\n"
  printf " - %s\n" "${findings[@]}"
  exit 1
fi

echo "Mermaid policy benchmark passed."
