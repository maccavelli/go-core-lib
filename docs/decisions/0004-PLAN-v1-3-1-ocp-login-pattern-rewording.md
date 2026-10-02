---
status: in-progress
date: 2026-10-02
associated-madr: "0004-MADR-evolve-selfupdate-api-and-tui-support.md"
---
# Re-word ocp-login to patterns (`v1.3.1`)

Associated MADR: [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md),
amendment P1.

## Goal

The MADR and two of its plans describe ocp-login, an org-internal program,
at file level in this public repository. Re-word every such passage to
patterns under amendment P1's rule, and release the result as `v1.3.1`.

Done means:

* no tracked file names an ocp-login source path, line number, unexported
  identifier or environment variable, or says how one of its defects works;
* every cross-reference to O1–O6 still reads;
* the owner can push and tag `v1.3.1`, whose code is identical to `v1.3.0`.
  *(Corrected 2026-10-02, deviation D1: the Go code, `go.mod` and
  `go.sum` are identical; `v1.3.1` also carries the govulncheck v1.8.0
  pin.)*

This PLAN does not quote the passages it replaces, because quoting them
would repeat the disclosure. It names them by file and section.

## Scope

### In scope

| File | Passage | Change |
| :--- | :--- | :--- |
| `docs/decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md` | "Method", the ocp-login bullet | name the three parts it read by role (its updater, its update command, its step runner), with no paths |
| same | §4 "Defects in the programs that use it", rows O1–O6 | keep each ID and a one-line finding; the evidence column says "Verified in ocp-login's source" or "Reasoning only", with no path or line; O4 as P1-Q1 decides |
| same | §5, the "Inline Bubble Tea front end" row | drop the file name |
| same | §4 Phase 2, the opening paragraph | describe the hand-built bridge (a bounded channel and a self-reissuing command) with no path |
| same | `## Amendments`, P1 | set it `accepted` and record the answers |
| `docs/decisions/0004-PLAN-v1-2-0-interaction-stream.md` | "Facts for that work", the ocp-login bullets | Bubble Tea v2, and "its reusable pieces are unexported", with no package path |
| same | "The precedent" | the same pattern, with no paths or line ranges; O3 and O6 named by ID |
| `docs/README.md` | Records table | this PLAN's row; the MADR row notes P1 |

Each re-worded passage is marked *(Re-worded to patterns by amendment P1.)*

`0004-PLAN-v1-1-0-core-api.md` needs no change. Amendment A6 names
GitLab's documented authentication header as an example of a header scheme,
which is public GitLab API documentation, not ocp-login detail.

### Out of scope

* **Rewriting history, or moving any tag.** Amendment P1 rejects both.
* **The module proxy's copies** of `v1.0.1` to `v1.3.0`, which cannot be
  changed.
* **Fixing O4 in ocp-login.** That belongs to ocp-login's own records.
* **Any change to Go code, CI or scripts.** `v1.3.1` is docs only.
  *(Corrected 2026-10-02, deviation D1: this PLAN changes docs only, and
  the release also carries the tooling pin already on `main`.)*
* `git push` and tags, which are the owner's.

## Rules

1. **Records first.** Nothing is edited until the owner accepts amendment
   P1 and this PLAN, and answers P1-Q1 and P1-Q2.
2. **One commit,** with `git commit --no-edit`, after the owner authorizes
   commits to `main` in that turn, and only after every check in Step 3
   passes.
3. **The checker is proven first.** It must report every passage in scope
   on the unedited tree before it is trusted to report none afterwards.
4. **The checker is not committed.** It lists the very identifiers it looks
   for, so it lives in the session's scratch space.

## Implementation Steps

### Step 1: records

The owner accepts amendment P1 and this PLAN, and answers P1-Q1 and P1-Q2.
Record the answers in P1, set P1 `accepted`, and set this PLAN
`in-progress`.

### Step 2: re-word

Edit each passage in the scope table with a script that asserts each old
passage occurs exactly once before replacing it, and fails otherwise.
Wording follows amendment P1's rule. Keep every sentence that states a
fact about `selfupdate`, and every defect ID.

### Step 3: checks

1. **The marker scan.** A scratch script searches every tracked file at
   HEAD for ocp-login markers:
   * its source paths and file names, as found by the read-only scan of
     2026-10-02;
   * line-number citations next to an ocp-login reference;
   * its environment variables and its module host.

   It must have found every scoped passage on the unedited tree (rule 3),
   and must find none after Step 2.
2. **Cross-references.** Every mention of O1 to O6 in the tree still has a
   row to point to.
3. ~~**Code unchanged.** `git diff v1.3.0 -- . ':!docs'` is empty.~~
   **Code unchanged** *(deviation D1)*: `git diff v1.3.0 -- '*.go' go.mod
   go.sum` is empty, and the rest of the non-docs diff is exactly the
   change of `ef05dfe`.
4. **Documents.** markdownlint-cli2 is clean on the changed files, and a
   link resolver finds no broken relative link or anchor in them.
5. **Identifiers.** The disclosure guard, run over the outgoing commit as
   AGENTS.md describes, passes.

### Step 4: close-out

* Write the release notes for `v1.3.1` in the execution record: a docs-only
  release that re-words ocp-login to patterns, with code identical to
  `v1.3.0`.
  *(Corrected 2026-10-02, deviation D1: the Go code, `go.mod` and
  `go.sum` are identical; `v1.3.1` also carries the govulncheck v1.8.0
  pin.)*
* Update `docs/README.md`, and mark this PLAN `complete` once CI is green on
  the pushed tree.
* The owner pushes and tags `v1.3.1`. CI runs on tags (`ci.yml`), so the tag
  gets its own green run.

## Verification

* The marker scan reports every scoped passage before Step 2, and none
  after it.
* Each O1–O6 reference resolves to its row.
* ~~`git diff v1.3.0 -- . ':!docs'` is empty.~~ No change to Go code,
  `go.mod` or `go.sum` since `v1.3.0`, and the rest of the non-docs diff
  is exactly `ef05dfe` (deviation D1).
* markdownlint, the link resolver and the disclosure guard pass.
* After the push, CI is green on the commit and on the `v1.3.1` tag.
* `go list -m github.com/maccavelli/go-core-lib@latest` reports `v1.3.1`
  once the proxy has seen the tag.

## Rollout and Rollback

* **Rollout.** The owner pushes the one commit and tags `v1.3.1`. Consumers
  need no change. Code is identical, and the reusable workflow is pinned by
  commit hash, not by tag.
  *(Corrected 2026-10-02, deviation D1: the Go code, `go.mod` and
  `go.sum` are identical; `v1.3.1` also carries the govulncheck v1.8.0
  pin.)*
* **Rollback.** None is needed for a docs-only release. If the wording is
  wrong, a later commit corrects it. History is not rewritten.

## Execution Record

### Step 1: records (2026-10-02)

* **Approval.** The owner answered: "questions: follow recommendations,
  proceed." P1-Q1: O4 keeps its category only. P1-Q2: `v1.3.1` is tagged.
* P1 is `accepted`, and this PLAN is `in-progress`.
* **Wording change before commit.** The recommended O4 text said "reported
  to ocp-login's maintainers". No report is recorded, so it reads "for
  ocp-login's maintainers to fix". P1 notes the change.

### Step 2: re-word (2026-10-02)

* A script asserted that each old passage occurred exactly once before it
  replaced it.
* **The MADR, five passages:**
  * the Method bullet;
  * the O1–O6 rows, each with its ID and a one-line finding, and with
    evidence "Verified in ocp-login's source" or "Read in ocp-login's
    source";
  * the §5 row;
  * the opening of Phase 2;
  * P1's status.
* **`0004-PLAN-v1-2-0-interaction-stream.md`:** the "Facts for that work"
  bullets and "The precedent".
* Each passage is marked.
* **Committed.** The owner committed and pushed Steps 1 and 2, with the O4
  wording change, as `cb7b206`. Deviation D1 and this record follow in a
  second commit.

### Step 3: checks (2026-10-02)

| Check | Result |
| :--- | :--- |
| Marker scan, unedited HEAD | 26 markers, every one in a scoped passage, so rule 3 holds |
| Marker scan, after Step 2 | 0 |
| O1–O6 cross-references | six rows, and every mention points to one |
| Go code since `v1.3.0` | `git diff v1.3.0 -- '*.go' go.mod go.sum`: 0 lines |
| Rest of the non-docs diff | `ci.yml`, `Makefile`, `scripts/go-precheck.sh`, one line each: the `@v1.8.0` pin of `ef05dfe` |
| markdownlint-cli2, on the four changed files | 0 issues |
| Link resolver | no broken relative link or anchor |
| Disclosure guard | passes, run on the change committed in a scratch clone; a planted home path and hostname made it exit 1 |

**Deviation D1 (2026-10-02).**

* **Found.** Check 3, as written, failed: `git diff v1.3.0 -- . ':!docs'`
  lists three files. They carry the govulncheck v1.8.0 pin
  ([0007-PLAN-adopt-govulncheck-v1-8.md](0007-PLAN-adopt-govulncheck-v1-8.md)),
  which landed after `v1.3.0` was tagged. The PLAN and P1 had said the code
  was identical.
* **Decision.** The owner chose to correct the claim ("option 1"). The
  alternative was to cut `v1.3.1` from a branch off `v1.3.0`.
* **Changed.**
  * The check is now: no change to Go code, `go.mod` or `go.sum`, and the
    rest of the non-docs diff is exactly `ef05dfe`.
  * P1's Release bullet, and this PLAN's Goal, Scope, Step 4, Verification
    and Rollout, are annotated.
  * No file was added to the scope.

### Release notes for `v1.3.1`

* **Documentation.**
  * `0004-MADR-evolve-selfupdate-api-and-tui-support.md` and
    `0004-PLAN-v1-2-0-interaction-stream.md` now describe ocp-login by
    pattern, not by file (amendment P1).
  * No decision changes.
* **Tooling.** CI, `make vuln` and the pre-add check's install hint pin
  govulncheck v1.8.0 (`0007-MADR-adopt-govulncheck-v1-8.md`).
* **Library.** No change. The Go code, `go.mod` and `go.sum` are those of
  `v1.3.0`.
* **Not changed.** Earlier versions keep their text, both in history and in
  the module proxy. Amendment P1 says why.

### Close-out

Waiting for the owner's push and the `v1.3.1` tag, and for CI on both. This
PLAN becomes `complete` then.
