---
status: accepted
date: 2026-09-29
decision-makers: go-core-lib maintainers
consulted: mcplib maintainers (the same code ships in mcplib v1.6.0)
informed: owners of the six selfupdate consumers
---
# Fix every debugging-pass finding before tagging v1.0.0

## Context and Problem Statement

On 2026-09-29, after
[0002-PLAN-rehome-selfupdate-from-mcplib.md](0002-PLAN-rehome-selfupdate-from-mcplib.md)
pushed `main` (`3700381`) and before `v1.0.0` was tagged, the owner asked
for "a debugging pass across the codebase. Find bugs. Find gaps. Find
missing wiring or functionality. Write findings as an madr for review".

This record lists what the pass found and proposes how and when to fix it.
The code is `mcplib` `v1.6.0`'s `selfupdate` with the four changes of
0002-MADR §3, so almost every code finding below is also present in `mcplib`
today, where six programs use it.

### Method

* Four read-only reviews ran in parallel, one per area:
  * **A**, network and integrity (`github.go`, `download.go`,
    `checksums.go`, `verify.go`, `assets.go`, `version.go`);
  * **B**, filesystem and install (`target.go`, `lock_*`, `replace*`,
    `cleanup*`, `session.go`, `standalone.go`, `managed.go`);
  * **C**, the coordinator and public API (`updater.go`, `confirmer.go`,
    `reporter.go`, `types.go`, `errors.go`, the examples), plus package
    coverage;
  * **D**, release tooling, CI and documentation.

  Each was measured against the intended contract in `mcplib`
  `docs/0005-PLAN-canonicalize-cli-self-update-in-mcplib.md` §4 and §20.
* Every experiment ran on scratch copies. The repository was not modified,
  and `git status` was clean after each review.
* The author re-ran the evidence on a fresh `git archive` of `004d13b`. That
  covered the reviewers' own probe tests for B and C, the author's own
  probes for A and D1, and direct checks for D3 and D4.

**Evidence** column: **R**, reproduced in that re-run; **R\***, reproduced
by the reviewer only; **C**, confirmed by reading the cited code; **—**,
reasoning only. A finding raised by two reviewers keeps one ID and notes the
other.

Baseline: `go test -race ./...` passes, coverage 75.5 %; lint is clean on
three targets; CI was green on `3700381` for Linux, macOS and Windows.

### High

| ID | Where | Finding | Evidence |
|---|---|---|---|
| B1 | `replace_unix.go:42-45`, `replace_windows.go:49-51`, `session.go:72-75`, `managed.go:84-86` | Suppose the post-rename directory sync fails **and** the rollback rename fails. The rollback error `rerr` is discarded, and `Install` returns `Applied:false` with only the sync error. Meanwhile the new binary is live and the backup leaks. The managed path then calls `recover` with `applyResult{}`, so it does not know a backup exists. | R: `res={… Applied:false} err=selfupdate: sync directory: injected dir sync failure`, `target content="new-bytes"`, `.demo.selfupdate-bak-*` left in the directory |
| B2 | `managed.go:113-131` | Recovery (`Restore`, `Start`, `WaitHealthy`) runs on the caller's `ctx`. When the failure *was* the deadline or a cancellation, the restart fails immediately, and a service that was running is left stopped. | R: `restart attempt (start #2) ran on a cancelled ctx -> left down` |

### Medium

| ID | Where | Finding | Evidence |
|---|---|---|---|
| B3 | `replace.go:41,57-62` | When the hard link fails, `backupFile` falls back to `copyFile`, which creates the backup `0o600` and never restores the mode. A rollback then puts back the old binary without its execute bit. `copyFile` has 0 % coverage. | R: `MODE LOST on rollback: -rwxr-xr-x -> -rw-------` |
| B4 | `session.go:72-73` | `delete(s.staging, path)` runs even when `replaceTarget` failed before the rename. `Close` then never removes the staging file, which is left in the install directory. | R: `LEAK: staging .demo.selfupdate-3052051208 still exists after Close` |
| B5 | `lock_unix.go:51`, `fs_test.go:61-79` | `O_NOFOLLOW` has no effect through `os.Root`: Go 1.27 follows a final-component symlink that stays inside the root. A relative lock symlink is therefore accepted, and flock is taken on its target. The existing test passes only because it uses an absolute link, which `os.Root` refuses as an escape. | R: `ACCEPTED relative lock symlink -> victim; session begun` |
| C1 (= A2) | `updater.go:143-150`, `github.go:171-176` | `--version` is not pinned. Neither `ByTag` nor `Run` checks that the returned tag equals the requested one, so another release is classified and applied. | R: `ByTag(v1.2.3) -> tag="v0.1.0" err=<nil>`; through `Run`: `asked=v1.0.5 installed target=v1.1.0 op=upgrade applied=true err=<nil>` |
| C2 | `updater.go:240-255` | An `Installer` that returns `(InstallResult{Applied:false}, nil)` produces a nil error, exit 0 and no `EventComplete`. The binary is unchanged. | R: `applied=false err=<nil> exit=0 lastEvent=installing` |
| C3 | `updater.go:45,259-260` | `New` accepts nil entries in `Verifiers`. `Run` panics on them after the full download, while holding the lock. | R: `New accepted []Verifier{nil}` / `Run panicked: … nil pointer dereference` |
| A1 | `github.go:241-247,279-301` | `mapRelease` validates **every** asset against the *Executable* limit, and also requires `uploaded` state, a positive size and a sha256 digest. One unrelated extra (a zero-byte file, an in-progress upload, anything larger than the limit) makes the whole release fail for check and apply alike. `SHA256SUMS` is also limited by `Executable`, not `Manifest`. PLAN §4.6 step 4 validates only the selected assets. Latent today: the largest real extra, `magic-cli-remote` `v0.20.0`'s 41 MB APK, is under the 512 MiB default. | R: `asset README.txt has non-positive size`, `big.apk size 1048576 exceeds limit 4096`, `install.sh is not uploaded` |
| A3 | `github.go:141-149` | `checkRedirect` strips the token across origins, but follows a redirect to plain `http` on a non-loopback host. PLAN §4.2: "Any non-loopback source must be HTTPS." For an asset without a GitHub digest, an on-path attacker could swap the binary and `SHA256SUMS` consistently. | R: `checkRedirect(https api -> http foreign) = <nil>` |
| D1 | `scripts/verify-selfupdate-release.sh:145-151` vs `checksums.go:43-49,87` | The publish gate accepts `SHA256SUMS` files that the client's parser rejects. Python `splitlines()` breaks on a lone `\r`, `\v` and similar with no length cap, while Go splits on `\n` with a 4096-byte cap. Such a release is immutable and can never be installed. | R: verifier `ok` for lone-CR, vertical-tab and 5000-byte-comment manifests; Go `want exactly two fields` / `token too long` |
| D4 | `scripts/check-workflow-gh-repo.sh:23-31` | The `GH_REPO` checker misses the step its rule exists for: `refuse-existing-release.sh` calls `gh` inside the script. It also counts a commented `# GH_REPO:`, and a step without `name:` inherits the previous step's state. | R: `GH_REPO` removed from "Refuse an existing release" gives `ok — every repository-scoped gh step sets GH_REPO`, exit 0. R\*: the comment and unnamed-step plants |
| D2 | `0002-MADR-rehome-selfupdate-from-mcplib.md` (Decision Outcome, Consequences, option A) | The MADR says "one deleted line in one consumer". All six consumers pass `bridge-release`, five of them as `false`, and all six must delete it. The guide already says so. A consumer record that follows the MADR would fail. | C: `bridge-release: false` in five consumer `ci.yml` files; MADR text |
| D3 | `docs/guides/migrating-from-mcplib-selfupdate.md` §3 | The guide resolves the pin with `git ls-remote … refs/tags/v1.0.0`. For an annotated tag, which is the fleet's practice, that returns the tag-object SHA, not the commit SHA that `uses:` needs. | R: `refs/tags/v1.6.0` gives `301440dc…` (a `tag` object); `refs/tags/v1.6.0^{}` gives `4e1f9a53…` |
| B8 | `managed_test.go:203-221` | `TestManagedRollbackErrorJoined` cannot fail on a missing join: its assertion is inside `if !errors.Is(err, restoreE) && …`. | C; R\*: passes with the join removed via `-overlay` |

### Low

| ID | Where | Finding | Evidence |
|---|---|---|---|
| C14 | `updater.go:159,249-253` | `EventComplete` is reported before the deferred `sess.Close()` releases the lock. PLAN §4.6 says it follows "the deferred session cleanup attempt", and that success paths call `Close` explicitly. An unlock failure therefore surfaces after "complete" is printed. Found by the author. | C |
| C4 | `updater.go:87-91`, `github.go:255-262` | The release tag is placed in error text unescaped, and the draft and immutability checks run before `versions.Validate`. A control-character tag from a non-GitHub or compromised source reaches the terminal. | R: `hasESC=true hasNL=true` |
| C5 | `updater.go:118-123` | Check mode on a local build returns exit 10 without saying that apply needs `--force` (PLAN §4.1). | R |
| C6 | `confirmer.go:53-67` | EOF (Ctrl-D) at `[y/N]` returns a bare `io.EOF` (exit 1), not a decline. | R: `input="" ok=false err=EOF` |
| C7 | `confirmer.go:51-65` | On cancellation, the scanner goroutine stays blocked and later swallows a line of input. | — |
| C8 | `updater.go:208-225` | `EventTransforming` is emitted before `EventVerified`. | R: `… transforming verified installing` |
| C9 (= A9) | `updater.go:164-190` | `SHA256SUMS` is parsed, and its entry looked up, only after the whole binary is downloaded and staged. PLAN §4.6 step 9 puts that before staging. | R\* |
| C10 | `updater.go:70-253` | Request-validation errors, reporter errors and the post-commit join return unwrapped. PLAN: "Every error wraps its operation and product". | C |
| C11 | `types.go:534-535` | `Config.Limits.ReleaseJSON` and `.ErrorBody` are validated, then never read. Only the source's `Limits` apply. | C |
| C12 | `example_test.go:67-75`, `updater.go:247` | `ExampleNewManagedInstaller` never calls `NewManagedInstaller`, and no example calls `Run` or `ExitCode`. The pending-backup text names a basename where PLAN Phase 4 promises the retained path. | C |
| A5 | `updater.go:96-123` | The coordinator never validates the *selected* assets' state or size itself, so a custom `ReleaseSource` can pass an `open` asset through. | R\* |
| A6 | `download.go:33-35` | `dec.More()` misses trailing `}` or `]` after the JSON document. | R: `decodeJSON("{\"id\":1}}") = <nil>` |
| A7 | `github.go:389-393` | A huge `Retry-After` in seconds overflows `time.Duration` into a negative value. | R: `RetryAfter=-2346317h47m54.709551616s` |
| A8 | `github.go:210-215` | Error bodies are read against the `ReleaseJSON` cap before the status is examined, so a large 429 body hides `RateLimitError`. | R: `response exceeds 4096-byte limit isRateLimited=false` |
| A10 | `github.go:78-86` | `.` and `..` are accepted as owner or repository names. | R: server saw `/repos/../../releases/latest` |
| A11 | `download.go:40-58` | Bidi and format controls (Cf, such as U+202E) pass through into diagnostics. | R\* |
| B6 | `cleanup_windows.go:65-69` | The receipt's backup-name check accepts any bare basename, including the target itself or the lock file, and absolute paths inside the directory. The backup prefix is never required. The precondition, write access to the install directory, already lets an attacker replace the binary, so this is defence in depth. | R (a portable replica of the expression): `"demo" rejected=false`, `".demo.selfupdate.lock" rejected=false` |
| B7 | `cleanup_windows.go:72-90` | A receipt whose backup is already gone (a crash between the two removes, or a manual delete) makes every later `Begin` fail until the receipt is deleted by hand. | — |
| B10 | `session.go:37,132`, `replace_*.go`; `replace_windows.go:58-79` | `os.Root` anchors only the lock and a few `Lstat` calls; staging, backup, rename and removal use absolute paths (PLAN §8 step 2). Windows `moveFileReplace` retries for `DefaultLockTimeout` regardless of `ctx`. | — |
| D5 | `verify-selfupdate-release.sh:101-106`, `publish-selfupdate-release.yml` upload step | Extra names with spaces or glob characters pass the verifier, then break the unquoted `$(find …)` upload *after* the draft exists. The draft blocks every rerun. | R\* |
| D6 | `verify-selfupdate-release.sh:127-131` | A symlinked binary in staging passes the verifier (which follows links) but is skipped by `find -type f`, so the release is missing a canonical binary. | R\* |
| D7 | `refuse-existing-release.sh:33-35` | A missing or unreadable repository reads as "release not found", so the guard proceeds. It is safe while `GH_REPO` is `github.repository`. | R\* |
| D8 | `publish-selfupdate-release.yml:36,40,…` | `${{ github.ref_name }}` is interpolated into shell before the strict-tag check. A tag pusher on the caller repository could inject commands. | R\* |
| D9 | `ci.yml:21,28`, `Makefile:17` | Gaps against the records: CI never runs `-race` (0002-PLAN V3), `shellcheck` or `markdownlint`; and CI pins `golangci-lint@v2.13.1` while the gate and records use 2.13.2. | C |
| D11 | CI history | One push produced two `CI` runs (two check suites, 1 s apart, on the push that first added the workflows). `ci.yml` has no `concurrency:` group. An orphan GitHub App check suite is still `queued`. | R\* (API) |
| D12 | `docs/README.md` "I want to…"; 0002-MADR Consequences | A row about CI failing "until the first package lands" is stale. The 0002 Consequences still say "the first commit names it", contradicting its §2 amendment. | C |

### Test gaps (grouped)

* **A4 — the network path.** These checks can be deleted without any test
  failing: the 403 `X-RateLimit-Remaining: 0` path, the `uploaded` and
  size checks, the `OpenAsset` non-2xx branch, and
  `assetBelongsToRelease`. `RateLimitError.Error` has 0 % coverage (R\*).
* **B9 — PLAN §8 "Required failure tests" missing.**
  * a relative lock symlink; staging collision; a stale backup;
  * malformed, reparse-point and undeletable Windows receipts, and a
    receipt consumed through `Begin`;
  * short-write, sync, close and directory-sync failures;
  * cancellation around staging; a joined rollback failure;
  * managed stop, start, restore and restart failures.

  `replaceTarget` is at 58.8 % coverage, `commitReplacement` at 57.1 %.
* **C13 — coordinator.** Missing: a call-order failure table, and
  `Run`-level tests of rollback, `--force`, local builds, latest-older and
  `--version`. No test uses a successful transformer (`hashFile` and
  `sessOwns` are at 0 %). `TestOverlappingRun` depends on a 20 ms sleep.
* **D10 — the verifier's fixture test** lacks `*`, uppercase, CRLF, lone
  CR, missing `.exe`, empty platforms, symlink, directory, empty-manifest
  and unsafe-extra cases.

### Checked and fine

* `SHA256SUMS` parsing (CRLF, the `*` marker, case, duplicates, traversal,
  line cap) and constant-time digest comparison.
* Token order and cross-origin token stripping.
* The strict version regex; bounded copies.
* Non-TTY apply fails before any download; `--check` with `--yes` or
  `--force` is rejected.
* `ExitCode` 0/10/1; decline is nil and `Declined`; rollback is labelled and
  confirmed.
* `sanitizeText` covers C0, C1, DEL, CR, LF and U+2028.
* The transformer seam is called.
* Reconcile state is restored on the error path.
* `actionlint` v1.7.12 finds nothing in either workflow.

## Decision Drivers

* `v1.0.0` is not tagged and no consumer depends on this module yet. A
  behaviour change now costs nothing downstream; after the tag it is a
  patch or minor release, and consumers pick it up on their own schedule.
* B1 and B2 can leave a machine in a state the library reports wrongly: a
  new binary reported as not applied, or a stopped service. D1 can produce a
  release that is immutable and never installable.
* 0002-MADR promised `v1.0.0` equals `mcplib` `v1.6.0` plus listed changes.
  Fixing before the tag changes that promise, and 0002 must say so.
* The same defects are live in `mcplib` for six programs. Fixing them here
  does not reach those programs until each migrates.

## Considered Options

* **A. Fix every high and medium finding before tagging v1.0.0; fix the low findings and test gaps in v1.0.x/v1.1.0.**
* **B. Fix everything before v1.0.0.**
* **C. Tag v1.0.0 now, at parity with mcplib v1.6.0, and fix in v1.0.1 onward.**
* **D. Fix in mcplib first and re-copy.**

## Decision Outcome

Chosen option: "B. Fix everything before v1.0.0", because the owner decided
so on 2026-09-29: "We will fix them all." That covers all 41 findings and
the four test-gap groups. `v1.0.0` is tagged only when none is open.

*The first draft of this record, the same day, proposed option A: the high
and medium findings before the tag, the rest after. Option A is kept below
as a considered option.*

### What B means

* **One PLAN,** `0003-PLAN-remediate-debugging-pass-findings.md`, fixes
  every item in the tables above. The fixes are grouped by area into phases,
  each ending in a commit.
  * Every behaviour fix lands with a test that is seen to **fail** on the
    pre-fix code, on a scratch copy, and to pass after.
  * A finding that is a test gap is closed by the tests it names. A mutation
    on a scratch copy shows each such test can fail.
  * A finding that is a record or guide error is corrected in place, with a
    dated annotation.
* **The API does not change.** No exported identifier is added, removed or
  re-typed; `go doc -all` differs only in doc comments. Observable behaviour
  does change, and every change is either a tightening or a return to the
  `mcplib` 0005 contract:
  * a tag mismatch, or an installer that reports no commit, becomes an
    error;
  * a nil verifier is rejected by `New`, and an http redirect is refused;
  * a lock or receipt path outside the expected names is refused;
  * EOF at the confirmation prompt is a decline;
  * extras no longer fail a release (a loosening, restoring the contract);
  * events and errors follow PLAN §4.6's order and wrapping.
* **Windows-only fixes** (B6, B7, and the Windows part of B10) are compiled
  and linted here. Where possible, their logic is factored into portable
  functions tested on every OS. ~~What stays Windows-only is proven on the
  Windows CI leg at the next push, and `v1.0.0` waits for that run.~~
  *Amended 2026-09-29, before approval: the owner provided SSH access to a
  Windows test host. Windows tests, fail-first runs included, run there
  before any push. The Windows CI leg at the push confirms them, and
  `v1.0.0` still waits for it.* The host is described in the PLAN's "Fixed
  inputs" by its properties, never by name.
* **0002 is amended.** `v1.0.0` is `mcplib` `v1.6.0` plus 0002 §3 plus this
  record's fixes. 0002-PLAN Phase 6 steps 3–5 wait for this PLAN to be
  `complete`. D2 and D12 correct 0002's own text.
* **Outside the repository.** The orphan GitHub App check suite (D11) is an
  installation setting on the GitHub side. It is reported to the owner, and
  no code change addresses it. `mcplib`'s maintainers get this record's ID
  list; back-porting is their record's decision.

### Consequences

* Good, because `v1.0.0` ships with no known finding from this pass.
* Good, because every fix carries a test proven to fail first, which closes
  B9, C13, A4 and D10 as a side effect.
* Neutral, because the API does not change.
* Bad, because the tag, and `prepare-commit-msg`'s move off `mcplib`, wait
  for the whole PLAN and for the Windows CI leg at the next push.
* Good, because Windows behaviour is proven on a real Windows host, with
  `-race` and symlinks, before anything is pushed, rather than first seen
  in CI.
* Bad, because 0002's "parity with `v1.6.0`" becomes "parity plus fixes",
  and this module's behaviour diverges from `mcplib`'s until `mcplib`
  back-ports or its consumers migrate.
* Bad, because B10's directory-identity check and the Windows receipt
  changes touch the most delicate code in the package, just before the
  first release.

### Confirmation

* Each finding ID maps to a PLAN step and to a named test or record edit.
  The PLAN's execution record shows, for each test, its failure on the
  pre-fix code and its pass after.
* The re-run probes (the B and C reviewer probes, and the author's A and D1
  probes) report the fixed behaviour.
* Lint is clean on three targets, `go test -race` passes, `shellcheck` and
  `markdownlint-cli2` are clean, and CI is green on three operating systems
  (the Windows leg included) before `v1.0.0` is tagged.

## Pros and Cons of the Options

### A. High and medium before the tag, low after

* Good, because the first tag carries no known state-corrupting defect.
* Bad, because low items such as C4 and D8 (both injection-shaped) wait.
  Their preconditions are a compromised source, or write access to the
  caller repository.

### B. Everything before the tag

* Good, because `v1.0.0` would have no known finding.
* Bad, because all 41 findings and four test-gap groups, several needing Windows-only verification
  (B6, B7, B10), hold the tag, and prepare-commit-msg's migration off
  `mcplib`, for longest.

### C. Tag now, fix after

* Good, because the tag and the consumer migrations proceed at once, with
  exact `mcplib` parity.
* Bad, because the first release ships B1, B2 and D1 knowingly, and every
  consumer must move again to get the fixes.

### D. Fix in mcplib first

* Good, because the six current consumers benefit first.
* Bad, because it reverses 0002's direction (the code now lives here), and
  it needs `mcplib`'s own records and a release before anything here moves.

## More Information

* Scope: `selfupdate/`, `scripts/`, `.github/workflows/`, `Makefile`,
  `AGENTS.md`, `README.md` and `docs/` at `004d13b`.
* Evidence files (probe tests and outputs) are in the author's session
  scratch space and are not committed. Each **R** row quotes the output it
  produced.
* Related: [0002-MADR-rehome-selfupdate-from-mcplib.md](0002-MADR-rehome-selfupdate-from-mcplib.md);
  [0002-PLAN-rehome-selfupdate-from-mcplib.md](0002-PLAN-rehome-selfupdate-from-mcplib.md)
  Phase 6; `mcplib`
  `docs/0005-PLAN-canonicalize-cli-self-update-in-mcplib.md` §4, §8 and §20.
