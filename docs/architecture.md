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
.gitattributes              LF line endings on every platform
.github/workflows/
  ci.yml                    CI
  publish-selfupdate-release.yml   reusable release workflow (workflow_call)
scripts/
  go-precheck.sh            the pre-add check
  verify-selfupdate-release.sh     validates a staged release set
  refuse-existing-release.sh       refuses a tag that already has a release
  check-workflow-gh-repo.sh        asserts every repository-scoped gh step sets GH_REPO
  *_test.sh                 offline tests for the two release scripts
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
| `selfupdate/` | `selfupdate` | 26 | 21, plus `testdata/SHA256SUMS.{valid,invalid}` | `x/mod/semver`, `x/sys/unix`, `x/sys/windows`, `x/term` |

- `selfupdate` is `mcplib` `v1.6.0`'s `selfupdate` (commit `4e1f9a53e265`),
  with the same exported API. It differs from that source in its import path,
  its package comment, and four Windows-only lint fixes.
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
2. refuses an existing release (`refuse-existing-release.sh`);
3. downloads the caller's artifact to `staging/`;
4. validates the staged set (`verify-selfupdate-release.sh`);
5. creates a draft, uploads the files, attests them, publishes, and waits
   for the release to be immutable and verified.

Every `gh` step that acts on the calling repository sets `GH_REPO`.

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
  the Go version read from `go.mod`: `go test`, and the refuse-existing-release
  test and workflow `GH_REPO` check under bash. On Linux it also runs
  `go vet`, `gofmt`, `go mod tidy -diff`, `make lint`, `govulncheck` v1.7.0,
  and the staged-release verifier's fixture test. Actions are pinned to
  commit SHAs.

## What is not here

- **A tag.** `v1.0.0` is the first.
- **Any consumer's migration,** and `mcplib`'s deprecation of its own copy.
  Each is recorded in that repository.
- **`docs/reports/`,** which is created with its first report.
