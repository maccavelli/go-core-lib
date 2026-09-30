---
status: proposed
date: 2026-09-29
associated-madr: "0003-MADR-remediate-debugging-pass-findings.md"
---
# Implement the fixes for every debugging-pass finding

Associated MADR: [0003-MADR-remediate-debugging-pass-findings.md](0003-MADR-remediate-debugging-pass-findings.md)

## Goal

Every finding in the MADR's tables is closed:

* all 41 findings: A1, A3, A5–A8, A10, A11 (A2 is C1, A9 is C9); B1–B8
  and B10; C1–C12 and C14; D1–D9, D11 and D12;
* the four test-gap groups A4, B9, C13 and D10.

Each fix has evidence in the execution record, and `v1.0.0` can then be
tagged under
[0002-PLAN-rehome-selfupdate-from-mcplib.md](0002-PLAN-rehome-selfupdate-from-mcplib.md)
Phase 6.

## Scope

### Finding → phase

| Phase | Area | Findings |
|---|---|---|
| 0 | records | D2, D12, and 0002's amendment |
| 1 | coordinator and confirmer | C1–C14 (C9 = A9), C13 |
| 2 | network and integrity | A1 with A5, A2 (= C1 at source), A3, A6, A7, A8, A10, A11, A4 |
| 3 | install path, portable | B1, B2, B3, B4, B5, B8, B10 (directory identity), B9 |
| 4 | install path, Windows | B6, B7, B10 (Windows retry and `ctx`) |
| 5 | release tooling | D1, D4, D5, D6, D7, D8, D10 |
| 6 | CI and tooling | D9, D11 |
| 7 | docs and close-out | D3, plus README and architecture for behaviour changes |

### Out of scope

* Any exported API change. No exported identifier is added, removed or
  re-typed (MADR "What B means"). New seams are unexported.
* `mcplib` and the consumers.
* The GitHub App check suite noted in D11. That is a GitHub installation
  setting for the owner, not code.
* `git push` and tags. Those stay under 0002-PLAN Phase 6 and need the
  owner's ask.

### Fixed inputs

* **The development host:** macOS, go1.27.1, `golangci-lint` 2.13.2,
  `govulncheck` 1.7.0, `shellcheck`, `markdownlint-cli2` 0.23.2, `python3`.
  Cross-OS commands set `CGO_ENABLED=0`.
* **The Windows test host,** reached over SSH and never named in a record:
  * Windows 10.0.26200 (Windows 11), amd64;
  * go1.27.1 windows/amd64 with `CGO_ENABLED=1`, and MinGW-w64 gcc 16.2,
    so `-race` works;
  * a Git Bash (MSYS2) login shell, PowerShell 7.6;
  * Developer Mode on, so unprivileged symlinks can be created.

  Measured on 2026-09-29, read-only.
* **Using the Windows host.**
  * A `git archive` of the commit under test is extracted into a new
    directory under the host's `%TEMP%`, named `go-core-lib-verify-<short
    sha>`. The directory is removed after the run.
  * Nothing else on the host is written, apart from Go's build and module
    caches.
  * Outputs are copied back into the development host's scratch space.
    Records quote them with the user-profile path replaced by `<user>`.

### Rules for every phase

* **Fail first.** For each behaviour fix, the new or changed test is copied
  into a scratch `git archive` of the phase's starting commit. It must fail
  there, and the failure line is recorded. It then passes in the tree.
* **Test-gap items** are proven by mutation on a scratch copy: remove the
  check the test covers, and the test must fail.
* **Gates before each commit:** `make pre-add-check` (three-target lint),
  `go test -race -count=1 ./...`, `go mod tidy -diff`, `shellcheck
  scripts/*.sh`, and `markdownlint-cli2` on touched Markdown. The last
  commit runs every gate.
* **The Windows gate,** for every phase that touches Go code or scripts
  (1–6): on the Windows host, `go vet ./...`,
  `go test -race -count=1 ./...`, and under its bash
  `./scripts/refuse-existing-release_test.sh` and
  `./scripts/check-workflow-gh-repo.sh`, plus from Phase 5 on the new
  script tests. This runs on the phase's working tree, before the commit.
  The tree is shipped as
  `git ls-files -co --exclude-standard | tar -cf - -T -`, which covers
  tracked and new, non-ignored files and never commits. A Windows fail-first run is also required for each
  test that exists only on Windows.
* **G-api** (normalised `go doc -all` against `mcplib` `v1.6.0`) is re-run
  each phase. It may differ only in doc comments.
* `git commit --no-edit`, one commit per phase. A phase too large to review
  may be split into lettered sub-phases (1a, 1b), each committed, and the
  split is recorded.

## Implementation Steps

### Phase 0: records

1. Owner approval. Set this PLAN `in-progress`, and index it in
   `docs/README.md`.
   * **Windows baseline:** run the Windows gate on `004d13b`, the code as
     pushed, and record which tests ran. That includes the two existing
     `//go:build windows` receipt tests, listed with `go test -v -run
     Windows`. This is the first Windows run outside CI, and every later
     Windows result is compared to it.
2. **D12.**
   * Replace the stale `docs/README.md` row "know why CI and `make lint`
     fail until the first package lands".
   * In 0002-MADR "Consequences", strike "the first commit names it", and
     annotate it with a pointer to the §2 amendment.
3. **D2.** In 0002-MADR, strike and annotate "one deleted line in one
   consumer", "`magic-cli-remote` also deletes its `bridge-release` line",
   and option A's "Bad" line. All six consumers delete `bridge-release`.
4. **0002 amendment.** Add a dated amendment to 0002-MADR: `v1.0.0` =
   `mcplib` `v1.6.0` + §3 + the 0003 fixes, and G-diff's baseline grows
   accordingly. Add a dated entry to 0002-PLAN Phase 6: steps 3–5 wait for
   this PLAN to be `complete`.
5. Commit the records.

### Phase 1: coordinator and confirmer (`updater.go`, `confirmer.go`, `types.go`, `example_test.go`)

Each item lists the change, then its test.

1. **C3.** `New` rejects a nil or typed-nil value in every required seam
   and in each `Verifiers` element, using an unexported `isNil` helper
   (`reflect` for typed nils).
   * `TestNewRejectsNilCollaborators`, rewritten as a table: each seam nil
     in turn, a typed-nil seam, a nil verifier, a typed-nil verifier, and
     zero `Limits`.
2. **C1, coordinator side.** In `fetchRelease`, when `TargetVersion` is set
   and `rel.Tag != TargetVersion`, return an error wrapping `ErrIntegrity`
   that names both tags with `%q`.
   * `TestRunRejectsTagMismatch`.
3. **C4.** Check order becomes PLAN §4.6 step 4's: immutable, then
   draft/prerelease, then `Validate`. Every error that carries a tag uses
   `%q`.
   * `TestRunEscapesUntrustedTag`: the error contains no ESC, BEL or
     newline.
4. **C9.** Parse `SHA256SUMS` and resolve the selected entry right after
   the manifest download, before `CreateStaging`.
   * `TestRunRejectsManifestBeforeStaging`: `CreateStaging` and the binary
     `OpenAsset` are never called.
5. **C8.** Emit `EventVerified` after the verifiers, then
   `EventTransforming`, then transform and revalidate.
   * `TestRunEventOrderWithTransformer`: the exact event sequence. It also
     covers `hashFile`, `sessOwns` and `hashAndValidateStaging`,
     including the size-limit branch.
6. **C2.** When `Install` returns `Applied:false` and a nil error, return
   an error saying the installer reported no commit.
   * `TestRunInstallerNoCommitIsError`: non-nil error, `ExitCode` 1.
7. **C14.** After a committed `Install`, call `sess.Close()` explicitly and
   join its error, *then* report `EventComplete`. The deferred `Close`
   stays as a safety net (it is idempotent, `session.go:126-128`).
   * `TestRunCompleteAfterClose`: ordering.
   * `TestRunCloseErrorJoinedAfterCommit`: `Applied` true, error joined.
8. **C10.** Every error return in `execute` and `apply` goes through
   `wrapRun`, including validation, reporter and post-commit errors.
   * `TestRunErrorsNameProduct`: a table over each return site; the message
     is prefixed `selfupdate: demo:` and `errors.Is` still holds.
9. **C5.** In check mode with `OperationReplaceLocal`, `EventSelected`
   carries `Detail` "local build: apply requires --force".
   * `TestRunLocalBuildCheckHintsForce`.
10. **C12.**
    * The pending-backup `Detail` names the full retained path
      (`filepath.Join(target.Dir, pending)`) and says it is validated
      before the next download.
    * `ExampleNewManagedInstaller` composes a real `NewManagedInstaller`
      with example `Lifecycle` and `Reconciler` types.
    * A new `ExampleExitCode`.
    * `TestRunPendingBackupDetail`; the examples' `// Output:` lines.
11. **C6.** EOF before any input is a decline, `(false, nil)`.
    * `TestTerminalConfirmerEOFDeclines`.
12. **C7.** `terminalConfirmer` starts a single reader goroutine, once, and
    delivers lines through a channel. A cancelled `Confirm` leaves the
    pending line for the next `Confirm` instead of losing it.
    * `TestTerminalConfirmerCancelKeepsLine`: cancel, write `y\n`, and a
      second `Confirm` returns true.
13. **C11.** The `Config.Limits` doc comment states that the Updater uses
    `Manifest` and `Executable`, and that `ReleaseJSON` and `ErrorBody`
    belong to the source's own `Limits`. This is a doc fix; there is no
    test.
14. **C13.**
    * `TestRunCallOrderFailureBoundaries`: a table injecting a failure at
      every collaborator call; no later call happens.
    * `TestRunOperationsMatrix`: rollback, reinstall with `--force`, local
      build with and without `--force`, latest-older-is-error, and
      `--version`, all through `Run`.
    * `TestOverlappingRun` rewritten to use a channel handshake instead of
      `time.Sleep`, and to assert that no collaborator is invoked by the
      rejected `Run`.
15. Gates, then commit.

### Phase 2: network and integrity (`github.go`, `download.go`, `reporter.go`, `updater.go`)

1. **A1 with A5**, in one commit so that validation is never absent.
   * `mapRelease` checks only structure for every asset: `ID > 0`, a
     non-empty basename, no control characters.
   * `execute` validates the *selected* binary against
     `Limits.Executable`, and the manifest against `Limits.Manifest`, with
     `validateAssetMetadata` (state, size, digest syntax). This happens
     before classification and check mode.
   * Tests: `TestGitHubExtraAssetsDoNotPoisonRelease` (zero-size, `open`,
     oversize and `sha512:` extras; `Latest` succeeds), and
     `TestRunValidatesSelectedAssets` (an `open` binary or an oversize
     manifest fails before any check result).
2. **A2.** `ByTag` rejects a response whose `tag_name` differs from the
   request.
   * `TestGitHubByTagRejectsMismatchedTag`.
3. **A3.** `checkRedirect` refuses a redirect whose scheme is not `https`
   unless the host is loopback.
   * `TestGitHubRedirectRequiresHTTPS`.
4. **A6.** After `Decode`, `dec.Token()` must return `io.EOF`.
   * `TestDecodeJSONRejectsTrailingDelimiters`.
5. **A7.** A `Retry-After` in seconds above
   `math.MaxInt64 / int64(time.Second)` is treated as malformed (zero).
   * `TestRateLimitRetryAfterOverflow`.
6. **A8.** In `getRelease`, and in `OpenAsset` where it reads an error
   body, check the status first. For a non-2xx response, read at most
   `ErrorBody` bytes, tolerate truncation, and map the status.
   * `TestGitHubLargeErrorBodyKeepsRateLimit`.
7. **A10.** `validateGitHubName` rejects `.` and `..`.
   * `TestNewGitHubSourceRejectsDotNames`.
8. **A11.** `sanitizeDiagnostic` and `sanitizeText` replace Unicode
   format controls (`unicode.Cf`) and U+2028/U+2029.
   * `TestSanitizeRemovesFormatControls`.
9. **A4.** New `httptest` cases:
   * 403 with `X-RateLimit-Remaining: 0` and no `Retry-After`;
   * `OpenAsset` 404 and 429;
   * an asset in `open` state; oversize and zero-size selected assets;
   * a foreign asset ID;
   * `RateLimitError.Error` text.

   Mutation proof: each of the five checks A4 names, removed on a scratch
   copy, makes a test fail.
10. Gates, then commit.

### Phase 3: install path, portable (`replace*.go`, `session.go`, `managed.go`, `lock_*.go`, `target.go`)

1. **B1.** When the rollback rename fails, `replaceTarget` (unix and
   windows) returns the backup and
   `errors.Join(syncErr, fmt.Errorf("selfupdate: restore backup: %w", rerr))`.
   * `installSession.Install` returns
     `InstallResult{Target, Backup}` with that error.
   * `managedSession.Install` passes the returned `applied` to `recover`,
     so recovery can retry the rollback.
   * Tests: `TestInstallSyncAndRollbackFailureReported`, and
     `TestManagedApplyFailureRetriesRollback` (through `replacePath` and
     `syncDirFn` seams).
2. **B2.** `recover` runs on
   `context.WithTimeout(context.WithoutCancel(ctx), recoveryTimeout)`, with
   an unexported `recoveryTimeout = 2 * time.Minute`.
   * `TestManagedRecoveryIgnoresCancelledContext`: `Start` and
     `WaitHealthy` see a live context.
3. **B3.** `copyFile` chmods the backup to the source's mode before `Sync`.
   * `TestBackupCopyFallbackPreservesMode`: `osLink` is forced to fail; the
     mode is kept after rollback.
4. **B4.** `applyResult` gains an unexported `renamed bool`.
   `Install` deregisters staging only when the rename happened, so `Close`
   removes staging after an earlier failure.
   * `TestInstallFailureBeforeRenameRemovesStagingOnClose`.
5. **B5.** An unexported `openLockFile(root, name)`, shared by
   `lock_unix.go` and `lock_windows.go`:
   * `root.Lstat` first, rejecting anything not regular (a symlink
     included);
   * when absent, create with `O_CREATE|O_EXCL` (which `os.Root` never
     follows);
   * after opening, require `os.SameFile` between the pre-open `Lstat` and
     `f.Stat()`.

   Tests: `TestLockRejectsRelativeSymlink` and
   `TestLockRejectsDanglingRelativeSymlink`. They skip only on Windows, and
   only when symlink creation is not permitted, with the reason logged.
6. **B10, directory identity.** `Begin` records `root.Stat(".")`. Before
   `replaceTarget` and before commit, `Install` requires `os.SameFile`
   with `os.Stat(target.Dir)`, else `ErrConcurrentUpdate`.
   * `TestInstallDetectsSwappedDirectory`.
   * This is the chosen fix for B10's first half. Re-routing every
     operation through `os.Root` is not done, because Windows `MoveFileEx`
     is path-based. The identity check closes the swap the finding
     describes on both platforms.
7. **B8.** `TestManagedRollbackErrorJoined` asserts `errors.Is(err,
   rec.restoreE)` unconditionally. Mutation proof: removing `recov` from
   the join now fails it.
8. **B9.** New tests, with unexported seams added only where a failure has
   none:
   * `TestStagingRejectsPlantedSymlink`;
   * `TestInstallInjectedFailures`: a table of chmod, backup, rename,
     dir-sync and commit failures; the target stays intact or the error is
     reported, as each case requires;
   * `TestInstallPermissionDenied` (unix, a read-only directory);
   * `TestInstallCancelledBeforeAndAfterStaging`;
   * a source body shorter than its declared size, or failing mid-stream,
     leaves no staging file after `Close`;
   * `TestManagedFailureMatrix`: `fakeLife` gains stop, start, installed
     and running error injection; stop, start, restore and restart
     failures are each joined, and recovery calls are counted.

   Windows receipt cases are in Phase 4.
9. Gates, then commit.

### Phase 4: install path, Windows (`cleanup.go`, `cleanup_windows.go`, `replace_windows.go`)

1. **B6.** A portable `validateReceiptBackup(target Target, name string)`
   in `cleanup.go` requires:
   * `filepath.IsLocal(name)`;
   * `filepath.Base(name) == name`;
   * the prefix `"." + target.Base + ".selfupdate-bak-"`;
   * `name != target.Base`.

   `processCleanupReceipt` uses it, with `root.Lstat` and `root.Remove`.
   * `TestValidateReceiptBackup` (portable): a table of the target, the
     lock, an absolute path, traversal, and a good name.
2. **B7.** When the named backup does not exist, remove the receipt, sync,
   and return nil. A digest mismatch, a reparse point or a failed delete
   still fail closed.
   * `TestWindowsCleanupReceiptMissingBackup` (Windows).
3. **B10, Windows.** `moveFileReplace(ctx, …)` stops retrying when `ctx` is
   done, through an unexported `moveFileExFn` seam.
   * `TestMoveFileReplaceHonoursContext` (Windows).
4. Also for Windows: `TestWindowsCleanupReceiptMalformed`,
   `TestWindowsCleanupReceiptReparseBackup` and
   `TestWindowsReceiptConsumedByBegin`.
5. **Checks on this host:** `CGO_ENABLED=0 GOOS=windows go vet` and
   `go test -c`, plus the three-target lint. ~~The Windows tests' first run
   is the next push (0002-PLAN Phase 6). Until it passes, this phase is
   recorded as "compiled and linted, Windows run pending", and `v1.0.0`
   waits.~~ *Amended 2026-09-29, before approval (Windows host provided):*
   * Every Windows-only test in this phase fails first on the Windows host
     against the phase's starting commit, then passes against the phase's
     working tree, with `-race`.
   * `TestLockRejectsRelativeSymlink` and
     `TestLockRejectsDanglingRelativeSymlink` (Phase 3) run there without
     skipping, because Developer Mode permits symlinks. A skip there is a
     failure of this step.
   * The Windows CI leg at the next push confirms, and `v1.0.0` waits for
     it.
6. Gates, then commit.

### Phase 5: release tooling (`scripts/`, `publish-selfupdate-release.yml`)

1. **D1 with D10: shared parity fixtures.**
   * Fixtures live in `selfupdate/testdata/manifest-parity/<case>/`, each
     with `SHA256SUMS` and `expect` (`accept` or `reject`). Cases:
     * LF, CRLF, `*` marker, uppercase hex, blank lines and comments;
     * lone CR, vertical tab, form feed, a 5000-byte line, a BOM;
     * a duplicate name, traversal, empty.
   * `TestManifestParityFixtures` (Go) asserts `expect` for each.
   * `verify-selfupdate-release_test.sh` builds a staging set from each
     fixture and asserts the same outcome.
   * The verifier's parser changes to: split on `\n` only, strip one
     trailing `\r`, reject any other `\r`, `\v`, `\f`, `\x1c`–`\x1f`,
     U+0085, U+2028 or U+2029, and reject a line over 4096 bytes.
   * Also added to the verifier test: a missing `.exe`, empty platforms, a
     symlink, a directory and unsafe extra names.
2. **D5.**
   * Extras must match `^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`. All six
     consumers' current extras match; this is asserted in the execution
     record.
   * The upload step builds an array from
     `find staging -type f -print0 | sort -z` and passes
     `"${files[@]}"`.
3. **D6.** The verifier rejects any staged entry that is a symlink or not
   a regular file (`os.lstat`, `stat.S_ISREG`).
4. **D7.**
   * `refuse-existing-release.sh` requires a non-empty `GH_REPO`, and
     first runs `"$GH" repo view "$GH_REPO" --json name`. A failure
     refuses.
   * `refuse-existing-release_test.sh`: the stub handles `repo view`, with
     new cases "`GH_REPO` unset" and "repository probe fails", both
     refusing.
5. **D8.**
   * Every `run:` block in the reusable workflow reads `TAG` and
     `REF_TYPE` from `env:`. No `${{ … }}` stays inside `run:`.
   * New `scripts/check-workflow-expressions.sh` fails on any `${{`
     inside a `run:` block, with `check-workflow-expressions_test.sh`
     (a planted expression is caught; the real workflow passes).
6. **D4.** `check-workflow-gh-repo.sh`:
   * skips comment lines before matching `GH_REPO:`;
   * resets its state on any `^      - ` step start;
   * treats a `.core-lib-release-tools/scripts/refuse-existing-release.sh`
     call as a `gh` call.

   New `check-workflow-gh-repo_test.sh` covers the three plants from the
   MADR (script-only step, commented `GH_REPO`, unnamed step), a control
   plant, and the real workflow.
7. Gates, then commit.

### Phase 6: CI and tooling (`ci.yml`, `scripts/go-precheck.sh`, `Makefile`)

1. **D9.**
   * Linux adds `go test -race -count=1 ./...`.
   * Linux adds `shellcheck scripts/*.sh`,
     `npx --yes markdownlint-cli2@0.23.2` (the local version), and
     actionlint `v1.7.12` via `go run`.
   * `golangci-lint` is pinned at `v2.13.2` in `ci.yml` and in
     `go-precheck.sh`'s install hint.
   * CI also runs the new script tests from Phase 5.
2. **D11.** `ci.yml` gains
   `concurrency: {group: ci-${{ github.ref }}, cancel-in-progress: true}`.
   The orphan GitHub App check suite is reported to the owner in the
   hand-off; it is not a repository change.
3. Local proof: YAML parses; `go run …actionlint@v1.7.12` is clean on
   both workflows; `-race` passes locally.
4. Gates, then commit.

### Phase 7: documentation and close-out

1. **D3.** The guide resolves the pin with
   `git ls-remote https://github.com/maccavelli/go-core-lib 'refs/tags/v1.0.0^{}'`,
   and says why: annotated tags.
2. `README.md` and `docs/architecture.md` describe:
   * the behaviour changes (tag pinning, the https-only redirect, EOF as a
     decline, extras not validated, the event order);
   * the new scripts and CI steps;
   * the concurrency group.
3. **Close-out evidence,** on a fresh `git archive` of the final commit:
   * re-run the B and C reviewer probes and the author's A and D1 probes;
     each must now show the fixed behaviour;
   * record G-diff (now `v1.6.0` + 0002 §3 + these fixes) and G-api (doc
     comments only);
   * run every gate.
4. Set this PLAN `complete`, update the index, and commit. 0002-PLAN
   Phase 6 then resumes, starting with a push that needs the owner's ask.

## Verification

* **V1.** Each ID in the MADR maps to a step above. The execution record
  lists, per ID, the commit, the test name and its fail-first line.
* **V2.** G-api differs from `mcplib` `v1.6.0` only in doc comments.
* **V3.** `go test -race -count=1 ./...` passes. Coverage is recorded
  against 75.5 %, and it must not fall.
* **V4.** Three-target lint, `shellcheck`, `markdownlint-cli2`,
  actionlint and `go mod tidy -diff` are clean.
* **V5.** The re-run probes show the fixed behaviour for A1–A3, A6–A8,
  A10, B1–B6, C1–C6, C8 and D1.
* **V6.** On the Windows host, the Windows gate passes on the final commit,
  every Windows-only test has a recorded fail-first run, and neither
  symlink test skips. Then the Windows CI leg passes before `v1.0.0` is
  tagged, under 0002-PLAN Phase 6.

## Rollout and Rollback

* **Rollout.** Local commits per phase; nothing is pushed by this PLAN.
  The next push, under 0002-PLAN Phase 6 on the owner's ask, carries the
  0002 record commit `004d13b` and every 0003 commit.
* **Rollback.** Each phase is one commit (or lettered sub-commits). Revert
  that commit here; nothing is published. After a push and before the tag,
  fix forward.

## Execution Record

2026-09-29, before approval: the owner said a Windows test host is
available over SSH, with PowerShell 7 and a full Go environment. It was
probed read-only, and its properties are in "Fixed inputs". Its login
shell turned out to be Git Bash, not PowerShell. The PLAN and MADR were
amended to use it (the Windows gate, Phase 0 baseline, Phase 4 step 5,
and V6).

Not started. Awaiting the owner's approval of this PLAN.
