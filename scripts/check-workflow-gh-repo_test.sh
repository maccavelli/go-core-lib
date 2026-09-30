#!/usr/bin/env bash
# Offline tests for check-workflow-gh-repo.sh: each plant removes GH_REPO in a
# way an earlier version of the checker missed, and must be reported
# (docs/decisions/0003-MADR-remediate-debugging-pass-findings.md D4).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK="$ROOT/scripts/check-workflow-gh-repo.sh"
WORKFLOW="$ROOT/.github/workflows/publish-selfupdate-release.yml"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

PASS=0
FAIL=0

expect() { # name want-rc file
	set +e
	"$CHECK" "$3" >/dev/null 2>&1
	rc=$?
	set -e
	if [ "$rc" -eq "$2" ]; then
		echo "  ok   $1"
		PASS=$((PASS + 1))
	else
		echo "  FAIL $1: want exit $2, got $rc"
		FAIL=$((FAIL + 1))
	fi
}

# drop_repo_in <step name>: the workflow without that step's GH_REPO line.
drop_repo_in() {
	awk -v step="- name: $1" '
		index($0, step) { in_step = 1 }
		in_step && /GH_REPO:/ { in_step = 0; next }
		{ print }
	' "$WORKFLOW"
}

# 0. The real workflow passes.
expect "real workflow" 0 "$WORKFLOW"

# 1. The control: a direct gh call loses GH_REPO.
drop_repo_in "Publish the draft" >"$WORK/control.yml"
expect "gh call without GH_REPO" 1 "$WORK/control.yml"

# 2. The step whose gh call is inside refuse-existing-release.sh.
drop_repo_in "Refuse an existing release" >"$WORK/refuse.yml"
expect "refuse script without GH_REPO" 1 "$WORK/refuse.yml"

# 3. GH_REPO commented out: a comment sets nothing.
awk '
	index($0, "- name: Publish the draft") { in_step = 1 }
	in_step && /GH_REPO:/ { sub(/GH_REPO:/, "# GH_REPO:"); in_step = 0 }
	{ print }
' "$WORKFLOW" >"$WORK/commented.yml"
expect "commented-out GH_REPO" 1 "$WORK/commented.yml"

# 4. An unnamed step after a step that sets GH_REPO.
cp "$WORKFLOW" "$WORK/unnamed.yml"
# The literal $TAG is YAML content for the planted step, not shell here.
# shellcheck disable=SC2016
printf '      - run: gh release view "$TAG"\n' >>"$WORK/unnamed.yml"
expect "unnamed step inherits nothing" 1 "$WORK/unnamed.yml"

# 5. A --help probe without GH_REPO stays exempt.
cp "$WORKFLOW" "$WORK/help.yml"
printf '      - name: Probe\n        run: gh release list --help\n' >>"$WORK/help.yml"
expect "--help probe exempt" 0 "$WORK/help.yml"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
