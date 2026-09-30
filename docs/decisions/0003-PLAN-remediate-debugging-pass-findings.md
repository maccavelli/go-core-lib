---
status: in-progress
date: 2026-09-29
associated-madr: "0003-MADR-remediate-debugging-pass-findings.md"
---
# Implement the fixes for every debugging-pass finding

Associated MADR: [0003-MADR-remediate-debugging-pass-findings.md](0003-MADR-remediate-debugging-pass-findings.md)

## Goal

Every finding in the MADR's tables is closed:

* all 41 findings: A1, A3, A5–A8, A10, A11 (A2 is C1, A9 is C9); B1–B8
  and B10; C1–C12 and C14; D1–D9, D11 and D12. Also B11, added by
  deviation D1, for 42;
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
| 0b | Windows running-image replace (deviation D1) | B11 |
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

### Phase 0b: Windows running-image replace (B11; added by deviation D1)

This phase runs first, because the Windows gate cannot pass until B11 is
fixed.

1. In `moveFileReplace`, retry while `isBusyRunningImage(last)` (a
   sharing violation **or** access denied), within the existing
   `DefaultLockTimeout` deadline. Taking the caller's `ctx` is B10's change
   and stays in Phase 4.
2. Tighten `TestNativeReplaceRunningCopy`: a nil `Install` error must leave
   exactly the new bytes at the target. The old-bytes allowance stays only
   for the pending-backup path.
3. **Proof on the Windows host.**
   * The tightened test fails at least once in 10 runs on the unchanged
     code.
   * With the fix, `-race` included, it passes **20 of 20** runs.
   * The full Windows gate passes.
4. On this host: `CGO_ENABLED=0 GOOS=windows go vet` and `go test -c`,
   three-target lint, and `go test -race`.
5. Commit.

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

### Phase 0: records (2026-09-29)

The owner approved: "Proceed". This PLAN is `in-progress`.

* **Windows baseline** on `004d13b`, the pushed code. It ran on the
  Windows test host from a `git archive`, with a scratch runner script that
  is not committed:
  * `go vet ./...`: exit 0.
  * The listed Windows tests pass: `TestExactAssetNameWindowsExtension`,
    `TestWindowsCleanupReceiptRoundTrip`,
    `TestWindowsCleanupReceiptDigestMismatch`,
    `TestIsUnsupportedDirSyncAccessDenied`,
    `TestBusyRunningImageIncludesAccessDenied`.
  * Under the host's bash, `refuse-existing-release_test.sh` gives
    `6 passed, 0 failed`, `check-workflow-gh-repo.sh` gives `ok`, and
    `verify-selfupdate-release_test.sh` gives `all fixtures passed` (the
    host has `python3`).
  * **`go test -race -count=1 ./...` failed:**
    `replace_native_test.go:115: selfupdate: replace target: Access is denied.`
    in `TestNativeReplaceRunningCopy`. That led to D1.
  * The remote directory was removed after every run (`cleanup ok`).
* **D12.** `docs/README.md`'s stale "fail until the first package lands"
  row is replaced by a row for 0001 §6 as history, and a row for this
  record is added. 0002-MADR's "the first commit names it" is struck and
  annotated.
* **D2.** 0002-MADR's three "one consumer" statements are struck and
  annotated: all six consumers delete `bridge-release`.
* **0002 amendment.** 0002-MADR "More Information" has a dated amendment:
  `v1.0.0` includes 0003's fixes, the API is unchanged, G-diff grows, and
  §8 waits. 0002-PLAN Phase 6 has a dated entry: `v1.0.0` goes on the
  commit that completes this PLAN, not on `3700381`.

#### Deviation D1 (2026-09-29): B11, running-image replace fails on Windows

* **Found** by the Phase 0 baseline, and pre-existing: the code is
  `mcplib` `v1.6.0`'s.
  * `TestNativeReplaceRunningCopy` failed on the Windows host in 3 of 6
    diagnostic runs, and then in **12 of 15** runs (10 without `-race`, 5
    with), always
    `selfupdate: replace target: Access is denied.`.
  * Host facts: Defender real-time protection `False`; Windows
    `10.0.26200.0`.
  * The same package passed the `windows-2025` CI leg once, on
    `3700381`. Whether that was chance or a build difference was not
    established.
* **Cause.** `moveFileReplace` (`replace_windows.go:74`) retries only on
  `ERROR_SHARING_VIOLATION`. A running image transiently returns
  `ERROR_ACCESS_DENIED`, which `isBusyRunningImage` (`:85`) already
  classifies as busy.
* **Experiment,** on a scratch copy only: the retry condition changed to
  `isBusyRunningImage(last)`, with the deadline unchanged. The result was
  **15 of 15** passes, against 3 of 15 unchanged, on the same host in the
  same session.
* **Owner's decision:** "Option 1 fix it".
  * B11 is added to the MADR (High).
  * Phase 0b is added before Phase 1, so the Windows gate can pass in
    every later phase.
  * Option 2 (move the running image aside first) was not chosen.
* **Scope added.** `replace_windows.go` and `replace_native_test.go` in
  Phase 0b. Phase 4's B10 step still adds the `ctx` parameter.

Commit: `11015fc`.

#### Note (2026-09-29): an outside commit and push during Phase 0

Commit `6b9fb99`, "docs(decisions): plan fixes for debugging-pass
findings", was made in this clone at 23:12:19 under the owner's identity,
not by this PLAN's executor. It holds the 0003 MADR and PLAN as they were
before Phase 0, and a `docs/README.md` row. It and `004d13b` were pushed to
`origin/main`.
* That push's CI run `36667778889` concluded `success` on all three
  operating systems.
* So `TestNativeReplaceRunningCopy` passed on `windows-2025` for a second
  time, while the Windows test host fails it most of the time. B11's
  trigger therefore depends on the host.
* Nothing was rewritten. `11015fc` builds on `6b9fb99`.

### Phase 0b: Windows running-image replace, B11 (2026-09-29)

* **Test tightened.** `TestNativeReplaceRunningCopy` keeps `Install`'s
  error in its own variable (`installErr`), and now fails if a clean
  `Install` (nil error, no pending backup) leaves anything but the new
  bytes.
  * The first draft reused `err`, which `os.ReadFile` then overwrote; it
    was caught on reading and corrected before any run.
* **Fail-first**, Windows host, working tree with only the test change, 10
  runs without `-race` and 5 with: `pass=6 fail=9`, each failure
  `replace_native_test.go:115: selfupdate: replace target: Access is denied.`.
* **Fix.** `replace_windows.go` `moveFileReplace` retries while
  `isBusyRunningImage(last)`. The `DefaultLockTimeout` deadline is
  unchanged, and a comment cites B11.
* **Proof,** Windows host, fixed tree:
  * `native x20 -race: pass=20 fail=0`;
  * then the full Windows gate: `go vet` 0, `go test -race` 0,
    `refuse-existing-release_test.sh` 0, `check-workflow-gh-repo.sh` 0,
    `verify-selfupdate-release_test.sh` 0; `overall=0`, `cleanup ok`.
* **On this host:** `gofmt -l` empty. `CGO_ENABLED=0 GOOS=<t>` `go vet` and
  `go test -c` pass for linux, darwin and windows. `make pre-add-check`
  gives `47 file(s) clean`, `go test -race` passes, and
  `go mod tidy -diff` exits 0.

Commit: `6f0b661`.

### Phase 1: coordinator and confirmer (2026-09-29)

**Changes.**

* **C3.** `New` uses an `isNil` helper (reflect) for every required seam. It
  rejects a nil or typed-nil verifier and a typed-nil transformer; an
  untyped nil transformer is still a no-op.
* **C1.** When `--version` is set, `fetchRelease` requires
  `rel.Tag == req.TargetVersion`, else an `ErrIntegrity` error that quotes
  both tags.
* **C4.** Order is now immutable, then draft/prerelease, then `Validate`.
  Every tag in an error is `%q`, in `updater.go` and in `github.go`'s four
  release and asset errors.
* **C9.** `SHA256SUMS` is parsed and the entry resolved before
  `EventDownloadingBinary` and `CreateStaging`.
* **C8.** `EventVerified` follows the verifiers; `EventTransforming`
  follows it.
* **C2.** `Applied:false` with a nil error becomes the unexported
  `errNotCommitted`.
* **C14.** A once-only `closeSession` runs explicitly before
  `EventComplete` and joins its error; the defer is the safety net.
* **C10.** Every error from `execute` and `apply` passes through `wrapRun`,
  with two exceptions. A validation error for an invalid product name is
  returned unprefixed, since that name is not safe to print. And
  `ErrUpdateAvailable` stays a bare status result, as before.
* **C5.** `EventSelected.Detail` is "local build: apply requires --force"
  in check mode.
* **C12.** The pending-backup detail names
  `filepath.Join(target.Dir, name)`. `ExampleNewManagedInstaller` builds a
  real `ManagedInstaller`, and `ExampleExitCode` is added.
* **C6.** EOF before an answer is a decline.
* **C7.** `terminalConfirmer` is a pointer with one reader goroutine and a
  line channel.
* **C11.** The `Config.Limits` doc comment is corrected.
* **Test helpers.** `scriptSource` gained `entered`, `openErr` and a shared
  `log`; `recReporter` gained `events` and `log`.
* **New tests** (`updater_contract_test.go` and `confirmer_test.go`):
  `TestRunCallOrderFailureBoundaries`, `TestRunOperationsMatrix`,
  `TestRunRejectsTagMismatch`, `TestRunEscapesUntrustedTag`,
  `TestRunRejectsManifestBeforeStaging`,
  `TestRunEventOrderWithTransformer`, `TestRunTransformerSizeLimit`,
  `TestRunInstallerNoCommitIsError`, `TestRunCompleteAfterClose`,
  `TestRunCloseErrorJoinedAfterCommit`, `TestRunErrorsNameProduct`,
  `TestRunLocalBuildCheckHintsForce`, `TestRunPendingBackupDetail`,
  `TestTerminalConfirmerEOFDeclines`,
  `TestTerminalConfirmerCancelKeepsLine`. `TestNewRejectsNilCollaborators`
  and `TestOverlappingRun` were rewritten.

**Fail-first.** The new test files ran against a `git archive` of `6f0b661`,
with a scratch-only shim declaring `errNotCommitted`.

* **Failed, as required** (14):
  * `TestNewRejectsNilCollaborators` (subtests nil verifier, typed-nil
    source, reporter, verifier and transformer);
  * `TestRunErrorsNameProduct` (all 9 subtests);
  * `TestRunRejectsTagMismatch`, `TestRunEscapesUntrustedTag`,
    `TestRunRejectsManifestBeforeStaging`,
    `TestRunEventOrderWithTransformer`,
    `TestRunInstallerNoCommitIsError`, `TestRunCompleteAfterClose`,
    `TestRunLocalBuildCheckHintsForce`, `TestRunPendingBackupDetail`,
    `TestTerminalConfirmerEOFDeclines`;
  * `TestTerminalConfirmerCancelKeepsLine`, after two corrections. As
    first written it **passed** on the old code, because `cancel()` could
    run before the old `Confirm` started its reader. The test now waits
    for the prompt to be written, gives the reader 100 ms to block,
    cancels, writes the answer, and confirms with a 3 s deadline. On the
    old code: `ok=false err=context deadline exceeded, want the line
    typed after cancellation` (3 of 3 runs). On the current code: 5 of 5
    passes with `-race`.
* **Passed on the old code, so proven by mutation on a scratch copy**
  (these tests close gaps, not bugs):
  * `ignore-confirm-error` fails `TestRunCallOrderFailureBoundaries`
    (`failure at Confirm`);
  * `rollback-as-upgrade` fails `TestRunOperationsMatrix`
    (`op=upgrade … want rollback`);
  * `no-transform-size-limit` fails `TestRunTransformerSizeLimit`
    (`err = <nil>`);
  * `drop-close-error` fails `TestRunCloseErrorJoinedAfterCommit`;
  * `no-overlap-guard` first failed only through the 10-minute `go test`
    timeout, a hang. The second `Run` is now bounded by a 5 s context, and
    the mutation fails at once: `overlap err = selfupdate: demo: context
    deadline exceeded`. The fake's `entered` handoff moved under its mutex
    to avoid a data race in that mutated run.

**Gates.**

* `make pre-add-check`: `48 file(s) clean`.
* `go test -race -count=1 ./...`: ok, coverage **81.4 %** (baseline
  75.5 %). `go mod tidy -diff`: 0.
* **G-api** against `v1.6.0`: 26 differing lines, all doc text: the import
  header, the package comment, and the `Config.Limits`,
  `NewTerminalConfirmer` and `New` comments. No `func`, `type`, `var` or
  `const` line differs.
* **Windows gate:** `go vet` 0; `go test -race` ok; the confirmer tests,
  `TestRunCallOrderFailureBoundaries`, `TestOverlappingRun` and
  `TestNativeReplaceRunningCopy` all PASS; the three script tests 0;
  `overall=0`; `cleanup ok`.

Commit: `f095505`.

### Phase 2: network and integrity (2026-09-29)

**Changes.**

* **A1 with A5.**
  * `mapRelease` now calls a new `validateAssetStructure` (ID, basename,
    no control characters) for every asset.
  * `validateAssetMetadata` builds on it for the full check.
  * `execute` runs `validateAssetMetadata` on `sel.Binary` against
    `Limits.Executable`, and on `sel.Manifest` against `Limits.Manifest`,
    right after `Select`. `OpenAsset` still checks the asset it opens.
* **A2.** `ByTag` rejects `rel.Tag != tag` with `ErrIntegrity`.
* **A3.** `checkRedirect` refuses any hop that is not `https` unless the
  host is loopback. The URL is not echoed.
* **A6.** `decodeJSON` requires `dec.Token()` to return `io.EOF`.
* **A7.** A `Retry-After` in seconds above `maxRetryAfterSeconds` is
  ignored. The first draft, `int64(math.MaxInt64 / int64(time.Second))`,
  failed the pre-add gate with `unnecessary conversion (unconvert)` on all
  three targets. It was corrected and re-gated.
* **A8.** `getRelease` checks the status before reading. Non-2xx bodies,
  there and in `OpenAsset`, go through the new `readTruncated`, which is
  capped at `ErrorBody` and tolerates truncation.
* **A10.** `validateGitHubName` rejects `.` and `..`.
* **A11.** The new `isInvisibleControl` (`unicode.Cf`, U+2028, U+2029) is
  applied in `sanitizeDiagnostic` and `sanitizeText`.
* **Tests.**
  * `github_hardening_test.go`: `TestGitHubExtraAssetsDoNotPoisonRelease`,
    `TestGitHubByTagRejectsMismatchedTag`,
    `TestGitHubRedirectRequiresHTTPS`,
    `TestDecodeJSONRejectsTrailingDelimiters`,
    `TestRateLimitRetryAfterOverflow`,
    `TestGitHubLargeErrorBodyKeepsRateLimit`,
    `TestNewGitHubSourceRejectsDotNames`,
    `TestSanitizeRemovesFormatControls`, and for A4
    `TestGitHubForbiddenRemainingZeroIsRateLimit`,
    `TestGitHubOpenAssetStatuses`, `TestGitHubOpenAssetValidatesAsset` and
    `TestRateLimitErrorText`.
  * `updater_contract_test.go`: `TestRunValidatesSelectedAssets`.
  * Authoring slip: the sanitizer test's `\u` escapes were written into
    the file as literal invisible runes, and `gofmt` refused them ("illegal
    byte order mark"). A scratch script rewrote the 12 runes as Go escapes.
    A scan found no literal control bytes in any Go or script file; `\x`
    escapes had been preserved.

**Fail-first** against a `git archive` of `f095505`.

* **Failed, as required** (9): `TestGitHubExtraAssetsDoNotPoisonRelease`,
  `TestGitHubByTagRejectsMismatchedTag`, `TestGitHubRedirectRequiresHTTPS`,
  `TestDecodeJSONRejectsTrailingDelimiters`,
  `TestRateLimitRetryAfterOverflow`,
  `TestGitHubLargeErrorBodyKeepsRateLimit`,
  `TestNewGitHubSourceRejectsDotNames`,
  `TestSanitizeRemovesFormatControls`, `TestRunValidatesSelectedAssets`.
* **The four A4 tests passed there** (they close gaps). Mutation proofs
  for the five checks A4 named:

  | Mutation | Test | Failure |
  |---|---|---|
  | `no-403-remaining` | `TestGitHubForbiddenRemainingZeroIsRateLimit` | `err = selfupdate: github http 403` |
  | `no-state-check` | `TestGitHubOpenAssetValidatesAsset` | `open state: accepted` |
  | `no-size-limit` | `TestGitHubOpenAssetValidatesAsset` | `oversize: accepted` |
  | `no-openasset-status` | `TestGitHubOpenAssetStatuses` | `status 429: no error` |
  | `no-belongs-check` | `TestGitHubOpenAssetValidatesAsset` | `foreign id: accepted` |

**Gates.**

* `make pre-add-check`: `49 file(s) clean`.
* `go test -race -count=1 -cover ./...`: coverage **83.3 %**.
* `go mod tidy -diff`: 0.
* G-api: byte-identical to Phase 1's (26 doc-text lines against
  `v1.6.0`; no signature line).
* Windows gate: `go vet` 0, `go test -race` 0, the named Phase 2 tests
  PASS, script tests 0, `overall=0`, `cleanup ok`.
