#!/usr/bin/env bash
# Scores ContextOS skill files against the authoring checklist.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
skills_dir="$repo_root/.github/skills"
github_readme="$repo_root/.github/README.md"
implementer_agent="$repo_root/.github/agents/contextos-implementer.agent.md"
architect_agent="$repo_root/.github/agents/contextos-architect.agent.md"

if [[ ! -d "$skills_dir" ]]; then
  echo "skills directory not found: $skills_dir" >&2
  exit 1
fi

total_score=0
skill_count=0
failed=0
max_score=125

printf "%-42s %5s  %s\n" "skill" "score" "findings"
printf "%-42s %5s  %s\n" "------------------------------------------" "-----" "--------"

for skill_file in "$skills_dir"/*/SKILL.md; do
  skill_dir="$(dirname "$skill_file")"
  skill_name="$(basename "$skill_dir")"
  score=0
  findings=()

  frontmatter_name="$(awk -F': ' '/^name: / {print $2; exit}' "$skill_file" | tr -d '"')"
  description="$(awk -F': ' '/^description: / {print $2; exit}' "$skill_file" | tr -d '"')"

  if [[ "$frontmatter_name" == "$skill_name" ]]; then
    score=$((score + 10))
  else
    findings+=("name-mismatch")
  fi

  if grep -q '^description: .*Use when' "$skill_file"; then
    score=$((score + 10))
  else
    findings+=("description-missing-use-when")
  fi

  if [[ ${#description} -le 1024 ]]; then
    score=$((score + 5))
  else
    findings+=("description-too-long")
  fi

  if grep -q '^argument-hint: ' "$skill_file"; then
    score=$((score + 5))
  else
    findings+=("missing-argument-hint")
  fi

  if grep -q '^user-invocable: true' "$skill_file"; then
    score=$((score + 5))
  else
    findings+=("missing-user-invocable")
  fi

  if grep -q '^## Outcome' "$skill_file"; then
    score=$((score + 8))
  else
    findings+=("missing-outcome")
  fi

  if grep -q '^## Procedure' "$skill_file"; then
    score=$((score + 8))
  else
    findings+=("missing-procedure")
  fi

  if grep -Eq '^## (Decision Points|Decision Table|When to Use|Purpose)' "$skill_file"; then
    score=$((score + 4))
  else
    findings+=("missing-decision-context")
  fi

  if grep -q '^## References' "$skill_file"; then
    score=$((score + 5))
  else
    findings+=("missing-references-section")
  fi

  if find "$skill_dir/assets" -type f -mindepth 1 -maxdepth 1 >/dev/null 2>&1 && [[ -n "$(find "$skill_dir/assets" -type f -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]]; then
    score=$((score + 10))
  else
    findings+=("missing-assets")
  fi

  if find "$skill_dir/references" -type f -mindepth 1 -maxdepth 1 >/dev/null 2>&1 && [[ -n "$(find "$skill_dir/references" -type f -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]]; then
    score=$((score + 10))
  else
    findings+=("missing-references-files")
  fi

  if [[ -d "$skill_dir/scripts" ]]; then
    if [[ -n "$(find "$skill_dir/scripts" -type f -mindepth 1 -maxdepth 1 -print -quit 2>/dev/null)" ]] && grep -q './scripts/' "$skill_file"; then
      score=$((score + 5))
    else
      findings+=("scripts-not-referenced-or-empty")
    fi
  else
    score=$((score + 5))
  fi

  if [[ -f "$github_readme" ]]; then
    if grep -q "\`$skill_name\`" "$github_readme"; then
      score=$((score + 10))
    else
      findings+=("missing-github-readme-map")
    fi
  fi

  if grep -q 'README' "$skill_file"; then
    score=$((score + 10))
  else
    findings+=("missing-readme-alignment")
  fi

  if grep -q "$skill_name" "$implementer_agent" "$architect_agent" 2>/dev/null; then
    score=$((score + 10))
  else
    findings+=("not-wired-to-agent")
  fi

  duplicate_descriptions="$(grep -R '^description: ' "$skills_dir"/*/SKILL.md | grep -F "$description" | wc -l | tr -d ' ')"
  if [[ "$duplicate_descriptions" == "1" ]]; then
    score=$((score + 10))
  else
    findings+=("duplicate-description")
  fi

  percent=$((score * 100 / max_score))
  total_score=$((total_score + percent))
  skill_count=$((skill_count + 1))

  if (( percent < 90 || ${#findings[@]} > 0 )); then
    failed=$((failed + 1))
  fi

  if [[ ${#findings[@]} -eq 0 ]]; then
    finding_text="ok"
  else
    finding_text="${findings[*]}"
  fi

  printf "%-42s %5s  %s\n" "$skill_name" "$percent" "$finding_text"
done

average=$((total_score / skill_count))

printf "\nAverage score: %s/100 across %s skills\n" "$average" "$skill_count"

if (( failed > 0 )); then
  echo "Benchmark failed: $failed skill(s) have findings or scored below 90."
  exit 1
fi

echo "Benchmark passed: all skills scored at least 90 with no findings."
