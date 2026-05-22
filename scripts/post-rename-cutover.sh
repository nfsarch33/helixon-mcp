#!/usr/bin/env bash
# post-rename-cutover.sh — Flip the remaining nfsarch33/ironclaw-mcp URL
# references over to nfsarch33/helixon-mcp once the GitHub repo rename
# has been applied by the operator.
#
# Idempotent. Safe to re-run. Operates only on text under the repo root.
#
# Usage:
#   bash scripts/post-rename-cutover.sh           # apply changes
#   bash scripts/post-rename-cutover.sh --dry-run # show pending edits, no writes

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

DRY_RUN=0
if [ "${1:-}" = "--dry-run" ]; then
  DRY_RUN=1
fi

OLD_URL_PATH='nfsarch33/ironclaw-mcp'
NEW_URL_PATH='nfsarch33/helixon-mcp'
OLD_MOD='github.com/nfsarch33/ironclaw-mcp'
NEW_MOD='github.com/nfsarch33/helixon-mcp'

# Files that we actively maintain and intend to flip.
# BOUNDARY.md is intentionally excluded — its OLD→NEW mapping table is a
# historical record. Operator hand-edits BOUNDARY.md "rename pending" status
# line separately.
TARGETS=(
  README.md
  llms.txt
  docs/quickstart.md
)

echo "post-rename-cutover: dry_run=${DRY_RUN}"

apply() {
  local file="$1"
  if [ ! -f "$file" ]; then
    echo "  skip: $file (missing)"
    return
  fi
  if grep -qE "${OLD_URL_PATH}|${OLD_MOD}" "$file"; then
    if [ "$DRY_RUN" -eq 1 ]; then
      echo "  would patch: $file"
      grep -nE "${OLD_URL_PATH}|${OLD_MOD}" "$file" | sed 's/^/    /'
    else
      # macOS BSD sed needs '' after -i; GNU sed accepts -i without arg.
      # Use a portable two-step.
      tmp="$(mktemp)"
      sed -e "s|${OLD_MOD}|${NEW_MOD}|g" -e "s|${OLD_URL_PATH}|${NEW_URL_PATH}|g" "$file" > "$tmp"
      mv "$tmp" "$file"
      echo "  patched: $file"
    fi
  else
    echo "  clean: $file"
  fi
}

for f in "${TARGETS[@]}"; do
  apply "$f"
done

# Sanity verification.
remaining=$(grep -rIE "${OLD_URL_PATH}|${OLD_MOD}" "${TARGETS[@]}" 2>/dev/null | wc -l | tr -d ' ')
echo "remaining references in maintained docs: ${remaining}"

if [ "$DRY_RUN" -eq 0 ] && [ "$remaining" != "0" ]; then
  echo "WARN: maintained docs still contain old references; manual review needed" >&2
  exit 2
fi

echo "post-rename-cutover: done"
