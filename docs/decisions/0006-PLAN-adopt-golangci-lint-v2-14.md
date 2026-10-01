---
status: in-progress
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
