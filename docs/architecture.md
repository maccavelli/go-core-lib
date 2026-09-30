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
`golang.org/x/sys v0.47.0` and `golang.org/x/term v0.43.0`.

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
  check-workflow-gh-repo.sh        asserts every repository-scoped gh step sets GH_REPO
  check-workflow-expressions.sh    asserts no ${{ }} is interpolated into a run block
  *_test.sh                 offline tests for each of those scripts
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
selfupdate/                 the self-update package
docs/
  README.md                 record index and the "I want to…" table
  architecture.md           this file
  decisions/                MADR and PLAN records
  guides/                   how-to guides
```

## Go code

| Directory | Package | Non-test files | Test files | Non-standard imports |
| :--- | :--- | :--- | :--- | :--- |
| `selfupdate/` | `selfupdate` | 27 | 25, plus `testdata/SHA256SUMS.{valid,invalid}` and 23 `testdata/manifest-parity/` cases | `x/mod/semver`, `x/sys/unix`, `x/sys/windows`, `x/term` |

- `selfupdate` is `mcplib` `v1.6.0`'s `selfupdate` (commit `4e1f9a53e265`),
  with the same exported API. It differs from that source in:
  - its import path and package comment;
  - four Windows-only lint fixes;
  - the fixes of
    [0003-MADR](decisions/0003-MADR-remediate-debugging-pass-findings.md).
- The coordinator (`updater.go`) owns the order of every step. It validates
  the selected binary and manifest itself, and parses `SHA256SUMS` before any
  staging. It pins an exact `--version`, and closes the session before
  reporting `complete`.
- The install path (`session.go`, `replace_*.go`, `lock*.go`, `cleanup*.go`,
  `managed.go`):
  - locks the target directory through `os.Root`, refusing a symlinked lock
    (`openLockFile`);
  - re-checks the directory's identity before the replace and before commit;
  - refuses a staging path that is not a regular file;
  - reports a failed restore together with the live backup;
  - runs managed recovery on a context of its own.

  On Windows, a busy running image is retried until `DefaultLockTimeout` or
  the caller's context ends. A cleanup receipt may name only a backup of its
  own target.
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
  `vuln`, `pre-add-check`, `help`.
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
    test, both workflow checkers and their tests.
  - **Linux also:** `go test -race`, `go vet`, `gofmt`, `go mod tidy -diff`,
    `make lint` (golangci-lint v2.13.2), `govulncheck` v1.7.0, `shellcheck`,
    `markdownlint-cli2` 0.23.2, `actionlint` v1.7.12, and the verifier's
    fixture test.
  - One run per ref (`concurrency`, cancel in progress). Actions are pinned
    to commit SHAs.

## What is not here

- **A tag.** `v1.0.0` is the first.
- **Any consumer's migration,** and `mcplib`'s deprecation of its own copy.
  Each is recorded in that repository.
- **`docs/reports/`,** which is created with its first report.
