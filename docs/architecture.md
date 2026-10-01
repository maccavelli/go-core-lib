# Architecture

How `go-core-lib` is put together, as it is now. This file carries no
history and no rationale: the records under [decisions/](decisions/) hold the
argument, and [README.md](README.md) indexes them.

## What it is

A Git repository for the Go module `github.com/maccavelli/go-core-lib`: a
library of general-purpose packages, one per top-level directory, with no
root package and no binary. It also hosts the reusable GitHub Actions
workflow that programs using `selfupdate` publish their releases through.

The module requires Go 1.27.1 and three modules: `golang.org/x/mod v0.40.0`,
`golang.org/x/sys v0.47.0` and `golang.org/x/term v0.43.0`. Its current
release is `v1.0.1`, an annotated tag on commit
`2ec2c6860ce92003e1a66024ce64503280da3f8d`.

## Tree

```text
README.md                   repository entry; links here
LICENSE                     Apache License 2.0
AGENTS.md                   rules for agents: dependencies, records, checks, commits
go.mod, go.sum              the module and its three requirements
Makefile                    development targets (below)
.golangci.yml               golangci-lint configuration
.markdownlint-cli2.jsonc    Markdown lint configuration
.gitattributes              LF line endings, except the byte-exact parity fixtures
.github/workflows/
  ci.yml                    CI
  publish-selfupdate-release.yml   reusable release workflow (workflow_call)
scripts/
  go-precheck.sh            the pre-add check
  verify-selfupdate-release.sh     validates a staged release set
  refuse-existing-release.sh       refuses a tag that already has a release
  check-workflows.sh        parses workflows as YAML: no ${{ }} in a run script,
                            and every repository-scoped gh step sets GH_REPO
  check-api-compat.sh       fails on an incompatible exported API change
                            against the newest v1.* tag (apidiff)
  requirements-workflow-check.txt  hash-pinned PyYAML for check-workflows.sh
  *_test.sh                 offline tests for each of those scripts
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
selfupdate/                 the self-update package
  selfupdatetest/           its exported test doubles
docs/
  README.md                 record index and the "I want to…" table
  architecture.md           this file
  decisions/                MADR and PLAN records
  reports/                  REPORT records
  guides/                   how-to guides
```

## Go code

| Directory | Package | Non-test files | Test files | Non-standard imports |
| :--- | :--- | :--- | :--- | :--- |
| `selfupdate/` | `selfupdate` | 36 | 45, including four fuzz targets, plus `testdata/SHA256SUMS.{valid,invalid}`, 23 `testdata/manifest-parity/` cases and 12 `testdata/golden/` files | `x/mod/semver`, `x/sys/unix`, `x/sys/windows`, `x/term` |
| `selfupdate/selfupdatetest/` | `selfupdatetest` | 2 | 1 | none (`selfupdate` itself) |

- `selfupdate` began as `mcplib` `v1.6.0`'s `selfupdate` (commit
  `4e1f9a53e265`), and its `v1.0.x` API is that package's. It differs from
  that source in:
  - its import path and package comment;
  - four Windows-only lint fixes;
  - the fixes of
    [0003-MADR](decisions/0003-MADR-remediate-debugging-pass-findings.md)
    and of [0004-MADR](decisions/0004-MADR-evolve-selfupdate-api-and-tui-support.md)
    Phase 0;
  - the additive Phase 1 API of that MADR, which `make apicheck` keeps
    compatible with `v1.0.1`.
- The Phase 1 API, by file:
  - **asking without installing:** `checker.go` (`Checker`, shared with
    `Run`'s discovery) and `checkcache.go` (`CheckCached`, `CheckStore`,
    `NewFileCheckStore`);
  - **output:** `jsonreporter.go`, `document.go` (`Result.Document`),
    `adapters.go` (the `…Func` adapters, `DiscardReporter`,
    `MultiReporter`, `NonInteractiveConfirmer`), and the progress and
    outcome events in `updater.go`;
  - **credentials:** `credentials.go`, with the lazy credential chain and
    its cross-origin stripping in `github.go`;
  - **integrity and probes:** `manifestverify.go`, `imageverify.go`
    (ELF, Mach-O, PE), and `probe.go` (staged and post-install probes);
  - **installers:** `TwoPhaseSession`, `StagingOwner` and
    `NewManagedInstallerFor` (`types.go`, `session.go`, `managed.go`), and
    `DryRun`, `KeepPrevious` and `CleanupPending` (`updater.go`,
    `session.go`, `standalone.go`).
- `selfupdatetest` provides `NewRelease`, `FakeSource`,
  `RecordingReporter`, `ScriptedConfirmer`, and `GitHubServer`, a fake
  GitHub API on one TLS origin whose asset requests redirect to a second.
- The coordinator (`updater.go`) owns the order of every step. It validates
  the selected binary and manifest itself, and parses `SHA256SUMS` before any
  staging. It pins an exact `--version`, and closes the session before
  reporting `complete`.
- The install path (`session.go`, `replace_*.go`, `lock*.go`, `cleanup*.go`,
  `managed.go`):
  - locks the target directory through `os.Root`, refusing a symlinked lock
    (`openLockFile`);
  - checks that the locked directory is the one at the path before it reads
    a cleanup receipt, and re-checks it, by the handle's identity, before
    the replace, after it and before commit;
  - undoes the rename through the directory handle when the directory
    changed after it;
  - refuses a staging path that is not a regular file;
  - reports a failed restore with the backup's path, in
    `Result.PendingBackup` with `Applied` false, and in the error;
  - runs the restore after a failed directory sync, and managed recovery,
    on contexts the caller's cancellation does not reach.

  On Windows, a busy running image is retried until the installer's lock
  timeout or the caller's context ends. An access-denied error on a
  read-only destination is not retried. A cleanup receipt may name only a
  backup of its own target, which is hashed and removed through the
  directory handle.
- The terminal confirmer reads one line per answer, a byte at a time, and
  leaves no read outstanding once a prompt is answered.
- Platform code is split by build tag: `*_unix.go` (`//go:build unix`),
  `*_windows.go`, and `cleanup_other.go` for non-Windows receipt handling.
- The package's GitHub `User-Agent` is supplied by the program. It reads
  `GH_TOKEN`, then `GITHUB_TOKEN`, when set.

## Release workflow

`publish-selfupdate-release.yml` is called with `artifact-name`,
`products-json`, `platforms-json` and `extra-assets-json`. On a strict
`vMAJOR.MINOR.PATCH` tag, in order, it:

1. checks out its own commit at `.core-lib-release-tools`, to run the scripts
   above from the same commit as the workflow;
2. refuses an existing release (`refuse-existing-release.sh`), after
   proving the repository itself is readable;
3. downloads the caller's artifact to `staging/`;
4. validates the staged set (`verify-selfupdate-release.sh`): regular files
   only, safe extra names, and a `SHA256SUMS` parsed exactly as the client
   parses it. Both parsers run the fixtures in
   `selfupdate/testdata/manifest-parity/`;
5. creates a draft, uploads the files (one argument each), attests them,
   publishes, and waits for the release to be immutable and verified.

Every `gh` step that acts on the calling repository sets `GH_REPO`. Every
`run:` block reads the ref from `env:` (`TAG`, `REF_TYPE`); no `${{ }}` is
interpolated into shell.

## Tooling

- **`make` targets:** `test`, `test-sum`, `fmt`, `vet`, `lint`, `tidy`,
  `vuln`, `apicheck`, `pre-add-check`, `help`.
- **`make apicheck`** runs `scripts/check-api-compat.sh`: the pinned
  `apidiff` compares the working tree with the newest `v1.*` tag (or
  `BASE=`), and any incompatible change fails it.
- **`make lint`** runs `golangci-lint run -c .golangci.yml ./...` three
  times: `GOOS=linux`, `darwin` and `windows`, each with `CGO_ENABLED=0`.
- **`scripts/go-precheck.sh`** runs `gofmt` on the given Go files, the same
  three golangci-lint runs, `go vet` and `go test` on their packages, and
  `govulncheck ./...`. `make pre-add-check` runs it, and so does the
  machine-wide agent gate before an agent `git commit` that stages Go files.
- **`.golangci.yml`** enables `revive`'s `exported`, `package-comments` and
  `var-naming` rules in place of `golint`. Test files are exempt from
  `errcheck`, `gosec`, `unparam`, `revive`, `gocritic` and `goconst`.
- **CI** (`.github/workflows/ci.yml`) runs on Linux, macOS and Windows, with
  the Go version read from `go.mod`.
  - **Every OS:** `go test`, plus, under bash, the refuse-existing-release
    test.
  - **Linux and macOS:** `go test -race`.
  - **Linux also:** a full-history checkout; `go test -shuffle=on -count=2`;
    `go vet` for `freebsd/amd64`, `openbsd/amd64` and `linux/386`;
    `go vet`, `gofmt`, `go mod tidy -diff`, `make lint` (golangci-lint
    v2.13.2); `make apicheck` and the gate's own test; `govulncheck` v1.7.0;
    `shellcheck` v0.11.0 (the latest release, pinned by SHA-256 and first
    on `PATH`, so actionlint's embedded checks use it too),
    `markdownlint-cli2` 0.23.2 and `actionlint` v1.7.12; the verifier's
    fixture test; and the workflow checker and its test.
  - One run per ref (`concurrency`, cancel in progress). Actions are pinned
    to commit SHAs.

## What is not here

- **Any consumer's migration,** and `mcplib`'s deprecation of its own copy.
  Each is recorded in that repository.
- **Release signing.** No publisher signature is verified; the
  `ManifestVerifier` hook is where one would be.
