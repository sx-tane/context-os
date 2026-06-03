#!/usr/bin/env bash
# Scores whether common prompts route to the intended ContextOS skills.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
customization_root="$repo_root/.codex"
skills_dir="$customization_root/skills"
scenario_file="$customization_root/skills/contextos-authoring/references/skill-routing-scenarios.tsv"
github_readme="$customization_root/README.md"
implementer_agent="$customization_root/agents/contextos-implementer.agent.md"
architect_agent="$customization_root/agents/contextos-architect.agent.md"

if [[ ! -f "$scenario_file" ]]; then
  echo "routing scenario file not found: $scenario_file" >&2
  exit 1
fi

total_score=0
scenario_count=0
failed=0

printf "%-24s %-38s %5s  %s\n" "scenario" "expected_skill" "score" "findings"
printf "%-24s %-38s %5s  %s\n" "------------------------" "--------------------------------------" "-----" "--------"

while IFS=$'\t' read -r scenario_id prompt_trigger expected_skill required_terms avoid_skills || [[ -n "${scenario_id:-}" ]]; do
  if [[ "$scenario_id" == "id" || -z "$scenario_id" ]]; then
    continue
  fi

  skill_file="$skills_dir/$expected_skill/SKILL.md"
  score=0
  findings=()

  if [[ -f "$skill_file" ]]; then
    score=$((score + 20))
  else
    findings+=("missing-skill")
  fi

  if [[ -f "$skill_file" ]]; then
    matched_term=false
    IFS=',' read -ra terms <<< "$required_terms"
    for term in "${terms[@]}"; do
      if grep -qiF "$term" "$skill_file"; then
        matched_term=true
        break
      fi
    done
    if [[ "$matched_term" == "true" ]]; then
      score=$((score + 25))
    else
      findings+=("trigger-terms-not-in-skill")
    fi
  fi

  if [[ -f "$github_readme" ]] && grep -q "\`$expected_skill\`" "$github_readme"; then
    score=$((score + 20))
  else
    findings+=("missing-readme-routing")
  fi

  if grep -q "$expected_skill" "$implementer_agent" "$architect_agent" 2>/dev/null; then
    score=$((score + 20))
  else
    findings+=("missing-agent-routing")
  fi

  confused=false
  IFS=',' read -ra avoided <<< "$avoid_skills"
  for avoid_skill in "${avoided[@]}"; do
    if [[ "$avoid_skill" == "$expected_skill" ]]; then
      confused=true
      break
    fi
  done
  if [[ "$confused" == "false" ]]; then
    score=$((score + 15))
  else
    findings+=("self-listed-as-avoid-skill")
  fi

  scenario_count=$((scenario_count + 1))
  total_score=$((total_score + score))

  if (( score < 100 || ${#findings[@]} > 0 )); then
    failed=$((failed + 1))
  fi

  if [[ ${#findings[@]} -eq 0 ]]; then
    finding_text="ok"
  else
    finding_text="${findings[*]}"
  fi

  printf "%-24s %-38s %5s  %s\n" "$scenario_id" "$expected_skill" "$score" "$finding_text"
done < "$scenario_file"

if (( scenario_count == 0 )); then
  echo "Routing benchmark failed: no scenarios were loaded." >&2
  exit 1
fi

average=$((total_score / scenario_count))

printf "\nAverage routing score: %s/100 across %s scenarios\n" "$average" "$scenario_count"

if (( failed > 0 )); then
  echo "Routing benchmark failed: $failed scenario(s) have findings."
  exit 1
fi

echo "Routing benchmark passed: all scenarios route cleanly."
