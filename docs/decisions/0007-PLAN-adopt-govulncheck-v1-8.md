---
status: complete
date: 2026-10-01
associated-madr: "0007-MADR-adopt-govulncheck-v1-8.md"
---
# Implement govulncheck v1.8.0 across CI, hints and hosts

Associated MADR: [0007-MADR-adopt-govulncheck-v1-8.md](0007-MADR-adopt-govulncheck-v1-8.md)

## Goal

Every place the vulnerability gate runs uses `govulncheck` v1.8.0: CI, the
install hints, and the four development hosts the MADR names.

## Scope

### In scope

* `.github/workflows/ci.yml`: `govulncheck@v1.7.0` becomes `@v1.8.0`.
* `Makefile` and `scripts/go-precheck.sh`: the install hints name
  `govulncheck@v1.8.0` instead of `@latest`.
* `docs/architecture.md`: v1.7.0 becomes v1.8.0.
* The binaries on the four hosts. These are outside the tree; the record
  names them by role only.

### Out of scope

* magic-cli-remote, ocp-login and every other fleet repository.
* The three WSL distributions with no Go.
* The historical records 0001–0006, which keep the versions they recorded.
* `git push`. The owner pushes.

## Implementation Steps

### Step 1: the files

1. Edit the three pins and the architecture line. Assert each edit landed
   with `git grep`. The only `govulncheck@` strings outside historical
   records are `@v1.8.0`.
2. `shellcheck scripts/go-precheck.sh` and actionlint v1.7.12 are clean.
3. A pin cannot be shown to fail on a broken input the way a test can. Its
   check is that CI installs and runs the pinned version: Verification V3.

### Step 2: the hosts

For each host: the macOS development host, the Linux host, the Windows test
host, and that host's default WSL distribution:

1. List every `govulncheck` on `PATH`, with its version
   (`which -a` / `where.exe`). Record the count and versions, redacted to
   roles and `<user>` paths.
2. Run `GOBIN=<dir> go install golang.org/x/vuln/cmd/govulncheck@v1.8.0`
   with that host's Go 1.27.1, once for each directory step 1 found.
3. Repeat step 1. Every copy must report `Scanner: govulncheck@v1.8.0`.
4. Run `govulncheck ./...` once on a go-core-lib tree on that host. On the
   Windows host and its WSL distribution, use a scratch copy, as the Windows
   gate does. It must report `No vulnerabilities found.`

Nothing is deleted. A shadowed copy is upgraded in place, never removed.

### Step 3: checks and commit

1. `make vuln` and `make pre-add-check` pass on the macOS development host.
2. Write the execution record: each host's before-and-after versions, by
   role, and the check outputs.
3. Commit the three files, `docs/architecture.md` and this PLAN with
   `git commit --no-edit`, after the owner authorizes a commit to `main` in
   that turn.

## Verification

* **V1.** `git grep -n 'govulncheck@'` outside historical records shows
  only `v1.8.0`.
* **V2.** Every `govulncheck` on `PATH`, on all four hosts, reports
  v1.8.0, and finds no vulnerabilities in go-core-lib.
* **V3.** After the owner's push, CI's govulncheck step installs v1.8.0 and
  passes.

## Rollout and Rollback

* **Rollout.** Step 2 changes the hosts at once. The files reach CI when the
  owner pushes.
* **Rollback.** Revert the commit. On a host, run the same `go install` with
  `@v1.7.0` and the same `GOBIN`.

## Execution Record

### Approval (2026-10-01)

The owner approved this PLAN: "approved, proceed, commit to main".

### Step 1: the files (2026-10-01)

* `ci.yml` installs `govulncheck@v1.8.0`. The `Makefile` and
  `scripts/go-precheck.sh` hints name `@v1.8.0`, and
  `docs/architecture.md` names v1.8.0.
* `git grep -n 'govulncheck@'` outside records 0001–0007 lists exactly
  those three `@v1.8.0` lines (V1).
* `shellcheck scripts/go-precheck.sh` and actionlint v1.7.12 exit 0.

### Step 2: the hosts (2026-10-01)

One script ran on each host, from a scratch go-core-lib tree:

* it listed every `govulncheck` on the login `PATH`, plus `$HOME/go/bin`
  and `$HOME/.local/bin`;
* it ran `GOBIN=<dir> go install golang.org/x/vuln/cmd/govulncheck@v1.8.0`
  once per directory found;
* it listed the copies again;
* it ran `CGO_ENABLED=0 govulncheck ./...`.

Its output was redacted to `<user>` paths before it was read. The tree was
removed afterwards.

| Host | Copies before | After | Scan |
| :--- | :--- | :--- | :--- |
| the macOS development host | 1, `$HOME/go/bin`, v1.7.0 | v1.8.0 | `No vulnerabilities found.`, rc 0 |
| the Linux host | 2, `$HOME/.local/bin` (first on `PATH`) and `$HOME/go/bin`, both v1.7.0 | both v1.8.0 | `No vulnerabilities found.`, rc 0 |
| the Windows test host | 1, `%USERPROFILE%\go\bin`, v1.7.0 | v1.8.0 | `No vulnerabilities found.`, rc 0 |
| its default WSL distribution (Ubuntu 24.04) | 1, `$HOME/go/bin`, v1.7.0 | v1.8.0 | `No vulnerabilities found.`, rc 0 |

Each host's Go was 1.27.1. The Linux host's second copy was shadowed by the
first. It was upgraded in place, as Step 2 says, and not removed. V2
holds.

The read-only probe before this PLAN started the Windows host's three other
WSL distributions, which have no Go. Nothing was installed in them.

### Step 3: checks (2026-10-01)

* On the macOS development host, `make vuln` gave `No vulnerabilities found.`
  (rc 0), with `Scanner: govulncheck@v1.8.0`.
* `make pre-add-check` gave `go-precheck: 97 file(s) clean (gofmt,
  golangci-lint, go vet, go test, govulncheck).`
* markdownlint-cli2 0.23.2 reported 0 issues.
* V3, CI on the pushed change, waits for the owner's push. This PLAN stays
  `in-progress` until then.

### Close-out (2026-10-01)

The owner pushed `ef05dfe`. CI run `36945362133` on `ef05dfec469f`
concluded `success` on `ubuntu-24.04`, `windows-2025` and `macos-15`. Its
govulncheck step ran `go install golang.org/x/vuln/cmd/govulncheck@v1.8.0`
and reported `No vulnerabilities found.` (V3). V1–V3 hold, so this PLAN is
`complete`.
