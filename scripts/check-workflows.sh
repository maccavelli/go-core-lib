#!/usr/bin/env bash
# Check GitHub Actions workflows by parsing them as YAML, the way GitHub does,
# rather than scanning lines (docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md
# R7, R8). Two rules:
#
#   expressions  no step's run script contains ${{ }}. GitHub substitutes an
#                expression into the script text before the shell runs it, so
#                an attacker-chosen value such as a tag name becomes shell
#                code. Values reach run scripts through env: instead
#                (docs/decisions/0003-MADR-remediate-debugging-pass-findings.md D8).
#
#   gh-repo      a step whose run script calls a repository-scoped gh command,
#                or runs refuse-existing-release.sh (which calls gh), has
#                GH_REPO in the step's, the job's or the workflow's env. The
#                reusable release job never checks the caller out, so gh has no
#                local repository to infer from, and without GH_REPO it fails or
#                silently checks nothing (0003-MADR D4). A command that is
#                itself a --help probe resolves no repository and is exempt.
#
# Usage: check-workflows.sh [--rule expressions|gh-repo|all] [workflow...]
# With no workflow, the reusable release workflow is checked. Exit 0 when
# clean, 1 on findings, 2 on a usage, parse or environment error. PyYAML is
# required: pip install -r scripts/requirements-workflow-check.txt
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RULE=all
if [ "${1:-}" = "--rule" ]; then
	RULE="${2:-}"
	shift 2 || true
fi
case "$RULE" in
expressions | gh-repo | all) ;;
*)
	echo "usage: check-workflows.sh [--rule expressions|gh-repo|all] [workflow...]" >&2
	exit 2
	;;
esac
if [ $# -eq 0 ]; then
	set -- "$ROOT/.github/workflows/publish-selfupdate-release.yml"
fi

python3 - "$RULE" "$@" <<'PY'
import re
import sys

try:
    import yaml
except ImportError:
    print("check-workflows: PyYAML is required: "
          "pip install -r scripts/requirements-workflow-check.txt", file=sys.stderr)
    sys.exit(2)


class UniqueKeyLoader(yaml.SafeLoader):
    """A SafeLoader that refuses duplicate mapping keys, which PyYAML would
    otherwise resolve silently to the last value."""


def construct_mapping(loader, node, deep=False):
    seen = set()
    for key_node, _ in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if key in seen:
            raise yaml.constructor.ConstructorError(
                None, None, "duplicate key %r" % (key,), key_node.start_mark)
        seen.add(key)
    return yaml.SafeLoader.construct_mapping(loader, node, deep)


UniqueKeyLoader.add_constructor(
    yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, construct_mapping)

# Repository-scoped gh command groups.
GH_CALL = re.compile(
    r"(?:^|[\s;&|(`])gh\s+(?:release|api|repo|run|workflow|pr|issue|attestation)\b")
REFUSE = re.compile(r"refuse-existing-release\.sh\b")
SEGMENT = re.compile(r"\n|;|&&|\|\||\|")
COMMENT = re.compile(r"(?:^|\s)#.*$")

rule = sys.argv[1]
paths = sys.argv[2:]


def commands(script):
    """Split a shell script into command segments, comments removed."""
    script = script.replace("\\\n", " ")
    lines = [COMMENT.sub("", line) for line in script.splitlines()]
    return [seg for seg in SEGMENT.split("\n".join(lines)) if seg.strip()]


def calls_gh(script):
    for seg in commands(script):
        if not (GH_CALL.search(seg) or REFUSE.search(seg)):
            continue
        if re.search(r"(?:^|\s)--help(?:\s|$)", seg):
            continue
        return seg.strip()
    return None


def has_repo(*envs):
    return any(isinstance(env, dict) and env.get("GH_REPO") not in (None, "")
               for env in envs)


findings = 0
for path in paths:
    try:
        with open(path, encoding="utf-8") as f:
            doc = yaml.load(f, Loader=UniqueKeyLoader)
    except (OSError, yaml.YAMLError) as e:
        print("check-workflows: %s: %s" % (path, e), file=sys.stderr)
        sys.exit(2)
    if not isinstance(doc, dict):
        print("check-workflows: %s: not a workflow mapping" % path, file=sys.stderr)
        sys.exit(2)
    jobs = doc.get("jobs") or {}
    for job_id, job in jobs.items():
        if not isinstance(job, dict):
            continue
        for i, step in enumerate(job.get("steps") or []):
            if not isinstance(step, dict):
                continue
            where = "%s: jobs.%s.steps[%d]" % (path, job_id, i)
            if step.get("name"):
                where += " (%s)" % step["name"]
            script = step.get("run")
            if not isinstance(script, str):
                continue
            if rule in ("expressions", "all") and "${{" in script:
                print("%s: ${{ }} inside a run script (use env:)" % where, file=sys.stderr)
                findings += 1
            if rule in ("gh-repo", "all"):
                cmd = calls_gh(script)
                if cmd and not has_repo(step.get("env"), job.get("env"), doc.get("env")):
                    print("%s: gh call without GH_REPO: %s" % (where, cmd), file=sys.stderr)
                    findings += 1

if findings:
    sys.exit(1)
print("check-workflows: ok (%s) — %d workflow(s)" % (rule, len(paths)))
PY
