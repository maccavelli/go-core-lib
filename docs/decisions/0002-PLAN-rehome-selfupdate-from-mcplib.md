---
status: in-progress
date: 2026-09-29
associated-madr: "0002-MADR-rehome-selfupdate-from-mcplib.md"
---
# Implement the re-home of `selfupdate` and its release tooling into go-core-lib

Associated MADR: [0002-MADR-rehome-selfupdate-from-mcplib.md](0002-MADR-rehome-selfupdate-from-mcplib.md)

## Goal

`github.com/maccavelli/go-core-lib/selfupdate` is `mcplib` `v1.6.0`'s
package, changed only as MADR §3 allows. The reusable release workflow and
its scripts run from this repository. Lint covers linux, darwin and windows.
The result, together with the 0001 scaffold, is pushed and tagged `v1.0.0`,
with CI green on all three operating systems.

## Scope

### In scope (this repository only)

| Path | Phase |
|---|---|
| `selfupdate/**` (49 files) | 1, 2 |
| `go.mod`, `go.sum` | 1 |
| `Makefile` (`lint` target), `scripts/go-precheck.sh` (step 2), `AGENTS.md` ("Pre-add checks") | 3 |
| `scripts/verify-selfupdate-release.sh`, `…_test.sh`, `scripts/refuse-existing-release.sh`, `…_test.sh`, `scripts/check-workflow-gh-repo.sh` | 4 |
| `.github/workflows/publish-selfupdate-release.yml` | 4 |
| `.github/workflows/ci.yml` | 3, 4 |
| `README.md`, `docs/README.md`, `docs/architecture.md`, `docs/guides/migrating-from-mcplib-selfupdate.md` | 0, 5 |
| `docs/decisions/0002-*` | all |

### Out of scope

* Any change in `mcplib`, `go-llmprovider-sdk` or a consumer (MADR §9).
* Moving any dependency version (MADR §4).
* Any exported API change (MADR §3).
* The records of `mcplib`, and its 0005 open items (MADR §2).

### Fixed inputs

* Source: `mcplib` at tag `v1.6.0`, commit `4e1f9a53e265`. Read it with
  `git -C ../mcplib show v1.6.0:<path>` or `git archive`, never from its
  working tree, which carries someone else's uncommitted edits to its own
  `0015` records.
* Go 1.27.1, `golangci-lint` 2.13.2, `govulncheck` v1.7.0, `shellcheck`,
  `markdownlint-cli2`, `python3`, and `gh` (Phase 6 only).
* Every cross-target command sets `CGO_ENABLED=0`, because this host's
  `go env` sets `CGO_ENABLED=1`.

## Implementation Steps

Every phase ends with its checks passing, `git add` of exactly its paths,
and `git commit --no-edit`. Long outputs go to scratch files with `$?`
captured before any filter.

### Phase 0: accept the records

1. The owner approves. Set the MADR `accepted` and this PLAN `in-progress`.
2. Add the two 0002 rows to the `docs/README.md` record index.
3. Commit the two records and `docs/README.md`.

### Phase 1: copy the package, change only the import path and `doc.go`

1. Extract `selfupdate/` from `mcplib` `v1.6.0` with
   `git -C ../mcplib archive v1.6.0 selfupdate | tar -x -C .`. Assert 49
   files.
2. Rewrite `github.com/maccavelli/mcplib/selfupdate` to
   `github.com/maccavelli/go-core-lib/selfupdate`. Assert exactly one
   occurrence, in `example_test.go`, before and after.
3. Edit the `doc.go` package comment as MADR §3 says: "fleet programs"; the
   workflow "at the exact go-core-lib module-tag commit".
4. `go get golang.org/x/mod@v0.40.0 golang.org/x/sys@v0.47.0 golang.org/x/term@v0.43.0`,
   then `go mod tidy`. Assert that `go.mod` has exactly those three direct
   requirements, and that `go.sum`'s `h1:` lines equal `mcplib` `v1.6.0`'s
   lines for them.
5. Checks:
   * **G-diff.** `diff -r <v1.6.0 archive>/selfupdate selfupdate` shows only
     the `example_test.go` import line and the `doc.go` comment.
   * **G-api.** `go doc -all ./selfupdate` here and in a `v1.6.0` archive,
     each written to a scratch file. Their `diff` shows only the package
     comment.
   * `go test -race -count=1 -coverprofile=… ./...` passes. Record the
     coverage against 75.5 %.
   * Host `make lint`, `make vuln`, `go mod tidy -diff` and
     `make pre-add-check`.
   * `CGO_ENABLED=0 GOOS=<t> go test -c -o /dev/null ./selfupdate` for
     linux, darwin and windows.
6. Commit. The staged `selfupdate/` is the verbatim copy plus the two
   changes, so the commit itself is the provenance record.

### Phase 2: the four Windows lint fixes

1. Make exactly MADR §3's four edits: `root.ReadFile(name)`;
   `windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)`;
   `osRename` moved to `replace_unix.go`; `applyResult.pendingBackup`
   deleted.
2. Checks:
   * `CGO_ENABLED=0 GOOS=<t> golangci-lint run -c .golangci.yml ./...`
     gives `0 issues.` for linux, darwin and windows. Before this phase,
     `GOOS=windows` reports the four findings. Record both outputs.
   * `CGO_ENABLED=0 GOOS=<t> go vet ./...` and `go test -c` for the three
     targets.
   * `go test -race -count=1 ./...` passes, with coverage recorded.
   * **G-api** is re-run; it is unchanged. **G-diff** now also shows exactly
     the four edits.
3. Commit.

### Phase 3: cross-target lint in `make lint`, the pre-add gate and CI

1. `Makefile` `lint`: loop over `linux darwin windows`, running
   `CGO_ENABLED=0 GOOS=$$os $(GOLANGCI_LINT) run -c $(FLEET_LINT_CFG) ./...`.
   Fail if any run fails, and name the failing target.
2. `scripts/go-precheck.sh` step 2: the same loop, each run's output under
   `golangci-lint (GOOS=<t>):`. The comments say why: MADR §5.
3. `ci.yml`: no change is needed for lint, since it already runs
   `make lint`. Assert that by reading the workflow.
4. Update `AGENTS.md` "Pre-add checks" to say the lint step runs for three
   targets, and remove its "until the first package lands" paragraph.
5. `shellcheck scripts/go-precheck.sh` exits 0.
6. **First-fail experiment,** on a scratch clone at this phase's working
   state:
   * with Phase 2 reverted in the clone (`git revert --no-edit`, scratch
     only), `make lint` and `make pre-add-check` fail, naming
     `GOOS=windows` and all four findings;
   * with each fix reverted alone, that finding alone is reported;
   * on the fixed clone, both pass.

   Record each exit code and its failure line. Time `make lint` before and
   after the loop (MADR "Consequences").
7. Commit.

### Phase 4: the release tooling

1. Extract the five scripts and the workflow from `v1.6.0` the same way,
   keeping the scripts' `0755` modes.
2. Workflow: `.mcplib-release-tools` becomes `.core-lib-release-tools`
   (assert 5 occurrences before and 0 after). Remove the `bridge-release`
   input, its `BRIDGE` env line and the `--bridge` argument. Make citations
   repository-named.
3. `verify-selfupdate-release.sh`: remove `--bridge` from the usage, the
   argument parser and the Python checks (the compatibility set,
   `SHA256SUMS-0.16.0` and the alias comparison). Make citations
   repository-named.
4. `verify-selfupdate-release_test.sh`: remove the four bridge cases. Add
   one case asserting that `--bridge true` is now a usage error (exit 2).
5. `refuse-existing-release.sh`, its test and `check-workflow-gh-repo.sh`:
   citations only.
6. `ci.yml`: add `./scripts/verify-selfupdate-release_test.sh` on Linux.
   Add `./scripts/refuse-existing-release_test.sh` and
   `./scripts/check-workflow-gh-repo.sh` on every OS, as in `mcplib`'s CI.
7. Checks:
   * the three tests pass; `shellcheck scripts/*.sh` exits 0; both YAML
     files parse with PyYAML;
   * **G-diff for tooling:** `diff` of each file against `v1.6.0` shows only
     steps 2–5;
   * `git grep -n -i 'mcplib' -- scripts .github selfupdate` shows only the
     repository-named citations;
   * **first-fail**, on scratch copies: a verifier that no longer rejects a
     missing `SHA256SUMS` fails its test; a workflow with `GH_REPO` removed
     from one `gh release` step fails `check-workflow-gh-repo.sh`; a `gh`
     stub that returns an unrecognised error fails the refuse test's
     fail-closed case.
8. Commit.

### Phase 5: documentation

1. `README.md`: status (the first package, releases from `v1.0.0`), a
   `selfupdate` section carrying the substance of `mcplib`'s README
   "Self-update" section, the reusable-workflow contract, and "I want to…"
   rows.
2. `docs/architecture.md`: the package, the tooling, the three-target lint
   and the new CI steps, with "What is not here" updated.
3. `docs/guides/migrating-from-mcplib-selfupdate.md`, per MADR §7. Its
   `uses:` SHA is a placeholder that names the `v1.0.0` commit and is filled
   in Phase 6.
4. `docs/README.md`: "I want to…" rows for the guide and the MADR.
5. Checks: `markdownlint-cli2` clean over non-record Markdown; the proven
   link resolver over every non-record Markdown file; identifier scan over
   the tree.
6. Commit. **This is the `v1.0.0` commit** (MADR §8).

### Phase 6: push, CI, tag `v1.0.0` (owner-run; each mutation needs the owner's same-turn ask)

1. Run the pre-push disclosure guard from `AGENTS.md` over
   `origin/main..main`. That covers every 0001 and 0002 commit.
2. **On the owner's ask:** `git push origin main`. Watch CI (`gh run watch`)
   until it finishes. It must pass on ubuntu-24.04, macos-15 and
   windows-2025. That is the first run of the Windows tests for the Phase 2
   fixes (MADR "Confirmation"). A failure stops the phase for a decision.
3. **On the owner's ask:** `git tag v1.0.0 <Phase 5 commit>` and
   `git push origin v1.0.0`. The tag's CI must pass.
4. `GOPROXY=https://proxy.golang.org go list -m github.com/maccavelli/go-core-lib@v1.0.0`
   resolves. Record the output.
5. Fill the guide's `uses:` SHA with the `v1.0.0` commit, set this PLAN
   `complete`, and commit. It is pushed on the owner's ask.

## Verification

* **V1. Same code.** G-diff and G-api outputs recorded, showing only MADR
  §3's changes.
* **V2. Module.** Three direct requirements at `mcplib`'s pins, `go.sum`
  `h1:` lines equal to `mcplib`'s, and `go mod tidy -diff` clean.
* **V3. Tests.** `go test -race` passes, with coverage recorded against
  75.5 %. The Windows leg passes in Phase 6.
* **V4. Lint on three targets.** Clean, and proven to fail with each fix
  reverted.
* **V5. Release tooling.** The three tests pass, and each was seen to fail
  on its planted input. `shellcheck` is clean. No `bridge` remains.
* **V6. Docs.** markdownlint and links are clean, and the resolver was
  proven first.
* **V7. Identifiers.** No account name, hostname or real-machine path in
  the tree.
* **V8. Release.** CI is green on three operating systems at the pushed
  `main` and at `v1.0.0`, and the proxy resolves `v1.0.0`.

## Rollout and Rollback

* **Rollout.** Phases 0–5 are local commits. Phase 6 pushes the 0001 and
  0002 commits together (0001 MADR §6) and tags `v1.0.0`, each on the
  owner's explicit ask. Consumers migrate later, under their own records.
* **Rollback before the push.** `git revert` of the phase commit, here only.
* **Rollback after the push, before the tag.** Fix forward with a new
  commit, and re-run CI.
* **After `v1.0.0`.** Never move or delete the tag. Fix in `v1.0.1`.
  Nothing consumes the module until a consumer's own record pins it.

## Execution Record

### Phase 0: accept the records (2026-09-29)

The owner approved: "Proceed, drop bridge release and fix windows
findings". The MADR is `accepted` and this PLAN is `in-progress`. The two
0002 rows are added to `docs/README.md`.

Commit: `49affdd`.

### Phase 1: copy the package (2026-09-29)

* **Extraction.** `git -C ../mcplib archive v1.6.0 selfupdate | tar -x`
  wrote 49 files, and `diff -r` against a separate `v1.6.0` reference
  archive was empty before the edits. There was exactly one
  `github.com/maccavelli/mcplib/selfupdate` occurrence (`example_test.go:10`)
  before the rewrite, and none after.
* **Module.** `go get` of the three pins, then `go mod tidy` (exit 0), gave
  three direct requirements: `golang.org/x/mod v0.40.0`,
  `golang.org/x/sys v0.47.0` and `golang.org/x/term v0.43.0`. The six
  `go.sum` lines are identical to `mcplib` `v1.6.0`'s lines for those
  modules. `go mod tidy -diff`: exit 0.
* **G-diff** against the `v1.6.0` reference (exit 1, as expected) shows
  three changed lines: `doc.go:2` (`mcplib consumers.` → `fleet
  programs.`), `doc.go:16` (`mcplib` → `go-core-lib`), and
  `example_test.go:10` (the import path).
* **G-api.** `go doc -all ./selfupdate`, 612 lines each side. The diff has
  two parts: the package comment's two sentences (as MADR §3 allows), and
  the header line
  `package selfupdate // import "github.com/maccavelli/go-core-lib/selfupdate"`.
  That is the module path MADR §1 decides, not an API change. No
  identifier, signature or other doc comment differs.
* `go test -race -count=1 ./...`:
  `ok github.com/maccavelli/go-core-lib/selfupdate 3.880s coverage: 75.5% of statements`.
  That is the same coverage as `mcplib`'s own run.
* `make lint` (host): `0 issues.` `make vuln`:
  `No vulnerabilities found.` `make pre-add-check`:
  `go-precheck: 47 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck).`
* `CGO_ENABLED=0 GOOS=<t> go test -c -o /dev/null ./selfupdate`: exit 0 for
  linux, darwin and windows.
* **Before Phase 2** (recorded now, on this copy):
  `CGO_ENABLED=0 GOOS=<t> golangci-lint run` exits 0 for linux and darwin.
  For windows it exits 1 with `4 issues:` at `cleanup_windows.go:54:15`
  G304, `cleanup_windows.go:123:16` SA1019, `replace.go:11:2` unused
  `osRename`, and `replace_windows.go:16:2` unused `pendingBackup`.
