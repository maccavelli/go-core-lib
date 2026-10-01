---
status: complete
date: 2026-10-01
associated-madr: "0006-MADR-adopt-golangci-lint-v2-14.md"
---
# Implement the golangci-lint v2.14.0 adoption

Associated MADR: [0006-MADR-adopt-golangci-lint-v2-14.md](0006-MADR-adopt-golangci-lint-v2-14.md)

## Goal

CI, the pre-add script and the docs name `golangci-lint` `v2.14.0`, and
`make lint` with it is clean on `linux`, `darwin` and `windows`, with no new
exemption.

## Scope

* `selfupdate/target_test.go`: one line, the Windows filesystem root.
* `.github/workflows/ci.yml`: the pin.
* `scripts/go-precheck.sh`: the install hint.
* `docs/architecture.md`: the version named in "Tooling".
* **Not in scope:** any production code, `.golangci.yml`, and the paused
  [0005-PLAN-opt-in-prerelease-channels.md](0005-PLAN-opt-in-prerelease-channels.md).
  That PLAN's uncommitted Step 2 work stays in the tree, untouched, and is
  not staged by this PLAN's commits.

## Implementation Steps

### Step 1: records

The MADR and this PLAN, and their `docs/README.md` rows. One commit; docs
only.

### Step 2: the taint source (`selfupdate/target_test.go`)

1. Replace `root = os.Getenv("SystemDrive") + "\"` with
   `root = filepath.VolumeName(t.TempDir()) + "\"`.
   * It is the same filesystem root on the drive that holds the test's
     temporary directory, which is the system drive on CI and on the test
     host.
   * Drop the `os` import if it is no longer used.
2. **Proofs**, on scratch copies:
   * the old line makes v2.14 report G703 at `target.go:127`, and the new
     line reports `0 issues`;
   * with `isFilesystemRoot` returning false,
     `TestCanonicalizeRootRejectsFilesystemRoot` fails locally (`/`) and on
     the Windows test host (the new root).
3. `make pre-add-check` on the file, now with v2.14. Then the Windows test
   host run, and one commit.

### Step 3: the pin (`ci.yml`, `scripts/go-precheck.sh`, `docs/architecture.md`)

1. Change `golangci-lint@v2.13.2` to `@v2.14.0` in `ci.yml` and in the
   pre-add script's install hint. In `docs/architecture.md`, change the
   version to `v2.14.0`.
2. **Checks:**
   * `actionlint`, `check-workflows.sh` (both rules), `shellcheck`;
   * `make lint` with v2.14.0 on all three targets;
   * a repository-wide `grep` showing no `v2.13.2` outside the historical
     records.
3. One commit.

### Step 4: close-out

After the owner's push, CI's lint step runs `v2.14.0` and the run is green.
Record the run, and mark this PLAN `complete`. Then resume the 0005 PLAN at
its Step 2.

## Verification

* The proofs in Step 2.
* `make lint` with v2.14.0, `make apicheck` and `go test ./...` pass.
* CI is green, with the lint step's log naming `v2.14.0`.

## Rollout and Rollback

Revert Step 3 to return CI to `v2.13.2`. Step 2 is correct under either
version.

## Execution Record

### Approval (2026-10-01)

The owner approved this PLAN ("proceed"), having chosen option A, "Pin CI
to v2.14 too".

### Step 1: records (2026-10-01)

The MADR (`accepted`) and this PLAN (`in-progress`) were written and
indexed.

### Step 2: the taint source (2026-10-01)

**What changed.** In `TestCanonicalizeRootRejectsFilesystemRoot`, the
Windows root is now `filepath.VolumeName(t.TempDir()) + "\"`, with a comment
saying why it is not `%SystemDrive%`. The `os` import stays, because the
file uses it elsewhere.

**Proofs.**

* **G703.**
  * A clean clone of the parent commit, under v2.14.0, reported
    `selfupdate/target.go:127:23: G703: Path traversal via taint analysis
    (gosec)` on `linux`, `darwin` and `windows`.
  * With this change, `make lint`'s three runs each report `0 issues`. That
    is on the working tree, which also holds the paused 0005 Step 2 work.
* **The test still guards the refusal.** With `isFilesystemRoot` returning
  false, it fails with `target_test.go:77: accepted filesystem root`:
  * locally, for `/`;
  * on the Windows test host, for the new root.

**Checks.**

* `make pre-add-check` passed on the file, with v2.14.0.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`, with the test passing.

### Step 3: the pin (2026-10-01)

**What changed.** `golangci-lint@v2.13.2` became `@v2.14.0` in:

* `.github/workflows/ci.yml`, the "vet, gofmt, tidy, lint" step;
* `scripts/go-precheck.sh`, the install hint;
* `docs/architecture.md`, "Tooling".

**Checks.**

* `actionlint` is clean, and so is `check-workflows.sh` (all rules, and
  `--rule expressions` on `ci.yml`). `shellcheck scripts/*.sh` is clean.
* `make lint` with v2.14.0: `0 issues` on `linux`, `darwin` and `windows`.
* A repository-wide `grep` for `v2.13.2` finds it only in records:
  * the 0003 PLAN's history of the pin (lines 420 and 1105);
  * this PLAN's and its MADR's own account.

  No live configuration names the old version.

### Step 4: close-out (2026-10-01)

The owner pushed Steps 1–3, together with the paused 0005 Step 2 code, which
the owner committed as `4c271e0`. CI run `36922093599` on `4c271e038604`
concluded `success` on `ubuntu-24.04`, `macos-15` and `windows-2025`.

The Linux log shows
`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0`,
then `0 issues.` for `GOOS=linux`, `darwin` and `windows`.

Every criterion is met, so this PLAN is `complete`.
[0005-PLAN-opt-in-prerelease-channels.md](0005-PLAN-opt-in-prerelease-channels.md)
resumes.
