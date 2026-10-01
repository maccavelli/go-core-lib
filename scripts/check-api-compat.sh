#!/usr/bin/env bash
# Fail when the working tree's exported API is incompatible with a released
# base (docs/decisions/0004-PLAN-v1-1-0-core-api.md Step 12; 0004-MADR
# Confirmation).
#
# Usage: check-api-compat.sh [BASE]
# BASE is a git revision; it defaults to the newest v1.* tag. The base is
# checked out as a detached worktree in a temporary directory, its export data
# is written with apidiff, and the working tree is compared against it.
#
# apidiff exits 0 even when it reports incompatible changes, so the gate is
# its output: any line under -incompatible fails the check. Exit 0 when
# compatible, 1 on an incompatible change, 2 on a usage or environment error.
set -euo pipefail

APIDIFF="${APIDIFF:-golang.org/x/exp/cmd/apidiff@v0.0.0-20260908205506-85c1c2202aba}"
MODULE="github.com/maccavelli/go-core-lib"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [ $# -gt 1 ]; then
	echo "usage: check-api-compat.sh [BASE]" >&2
	exit 2
fi
BASE="${1:-}"
if [ -z "$BASE" ]; then
	if ! BASE="$(git describe --tags --abbrev=0 --match 'v1.*')"; then
		echo "check-api-compat: no v1.* tag to compare against; pass BASE" >&2
		exit 2
	fi
fi
if ! git rev-parse --verify --quiet "${BASE}^{commit}" >/dev/null; then
	echo "check-api-compat: $BASE is not a commit" >&2
	exit 2
fi

WORK="$(mktemp -d)"
cleanup() {
	git -C "$ROOT" worktree remove --force "$WORK/base" >/dev/null 2>&1 || true
	rm -rf "$WORK"
}
trap cleanup EXIT

# A tool failure exits 2, so it is never mistaken for an incompatible change.
if ! git worktree add --detach --quiet "$WORK/base" "$BASE"; then
	echo "check-api-compat: cannot check out $BASE" >&2
	exit 2
fi
if ! (cd "$WORK/base" && go run "$APIDIFF" -m -w "$WORK/base.api" "$MODULE"); then
	echo "check-api-compat: apidiff could not write the export data of $BASE" >&2
	exit 2
fi
if ! report="$(go run "$APIDIFF" -m -incompatible "$WORK/base.api" "$MODULE")"; then
	echo "check-api-compat: apidiff could not compare the working tree" >&2
	exit 2
fi

if [ -n "$report" ]; then
	echo "check-api-compat: incompatible with $BASE:" >&2
	printf '%s\n' "$report" >&2
	exit 1
fi
echo "check-api-compat: compatible with $BASE"
