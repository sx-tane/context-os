#!/usr/bin/env bash
# Scores README quality using folder context rather than one shared template.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
exclude_file="$repo_root/.github/skills/contextos-authoring/references/readme-coverage-exclusions.txt"

if [[ ! -d "$repo_root/.git" ]]; then
  echo "not inside a git repository: $repo_root" >&2
  exit 1
fi

mapfile -t excludes < <(grep -v '^[[:space:]]*#' "$exclude_file" 2>/dev/null | sed '/^[[:space:]]*$/d')

is_excluded() {
  local dir="$1"
  for ex in "${excludes[@]:-}"; do
    if [[ "$dir" == "$ex" || "$dir" == "$ex"/* ]]; then
      return 0
    fi
  done
  return 1
}

has_intro() {
  local file="$1"
  awk 'NR > 1 && $0 !~ /^#/ && $0 !~ /^[[:space:]]*$/ && length($0) > 30 { found=1 } END { exit(found ? 0 : 1) }' "$file"
}

has_structure_artifact() {
  local file="$1"
  grep -Eq '(\| .+ \||```|^[-*] )' "$file"
}

is_high_level_dir() {
  case "$1" in
    apps|apps/api|apps/frontend|apps/frontend/src|docs|domain|internal|storage|tests) return 0 ;;
    *) return 1 ;;
  esac
}

needs_ops_section() {
  case "$1" in
    apps|apps/ai-worker|apps/api|apps/frontend|docker|docs|domain|internal|migrations|prompts|scripts|storage|tests) return 0 ;;
    *) return 1 ;;
  esac
}

folder_specific_check() {
  local dir="$1"
  local file="$2"

  case "$dir" in
    apps)
      grep -Eq '(api/|frontend/|ai-worker/)' "$file"
      ;;
    apps/api)
      grep -Eq '(handler/|request/|response/|middleware/|docs/)' "$file"
      ;;
    apps/api/request)
      grep -Eqi '(request|payload|json|ingest\.go)' "$file"
      ;;
    apps/api/response)
      grep -Eqi '(response|payload|error\.go|ingest\.go)' "$file"
      ;;
    apps/frontend)
      grep -Eqi '(SvelteKit|codegen|test|src/)' "$file"
      ;;
    apps/frontend/src)
      grep -Eq '(lib/|routes/)' "$file"
      ;;
    internal)
      grep -Eq '(ingestion|normalization|classification|extraction|identity|relationship|graph|reasoning|execution|presentation)' "$file"
      ;;
    internal/pipeline)
      grep -Eqi '(orchestration|stage|pipeline)' "$file"
      ;;
    domain)
      grep -Eq '(contracts|events|types|entities|pipelines)' "$file"
      ;;
    domain/types)
      grep -Eqi '(NormalizedDocument|Classification|Entity|Relationship|Mismatch)' "$file"
      ;;
    storage)
      grep -Eq '(raw/|parsed/|embeddings/|snapshots/)' "$file"
      ;;
    migrations)
      grep -Eqi '(migration|schema|PostgreSQL|pgvector|currently contains only this README)' "$file"
      ;;
    prompts)
      grep -Eqi '(prompt|AI|worker|automation|currently contains only this README)' "$file"
      ;;
    storage/embeddings)
      grep -Eqi '(embedding|pgvector|vector|currently contains only this README)' "$file"
      ;;
    storage/parsed)
      grep -Eqi '(parsed|normalized|stage output|currently contains only this README)' "$file"
      ;;
    storage/raw)
      grep -Eqi '(raw|uploads/|source snapshots|ingest staging)' "$file"
      ;;
    storage/snapshots)
      grep -Eqi '(snapshot|graph|reasoning|comparison|currently contains only this README)' "$file"
      ;;
    tests)
      grep -Eqi '(go test|harness|pipeline)' "$file"
      ;;
    *)
      return 1
      ;;
  esac
}

mentions_direct_child() {
  local dir="$1"
  local file="$2"
  local child
  while IFS= read -r child; do
    [[ -z "$child" ]] && continue
    [[ "$child" == "README.md" || "$child" == "readme.md" ]] && continue
    if grep -Fqi "$child" "$file"; then
      return 0
    fi
    child="${child%.*}"
    if grep -Fqi "$child" "$file"; then
      return 0
    fi
  done < <(
    git -C "$repo_root" ls-files "$dir" \
      | awk -v prefix="$dir/" 'index($0, prefix) == 1 { rest = substr($0, length(prefix) + 1); split(rest, parts, "/"); if (parts[1] != "") print parts[1] }' \
      | sort -u
  )
  return 1
}

score_readme() {
  local dir="$1"
  local file="$2"
  local score=0
  local findings=()

  if grep -q '^# ' "$file"; then
    score=$((score + 20))
  else
    findings+=("missing-title")
  fi

  if has_intro "$file"; then
    score=$((score + 20))
  else
    findings+=("missing-meaningful-body")
  fi

  if folder_specific_check "$dir" "$file" || mentions_direct_child "$dir" "$file"; then
    score=$((score + 20))
  else
    findings+=("missing-code-context")
  fi

  if is_high_level_dir "$dir"; then
    if grep -qi '```mermaid' "$file"; then
      score=$((score + 20))
    else
      findings+=("missing-mermaid")
    fi
  elif has_structure_artifact "$file" || [[ "$(grep -Ec '^## ' "$file")" -ge 1 ]]; then
    score=$((score + 20))
  else
    findings+=("missing-structure-aid")
  fi

  if needs_ops_section "$dir"; then
    if grep -Eqi '^## .*?(Run Commands|Usage|Testing|Verification|Update Workflow|Maintenance|Maintenance Checklist|Retention Notes|Golden Update Policy|Implementation Notes|Running locally|Running tests)$' "$file" || grep -Eq '```(bash|sh)' "$file"; then
      score=$((score + 20))
    else
      findings+=("missing-operational-guidance")
    fi
  elif [[ "$(grep -Ec '^## ' "$file")" -ge 1 ]]; then
    score=$((score + 20))
  else
    findings+=("missing-secondary-sections")
  fi

  local findings_str="ok"
  if (( ${#findings[@]} > 0 )); then
    findings_str="$(IFS=,; echo "${findings[*]}")"
  fi

  printf '%s\t%s\t%s\n' "$dir" "$score" "$findings_str"
}

results_file="/tmp/contextos-readme-quality.tsv"
: > "$results_file"

while IFS= read -r dir; do
  if is_excluded "$dir"; then
    continue
  fi

  readme="$repo_root/$dir/README.md"
  if [[ -f "$readme" ]]; then
    score_readme "$dir" "$readme" >> "$results_file"
  fi
done < <(
  git -C "$repo_root" ls-files \
    | awk -F/ 'NF>1 { p=""; for(i=1;i<NF;i++){ p=(p?p"/":"")$i; print p } }' \
    | sort -u
)

if [[ ! -s "$results_file" ]]; then
  echo "README quality benchmark failed: no README files found in required directories."
  exit 1
fi

printf '%-40s %5s  %s\n' "directory" "score" "findings"
printf '%-40s %5s  %s\n' "----------------------------------------" "-----" "--------"
awk -F'\t' '{ printf "%-40s %5s  %s\n", $1, $2, $3 }' "$results_file"

total="$(awk -F'\t' '{sum += $2} END {print sum+0}' "$results_file")"
count="$(awk 'END {print NR+0}' "$results_file")"
avg=$((total / count))

echo
echo "Average README quality score: $avg/100 across $count directories"

if awk -F'\t' '$2 < 100 { found=1 } END { exit(found ? 0 : 1) }' "$results_file"; then
  echo "README quality benchmark failed: one or more README files scored below 100."
  exit 1
fi

echo "README quality benchmark passed: all required README files scored 100."