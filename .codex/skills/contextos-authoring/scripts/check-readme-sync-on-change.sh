#!/usr/bin/env bash
# Fails if changed code files are not accompanied by meaningful nearest-README updates.

set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
range="${1:-HEAD}"
exclude_file="$repo_root/.codex/skills/contextos-authoring/references/readme-coverage-exclusions.txt"

if [[ ! -d "$repo_root/.git" ]]; then
  echo "not inside a git repository: $repo_root" >&2
  exit 1
fi

mapfile -t excludes < <(grep -v '^[[:space:]]*#' "$exclude_file" 2>/dev/null | sed '/^[[:space:]]*$/d')
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
declare -A shallow_readme_updates=()
declare -A missing_new_file_context=()
declare -A readmes_to_check=()

is_excluded() {
  local path="$1"
  for ex in "${excludes[@]:-}"; do
    if [[ "$path" == "$ex" || "$path" == "$ex"/* ]]; then
      return 0
    fi
  done
  return 1
}

should_skip_file() {
  local file="$1"
  local base="$(basename "$file")"

  if [[ "$base" == "README.md" || "$base" == "readme.md" ]]; then
    return 0
  fi

  if is_excluded "$file"; then
    return 0
  fi

  case "$file" in
    */node_modules/*|*/.svelte-kit/*|*/generated/*) return 0 ;;
    *_test.go|*.test.ts) return 0 ;;
    *.png|*.jpg|*.jpeg|*.gif|*.webp|*.svg|*.ico) return 0 ;;
    *.lock|*.lockb|*.sum|*.mod) return 0 ;;
    *.json|*.yaml|*.yml) return 1 ;;
  esac

  return 1
}

nearest_readme() {
  local dir="$1"

  while [[ "$dir" != "." && "$dir" != "/" ]]; do
    if is_excluded "$dir"; then
      return 1
    fi

    if [[ -f "$repo_root/$dir/README.md" ]]; then
      printf '%s/README.md\n' "$dir"
      return 0
    fi

    dir="$(dirname "$dir")"
  done

  return 1
}

has_meaningful_readme_delta() {
  local readme="$1"

  git -C "$repo_root" diff --unified=0 "$range" -- "$readme" \
    | awk '
        /^\+\+\+/ { next }
        /^\+[[:space:]]*$/ { next }
        /^\+[[:space:]]*[-*#>`_|:,.]*[[:space:]]*$/ { next }
        /^\+/ {
          line = substr($0, 2)
          gsub(/[`*_#>\[\](){}|:.,;/-]/, " ", line)
          words = split(line, parts, /[[:space:]]+/)
          useful = 0
          for (i = 1; i <= words; i++) {
            if (length(parts[i]) >= 3 && parts[i] ~ /[[:alpha:]]/) {
              useful++
            }
          }
          if (useful >= 3) {
            found = 1
          }
        }
        END { exit(found ? 0 : 1) }
      '
}

readme_mentions_new_file() {
  local readme="$1"
  local file="$2"
  local base="$(basename "$file")"
  local stem="${base%.*}"
  local parent="$(basename "$(dirname "$file")")"

  grep -Fqi "$base" "$repo_root/$readme" && return 0
  grep -Fqi "$stem" "$repo_root/$readme" && return 0
  grep -Fqi "$parent/" "$repo_root/$readme" && return 0

  return 1
}

record_changed_code() {
  local status="$1"
  local file="$2"

  if should_skip_file "$file"; then
    return 0
  fi

  local readme_path
  if ! readme_path="$(nearest_readme "$(dirname "$file")")"; then
    return 0
  fi

  readmes_to_check["$readme_path"]=1

  if [[ -z "${changed_map[$readme_path]:-}" ]]; then
    missing_readme_updates["$readme_path"]+=" $file"
    return 0
  fi

  if [[ "$status" == A* || "$status" == C* || "$status" == R* ]]; then
    if ! readme_mentions_new_file "$readme_path" "$file"; then
      missing_new_file_context["$readme_path"]+=" $file"
    fi
  fi
}

while IFS=$'\t' read -r status first second; do
  file="$first"
  if [[ "$status" == R* || "$status" == C* ]]; then
    file="$second"
  fi

  [[ -z "${file:-}" ]] && continue
  record_changed_code "$status" "$file"
done < <(git -C "$repo_root" diff --name-status --diff-filter=ACMRTUXB "$range")

for readme_path in "${!readmes_to_check[@]}"; do
  if [[ -n "${changed_map[$readme_path]:-}" ]]; then
    if ! has_meaningful_readme_delta "$readme_path"; then
      shallow_readme_updates["$readme_path"]=1
    fi
  fi
done

if (( ${#missing_readme_updates[@]} > 0 || ${#shallow_readme_updates[@]} > 0 || ${#missing_new_file_context[@]} > 0 )); then
  echo "README sync check failed for range $range"

  if (( ${#missing_readme_updates[@]} > 0 )); then
    echo
    echo "Changed code without nearest README updates:"
    for r in "${!missing_readme_updates[@]}"; do
      echo "$r <-${missing_readme_updates[$r]}"
    done | sort
  fi

  if (( ${#shallow_readme_updates[@]} > 0 )); then
    echo
    echo "README updates without meaningful added documentation:"
    for r in "${!shallow_readme_updates[@]}"; do
      echo "$r"
    done | sort
  fi

  if (( ${#missing_new_file_context[@]} > 0 )); then
    echo
    echo "New code files not mentioned by nearest README:"
    for r in "${!missing_new_file_context[@]}"; do
      echo "$r <-${missing_new_file_context[$r]}"
    done | sort
  fi

  exit 1
fi

echo "README sync check passed for range $range"
