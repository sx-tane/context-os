#!/usr/bin/env bash
# Fails if changed code files are not accompanied by README updates in the same directory.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
range="${1:-HEAD~1..HEAD}"

if [[ ! -d "$repo_root/.git" ]]; then
  echo "not inside a git repository: $repo_root" >&2
  exit 1
fi

mapfile -t changed < <(git -C "$repo_root" diff --name-only --diff-filter=ACMRTUXB "$range")

if [[ ${#changed[@]} -eq 0 ]]; then
  echo "README sync check: no changed files for range $range"
  exit 0
fi

declare -A changed_map=()
for f in "${changed[@]}"; do
  changed_map["$f"]=1
done

declare -A missing_readme_updates=()

for f in "${changed[@]}"; do
  base="$(basename "$f")"

  if [[ "$base" == "README.md" || "$base" == "readme.md" ]]; then
    continue
  fi

  case "$f" in
    *.png|*.jpg|*.jpeg|*.gif|*.webp|*.svg|*.lock|*.sum|*.mod) continue ;;
  esac

  dir="$(dirname "$f")"
  readme_path="$dir/README.md"

  if [[ -f "$repo_root/$readme_path" ]]; then
    if [[ -z "${changed_map[$readme_path]:-}" ]]; then
      missing_readme_updates["$readme_path"]=1
    fi
  fi
done

if (( ${#missing_readme_updates[@]} > 0 )); then
  echo "README sync check failed for range $range"
  echo "Changed code without same-folder README updates:"
  for r in "${!missing_readme_updates[@]}"; do
    echo "$r"
  done | sort
  exit 1
fi

echo "README sync check passed for range $range"