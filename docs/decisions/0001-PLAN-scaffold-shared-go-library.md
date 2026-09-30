---
status: in-progress
date: 2026-09-29
associated-madr: "0001-MADR-scaffold-shared-go-library.md"
---
# Implement the go-core-lib Go 1.27.1 library scaffold

Associated MADR: [0001-MADR-scaffold-shared-go-library.md](0001-MADR-scaffold-shared-go-library.md)

## Goal

Bring `go-core-lib` to the standard of `go-llmprovider-sdk`: agent rules,
lint and pre-add tooling, CI, a Go 1.27.1 module and a documentation tree.
Adapt it as the MADR's §4 and §5 say, so that the `selfupdate` re-home pair
can land code into it without deciding anything about the repository itself.

Done means every item under Verification holds, and the execution record
below carries the output.

## Scope

### In scope (this repository only)

| Path | Source | Change from source |
|---|---|---|
| `AGENTS.md` | SDK | Rewritten intro and Dependencies section. No live-test section, and no migration or tidy embargo. Record citations point here. |
| `.claude/.gitignore` | SDK | none |
| `.claude/rules/madr-and-plan-skill.md` | SDK | none |
| `.grok/rules/madr-plan-before-mutating-work.md` | SDK | none |
| `.opencode/rules.md` | SDK | none |
| `opencode.json` | SDK | none |
| `.gitignore` | SDK | none |
| `LICENSE` | `magic-cli-remote` | none (Apache License 2.0, MADR §7) |
| `.gitattributes` | `magic-cli-remote` | Comment rewritten for this repository. Binary list trimmed to image, archive, font and PDF patterns. |
| `.markdownlint-cli2.jsonc` | SDK | none |
| `.golangci.yml` | SDK | The filename-based exclusion rule (`path: (ui\.go\|…\|kibana\.go)`) is removed. |
| `Makefile` | SDK | Header comment only |
| `scripts/go-precheck.sh` | SDK | Provenance comment only; mode `0755` |
| `.github/workflows/ci.yml` | SDK | `live_gateways` vet step removed; `govulncheck@v1.7.0` step added on Linux |
| `go.mod` | new | `module github.com/maccavelli/go-core-lib`, `go 1.27.1`, nothing else |
| `README.md` | replaces the one-line file | Real content |
| `docs/README.md` | new | Record index and "I want to…" table |
| `docs/architecture.md` | new | The tree as it is at Phase 3's commit |
| `docs/decisions/0001-*` | this pair | Status and execution record |

"SDK" means `go-llmprovider-sdk` at `bc76ddd`, read from the sibling
checkout, never from memory.

### Out of scope

* Any Go source, test or `testdata/`. The `selfupdate` re-home is the next
  greenfield pair, `0002`, written after this PLAN is complete.
* `go.sum` and any `require` line.
* `mcplib`'s reusable release workflow and its scripts.
* `.github/dependabot.yml`, `docs/reports/`, `docs/guides/`.
* Any change in `mcplib`, `go-llmprovider-sdk` or a consumer.
* `git push` and tags. Under MADR §6 the scaffold commits are pushed only
  together with the `selfupdate` re-home, and then only when the owner asks
  in the same turn.

### Fixed inputs

* Toolchain `go1.27.1`; `golangci-lint` 2.13.2; `govulncheck` v1.7.0;
  `markdownlint-cli2` and `shellcheck` on `PATH`.
* Hooks path resolves to `~/.global-git-hooks`, which was checked on
  2026-09-29.

## Implementation Steps

Every phase ends with the phase's checks passing, then `git add` of exactly
the phase's paths, then `git commit --no-edit`. The global
`prepare-commit-msg` hook writes the message. No `-m`, `-F` or `--amend`.

### Phase 0: accept the records

1. The owner approves the MADR and this PLAN.
2. Set the MADR `status: accepted` and this PLAN `status: in-progress`, and
   update `date:`. Record the approval, quoted, in the execution record.
3. Commit the two records alone. This is the bootstrap exception: no other
   path is in this commit.

### Phase 1: agent rules and repository hygiene

1. Copy `.claude/.gitignore`, `.claude/rules/madr-and-plan-skill.md`,
   `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md`,
   `opencode.json`, `.gitignore` and `.markdownlint-cli2.jsonc` from the SDK
   byte for byte. Assert this with `cmp` on each file.
2. Copy `LICENSE` from `magic-cli-remote` byte for byte, and assert it
   with `cmp`. Leave the appendix template unfilled, as in the source.
3. Write `.gitattributes`: `* text=auto eol=lf` and the trimmed binary
   patterns, with a comment giving the reason (Windows CI, `SHA256SUMS`
   fixtures, shell scripts).
4. Write `AGENTS.md` from the SDK's, keeping the sections and their
   normative wording:
   * **Intro.** Describe `go-core-lib` (`github.com/maccavelli/go-core-lib`):
     a library only, one directory per capability, no root package,
     Go 1.27.1.
   * **Dependencies.** No module without a MADR here. Never `mcplib`, the
     MCP go-sdk or `go-llmprovider-sdk`. `go.mod`/`go.sum` change with the
     code that needs them, and `go mod tidy -diff` is clean at every commit.
   * **MADR and PLAN before mutating work, and Records.** Unchanged, except
     that the renumbering bullet about `mcplib` records is dropped.
   * **Pre-add checks.** Unchanged, plus one sentence: until the first
     package lands, `make lint`, `make vet`, `make test` and `make vuln`
     fail with "no packages", as MADR §6 records.
   * **Live tests.** Dropped. The first record that adds live tests adds the
     section.
   * **Identifiers and Commits.** Unchanged.
5. Checks: `cmp` for the verbatim files; `diff` of `AGENTS.md` against the
   SDK's, read in full, showing only the changes in step 4;
   `markdownlint-cli2 AGENTS.md`; the identifier scan (Verification V6).
6. Commit.

### Phase 2: Go module, lint, pre-add gate and CI

1. Write `go.mod` as three lines: `module github.com/maccavelli/go-core-lib`,
   a blank line, `go 1.27.1`. Run `go mod verify` and `go mod tidy -diff`;
   both exit 0. Assert that `go.mod` is byte-identical after tidy, and that
   no `go.sum` appears.
2. Copy `.golangci.yml` from the SDK, and remove only the exclusion rule
   whose `path:` is the application-filename list. Assert with `diff` that
   this is the only change.
3. Copy `Makefile`, and change only its header comment to name
   `go-core-lib`.
4. Copy `scripts/go-precheck.sh` and keep mode `0755`. Change only the
   provenance comment, so that it says it was adapted from
   `go-llmprovider-sdk`'s script under
   `docs/decisions/0001-MADR-scaffold-shared-go-library.md`. Run
   `shellcheck scripts/go-precheck.sh`. *(Annotated 2026-09-29: this found a
   pre-existing SC2001 note, fixed at the source. See Deviation D1 in the
   execution record.)*
5. Copy `.github/workflows/ci.yml`. Remove the `vet live-tagged tests` step.
   Add a Linux-only step that runs
   `go install golang.org/x/vuln/cmd/govulncheck@v1.7.0` and then
   `"$(go env GOPATH)/bin/govulncheck" ./...`. The action SHAs stay the
   SDK's. Check that the YAML parses with `python3 -c 'import yaml…'` if
   PyYAML is present; otherwise record that no YAML check was available.
   `actionlint` is not installed.
6. **Expected failures, on the committed tree.** Run `make test`,
   `make vet`, `make lint` and `make vuln`, each redirected to a scratch
   file with its `$?` captured. Each must fail with the "no packages"
   message from the MADR's table. Record the exit codes and first lines.
   Run `make pre-add-check`; it must exit 0 with
   `go-precheck: no Go files to check.`
7. **First-fail experiment, on a scratch clone** (never the tree). Clone
   the repository at the Phase 2 working state into the scratchpad. Plant
   `probe/probe.go`, declaring `package probe` with a package comment, and:
   * **(a)** an exported function with no doc comment. Run
     `make pre-add-check` and `make lint`. Both must exit non-zero, and the
     output must name `exported: exported function … should have comment`
     (`revive`).
   * **(b)** the same file, badly formatted. `make pre-add-check` must
     report the file under `gofmt:`.
   * **(c)** the file fixed: documented and `gofmt`-clean, with a
     `probe_test.go` that passes. `make pre-add-check`, `make lint`,
     `make vet`, `make test` and `make vuln` must all exit 0, and
     `go mod tidy -diff` must exit 0.
   * **(d)** `probe/client.go` holding an undocumented exported function.
     `make lint` must report it. This proves §5's dropped exclusion. Repeat
     against the SDK's unmodified `.golangci.yml` and record that it is not
     reported there.

   Record each result's exit code and its failure line. Delete the scratch
   clone afterwards; it is in the scratchpad, not the tree.
8. Commit `go.mod`, `.golangci.yml`, `Makefile`, `scripts/go-precheck.sh`
   and `.github/workflows/ci.yml`.

### Phase 3: documentation tree

1. Replace `README.md` with what the repository is (MADR §1), its status
   (no package yet; CI fails with "no packages" until the `selfupdate`
   re-home; no release), a link to `docs/README.md`, an "I want to…"
   table, and a "License" line naming the Apache License 2.0 and linking
   `LICENSE`.
2. Write `docs/README.md`: the record index (the 0001 MADR and PLAN, with
   their status) and the "I want to…" table, phrased as reader tasks.
3. Write `docs/architecture.md`: the tree and tooling as they are at this
   commit, with no history and no rationale, and a "What is not here"
   list that names the `selfupdate` re-home, `docs/reports/` and
   `docs/guides/`.
4. Checks: `markdownlint-cli2` over `README.md`, `docs/README.md` and
   `docs/architecture.md` exits 0. A throwaway link resolver over those
   three files resolves every relative link. Prove the resolver first on a
   scratch copy with one planted dead link, which it must report.
5. Set this PLAN `status: complete` once V1–V7 hold, and fill in the
   execution record. Commit Phase 3's files and this PLAN together.

## Verification

* **V1. Provenance.** `cmp` passes on the seven verbatim SDK files and on
  `LICENSE` against `magic-cli-remote`'s. `diff`
  against the SDK shows only the allowed changes for `AGENTS.md`,
  `.golangci.yml`, `Makefile`, `scripts/go-precheck.sh` and `ci.yml`.
* **V2. Module.** `go.mod` is exactly three lines. `go mod verify` and
  `go mod tidy -diff` exit 0. There is no `go.sum`.
* **V3. Honest gates.** Phase 2 step 6: the four `make` targets fail with
  "no packages", and `make pre-add-check` exits 0 on no Go files.
* **V4. Gate proven.** Phase 2 step 7 (a)–(d) behave as stated, with the
  output recorded.
* **V5. Markdown.** `markdownlint-cli2` is clean over every non-record
  Markdown file. Relative links resolve, and the resolver was proven on a
  planted dead link.
* **V6. Identifiers.** No committed file contains the local account name,
  the hostname, or a `/Users/` or `/home/<real user>` path. Checked with
  `git grep` on the committed tree, and described here without quoting the
  values searched for.
* **V7. Commits.** One commit per phase. The message was written by the
  hook. `git status` is clean after each phase.

## Rollout and Rollback

* **Rollout.** Local commits only. Under MADR §6 they are not pushed on
  their own. They go to `origin` together with the `selfupdate` re-home's
  commits, when the owner asks, so CI first runs on a module that has a
  package. This PLAN is `complete` without a push.
* **Rollback.** Each phase is one commit with no Go code and no
  requirements. Rolling back is `git revert <commit>` of that phase, which
  touches no other repository. Nothing here is published, tagged or
  consumed.

## Execution Record

2026-09-29: owner's answers to the proposed draft, "wait to push scaffold
with re-home. apache 2 license." Folded into the MADR's §6 and §7 and into
this PLAN's Phase 1 step 2, Phase 3 step 1, V1 and Rollout before approval.

### Phase 0: accept the records (2026-09-29)

The owner approved the amended pair: "proceed". The MADR is `accepted` and
this PLAN is `in-progress`. The two records are committed alone, under the
bootstrap exception.

Commit: `c08511c`.

### Phase 1: agent rules and repository hygiene (2026-09-29)

* `cmp` passes on the seven verbatim SDK files (`.claude/.gitignore`,
  `.claude/rules/madr-and-plan-skill.md`,
  `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md`,
  `opencode.json`, `.gitignore`, `.markdownlint-cli2.jsonc`), and on
  `LICENSE` against `magic-cli-remote`'s.
* `diff` of `AGENTS.md` against the SDK's, read in full, shows four hunks
  and no others. They are the intro, the Dependencies section, the removed
  `mcplib` renumbering bullet, and the Live tests section replaced by the
  "no packages" paragraph.
* `markdownlint-cli2 AGENTS.md`: `Summary: 0 issues in 0 files`, exit 0.
* Identifier scan over the ten files (the local account name, the short
  hostname, `/Users/`, `/home/<lowercase>`): no match, exit 1.

Commit: `53994bb`.

### Phase 2: Go module, lint, pre-add gate and CI (2026-09-29)

* **Step 1.** `go mod verify`: `all modules verified`, exit 0.
  `go mod tidy -diff` and `go mod tidy`: exit 0, with
  `go: warning: "all" matched no packages`. `go.mod` is `cmp`-identical
  before and after tidy. No `go.sum` was created.
* **Steps 2, 3 and 5.** `diff` against the SDK shows only the planned
  changes:
  * `.golangci.yml`: the one exclusion rule whose `path:` is the
    application-filename list, 14 lines;
  * `Makefile`: the two header-comment lines;
  * `ci.yml`: the `vet live-tagged tests` step replaced by `govulncheck`
    (`go install golang.org/x/vuln/cmd/govulncheck@v1.7.0`, then
    `govulncheck ./...`, Linux only).

  PyYAML loads `ci.yml` and `.golangci.yml`. `actionlint` is not installed,
  so no workflow lint was run.
* **Step 4.** Mode is `-rwxr-xr-x`. After D1, `diff` against the SDK's
  `113f771` copy shows 16 changed lines, all of them in the provenance
  comment. `shellcheck`: exit 0.
* **Step 6, expected failures on the tree** (`make` exits 2 when its
  recipe fails; the recipe's own status is shown):

  | Target | Recipe exit | First message |
  |---|---|---|
  | `make test` | 1 | `go: warning: "./..." matched no packages` / `no packages to test` |
  | `make vet` | 1 | `no packages to vet` |
  | `make lint` | 5 | `level=error msg="Running error: context loading failed: no go files to analyze: …"` |
  | `make vuln` | 2 | `govulncheck: no packages matched the provided patterns` |
  | `make pre-add-check` | 0 | `go-precheck: no Go files to check.` |

* **Step 7, first-fail experiment** on a scratch clone, with the Phase 2
  files and a planted `probe` package:
  * **(a)** An undocumented exported `PlantedUndocumented`:
    `make pre-add-check` and `make lint` both failed with
    `probe/probe.go:4:1: exported: exported function PlantedUndocumented should have comment or be unexported (revive)`.
  * **(b)** The same file, badly formatted: `go-precheck.sh probe/probe.go`
    exited 1 with `gofmt: these files are not formatted …` / `probe/probe.go`.
  * **(c)** A documented, formatted `Planted()` with a passing test.
    `make pre-add-check` gave `2 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)`,
    `make lint` gave `0 issues.`, `make vuln` gave
    `No vulnerabilities found.`, and `make vet`, `make test` and
    `go mod tidy -diff` exited 0. No `go.sum` was created.
  * **(d)** An undocumented exported function in `probe/client.go`. With
    this repository's `.golangci.yml`, `make lint` failed with
    `probe/client.go:3:1: exported: exported function PlantedClient should have comment or be unexported (revive)`.
    With the SDK's unmodified `.golangci.yml`: `0 issues.`, exit 0.

  The scratch clone was deleted afterwards.

#### Deviation D1 (2026-09-29): shellcheck SC2001 in `go-precheck.sh`

* **Found.** Step 4's `shellcheck scripts/go-precheck.sh` exited 1 with one
  style note:
  `echo "$unformatted" | sed 's/^/  /' >&2` — `SC2001 (style): See if you can use ${variable//search/replace} instead.`
  It was pre-existing: the same line fails the same way in the copies in
  `go-llmprovider-sdk`, `magic-cli-remote`, `ocp-login` and
  `ocp-login-macos`. The `ocp-login` copies also had
  `SC2181 (note)` on the golangci-lint exit-status check.
* **Decision.** The owner: "Fix it everywhere. No new records. Just fix it."
* **Done.**
  * Every copy now uses `printf '%s\n' "$unformatted" | sed 's/^/  /' >&2`,
    the idiom the script already uses for its other output. On the same
    input its output is byte-identical to the old line.
  * In the two `ocp-login` copies, the SC2181 check became
    `if ! lint_out="$(… 2>&1)"; then`, the SDK script's own idiom.
  * Commits, made with no new record in any repository, as the owner
    directed:

    | Repository | Branch | Commit |
    |---|---|---|
    | `go-llmprovider-sdk` | `main` | `113f771` |
    | `magic-cli-remote` | `master` | `7432f111` |
    | `ocp-login` | `main` | `1c7e8e3` |
    | `ocp-login-macos` | `main` | `09d9929` |

  * Both `ocp-login` repositories had an unrelated feature branch checked
    out, with a clean tree. Their fixes were committed on `main` through a
    temporary `git worktree`, which was then removed. The checked-out
    branches were not touched.
  * Nothing was pushed.
* **Verified.**
  * `shellcheck` exits 0 on all five copies.
  * With an unformatted planted file, every copy exits 1 and lists the file
    under `gofmt:`, indented two spaces.
  * With a stub `golangci-lint` that exits 1, both `ocp-login` copies exit 1
    and print the stub's line under `golangci-lint:`. With a stub that exits
    0, they report the file clean and exit 0.
* **Scope added to Phase 2.** None in this repository beyond the fixed line.
  The four commits above are outside this repository.

Commit: this Phase 2 commit.
