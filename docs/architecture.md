# Architecture

How `go-core-lib` is put together, as it is now. This file carries no
history and no rationale: the records under [decisions/](decisions/) hold the
argument, and [README.md](README.md) indexes them.

## What it is

A Git repository for the Go module `github.com/maccavelli/go-core-lib`: a
library of general-purpose packages, one per top-level directory, with no
root package and no binary.

The module requires Go 1.27.1 and no other module. It contains no Go package
yet.

## Tree

```text
README.md                   repository entry; links here
LICENSE                     Apache License 2.0
AGENTS.md                   rules for agents: dependencies, records, checks, commits
go.mod                      the module: path and go directive only
Makefile                    development targets (below)
.golangci.yml               golangci-lint configuration
.markdownlint-cli2.jsonc    Markdown lint configuration
.gitattributes              LF line endings on every platform
.github/workflows/ci.yml    CI
scripts/go-precheck.sh      the pre-add check
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json
docs/
  README.md                 record index and the "I want to…" table
  architecture.md           this file
  decisions/                MADR and PLAN records
```

## Go code

None. With no package, `go test ./...`, `go vet ./...`, `golangci-lint` and
`govulncheck` fail with "no packages" (or "no go files to analyze").

## Tooling

- **`make` targets:** `test`, `test-sum`, `fmt`, `vet`, `lint`, `tidy`,
  `vuln`, `pre-add-check`, `help`.
- **`scripts/go-precheck.sh`** runs `gofmt` on the given Go files,
  `golangci-lint run -c .golangci.yml ./...`, `go vet` and `go test` on their
  packages, and `govulncheck ./...`. `make pre-add-check` runs it, and so
  does the machine-wide agent gate before an agent `git commit` that stages
  Go files. With no Go files it prints `go-precheck: no Go files to check.`
  and exits 0.
- **`.golangci.yml`** enables `revive`'s `exported`, `package-comments` and
  `var-naming` rules in place of `golint`. Test files are exempt from
  `errcheck`, `gosec`, `unparam`, `revive`, `gocritic` and `goconst`.
- **CI** (`.github/workflows/ci.yml`) runs on Linux, macOS and Windows, with
  the Go version read from `go.mod`: `go test`; on Linux also `go vet`,
  `gofmt`, `go mod tidy -diff`, `make lint` and `govulncheck` v1.7.0.
  Actions are pinned to commit SHAs.

## What is not here

- **Any package.** The first is `selfupdate`, re-homed from `mcplib` under
  its own record.
- **`docs/reports/` and `docs/guides/`**, which are created with their first
  document.
- **A release.** There is no tag.
