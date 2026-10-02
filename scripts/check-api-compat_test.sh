#!/usr/bin/env bash
# Tests for check-api-compat.sh, run on a scratch clone so the working tree
# is never changed (docs/decisions/0004-PLAN-v1-1-0-core-api.md Step 12).
#
# The clone holds the committed HEAD and the tags, plus the working tree's
# copy of the gate, so the gate under test is the one being changed. The
# planted removal is the case that matters: apidiff exits 0 even when it
# reports an incompatible change, so a gate that read its exit status would
# pass it.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

PASS=0
FAIL=0

check() { # name want got
	if [ "$2" = "$3" ]; then
		echo "  ok   $1"
		PASS=$((PASS + 1))
	else
		echo "  FAIL $1: want exit $2, got $3"
		FAIL=$((FAIL + 1))
	fi
}

CLONE="$WORK/clone"
git clone --quiet "$ROOT" "$CLONE"
cp "$ROOT/scripts/check-api-compat.sh" "$CLONE/scripts/check-api-compat.sh"
BASE="$(git -C "$CLONE" describe --tags --abbrev=0 --match 'v1.*')"

run_gate() { # [BASE]
	set +e
	"$CLONE/scripts/check-api-compat.sh" "$@" >"$WORK/out" 2>&1
	rc=$?
	set -e
	echo "$rc"
}

# 1. The committed tree is compatible with its own newest release.
check "unchanged tree is compatible" 0 "$(run_gate "$BASE")"

# 2. An added exported identifier is a compatible change.
printf 'package selfupdate\n\n// PlantedAddition is a test fixture.\nfunc PlantedAddition() {}\n' \
	>"$CLONE/selfupdate/zz_planted_addition.go"
check "an addition is compatible" 0 "$(run_gate "$BASE")"
rm "$CLONE/selfupdate/zz_planted_addition.go"

# 3. THE CASE THAT MATTERS. NewStrictVersionPolicy, released in v1.0.0, is
#    renamed to an unexported name in package selfupdate, so the module
#    still builds but the identifier is gone. That holds only while no
#    non-test code outside selfupdate/ uses it: such a use would stop the
#    renamed module from building, and the gate would fail to load it
#    rather than report the removal. The precondition says so by name
#    (docs/decisions/0004-PLAN-v1-4-0-command-surface.md, deviation D3).
PLANTED=NewStrictVersionPolicy
users="$(grep -rlw --include='*.go' --exclude='*_test.go' "$PLANTED" "$CLONE" |
	grep -v "^$CLONE/selfupdate/[^/]*\.go$" || true)"
if [ -n "$users" ]; then
	echo "  FAIL $PLANTED is used outside package selfupdate, so it cannot be the planted removal; choose another:"
	echo "       ${users//"$CLONE"\//}"
	FAIL=$((FAIL + 1))
else
	find "$CLONE/selfupdate" -maxdepth 1 -name '*.go' -exec perl -pi -e "s/\\b$PLANTED\\b/newStrictVersionPolicy/g" {} +
	check "a removed identifier is refused" 1 "$(run_gate "$BASE")"
	if grep -q "$PLANTED: removed" "$WORK/out"; then
		echo "  ok   the report names the removal"
		PASS=$((PASS + 1))
	else
		echo "  FAIL the report does not name the removal:"
		sed 's/^/       /' "$WORK/out"
		FAIL=$((FAIL + 1))
	fi
fi
git -C "$CLONE" checkout --quiet -- .
cp "$ROOT/scripts/check-api-compat.sh" "$CLONE/scripts/check-api-compat.sh"

# 4. A base that is not a commit is a usage error, not a pass.
check "an unknown base is an error" 2 "$(run_gate no-such-revision)"

# 5. Too many arguments.
check "two arguments are an error" 2 "$(run_gate "$BASE" "$BASE")"

echo "check-api-compat_test: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ]
