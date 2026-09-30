#!/usr/bin/env bash
# Assert every step of the reusable release workflow that calls gh against a
# repository also sets GH_REPO.
#
# The job never checks the calling repository out at top level, so gh has no
# local repository to infer from. A step that omits GH_REPO either fails
# outright or, worse, degrades silently — that is how a guard shipped which
# reported success while checking nothing
# (mcplib docs/0007-MADR-restore-repository-context-in-the-reusable-release-workflow.md).
#
# --help probes are exempt: they resolve no repository.
#
# Rules (docs/decisions/0003-MADR-remediate-debugging-pass-findings.md D4):
#   - a comment neither calls gh nor sets GH_REPO, so a commented-out
#     "# GH_REPO:" does not count;
#   - every step starts fresh, named or not ("      - "), so an unnamed step
#     never inherits the previous step's GH_REPO;
#   - running the tools' refuse-existing-release.sh counts as a gh call: it
#     queries the calling repository through gh.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORKFLOW="${1:-$ROOT/.github/workflows/publish-selfupdate-release.yml}"

[ -f "$WORKFLOW" ] || {
	echo "check-workflow-gh-repo: no such workflow: $WORKFLOW" >&2
	exit 1
}

missing=$(awk '
	/^ *#/           { next }                            # comments count for nothing
	/^      - / {                                        # any step starts fresh
		step = $0; has_repo = 0
		if ($0 ~ /^      - name:/) next                  # a step NAME can contain
		                                                 # "gh release"; never
		                                                 # treat it as a command
	}
	/GH_REPO:/       { has_repo = 1 }
	/gh (release|api|repo) / || /^ +\.core-lib-release-tools\/scripts\/refuse-existing-release\.sh/ {
		if ($0 ~ /--help/) next
		if (!has_repo) printf "  %s\n      %s\n", step, $0
	}
' "$WORKFLOW")

if [ -n "$missing" ]; then
	echo "check-workflow-gh-repo: gh steps missing GH_REPO:" >&2
	echo "$missing" >&2
	exit 1
fi

echo "check-workflow-gh-repo: ok — every repository-scoped gh step sets GH_REPO"
