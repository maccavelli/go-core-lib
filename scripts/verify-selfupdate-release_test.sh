#!/bin/sh
# Offline fixtures for scripts/verify-selfupdate-release.sh. Performs no publication.
set -eu

ROOT=$(cd -- "$(dirname "$0")/.." && pwd)
SCRIPT="$ROOT/scripts/verify-selfupdate-release.sh"
WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

pass() {
	echo "ok - $1"
}

fail() {
	echo "not ok - $1" >&2
	exit 1
}

PRODUCTS='["demo"]'
PLATFORMS='[{"os":"linux","arch":"amd64"},{"os":"windows","arch":"amd64"}]'
EXTRAS='["install.sh"]'

digest_of() {
	python3 -c 'import hashlib,sys; print(hashlib.sha256(open(sys.argv[1],"rb").read()).hexdigest())' "$1"
}

make_valid() {
	d="$1"
	mkdir -p "$d"
	printf 'linux-body' >"$d/demo-linux-amd64"
	printf 'win-body' >"$d/demo-windows-amd64.exe"
	printf 'installer' >"$d/install.sh"
	linux=$(digest_of "$d/demo-linux-amd64")
	win=$(digest_of "$d/demo-windows-amd64.exe")
	printf '%s  demo-linux-amd64\n%s  demo-windows-amd64.exe\n' "$linux" "$win" >"$d/SHA256SUMS"
}

run_ok() {
	label="$1"
	shift
	if "$SCRIPT" "$@"; then
		pass "$label"
	else
		fail "$label"
	fi
}

run_fail() {
	label="$1"
	shift
	if "$SCRIPT" "$@" >/dev/null 2>&1; then
		fail "$label (expected failure)"
	else
		pass "$label"
	fi
}

# A usage error is exit 2, distinct from a validation failure (exit 1).
run_usage() {
	label="$1"
	shift
	rc=0
	"$SCRIPT" "$@" >/dev/null 2>&1 || rc=$?
	if [ "$rc" -eq 2 ]; then
		pass "$label"
	else
		fail "$label (expected usage exit 2, got $rc)"
	fi
}

VALID="$WORKDIR/valid"
make_valid "$VALID"
run_ok "valid artifact" \
	--dir "$VALID" --products "$PRODUCTS" --platforms "$PLATFORMS" --extras "$EXTRAS"

MISSING="$WORKDIR/missing"
cp -R "$VALID" "$MISSING"
rm -f "$MISSING/demo-linux-amd64"
run_fail "missing binary" \
	--dir "$MISSING" --products "$PRODUCTS" --platforms "$PLATFORMS" --extras "$EXTRAS"

DUP="$WORKDIR/dup"
cp -R "$VALID" "$DUP"
linux=$(digest_of "$DUP/demo-linux-amd64")
printf '%s  demo-linux-amd64\n%s  demo-linux-amd64\n' "$linux" "$linux" >"$DUP/SHA256SUMS"
run_fail "duplicate checksum entry" \
	--dir "$DUP" --products "$PRODUCTS" --platforms "$PLATFORMS" --extras "$EXTRAS"

EXTRA="$WORKDIR/extra"
cp -R "$VALID" "$EXTRA"
printf 'nope' >"$EXTRA/unexpected"
run_fail "undeclared extra file" \
	--dir "$EXTRA" --products "$PRODUCTS" --platforms "$PLATFORMS" --extras "$EXTRAS"

MALFORMED="$WORKDIR/malformed"
cp -R "$VALID" "$MALFORMED"
printf 'not-a-digest  demo-linux-amd64\n' >"$MALFORMED/SHA256SUMS"
run_fail "malformed SHA256SUMS" \
	--dir "$MALFORMED" --products "$PRODUCTS" --platforms "$PLATFORMS" --extras "$EXTRAS"

run_ok "strict tag accepted" \
	--dir "$VALID" --products "$PRODUCTS" --platforms "$PLATFORMS" --extras "$EXTRAS" \
	--tag v1.2.3

run_fail "non-strict tag rejected" \
	--dir "$VALID" --products "$PRODUCTS" --platforms "$PLATFORMS" --extras "$EXTRAS" \
	--tag v1.2

# The v0.16.0 compatibility bridge was not carried over from mcplib
# (docs/decisions/0002-MADR-rehome-selfupdate-from-mcplib.md §3).
run_usage "--bridge is not an option" \
	--dir "$VALID" --products "$PRODUCTS" --platforms "$PLATFORMS" --extras "$EXTRAS" \
	--bridge true

run_usage "--repository is not an option" \
	--dir "$VALID" --products "$PRODUCTS" --platforms "$PLATFORMS" --extras "$EXTRAS" \
	--repository maccavelli/magic-cli-remote

echo "verify-selfupdate-release_test: all fixtures passed"
