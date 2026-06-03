#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   create-parent-child-issues.sh <repo> <area-label> <parent-title> <parent-file> <child-spec-file>
#
# child-spec-file format (tab-delimited):
#   <child-title>\t<child-body-file>

if [[ $# -ne 5 ]]; then
  echo "Usage: $0 <repo> <area-label> <parent-title> <parent-file> <child-spec-file>" >&2
  exit 1
fi

repo="$1"
area_label="$2"
parent_title="$3"
parent_file="$4"
child_spec_file="$5"

if [[ ! -f "$parent_file" ]]; then
  echo "Parent body file not found: $parent_file" >&2
  exit 1
fi

if [[ ! -f "$child_spec_file" ]]; then
  echo "Child spec file not found: $child_spec_file" >&2
  exit 1
fi

parent_url=$(gh issue create \
  --repo "$repo" \
  --title "$parent_title" \
  --body-file "$parent_file" \
  --label "type: epic" \
  --label "$area_label")

parent_number=$(echo "$parent_url" | awk -F'/' '{print $NF}')
echo "Created parent issue #$parent_number"

while IFS=$'\t' read -r child_title child_body_file; do
  [[ -z "$child_title" ]] && continue
  [[ "$child_title" =~ ^# ]] && continue

  if [[ ! -f "$child_body_file" ]]; then
    echo "Child body file not found: $child_body_file" >&2
    exit 1
  fi

  child_url=$(gh issue create \
    --repo "$repo" \
    --title "$child_title" \
    --body-file "$child_body_file" \
    --label "type: feature" \
    --label "$area_label")

  child_number=$(echo "$child_url" | awk -F'/' '{print $NF}')
  echo "Created child issue #$child_number"

  gh issue comment "$child_number" \
    --repo "$repo" \
    --body "Linked to parent issue #$parent_number."
done < "$child_spec_file"

echo "Parent issue: #$parent_number"