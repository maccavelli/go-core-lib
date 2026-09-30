#!/usr/bin/env bash
# Assert no ${{ }} expression is interpolated inside a run: block of the
# reusable release workflow.
#
# GitHub substitutes ${{ }} into the script text before the shell runs it, so
# an attacker-chosen value, such as a tag name, becomes shell code before any
# check can see it. Values reach run blocks through env: instead
# (docs/decisions/0003-MADR-remediate-debugging-pass-findings.md D8).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKFLOW="${1:-$ROOT/.github/workflows/publish-selfupdate-release.yml}"

[ -f "$WORKFLOW" ] || {
	echo "check-workflow-expressions: no such workflow: $WORKFLOW" >&2
	exit 1
}

found=$(awk '
	function indent(s) { match(s, /^ */); return RLENGTH }
	# Leaving a block scalar: a non-blank line indented no deeper than run:.
	in_run && /[^ ]/ && indent($0) <= run_indent { in_run = 0 }
	/^ *(- )?run: *[|>][-+]? *$/ { in_run = 1; run_indent = indent($0); next }
	/^ *(- )?run: / && /\$\{\{/ { printf "  line %d: %s\n", NR, $0; next }
	in_run && /\$\{\{/ { printf "  line %d: %s\n", NR, $0 }
' "$WORKFLOW")

if [ -n "$found" ]; then
	echo "check-workflow-expressions: \${{ }} inside run blocks (use env:):" >&2
	echo "$found" >&2
	exit 1
fi

echo "check-workflow-expressions: ok — no \${{ }} inside run blocks"
