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
   * ~~a verifier sees the manifest digest;~~ *(dropped, deviation D1: the
     value cannot differ, so no test of it can fail)*
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
     the expressions rule on `ci.yml`. *(Linux only, per deviation D2; the
     old checkers ran on all three runner OSes.)*
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

### Step 4: coordinator (2026-09-30)

**What changed.**

* **G8, size.**
  * `hashAndValidateStaging` returns the size as well.
  * `apply` keeps `installedSize`: the advertised size, which the download
    enforced, until a transform changes it.
  * `StagedArtifact.Size` now carries that size.
* **G8, digest.** `runVerifiers` takes the manifest entry's digest and
  passes it as `ManifestSHA256`.
* **R9.** `validateAssetStructure` refuses `.` and `..`.

**Tests** (`coordinator_fix_test.go`). Each was run on a scratch copy of
`33761dd`, unchanged:

| Test | On the unfixed copy |
| :--- | :--- |
| `TestInstallRequestCarriesTransformedSize` | FAIL: `StagedArtifact.Size = 9, want the transformed length 16 (advertised 9)` |
| `TestAssetStructureRefusesDotNames` | FAIL: `asset name "." accepted`, `asset name ".." accepted` |

**Checks.**

* `go test -race -count=1 ./...` passed.
* `make pre-add-check` passed on the three files.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`.

**Deviation D1 (2026-09-30): no test for the `ManifestSHA256` value.**

* **Found.** Step 4.4 asked for a test that a verifier sees the manifest
  digest, seen to fail on the unfixed code. No such test can fail:
  * `verifyIntegrity` runs before any verifier and rejects the run unless
    the staged digest and the manifest digest are byte-equal
    (`equalDigest`, a constant-time exact compare);
  * `parseSHA256SUMS` lowercases every manifest digest;
  * so the old value (the staged digest) and the new one are always the
    same string by the time a verifier sees it.
* **Decision.** The change is kept, because it makes the value match the
  field's documentation, and the unfalsifiable test is dropped. Step 4.4
  is annotated. No file was added to the step.
* **MADR.** No MADR amendment is needed. Its G8 *(clarified)* note already
  describes the fix as "the value", and nothing it asserts changes.

### Step 5: release tooling (2026-09-30)

**What changed.**

* **R6.** The verifier's six regex checks use `fullmatch`.
  `verify-selfupdate-release_test.sh` gains two cases: a tag, and an
  extra-asset name, each with a trailing newline.
* **R7, R8.**
  * The new `scripts/check-workflows.sh` parses each workflow with PyYAML
    and refuses duplicate keys.
  * Its two rules are `expressions` and `gh-repo`. `gh-repo` splits the
    script into command segments, so only a segment that is itself a
    `--help` probe is exempt, and it reads `GH_REPO` from the step's,
    job's or workflow's `env`.
  * It exits 2 when PyYAML is missing.
  * The old `check-workflow-expressions.sh`, `check-workflow-gh-repo.sh`
    and their tests were deleted.
* **Pinned install.** `scripts/requirements-workflow-check.txt` pins
  `PyYAML==6.0.3` with its cp312 manylinux wheel hash and its sdist hash.
* **CI.** `ci.yml` splits the old step into two:
  * "Verify the release guard" still runs on all three OSes;
  * "Verify the workflow contract" is Linux only, and runs the pinned venv
    install, the checker and its test.
* **Records outside the plan's file list.** `docs/architecture.md` listed
  the deleted checkers, so it was updated here rather than in Step 7.

**Tests and proofs.**

* **R6, unfixed verifier** (`33761dd`):
  * `not ok - tag with a trailing newline (expected failure)`.
  * The test stops at its first failure, so the extra-name case was run
    alone on a second copy, with the tag case neutralised:
    `not ok - extra name with a trailing newline (expected failure)`.
* **R6, fixed verifier.** It passes all fixtures on macOS and on the
  Windows test host, where Git Bash can create the newline-named file,
  so neither case skipped.
* **R7, R8, old checkers.** `check-workflows_test.sh` was run on a scratch
  copy with each case sent to the old checker for its rule:
  `11 passed, 13 failed`.
  * All 11 carried-over cases passed there, so the old verdicts are
    preserved.
  * Every R7 and R8 shape failed, for example:
    * `FAIL R7: block header with a comment: want exit 1, got 0`;
    * `FAIL R7: env after a dash-run block is allowed: want exit 0, got 1`;
    * `FAIL R8: a --help tail exempts only itself: want exit 1, got 0`;
    * `FAIL R8: job-level GH_REPO counts: want exit 0, got 1`.
  * The duplicate-key case is a guard the new checker adds, not an R7
    item. The old checker exited 1 on it, finding the expression.
* **R7, R8, new checker.** `24 passed, 0 failed` on macOS and on the
  Windows test host.
* **Missing PyYAML.** With a `python3` that has no site-packages,
  the checker printed
  `check-workflows: PyYAML is required: pip install -r scripts/requirements-workflow-check.txt`
  and exited 2.
  * The first two attempts passed wrongly. The macOS system `python3` shim
    found PyYAML elsewhere. Then the shell's `BASH_ENV` put Homebrew back
    at the front of `PATH`.
  * The proof that counts ran with `BASH_ENV` unset.
* **Pinned install.** A platform-targeted
  `pip download --platform manylinux2014_x86_64 --python-version 3.12 --only-binary=:all: --require-hashes`
  resolved the pinned wheel. The same command with one hash character
  changed failed: `THESE PACKAGES DO NOT MATCH THE HASHES`.
* **Lint.** `actionlint` (v1.7.12) and `shellcheck scripts/*.sh` were
  clean, and `markdownlint-cli2` was clean on `docs/architecture.md`.
* **Windows test host.** `go vet`, `go test -race`, and all four script
  tests passed.
* **Not yet verified.** The venv step has not run on a GitHub runner. It
  first runs on the push the owner asks for.

**Deviation D2 (2026-09-30): the workflow checker runs on Linux only in
CI.**

* **Found.** The old checkers ran in a step without an OS condition, so on
  all three runner OSes. The new checker needs PyYAML, which no runner
  image ships. A hash-pinned install differs by OS and by Python version:
  another wheel, another hash, and on Windows another venv layout and no
  `python3` command.
* **Decision.** Run the checker and its test on Linux only, in a
  hash-pinned venv. That follows the precedent `ci.yml` already sets for
  the Python-based release-verifier test. The refuse-guard test keeps all
  three OSes.
* **Why this does not weaken the gate.** The checker's verdict depends
  only on the workflow text. The Windows test host still runs the checker
  and its test locally.
* **What the owner can reverse.** If the checker should run on every
  runner OS, the requirements file needs hashes for every wheel, and the
  step needs per-OS venv paths.
* **MADR.** Unaffected: its R7/R8 *(clarified)* note says only "CI installs
  `PyYAML==6.0.3` into a virtual environment from a hash-pinned
  requirements file".

### Step 6: harness (2026-09-30)

**What changed.**

* **Fuzz targets** (`fuzz_test.go`): `FuzzParseSHA256SUMS`,
  `FuzzGitHubReleaseJSON` (with the R9 invariant on, and `.` and `..`
  seeds), `FuzzSanitize` and `FuzzVersionPolicy`.
  * `FuzzParseSHA256SUMS` is seeded with all 23 `testdata/manifest-parity`
    manifests plus eight hand seeds, one of them an uppercase digest.
  * Every seed string is written with hex escapes or numeric code points,
    so the file holds only ASCII.
* **Branch tests** (`branch_coverage_test.go`):
  * `openLockFile`: the create race, retried; a lock replaced between
    `Lstat` and open; a lock that keeps changing; a non-regular lock.
  * The managed commit's `checkDir`.
  * Commit failures to remove or to sync.
  * `rollbackReplacement` with no backup, a failed restore and a failed
    sync.
  * `copyFile` failing at chmod, sync and close.
  * `readTruncated`.
* **Test seams.**
  * `lockOpenHook` (stages `before-lstat` and `after-lstat`) in
    `lock.go`.
  * `fileChmod`, `fileSync` and `fileClose` in `replace.go`. These three
    are method-value seams the plan did not name, but covering
    `copyFile`'s chmod, sync and close failures needs them.
* **Seams through `setSeam`.** Six tests that swapped a seam by hand now
  use `setSeam`: `confirmer_test.go` (three), `fs_test.go`,
  `github_test.go`, `replace_windows_test.go` (two) and `target_test.go`.
  `setSeam` carries the no-`t.Parallel` rule.
* **The C7 test.**
  * `TestTerminalConfirmerCancelKeepsLine` lost its 100 ms ordering pause.
    With Step 3's confirmer, the read starts before `Confirm` waits, so
    the cancel can land at any point.
  * It also gained a check that no read is outstanding after the answered
    `Confirm`.
  * Its 200 ms pause after the late `y` stays, because it orders nothing:
    it only gives a lingering reader time to misbehave.
* **Lint fix.** `unparam` failed for Windows: every Windows caller passed
  `0` for `openLockFile`'s `extraFlags`. The parameter became a per-OS
  constant, `lockOpenFlags`: `unix.O_NOFOLLOW` on Unix and `0` on Windows.
  The behaviour is unchanged.
* **Already covered.** Two items the plan lists here were covered in
  Step 2:
  * `beginSession`'s "changed while locking", by
    `TestBeginChecksDirectoryBeforeReceipt` (macOS);
  * a rollback failure inside managed recovery, by
    `TestManagedReportsKeptBackup`.

**Proofs.** On a scratch copy of the working tree, each mutation broke the
code one test guards, and was run against that test alone. All 17 were
killed; the first failing line of each:

| Mutation | Failure |
| :--- | :--- |
| no retry after the create race | `err = selfupdate: create lock: openat .demo.selfupdate.lock: file exists` |
| no `SameFile` check on the lock | `err = <nil>, want one containing "lock changed while opening"` |
| no regular-file check on the lock | `… open lock: … is a directory, want one containing "lock is not a regular file"` |
| commit skips `checkDir` | `Applied=true err=selfupdate: remove backup: … no such file or directory; want an applied install whose commit was refused` |
| backup removal error ignored | `err = <nil>, want one containing "remove backup"` |
| commit sync error ignored | `err = <nil>, want one containing "injected commit sync failure"` |
| rollback sync error ignored | `err = <nil>, want one containing "injected rollback sync failure"` |
| `copyFile` chmod, sync and close errors ignored (three mutations) | `err = <nil>, want the injected chmod failure` (and sync, close) |
| `copyFile` keeps a partial copy | `a failed copy left …/dst-chmod behind: <nil>` |
| `readTruncated` one byte over | `over the limit: "01234", <nil>` |
| digests not lowercased | `digest "ABAB…" is not lowercase 64-hex` |
| R9 reverted | `dot asset name accepted: "."` |
| `sanitizeText` lets ESC through | `text: control U+001B survived in "a\x1b[31mb?c d?"` |
| strict version grammar skipped | `Validate accepted "v1.2.3+meta"` |
| pending read never cleared | `a read is still outstanding after the answered Confirm` |

After the `lockOpenFlags` refactor, the 17 mutations were run again, and
none survived.

**Checks.**

* **Fuzzing.** Each target ran for 10 s on macOS with no crasher:
  * `FuzzParseSHA256SUMS`, 289,812 execs;
  * `FuzzGitHubReleaseJSON`, 254,750;
  * `FuzzSanitize`, 77,172;
  * `FuzzVersionPolicy`, 606,864.

  No corpus was written under `testdata/`.
* **Coverage.** `go tool cover -func` total, before this step and after:
  85.5% → 86.6%. The round-2 review measured 85.2% on `v1.0.0`. Per
  function, before → after:
  * `readTruncated` 66.7 → 100;
  * `rollbackReplacement` 80 → 100;
  * `openLockFile` 65 → 84;
  * `copyFile` 61.1 → 80;
  * `commitReplacement` 71.4 → 85.7;
  * `syncDirectory` 50 → 75.
* **Pre-add.** `make pre-add-check` passed on the 12 files.
* **Windows test host.**
  * `go vet`, `go test -race`, all script tests and every named test
    passed, including every fuzz seed.
  * `TestManagedCommitRefusesMovedDirectory` logs that Windows refused
    the directory swap, as the Step 2 tests do.

### Step 7: close-out (2026-09-30)

**What changed.**

* **`docs/architecture.md`.**
  * The install-path section now describes the receipt ordering, the
    handle-identity checks, the rollback through the handle, the R1
    report, and the restore on a context the caller's cancellation does
    not reach.
  * It gives the Windows retry bound as the installer's lock timeout, and
    says a read-only destination is not retried.
  * It describes the confirmer's one-line reads.
  * The test-file count is now 31, including the fuzz targets.
* **`InstallOptions.LockTimeout`.** Its doc comment says it also bounds the
  Windows busy-image retry (R4).
* **`doc.go`.** Unchanged: nothing Phase 0 changed contradicts it.
  `Result.PendingBackup`'s new meaning is documented on the field
  (Step 2).

**Verification**, run on the finished tree:

| Check | Result |
| :--- | :--- |
| `go test -race -count=3 ./...` | `ok` |
| `go test -shuffle=on -count=2 ./...` | `ok` |
| `make lint`, for Linux, macOS and Windows | `0 issues.` each |
| `make vuln` | `No vulnerabilities found.` |
| `go mod tidy -diff` | clean |
| `make pre-add-check` on `types.go` | clean |
| `markdownlint-cli2` on `docs/architecture.md` and `docs/README.md` | clean |
| Windows test host | `go vet`, `go test -race` and all four script tests passed |

**Status.** This PLAN stays `in-progress`. Its Verification section also
requires CI to be green on all three operating systems after the push the
owner asks for, including the first run of the new venv step (deviation
D2), and that has not happened yet. It becomes `complete` when that run is
green. `v1.0.1` is not tagged: that needs the owner's explicit ask.
