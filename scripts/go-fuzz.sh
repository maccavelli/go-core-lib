#!/usr/bin/env bash
# Fuzz every fuzz target in one package, for a fixed time each
# (docs/decisions/0004-PLAN-h2-fuzzing-and-manifest-differential.md Step 4;
# 0004-MADR H2, amendment C2). go test -fuzz takes exactly one fuzz target per
# run, so the targets run one after another.
#
# Usage: go-fuzz.sh [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PKG
#   -t FUZZTIME      time per target, as go test -fuzztime takes it
#                    (default 20s)
#   -z MINIMIZETIME  the cap on minimizing one input, as go test
#                    -fuzzminimizetime takes it (default 5s). Go's own
#                    default, 60s, lets the minimizer pause fuzzing for most
#                    of a short run (deviation D2 of
#                    0004-PLAN-h2-fuzzing-and-manifest-differential.md)
#   -m MIN           the fewest targets accepted (default 4, at least 1): a
#                    loop that finds nothing must not pass
#
# GO names the go command (default go); the tests inject a recorder
# through it.
#
# Exit 0 when every target ran clean; on a fuzz failure, go test's status; 1
# when too few targets are found or they cannot be listed; 2 on a usage
# error. Go leaves a failing input in PKG/testdata/fuzz/<Name>/.
set -euo pipefail

usage() {
	echo "usage: go-fuzz.sh [-t FUZZTIME] [-z MINIMIZETIME] [-m MIN] PKG" >&2
	exit 2
}

GO="${GO:-go}"
fuzztime=20s
minimize=5s
min=4
while getopts ':t:z:m:' opt; do
	case "$opt" in
	t) fuzztime="$OPTARG" ;;
	z) minimize="$OPTARG" ;;
	m) min="$OPTARG" ;;
	*) usage ;;
	esac
done
shift $((OPTIND - 1))
[ $# -eq 1 ] || usage
pkg="$1"
case "$min" in
'' | *[!0-9]* | 0) usage ;;
esac

# The listing's status is taken before anything filters it.
if ! listing="$("$GO" test -list '^Fuzz' "$pkg")"; then
	echo "go-fuzz: cannot list the fuzz targets in $pkg" >&2
	exit 1
fi
targets=()
while IFS= read -r line; do
	case "$line" in
	Fuzz*) targets+=("$line") ;;
	esac
done <<<"$listing"

if [ "${#targets[@]}" -lt "$min" ]; then
	echo "go-fuzz: found ${#targets[@]} fuzz targets in $pkg, want at least $min" >&2
	exit 1
fi

for target in "${targets[@]}"; do
	echo "go-fuzz: $target for $fuzztime (minimizing at most $minimize)"
	rc=0
	"$GO" test -run '^$' -fuzz "^${target}\$" -fuzztime "$fuzztime" -fuzzminimizetime "$minimize" "$pkg" || rc=$?
	if [ "$rc" -ne 0 ]; then
		echo "go-fuzz: $target failed (exit $rc); Go wrote the failing input under $pkg/testdata/fuzz/$target/" >&2
		exit "$rc"
	fi
done
echo "go-fuzz: ${#targets[@]} fuzz targets ran clean in $pkg"
