#!/usr/bin/env bash
# Offline tests for check-workflow-expressions.sh
# (docs/decisions/0003-MADR-remediate-debugging-pass-findings.md D8).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CHECK="$ROOT/scripts/check-workflow-expressions.sh"
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

# The expression text is built at run time so this file holds no literal
# expression of its own.
EXPR='$'"{{ github.ref_name }}"

expect "real workflow" 0 "$WORKFLOW"

# 1. An expression inside a multi-line run block.
awk -v expr="$EXPR" '
	/gh release edit "\$TAG" --draft=false/ { sub(/"\$TAG"/, "\"" expr "\"") }
	{ print }
' "$WORKFLOW" >"$WORK/block.yml"
grep -qF "$EXPR\" --draft=false" "$WORK/block.yml"
expect "expression in a run block" 1 "$WORK/block.yml"

# 2. An expression on a one-line run.
cp "$WORKFLOW" "$WORK/oneline.yml"
printf '      - name: One line\n        run: echo "%s"\n' "$EXPR" >>"$WORK/oneline.yml"
expect "expression on a one-line run" 1 "$WORK/oneline.yml"

# 3. An expression in env: stays allowed.
cp "$WORKFLOW" "$WORK/env.yml"
# The literal $X is YAML content for the planted step, not shell here.
# shellcheck disable=SC2016
printf '      - name: Env\n        env:\n          X: %s\n        run: |\n          echo "$X"\n' "$EXPR" >>"$WORK/env.yml"
expect "expression in env is allowed" 0 "$WORK/env.yml"

printf '\n%d passed, %d failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
