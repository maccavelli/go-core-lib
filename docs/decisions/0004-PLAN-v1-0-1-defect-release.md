---
status: in-progress
date: 2026-09-30
associated-madr: "0004-MADR-evolve-selfupdate-api-and-tui-support.md"
---
# Implement Phase 0: the `v1.0.1` defect release

Associated MADR: [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md)

This PLAN implements the MADR's Decision Outcome §2, "Phase 0: defect
release (`v1.0.1`)", including the five bullets marked *(clarified)*. The
MADR's later phases each get their own PLAN under the same number.

## Goal

Every Phase 0 item is closed, each with a test that was seen to fail on
the unfixed code:

* R1–R9 from the MADR's "Defects in `v1.0.0`";
* G8;
* the Phase 0 harness items.

The exported API does not change. `v1.0.1` is tagged only when the owner
asks for it.

## Scope

### Item → step

| Item | Step | Files |
| :--- | :--- | :--- |
| Records | 1 | this PLAN, the MADR, `docs/README.md` |
| R1 (standalone and managed) | 2 | `updater.go`, `managed.go`, `types.go` (doc only) |
| R3 | 2 | `session.go`, `replace.go` |
| R4 | 2 | `replace_windows.go`, `replace_unix.go`, `session.go` |
| R5 | 2 | `session.go`, `cleanup_windows.go` |
| R2 | 3 | `confirmer.go` |
| G8, R9 | 4 | `updater.go`, `github.go` |
| R6 | 5 | `scripts/verify-selfupdate-release.sh` and its test |
| R7, R8 | 5 | a new `scripts/check-workflows.sh`, which replaces both old checkers; its test; a new `scripts/requirements-workflow-check.txt`; `ci.yml` |
| Harness | 6 | `*_test.go`, `lock.go` (one test seam), `testdata/` |
| Close-out | 7 | `doc.go`, `docs/architecture.md`, this PLAN, `docs/README.md` |

### Out of scope

* Everything in the MADR's Phases 1–4.
* Performing every replacement step through the directory handle. R3's
  rollback uses the handle. The forward path stays path-based and
  checked, as the MADR's R3 bullet specifies.
* Tagging or pushing. Both need an explicit ask.

### Rules for every step

* Each new test is first run against the unfixed code **on a scratch copy**,
  and its failure is recorded here. The working tree is never broken
  deliberately.
* `make pre-add-check` passes before each commit. Every step that touches
  Windows code, or code that runs on Windows, passes `go vet` and
  `go test -race` on the Windows test host, reached with the scratch script
  that copies the tree to a temporary directory and removes it afterwards.
* Each step ends with a `git commit --no-edit`. The global hook writes the
  message.
* Anything found outside this scope stops the work, and is recorded as a
  dated deviation before any further change.

## Implementation Steps

### Step 1: records

1. Mark the MADR `accepted`, and add the *(clarified)* notes to its Phase 0
   bullets.
2. Write this PLAN, and index it in `docs/README.md`.

### Step 2: install path (R1, R3, R4, R5)

1. **R1, standalone.**
   * In `Updater.apply`, when `Applied` is false and `Backup` is not
     empty, set `Result.PendingBackup` to the backup path.
   * Join an error that names the path, sanitised, and says that the
     previous binary was kept there because restoring it failed.
   * Extend the `Result.PendingBackup` doc comment to cover this case: on
     any OS, with `Applied` false, the caller must restore or remove the
     file.
2. **R1, managed.** When `recover` fails to roll back, return an
   `InstallResult` that carries the backup path, so the same `apply` rule
   reaches the caller.
3. **R3.**
   * Keep the identity from `root.Stat(".")` as `dirInfo`.
   * When `checkDir` fails after the rename, roll back through a new
     helper. The helper renames the backup's base name over the target's
     base name with `root.Rename`, then syncs the handle's directory.
   * If the rollback succeeds, return `Applied: false` and the
     `ErrConcurrentUpdate` error. If it fails, keep today's result
     (`Applied: true`, with the backup) and join the rollback error.
4. **R4.**
   * The session carries its lock timeout. `replaceLocked` and `rollback`
     pass it to `moveFileReplace` through an unexported context value.
     When no value is set, `DefaultLockTimeout` applies.
   * `moveFileReplace` stops at once on an access-denied error when the
     destination has `FILE_ATTRIBUTE_READONLY`.
   * The restore inside `replaceTarget` (Unix and Windows) runs on
     `context.WithoutCancel(ctx)`. The Windows retry is bounded by the
     same timeout.
5. **R5.**
   * In `beginSession`, check that the root and the path name the same
     directory before `processCleanupReceipt` runs.
   * In `processCleanupReceipt`, hash the pending backup through
     `root.Open`, and compare the opened file with the `root.Lstat` result
     using `os.SameFile`.
6. **Tests.** Each is seen to fail on a scratch copy of the unfixed code.
   * R1 through `Run`: the round-2 probe, which fails a directory sync and
     then a restore, turned into assertions.
   * R1 through the managed installer.
   * R3: the directory is swapped right after the rename, and the test
     asserts the rollback happened in the moved directory. On Windows, if
     the OS refuses to rename a directory the session holds open, the test
     asserts that refusal instead, because the swap cannot happen there.
   * R4: a stubbed `moveFileExFn` returns access-denied on a read-only
     destination and must fail without retrying. A short lock timeout must
     bound the retry. The restore must run after the caller's context is
     cancelled.
   * R5: a directory swapped before the receipt is processed is refused
     before the receipt is read (Windows).

### Step 3: confirmer (R2)

1. Replace the long-lived reader goroutine:
   * `Confirm` starts a read only when none is outstanding. The read is
     one line, read byte by byte, capped at 4 KiB of kept text. Its
     goroutine exits after delivering that line.
   * A cancelled `Confirm` leaves its read outstanding, so the next
     `Confirm` receives that line, which keeps 0003 C7.
   * An answered `Confirm` leaves no read outstanding.
2. **Tests**, seen to fail on the unfixed code:
   * the host reads its own next line after an answered `Confirm`;
   * two confirmers on one descriptor each get their own line;
   * the goroutine count returns to its baseline after an answered
     `Confirm`;
   * the existing C7 test still passes.

### Step 4: coordinator (G8, R9)

1. `hashAndValidateStaging` also returns the staged size. `apply` passes
   the post-transform size in `StagedArtifact.Size`.
2. `runVerifiers` receives the manifest entry's digest and passes it as
   `ManifestSHA256`.
3. `validateAssetStructure` refuses `.` and `..`.
4. **Tests**, seen to fail on the unfixed code:
   * a transformer that grows the file changes `StagedArtifact.Size`;
   * a verifier sees the manifest digest;
   * `.` and `..` are refused.

### Step 5: release tooling (R6, R7, R8)

1. **R6.** The verifier's regexes use `fullmatch`. The test gains a tag
   with a trailing newline and an extra-asset name with a trailing
   newline; both must be refused.
2. **R7 and R8.** A new `scripts/check-workflows.sh` embeds a Python
   program that parses each workflow with PyYAML and applies two rules:
   * **expressions:** no step's `run` value contains `${{`.
   * **gh-repo:**
     * A step whose `run` script calls a repository-scoped `gh` command
       (`release`, `api`, `repo`, `run`, `workflow`, `pr`, `issue`,
       `attestation`) or runs `refuse-existing-release.sh` must have
       `GH_REPO` in the step's, the job's or the workflow's `env`.
     * The script is split into commands on newlines, `;`, `&&`, `||` and
       `|`, with comments removed.
     * Only a command segment that is itself a `--help` probe is exempt.

   Usage is `check-workflows.sh [--rule expressions|gh-repo|all] [file...]`.
   With no file, it checks the publish workflow with both rules. A missing
   PyYAML exits 2 with a message that names the requirements file.
3. **Tests.** A new `scripts/check-workflows_test.sh` carries every case
   from the two old tests, plus the R7 and R8 shapes, and the real
   workflows, which must pass. Each R7 and R8 shape is shown to be missed
   (or falsely flagged) by the old checkers, on a scratch copy.
4. Delete the two old checkers and their tests.
5. **CI.**
   * `ci.yml` creates a virtual environment in `$RUNNER_TEMP` and runs
     `pip install --require-hashes --only-binary=:all:` from
     `scripts/requirements-workflow-check.txt` (`PyYAML==6.0.3`, with
     wheel and sdist hashes).
   * CI then runs the new checker: both rules on the publish workflow, and
     the expressions rule on `ci.yml`.
   * Shellcheck still covers every `scripts/*.sh`.
6. The Windows test script names the new checker.

### Step 6: harness

1. **Fuzz targets.** Commit `FuzzParseSHA256SUMS`, `FuzzSanitize`,
   `FuzzVersionPolicy` and `FuzzGitHubReleaseJSON`, the last with the R9
   invariant enabled. Their seed corpus includes every
   `testdata/manifest-parity` input. `go test` runs the seeds on every
   platform.
2. **Dark 0003 branches.** Add a test seam, `lockOpenHook`, called between
   `Lstat` and `Open` in `openLockFile`, and cover:
   * the lock appearing between `Lstat` and create (the retry);
   * the lock being replaced between `Lstat` and open ("changed while
     opening");
   * the lock changing on both attempts ("kept changing");
   * a non-regular lock;
   * `beginSession`'s "changed while locking";
   * `commit`'s `checkDir` failure;
   * `commitReplacement` failing to remove or sync;
   * `rollbackReplacement` failing, with no backup and with a failed
     restore;
   * a rollback failure inside managed recovery;
   * `copyFile` failing on chmod, sync and close;
   * `readTruncated` at its limit.
3. **Seams.** Every test that swaps a package-level seam uses `setSeam`.
   A comment beside `setSeam` states the rule: no `t.Parallel` in this
   package, because seams are package state.
4. **Leak checks.** The confirmer tests use channel handshakes instead of
   `time.Sleep` where the sleep orders events, and check the goroutine
   count.
5. Record coverage before and after the step.

### Step 7: close-out

1. `doc.go` and `docs/architecture.md` describe any behaviour this PLAN
   changed: the R1 result, R3 rollback and the R4 retry bound.
2. Record the execution in this PLAN, set its status to `complete`, and
   update `docs/README.md`.

## Verification

* `make pre-add-check` passes at every commit. `make lint` runs for
  Linux, macOS and Windows, and `make vuln` passes.
* `go test -race -count=3 ./...` and `go test -shuffle=on -count=2 ./...`
  pass locally.
* The Windows test host passes `go vet ./...`, `go test -race -count=1 ./...`
  and the script tests.
* Every new test's failure on the unfixed code is recorded in the
  execution record, with the failure text.
* After the push the owner asks for, CI is green on all three operating
  systems. That includes the new virtual-environment step.

## Rollout and Rollback

* Each step is its own commit, so a step can be reverted alone.
* `v1.0.1` is tagged and published only on the owner's explicit ask. The
  change is compatible with `v1.0.0`: no exported identifier changes. The
  behaviour changes are listed in the tag's release notes:
  * R1: the path appears in `PendingBackup` with `Applied` false;
  * R3: a rollback on a directory swap;
  * R4: a read-only destination fails fast;
  * R9: dot asset names are refused.
* Consumers pick `v1.0.1` up when each one migrates, per the MADR's owner
  decision 5.

## Execution Record

### Step 1: records (2026-09-30)

* The MADR was marked `accepted`. Its Phase 0 section gained five
  *(clarified)* notes: R2, R3, R4, R7/R8 and G8.
* This PLAN was written and indexed.

### Step 2: install path (2026-09-30)

**What changed.**

* **R1.**
  * `Updater.apply` moves an unapplied result's `Backup` into
    `Result.PendingBackup`, and joins an error naming the path.
  * `managedSession.recover` now returns an `InstallResult` carrying the
    backup when the rollback failed and the file still exists.
  * The `Result.PendingBackup` doc covers both meanings.
  * The absolute-path logic moved into `retainedPath`.
* **R3.**
  * `dirInfo` comes from `root.Stat(".")`.
  * `Install` undoes the rename through the new `rollbackInRoot`
    (`root.Rename`, then `syncRoot`) when `checkDir` fails after the
    rename.
  * `syncDirectory` now delegates to `syncRoot`.
* **R4.**
  * The session keeps its lock timeout. `replaceLocked` and `rollback`
    pass it through `withRetryBudget`; `retryBudget` reads it on Windows.
  * `moveFileReplace` stops at once through `isReadOnlyDenial`.
  * Both `replaceTarget` restores run on `context.WithoutCancel`.
* **R5.**
  * `beginSession` checks root-versus-path identity before
    `processCleanupReceipt`.
  * On Windows the backup is hashed through `root.Open`, with an
    `os.SameFile` check (`rootFileSHA256`).
* **Notes, within R5's scope.**
  * `cleanup_other.go` had the same flaw: it found the receipt through the
    root and removed it by path. It now uses `root.Remove`.
  * The R5 test needs a moment to swap the directory, so `beginSession`
    gained the test seam `afterLockHook`, which is nil in production.

**Tests** (`recovery_test.go`, `replace_windows_budget_test.go`). Each was
run on a scratch copy of `33761dd`. Only the two test seams were added
there, `afterLockHook` and a no-op `withRetryBudget`, without the fixes.

| Test | On the unfixed copy |
| :--- | :--- |
| `TestRunReportsKeptBackup` | FAIL (macOS, Windows): `PendingBackup is empty; the kept backup was not reported (err=… sync directory: injected directory sync failure` |
| `TestManagedReportsKeptBackup` | FAIL (macOS, Windows): `Backup is empty after a failed rollback` |
| `TestInstallRollsBackWhenDirectoryMovesAfterRename` | FAIL (macOS): `Applied=true err=… target directory changed during the update: … concurrent update; want an unapplied concurrent-update failure` |
| `TestReplaceRestoreSurvivesCancellation` | FAIL (macOS, Windows): `the restore ran on a cancelled context: context canceled` |
| `TestBeginChecksDirectoryBeforeReceipt` | FAIL (macOS): `the receipt in the swapped-in directory was touched: lstat …: no such file or directory` |
| `TestMoveFileReplaceReadOnlyFailsFast` | FAIL (Windows): `calls = 475 after 5.0000633s; a read-only destination must not be retried` |
| `TestMoveFileReplaceHonoursRetryBudget` | FAIL (Windows): `retried for 5.0040745s with a 50ms budget` |

* **Fixing the R5 test itself.** The first version of the R5 test passed
  on the unfixed copy: the unfixed code acts only when the locked
  directory also holds a receipt. The test now plants one there, and it
  fails as shown above. On Windows, its first run failed on the fixed
  tree too: with no swap, Begin rightly rejected the placeholder receipt
  as malformed. When no swap happens, the test now removes its
  placeholder.
* **R3 and R5 on Windows.** Windows refuses to rename a directory while the
  session holds handles inside it: "The process cannot access the file
  because it is being used by another process". Both tests therefore
  assert and log that refusal there. The swap they guard against cannot
  happen on that OS.
* **Checks.**
  * macOS: `make pre-add-check` passed on the 11 files (after two lint
    fixes: `retryBudget` moved to the only file that uses it, and one test
    stub was rewritten for `nilerr`).
  * Windows test host: `go vet ./...` and `go test -race -count=1 ./...`
    passed, and the script tests passed.

### Step 3: confirmer (2026-09-30)

**What changed.**

* The long-lived `bufio.Scanner` goroutine is gone.
* `Confirm` takes the outstanding read from `nextLine`, starting one only
  when there is none, and clears it with `consumed` once a line arrives.
* `readLine` reads byte by byte to the newline. It keeps at most
  `maxAnswer` (4 KiB) of text, trims a trailing `\r` as `ScanLines` did,
  and returns a final unterminated line before EOF.
* A cancelled `Confirm` leaves its read pending for the next one (C7).

**Tests** (`confirmer_read_test.go`). Each was run on a scratch copy of
`33761dd`, unchanged, on macOS and on the Windows test host:

| Test | On the unfixed copy |
| :--- | :--- |
| `TestConfirmLeavesHostInput` | FAIL: `the host's input was consumed by the confirmer` |
| `TestConfirmersShareInput` | FAIL: `second: ok=false err=context deadline exceeded, want its own "y"` |
| `TestConfirmLeaksNoReader` | FAIL: `goroutines: 5 after an answered Confirm, 4 before` |

* **Checks.**
  * On the fixed tree, every confirmer test passed three times under
    `-race`, including the existing C7 test
    (`TestTerminalConfirmerCancelKeepsLine`).
  * `make pre-add-check` passed on both files.
  * The Windows test host passed `go vet ./...` and
    `go test -race -count=1 ./...`.
* `TestConfirmersShareInput` keeps one 100 ms pause, with a comment. It
  orders nothing: it gives a wrongly lingering reader time to block, so the
  old failure is observed rather than raced. The fixed code passes with or
  without it.
