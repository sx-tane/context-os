#!/usr/bin/env bash
# Scores README coverage across tracked directories.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
exclude_file="$repo_root/.codex/skills/contextos-authoring/references/readme-coverage-exclusions.txt"
out_file="/tmp/contextos-missing-readmes.txt"

: > "$out_file"

if [[ ! -d "$repo_root/.git" ]]; then
  echo "not inside a git repository: $repo_root" >&2
  exit 1
fi

mapfile -t excludes < <(grep -v '^\s*#' "$exclude_file" 2>/dev/null | sed '/^\s*$/d')

is_excluded() {
  local dir="$1"
  for ex in "${excludes[@]:-}"; do
    if [[ "$dir" == "$ex" || "$dir" == "$ex"/* ]]; then
      return 0
    fi
  done
  return 1
}

total_required=0
covered=0

git -C "$repo_root" ls-files \
  | awk -F/ 'NF>1 { p=""; for(i=1;i<NF;i++){ p=(p?p"/":"")$i; print p } }' \
  | sort -u \
  | while IFS= read -r dir; do
      if is_excluded "$dir"; then
        continue
      fi

      total_required=$((total_required + 1))
      if [[ -f "$repo_root/$dir/README.md" || -f "$repo_root/$dir/readme.md" ]]; then
        covered=$((covered + 1))
      else
        echo "$dir" >> "$out_file"
      fi

      printf "%s\t%s\n" "$total_required" "$covered" > /tmp/contextos-readme-coverage-counters.$$ 
    done

if [[ -f /tmp/contextos-readme-coverage-counters.$$ ]]; then
  total_required="$(cut -f1 /tmp/contextos-readme-coverage-counters.$$)"
  covered="$(cut -f2 /tmp/contextos-readme-coverage-counters.$$)"
  rm -f /tmp/contextos-readme-coverage-counters.$$
else
  total_required=0
  covered=0
fi

if (( total_required == 0 )); then
  echo "README coverage benchmark failed: no tracked directories found."
  exit 1
fi

missing=$((total_required - covered))
score=$((covered * 100 / total_required))

echo "README coverage score: $score/100"
echo "Required directories: $total_required"
echo "Covered directories: $covered"
echo "Missing README directories: $missing"

if (( missing > 0 )); then
  echo
  echo "Top missing directories:"
  head -n 120 "$out_file"
  echo
  echo "README coverage benchmark failed: add missing README files or update exclusions."
  exit 1
fi

echo "README coverage benchmark passed."
