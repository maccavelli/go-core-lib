---
status: in-progress
date: 2026-10-01
associated-madr: "0004-MADR-evolve-selfupdate-api-and-tui-support.md"
---
# Implement harness item H4: an end-to-end update of a running copy

Associated MADR: [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md)

This PLAN implements item H4 of the MADR's §8, "Harness, across every phase":

> An end-to-end test on all three CI operating systems. A copied test binary
> reports its version, H3 serves a "v2" build, and `Updater.Run` drives the
> real standalone installer while the old copy is running. Negative cases
> assert the target is byte-identical afterwards.

It is the last open §8 item. H1–H3 and H5–H7 are done (Phase 0, Phase 1,
Phase 2, and [0004-PLAN-h2-fuzzing-and-manifest-differential.md](0004-PLAN-h2-fuzzing-and-manifest-differential.md)).

## Goal

One test file proves, on Linux, macOS and Windows, the whole update path the
library exists for:

* GitHub discovery through a redirecting second origin, with a token;
* integrity, the executable-image check, and the staged and post-install
  probes;
* the real standalone installer, replacing an executable **that is running**;

and then shows the result the user sees: the target reports the new version,
and the old process runs on and exits cleanly. Each negative case shows the
target byte-identical to the old build afterwards, and the old process
unharmed.

There is no change to the exported API (`make apicheck`), to `go.mod`, or to
CI: the test runs inside `go test` on all three CI legs.

## Scope

### What exists, and what is missing

| Fact | Evidence |
| :--- | :--- |
| Running-image replacement is tested only below `Updater.Run`: a session's `Install` over a helper copy of the test binary. That test accepts either outcome on Windows, a clean commit or a pending backup. | `selfupdate/replace_native_test.go`, `TestNativeReplaceRunningCopy` |
| `TestKeepPreviousRunningImage` (Windows) exercises the rename of a hard link to a running image, again below `Run`. | `selfupdate/lifecycle_windows_test.go` |
| The end-to-end GitHub test replaces a plain temporary file, not a running program, and its comment defers that case to H4. | `selfupdate/e2e_github_test.go`, `TestE2EGitHubRedirectedDownload` |
| The probe tests run the test binary, whose version comes from an environment variable, `SELFUPDATE_TEST_PRINT_VERSION`. So the old and new "versions" are the same bytes. | `selfupdate/replace_native_test.go` `TestMain`; `selfupdate/probe_test.go` |
| Tests already build real executables with `go build` in a temporary module. | `selfupdate/imageverify_test.go`, `buildFixture` |
| `selfupdatetest.GitHubServer` serves a release through a 302 to a second TLS origin, with `RequireToken`, `TruncateAssets` and `RateLimit`. | `selfupdate/selfupdatetest/githubserver.go` |

### What the Windows host showed (2026-10-01)

A scratch experiment ran on the Windows test host, never committed: a
standalone `Install` over a running copy of the test binary, with
`CleanupPending` called before and after the copy exited.

| Moment | Observed |
| :--- | :--- |
| `Install` | `applied=true`, `PendingBackup` set to `.helper.exe.selfupdate-bak-…`, `err=<nil>`; the cleanup receipt is present |
| `CleanupPending` while the old image runs | `selfupdate: remove pending backup: removeat .helper.exe.selfupdate-bak-…: Access is denied.`, and the receipt is kept |
| the old process | exits `<nil>` |
| `CleanupPending` after it exits | `<nil>`; the receipt and the backup are both gone |

So on Windows the test can require a pending backup, the refusal while the
old image runs, and the cleanup after it exits, instead of accepting either
outcome. On Linux and macOS the commit removes the backup at once.

### Item → step

| Item | Step |
| :--- | :--- |
| records; MADR amendment D1; the H2 PLAN's close-out | 1 |
| the helper program and the harness; the positive end-to-end test | 2 |
| the negative cases | 3 |
| documentation and close-out | 4 |

### Out of scope

* **Service-managed updates** (`ManagedInstaller` over a real service
  manager). They need a systemd, launchd or SCM instance, which belongs
  with Phase 4's `selfupdate/service`.
* **Signal delivery to a CLI** (ctrl+c during an update). That is Phase 3's
  `selfupdate/cli`. Phase 2's `TestStreamCancelDuringDownload` covers
  cancellation at the library level.
* **A Windows target that is not named `.exe`.** The asset contract appends
  `.exe` for Windows (`ExactAssetName`).
* **The CleanupPending finding below.** It is recorded, and nothing is
  changed.
* Any `git push` or tag.

### Fixed inputs

| Input | Value |
| :--- | :--- |
| Go | 1.27.1 (`go.mod`); the `go` command is on `PATH` wherever `go test` runs |
| Platforms | the runtime platform of each CI leg: `linux/amd64`, `darwin/arm64` and `windows/amd64`. Each is supported by `NewImageVerifier`. |
| The "foreign" build | the helper built for another OS: `linux` on darwin and windows, `windows` on linux |

### Rules for every step

These are the H2 PLAN's four rules. Every step that changes Go code also
passes on the Windows test host, with `go vet ./...` and
`go test -race -count=1 ./...`.

## Proposed MADR amendment

Step 1 applies it to the MADR's §8, marked *(amended 2026-MM-DD,
0004-PLAN-h4-running-copy-end-to-end)*. Approving this PLAN approves it.

| ID | MADR text today | Amendment | Why |
| :--- | :--- | :--- | :--- |
| D1 | H4: "A copied test binary reports its version, H3 serves a "v2" build" | The test builds a small helper program twice, with the version stamped by `-ldflags -X main.version=…`: `v1.0.0` and `v1.1.0`. The v1 build is copied to the target and started. H3 serves the v2 build. | The version must be a property of the bytes. With the test binary, the version comes from the environment, so v1 and v2 would be the same bytes. A probe could not tell them apart, and "byte-identical to the old build" would hold whether or not anything was replaced. Building twice is already the practice in `imageverify_test.go`. |

## Implementation Steps

### Step 1: records

1. Apply amendment D1 to the MADR's §8.
2. Add this PLAN's row to `docs/README.md`, with status `in-progress`.
3. Close
   [0004-PLAN-h2-fuzzing-and-manifest-differential.md](0004-PLAN-h2-fuzzing-and-manifest-differential.md)
   once CI on its pushed tree is green. Record the run, the four fuzz runs
   and the unskipped differential from the Linux log, and set its index row
   to `complete`. If that CI is not green, this item stops for a deviation.
4. One commit; docs only.

### Step 2: the harness and the positive test (`selfupdate/e2e_running_test.go` (new))

Package `selfupdate_test`.

**The helper program** is a Go source constant in the test file. It is
written, with its own `go.mod`, into a temporary directory and built there,
as `buildFixture` does. It has two modes:

* `--version` prints `demo <version>` and exits 0;
* `serve READY DONE` writes `READY`, then waits until `DONE` exists (polling
  every 20 ms, for at most 60 s), then exits 0.

`version` defaults to `dev`, and the build stamps it.

**The harness:**

* `buildHelper(t, version, goos)` returns the binary's bytes. It is built
  with `CGO_ENABLED=0`, `GOWORK=off` and `GOFLAGS=`, for the runtime
  `GOARCH`.
* `startCopy(t, path)` starts `path serve READY DONE`, waits for `READY`
  (up to 10 s), and returns a handle. `stop()` creates `DONE` and returns
  the process's exit error, at most once. A cleanup stops a copy the test
  left running.
* `newE2E(t, served []byte)` writes the v1 build to the target:
  `demo`, or `demo.exe` on Windows, in an `EvalSymlinks`'d temporary
  directory. It starts the copy, and serves one release, `v1.1.0`, whose
  binary is `served`, through `selfupdatetest.GitHubServer` with
  `RequireToken`. It returns an `Updater` composed of:
  * `NewGitHubSource`, with the server's client and token;
  * a standalone installer with `TargetPolicy{ExecutablePath, AllowedRoots}`;
  * `Verifiers: NewImageVerifier(runtime platform)`;
  * `Probes` and `InstallOptions.PostInstall`, both
    `NewVersionProber([]string{"--version"}, nil, 30*time.Second)`;
  * a `selfupdatetest.RecordingReporter` and `NonInteractiveConfirmer`.
* Every test starts with `selfupdate.CheckNoLeak(t)`.

**`TestE2EUpdateRunningCopy`.** `Run` with `Yes`, `CurrentVersion` `v1.0.0`
and `ReleaseBuild`, while the v1 copy runs.

* `ExitCode` is 0, and `res.Applied` is true.
* The target's bytes equal the v2 build, and `target --version` prints
  `demo v1.1.0`.
* Both probes ran: the recorded events show the run reached `installing`
  and `complete`, and a version prober cannot pass on the v1 bytes. Step 2's
  proofs show this.
* `stop()` returns `<nil>`: the old process ran on and exited cleanly.
* **Linux and macOS:** `res.PendingBackup` is empty, and no
  `.demo.selfupdate-*` file is left.
* **Windows**, as the experiment above showed:
  * `res.PendingBackup` is set and the cleanup receipt exists;
  * `CleanupPending` before `stop()` fails, mentioning
    `remove pending backup`, and keeps the receipt;
  * after `stop()`, it returns nil, and the receipt and backup are gone.

**Proofs** (scratch copies):

| Mutation | Must fail |
| :--- | :--- |
| `apply` returns before `Install`, as a dry run does | the version check (`demo v1.0.0`) |
| the helper's version is not stamped (the `-ldflags` dropped) | the staged probe (`printed "demo dev", want "v1.1.0"`) |
| (Windows host) `commitReplacement` returns the pending backup without writing the receipt | "the receipt exists", and then "the backup is gone after stop" |

### Step 3: the negative cases (`selfupdate/e2e_running_test.go`)

`TestE2ERunningCopyRefusals` runs one subtest per case, each with its own
server, target and running copy. Each asserts:

* the run fails with the named error, and `res.Applied` is false;
* the target's bytes equal the v1 build, and `target --version` prints
  `demo v1.0.0`;
* no staging or backup file is left, and there is no cleanup receipt;
* `stop()` returns `<nil>`.

| Case | How | Error |
| :--- | :--- | :--- |
| the binary does not match `SHA256SUMS` | the release's binary asset holds other bytes than its manifest entry | `ErrIntegrity` |
| the binary is for another OS | the foreign build, with a matching manifest | `ErrIntegrity`, from the image verifier |
| the new binary reports the wrong version | the v1 build served as `v1.1.0` | the staged probe: `failed a probe` |
| the installed binary fails its post-install probe | the v2 build, with `PostInstall` replaced by a prober that always fails | `failed its probe`; `res.RolledBack` is true, and `EventRolledBack` is recorded |
| the download is cut short | `TruncateAssets(true)` | a read error; no `ErrIntegrity` is required, and no file is written |
| the API refuses the token | `RequireToken("other")` | `github http 401` |

The post-install case is the one that exercises the running image twice.
The replacement goes in while v1 runs, then the rollback moves v1 back. On
Windows that rollback renames a hard link to the running image, which
the Windows experiment in [0004-PLAN-v1-1-0-core-api.md](0004-PLAN-v1-1-0-core-api.md),
Step 10, showed works.

**Proofs** (scratch copies):

| Mutation | Must fail |
| :--- | :--- |
| `runVerifiers` returns nil without running the verifiers | the foreign-build case (the target changes) |
| the staged probes are skipped | the wrong-version case |
| the post-install rollback is skipped (`probeInstalled` returns the error without rolling back) | the post-install case (the target holds v2) |
| the integrity check is skipped (`verifyIntegrity` returns nil) | the `SHA256SUMS` case |

### Step 4: documentation and close-out (`docs/architecture.md`, `docs/README.md`, this PLAN)

1. **`docs/architecture.md`, "Tooling":** the end-to-end test, the helper
   build, and the per-OS assertions.
2. **`docs/README.md`:** the row "see the update path proven end to end on
   each OS".
3. **A finding, recorded and not acted on.** While an old image still runs
   on Windows, `CleanupPending`, and therefore any later `Begin`, fails with
   `remove pending backup: … Access is denied`, not `ErrConcurrentUpdate`.
   * Its doc comment names only `ErrConcurrentUpdate` as the benign
     failure.
   * A program that calls `CleanupPending` at startup, as the doc
     suggests, gets this error until the old process exits.
   * Whether to classify it, for example as a new benign sentinel, is a
     decision for a later record.
4. **Verification** is the H2 PLAN's list.
5. **Status.** This PLAN is marked `complete` only after CI is green on the
   pushed tree, with `TestE2EUpdateRunningCopy` passing (not skipped) on all
   three legs.

## Verification

* Every step's proofs fail as listed.
* `make pre-add-check`, `make lint` and `make apicheck` pass.
* `go test -race -count=1 ./...` and `go test -shuffle=on -count=2 ./...`
  pass on macOS.
* The Windows test host passes `go vet ./...` and
  `go test -race -count=1 ./...`, running both new tests.
* **After the owner's push:** CI is green on all three operating systems,
  and the logs show the two new tests passing.

## Rollout and Rollback

* **Rollback.** Each step is one commit and can be reverted alone.
* **CI time.** The helper is built three times per test binary: v1, v2 and
  the foreign build. Each run then takes a few seconds per OS.
* **No tag is required.** Nothing in the Go API changes.

## Execution Record

### Approval (2026-10-01)

The owner approved this PLAN and amendment D1 ("approved proceed").

### Step 1: records (2026-10-01)

* The MADR's H4 bullet gained amendment D1.
* This PLAN was indexed as `in-progress`.
* [0004-PLAN-h2-fuzzing-and-manifest-differential.md](0004-PLAN-h2-fuzzing-and-manifest-differential.md)
  was closed against CI run `36888223078` (green on all three operating
  systems), with the fuzz runs from the Linux log. Its index row reads
  `complete`.

### Step 2: the harness and the positive test (2026-10-01)

**What changed.** `selfupdate/e2e_running_test.go` (new, package
`selfupdate_test`):

* **`helperSource`** is the PLAN's two-mode program. `stampFlag`,
  `-X main.version=`, stamps the version.
* **`buildHelper(t, version, goos)`** builds it in its own temporary module,
  with `CGO_ENABLED=0`, `GOWORK=off`, `GOFLAGS=` and the runtime `GOARCH`,
  and returns the bytes.
* **`startCopy`** runs `serve READY DONE` with its signal files in a
  separate temporary directory, so they never show among the target's
  siblings. It waits up to 10 s for `READY`. `stop()` writes `DONE` and
  returns `Wait`'s error, once, and a cleanup stops any copy left running.
* **`newE2E`** wires the PLAN's composition:
  * the target is `demo` (`demo.exe` on Windows) in an `EvalSymlinks`'d
    directory;
  * one `v1.1.0` release, served through `GitHubServer` with
    `RequireToken("e2e-token")`; `GH_TOKEN` and `GITHUB_TOKEN` are cleared;
  * `NewGitHubSource` with that token, and `NewImageVerifier` for the
    runtime platform;
  * `NewVersionProber(["--version"])` as the staged probe and as
    `PostInstall`, either of which a case can replace;
  * a `RecordingReporter`.
* **`leftovers`** lists `.<base>.selfupdate-*` siblings, which are staging
  and backups. It leaves out the lock file, which stays by design.
* **`TestE2EUpdateRunningCopy`:**
  * the target reports `demo v1.0.0` before the run;
  * the run gives `ExitCode` 0, `Applied`, the v2 bytes,
    `demo v1.1.0`, and both `installing` and `complete` events;
  * then the per-OS branch: a clean commit on Linux and macOS, and on
    Windows the pending backup, the refused cleanup while v1 runs, and the
    cleanup after it exits.

**Proofs:**

| Mutation | Failure |
| :--- | :--- |
| `apply` returns before `Install`, as a dry run does | `ExitCode = 0, res = {… Applied:false …}` |
| the helper's version is not stamped (`-X main.unstamped=`) | `before the update the target reports "demo dev"`. This was caught earlier than the PLAN's staged-probe line, by the pre-run check. |
| (Windows host) `commitReplacement` returns the pending backup without writing the receipt | `pending "…", receipt false; want both while the old image runs` |

**Checks.**

* `go test -race` ran the test in 1.5 s on macOS.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`, with `TestE2EUpdateRunningCopy` passing in
  1.6 s. That run took the Windows branch: the pending backup, the refused
  cleanup, and the cleanup after the old image exited.
* `make pre-add-check` passed on the file.
