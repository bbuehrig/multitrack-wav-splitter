#!/usr/bin/env bash
# Build GitHub Release notes from CHANGELOG.md.
# Usage: scripts/release-notes.sh <version> > release-notes.md
# Version may include a leading "v" (e.g. v0.2.0 or 0.2.0).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHANGELOG="${ROOT}/CHANGELOG.md"
VERSION="${1:-}"
VERSION="${VERSION#v}"

if [[ -z "${VERSION}" ]]; then
  echo "usage: $0 <version>" >&2
  exit 1
fi

if [[ ! -f "${CHANGELOG}" ]]; then
  echo "CHANGELOG.md not found" >&2
  exit 1
fi

# Extract the section for this version (### X.Y.Z … until next ### or EOF).
SECTION="$(awk -v ver="${VERSION}" '
  BEGIN { in_section = 0 }
  /^### / {
    if (in_section) exit
    # Match "### 0.2.0" or "### 0.2.0 – title"
    if ($0 ~ "^### " ver "([[:space:]–—-].*)?$") {
      in_section = 1
      next
    }
  }
  in_section { print }
' "${CHANGELOG}")"

echo "## What's new in v${VERSION}"
echo
if [[ -n "${SECTION}" ]]; then
  echo "${SECTION}"
else
  echo "_No CHANGELOG section found for ${VERSION}. See commit history._"
fi

echo
echo "---"
echo
echo "## Full changelog history"
echo
# Drop the top title line if present; keep the rest as history.
awk '
  NR == 1 && /^## Changelog/ { next }
  { print }
' "${CHANGELOG}"
