---
status: accepted
date: 2026-09-29
decision-makers: go-core-lib maintainers
consulted: go-llmprovider-sdk and magic-cli-remote repository conventions; mcplib maintainers
informed: fleet consumers of github.com/maccavelli/mcplib/selfupdate
---
# Scaffold go-core-lib as a Go 1.27.1 shared library to the go-llmprovider-sdk standard, with honest gates until the first package lands

## Context and Problem Statement

`go-core-lib` (`https://github.com/maccavelli/go-core-lib`) is a new
repository. Its only commit, `8ebd95e`, adds a one-line `README.md`. The
owner wants it to be the fleet's shared Go library for general-purpose code
that is neither MCP-specific (which stays in `mcplib`) nor LLM-provider code
(which lives in `go-llmprovider-sdk`). Its first code will be `mcplib`'s
`selfupdate` package, re-homed. The owner asked on 2026-09-29 for the
repository to be scaffolded "to the same standards as the other repos, eg:
go-llmprovider-sdk or magic-cli-remote", and "for go v1.27.1 development".

The re-home is recorded elsewhere. `go-llmprovider-sdk`
`docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md`, sixth
amendment, "The owner's further decisions (2026-09-29)", states that
`selfupdate` comes from `github.com/maccavelli/go-core-lib`, "extracted under
that repository's own records". This record covers the scaffold that those
records will build on. The `selfupdate` extraction itself is a separate pair.

Evidence gathered for this record (read-only, 2026-09-29):

* **The closest precedent is `go-llmprovider-sdk`.** It is a Go library carved
  out of `mcplib`, with no binary. It reached its current standard through
  `0002-MADR-migrate-llmprovider-from-mcplib.md` Phase 2 and amendments three
  to five: commits `699e2b8`, `3e764d7`, `21f01c3` and `08de832`. Its scaffold
  is:
  * `AGENTS.md`, holding the normative agent rules: dependencies, the MADR and
    PLAN gate, records, pre-add checks, identifiers and commits;
  * per-agent pointers `.claude/rules/madr-and-plan-skill.md`,
    `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md`,
    and `opencode.json`, plus `.claude/.gitignore`;
  * `.gitignore`, `.markdownlint-cli2.jsonc` (the fleet file, identical in
    13 of 14 fleet repositories per that record's fourth amendment), and
    `.golangci.yml` (golangci-lint v2, with `revive`'s `exported`,
    `package-comments` and `var-naming` rules standing in for `golint`);
  * `Makefile` (`test`, `test-sum`, `fmt`, `vet`, `lint`, `tidy`, `vuln`,
    `pre-add-check`, `help`) and `scripts/go-precheck.sh` (`gofmt`,
    `golangci-lint`, `go vet`, `go test`, `govulncheck`);
  * `.github/workflows/ci.yml`: Linux, macOS and Windows, Go read from
    `go.mod`, actions pinned to commit SHAs;
  * `README.md` linking `docs/README.md`, `docs/architecture.md`, and
    `docs/decisions/` and `docs/reports/`;
  * `go.mod` at `go 1.27.1`. There is no `LICENSE`, by the owner's choice.
    There is no Dependabot configuration, by the owner's decision of
    2026-09-29 ("No dependabot").
* **`magic-cli-remote` adds four things the SDK lacks.** One is a
  `LICENSE`: the Apache License 2.0, 201 lines, with the appendix left as
  its unfilled template. Its MD5 is the one shared by 18 Apache-2.0
  `LICENSE` files in the local Go module cache, the most common copy there.
  Another is a
  `.gitattributes` that forces LF on every platform, because a Windows
  runner's `core.autocrlf=true` broke byte-exact golden tests (its MADR 0116
  P11). The third is `govulncheck` in CI, pinned at `v1.7.0` (commit
  `81c5f3f5`). The fourth is a Dependabot file for GitHub Actions, which the
  owner declined for the SDK. The rest of it is an application and a Flutter
  app, and does not apply to a library.
* **The toolchain is Go 1.27.1.** The installed toolchain is `go1.27.1`.
  `magic-cli-remote` and `go-llmprovider-sdk` both say `go 1.27.1`. The
  fleet rule is `magic-cli-remote`
  `docs/decisions/0169-MADR-standardize-toolchains-on-current-supported-advisory-free-releases.md`
  D2. `golangci-lint` 2.13.2 (built with go1.27.1) and `govulncheck` v1.7.0
  are installed.
* **What `selfupdate` will need** (`mcplib` at `2069912`, `v1.6.0-2`): 26
  non-test and 21 test files, about 6,100 lines, and `testdata/` with two
  `SHA256SUMS` fixtures. Non-standard imports across darwin, linux and
  windows are `golang.org/x/mod/semver`, `golang.org/x/sys/unix`,
  `golang.org/x/sys/windows` and `golang.org/x/term`. It has
  platform-specific files (`*_unix.go`, `*_windows.go`). No other `mcplib`
  package imports it. Six fleet repositories import it, and six call
  `mcplib`'s reusable `publish-selfupdate-release.yml` workflow and its
  scripts.
* **Every gate fails on a module with no packages.** A scratch module
  holding only `module github.com/maccavelli/go-core-lib` / `go 1.27.1`,
  with Go 1.27.1, gave:

  | Command | Exit | Output |
  |---|---|---|
  | `go test ./...` | 1 | `go: warning: "./..." matched no packages` / `no packages to test` |
  | `go vet ./...` | 1 | `no packages to vet` |
  | `golangci-lint run -c .golangci.yml ./...` | 5 | `no go files to analyze` |
  | `govulncheck ./...` | 2 | `no packages matched the provided patterns` |
  | `go mod tidy -diff` | 0 | `warning: "all" matched no packages` |
  | `go mod verify` | 0 | `all modules verified` |

  `go-llmprovider-sdk`'s `scripts/go-precheck.sh` exits 0 before running
  any tool when no Go files are given or tracked. Its CI and `make` targets
  have no such guard.
* **The repository's git setup is ready.** Hooks resolve to
  `~/.global-git-hooks`. A local `user.name` / `user.email` is set. `origin`
  is the GitHub repository above.

## Decision Drivers

* The same standard as the sibling libraries, so that an agent or maintainer
  moving between them finds the same files, rules and commands.
* Go 1.27.1 from the first commit, so that no module directive has to be
  raised later.
* A dependency floor below every other fleet library. Consumers include
  `prepare-commit-msg`, which is dropping `mcplib` entirely. A core library
  that pulled in `mcplib`, the MCP go-sdk or `go-llmprovider-sdk` would
  defeat that.
* Gates that report the truth. A check that passes because it was taught to
  pass on nothing is not a check.
* The `selfupdate` re-home, and later packages, land into a tree that
  already has its rules, so that their records decide only their own
  content.

## Considered Options

* **A. The go-llmprovider-sdk scaffold, adapted, on an empty module; gates stay honest and fail until the first package.**
* **B. As A, plus a placeholder root package** (a `doc.go` declaring `package corelib`) so that every gate has something to check.
* **C. As A, plus "no packages, skip" guards** in CI and the `Makefile`.
* **D. No separate scaffold.** Build the scaffold as the first phases of the `selfupdate` re-home pair.

## Decision Outcome

Chosen option: "A", because it meets the standard now, invents no code and
no guard, and leaves the one red signal (no package yet) pointing at the work
that removes it. The `selfupdate` re-home turns CI green by adding code,
not by changing a gate.

The decision has these parts.

### 1. Identity and scope

* Module path: `github.com/maccavelli/go-core-lib`.
* A library only. No binary, no `cmd/`, no packaging or release targets.
* Scope: general-purpose Go packages shared by fleet programs. MCP code stays
  in `mcplib`. LLM-provider code stays in `go-llmprovider-sdk`.

### 2. Layout

* One top-level directory per capability, with the package named after the
  directory (the first is `selfupdate/`). There is no root package.
* Unexported helpers shared between packages go under `internal/`.
* Documentation follows the fleet tree: `README.md` → `docs/README.md`,
  `docs/architecture.md`, `docs/decisions/`, `docs/reports/`,
  `docs/guides/`. It is one tree, because nothing in the repository is
  adjacent to its purpose. `docs/reports/` and `docs/guides/` are created by
  their first document, not as empty directories.
* Records use one repository-wide `NNNN` sequence starting at `0001` (this
  record).

### 3. Toolchain and dependencies

* `go.mod` says `go 1.27.1` and has no `toolchain` line, as in
  `go-llmprovider-sdk`.
* The scaffold adds no requirement and no `go.sum`. A module is required only
  in the commit that adds its first import, and only after a MADR here names
  it. `golang.org/x/mod`, `golang.org/x/sys` and `golang.org/x/term` are
  expected to be named by the `selfupdate` re-home record. Pinning them
  here would decide that record's content in advance.
* This module never imports `github.com/maccavelli/mcplib`,
  `github.com/modelcontextprotocol/go-sdk` or
  `github.com/maccavelli/go-llmprovider-sdk`. It sits below them in the
  dependency graph.
* `go mod tidy -diff` is clean at every commit. Unlike the SDK during its
  migration, there is no period in which `tidy` must not run.

### 4. Files taken from go-llmprovider-sdk

These are taken with the SDK's wording and changed only where they name the
SDK, its packages, its live tests or its migration records:

* `AGENTS.md`;
* `.claude/.gitignore`, `.claude/rules/madr-and-plan-skill.md`,
  `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md`,
  `opencode.json`;
* `.gitignore` and `.markdownlint-cli2.jsonc` (verbatim);
* `Makefile` (same targets);
* `scripts/go-precheck.sh`, with its provenance comment re-pointed at this
  record.

### 5. Deliberate differences from go-llmprovider-sdk

* **`.golangci.yml` drops one exclusion rule.** The SDK's copy switches off
  twelve linters for any file named `ui.go`, `client.go`, `config.go`,
  `main.go` or one of nine other application filenames. That rule came from
  fleet application repositories. In a library it would silently exempt a
  future `selfupdate/client.go` or `config.go`. Everything else is the SDK's
  file.
* **`.gitattributes` from `magic-cli-remote`**, trimmed to `* text=auto eol=lf`
  plus the binary patterns. CI runs on Windows, and `selfupdate` brings
  `testdata/SHA256SUMS.*` fixtures and POSIX shell scripts, which CRLF would
  break.
* **CI runs `govulncheck`** on Linux, pinned at `v1.7.0` as in
  `magic-cli-remote`. The pre-add gate already runs it, and a library that
  downloads and replaces executables should not have a weaker check in CI
  than on the desk.
* **No `go vet -tags live_gateways` step.** This repository has no live
  tests yet. The record that adds one adds its CI step.

### 6. Gates are not taught to pass on nothing

* CI, `make test`, `make vet`, `make lint` and `make vuln` run unchanged
  against the empty module. They fail with the messages in the table above
  until the first package lands. `docs/architecture.md` and `README.md`
  say so.
* `scripts/go-precheck.sh` keeps its existing "no Go files to check" exit 0.
  That case is about which files are *staged* and predates this record. It
  is not a new guard.
* **The scaffold is not pushed on its own.** Its commits stay local and are
  pushed together with the `selfupdate` re-home, when the owner asks for
  that push (the owner, 2026-09-29: "wait to push scaffold with re-home").
  No commit reaches `origin` while the module has no package, so CI never
  runs on the empty module.

### 7. Licence

* **`LICENSE` is the Apache License 2.0** (the owner, 2026-09-29: "apache 2
  license"). It is `magic-cli-remote`'s file, byte for byte: the canonical
  text with the appendix left as its template. This differs from
  `go-llmprovider-sdk`, which has no licence file.

### 8. Not added

* **No `.github/dependabot.yml`.** This follows the owner's decision for
  `go-llmprovider-sdk`. SHA-pinned actions are updated by hand, as a
  recorded act.
* **No tag or version.** The first release, and whether it is `v0` or `v1`,
  is decided by the `selfupdate` re-home record.
* **No `selfupdate` code, and none of `mcplib`'s release workflow or its
  scripts.** Whether `publish-selfupdate-release.yml`,
  `verify-selfupdate-release.sh`, `refuse-existing-release.sh` and
  `check-workflow-gh-repo.sh` move with the package, and how history is
  imported, is that record's to decide.

### Consequences

* Good, because every fleet convention an agent relies on is present, in the
  same files and words as in `go-llmprovider-sdk`.
* Good, because the module is on Go 1.27.1 with no requirements. The
  `selfupdate` record starts from a clean dependency slate and names every
  module it adds.
* Good, because no placeholder code or skip-guard has to be found and
  removed later, and no gate can go quiet if every package were deleted.
* Neutral, because `.golangci.yml` and `ci.yml` differ from the SDK's in the
  three ways §5 lists, each stated with its reason.
* Good, because the licence is stated from the first published commit, so
  consumers importing the module know its terms.
* Neutral, because CI would fail on the empty module, but under §6 no
  commit reaches `origin` before the first package. The local `make`
  targets still fail with "no packages" until then.
* Bad, because the scaffold commits wait unpushed until the `selfupdate`
  re-home is done, so they exist only in this checkout until then.
* Bad, because `make lint` and `make vuln` exit non-zero until then, so the
  pre-add gate cannot be run end to end on real code until the first
  package. This record's PLAN proves the gate on a scratch copy with a
  planted package instead.

### Confirmation

* Each file in §4 is diffed against its `go-llmprovider-sdk` source, and the
  only differences are those §4 and §5 allow. `LICENSE` is `cmp`-identical
  to `magic-cli-remote`'s.
* `go mod verify` and `go mod tidy -diff` exit 0. `gofmt -l .` is empty.
  `markdownlint-cli2` is clean over the non-record Markdown.
* The empty-module gate failures in the table above are reproduced on the
  committed tree, as expected failures, and recorded in the PLAN.
* `scripts/go-precheck.sh` and `make lint` are seen to **fail** on a scratch
  copy with a planted undocumented exported function and an unformatted
  file, and to **pass** on the same copy with a clean planted package.
* The identifier scan finds no hostname, account name or real-machine
  absolute path in the committed files.

## Pros and Cons of the Options

### A. Honest empty scaffold

* Good, because it meets the standard without code that exists only to
  satisfy a tool.
* Good, because the next record lands into finished rules.
* Bad, because CI would be red on the empty module. The owner's §6 decision
  (push only with the re-home) means it never runs there.

### B. Placeholder root package

* Good, because every gate passes from the first commit.
* Bad, because it publishes an importable `github.com/maccavelli/go-core-lib`
  package with no content. §2 says there is no root package, so the
  placeholder has to be deleted again, or it becomes the decision by
  default.
* Bad, because the green CI proves only that an empty file compiles.

### C. Skip guards when no packages exist

* Good, because CI is green without code.
* Bad, because the guard outlives its reason. If a later change removed
  every package, CI would go green instead of failing. A skipped test is
  reported as a pass, the workaround shape the owner's rules forbid.
* Bad, because it adds a branch that neither sibling library has.

### D. Scaffold inside the `selfupdate` re-home pair

* Good, because CI is never red on a pushed commit.
* Bad, because one record then decides two unrelated things: repository
  standards, and how a package leaves `mcplib`. The standards would be
  amended in a record about `selfupdate`.
* Bad, because the owner asked for the scaffold now, and the re-home needs
  its own investigation first: history import, the reusable workflow, and
  six consumers.

## More Information

* `go-llmprovider-sdk`
  `docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md`: first
  amendment (pre-add gate and agent pointers), third (scaffold to standards,
  no Dependabot), fourth (golangci-lint replaces golint), fifth (`go.mod` at
  1.27.1), sixth ("The owner's further decisions", `selfupdate` to
  go-core-lib).
* `mcplib` `docs/0005-MADR-canonicalize-cli-self-update-in-mcplib.md` and
  `docs/0007-MADR-restore-repository-context-in-the-reusable-release-workflow.md`:
  the design of the code the next record re-homes.
* `magic-cli-remote`
  `docs/decisions/0169-MADR-standardize-toolchains-on-current-supported-advisory-free-releases.md`:
  the fleet toolchain rule.
* Owner's decisions on the proposed draft (2026-09-29): "wait to push
  scaffold with re-home. apache 2 license." They are recorded in §6 and §7.
  The draft had left the push to the owner and had added no licence.
