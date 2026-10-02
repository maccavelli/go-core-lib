---
status: complete
date: 2026-10-02
associated-madr: "0008-MADR-enforce-import-rules-with-depguard.md"
---
# Implement depguard import rules

Associated MADR: [0008-MADR-enforce-import-rules-with-depguard.md](0008-MADR-enforce-import-rules-with-depguard.md)

## Goal

`make lint`, the pre-add check and CI refuse every import the MADR forbids or
does not allow, with a message that names the record. The current tree is
clean.

Done means every item under Verification holds and CI is green on the pushed
commit.

## Scope

### In scope

| Step | Paths | What |
| :--- | :--- | :--- |
| 1 | `docs/decisions/0008-*`, `docs/README.md` | accept the records |
| 2 | `.golangci.yml` | enable `depguard` with the MADR's six rules, and prove each |
| 3 | `AGENTS.md`, `docs/architecture.md` | say what enforces the rules |

### Out of scope

* Any Go code, `go.mod`, CI workflow or Makefile change. `make lint`
  already reads `.golangci.yml`.
* go-tui-lib's rules, which are that repository's.
* A tag (MADR Q2).

## Rules for every step

1. **Proofs on a scratch copy.** Each planted import goes into a copy of
   the tree, never the tree itself.
2. **Loading must succeed.** A planted import of a module the copy cannot
   resolve is reached through a small local module and a `replace`. A
   proof is invalid if golangci-lint reports a load or type error instead
   of the depguard finding.
3. **Checks per step:** `make lint` (three `GOOS`), and `make pre-add-check`
   when Go files change. No Go file changes in this PLAN.
4. **Commit.** One commit per step, with `git commit --no-edit`, after the
   owner authorizes commits to `main` in that turn. The disclosure guard
   must pass in the same command first.

## Implementation Steps

### Step 1: records

The owner accepts the MADR and answers Q1 and Q2. Record the answers, set
the MADR `accepted` and this PLAN `in-progress`, and add both rows to
`docs/README.md`.

### Step 2: `.golangci.yml`

Add `depguard` to the enabled linters, and this block under `settings`:

```yaml
    # The module's import rules
    # (docs/decisions/0008-MADR-enforce-import-rules-with-depguard.md).
    depguard:
      rules:
        forbidden: # renamed banned: deviation D1
          list-mode: lax
          files:
            - $all
          deny:
            - pkg: github.com/maccavelli/mcplib
              desc: "this module never imports mcplib (AGENTS.md, Dependencies)"
            - pkg: github.com/modelcontextprotocol/go-sdk
              desc: "this module never imports the MCP go-sdk (AGENTS.md, Dependencies)"
            - pkg: github.com/maccavelli/go-llmprovider-sdk
              desc: "this module never imports go-llmprovider-sdk (AGENTS.md, Dependencies)"
            - pkg: charm.land
              desc: "Charm lives in go-tui-lib, not here (0004-MADR, Decision Outcome)"
            - pkg: github.com/charmbracelet
              desc: "Charm lives in go-tui-lib, not here (0004-MADR, Decision Outcome)"
        module:
          list-mode: strict
          files:
            - $all
          allow:
            - $gostd
            - github.com/maccavelli/go-core-lib
            - golang.org/x/mod
            - golang.org/x/sys
            - golang.org/x/term
        buildinfo:
          list-mode: strict
          files:
            - "**/buildinfo/*.go"
            - "!$test"
          allow:
            - $gostd
        selfupdate:
          list-mode: strict
          files:
            - "**/selfupdate/*.go"
            - "!$test"
          allow:
            - $gostd
            - golang.org/x/mod/semver
            - golang.org/x/sys
            - golang.org/x/term
        selfupdate-cli:
          list-mode: strict
          files:
            - "**/selfupdate/cli/*.go"
            - "!$test"
          allow:
            - $gostd
            - golang.org/x/term
            - github.com/maccavelli/go-core-lib/selfupdate$
            - github.com/maccavelli/go-core-lib/buildinfo$
        selfupdatetest:
          list-mode: strict
          files:
            - "**/selfupdate/selfupdatetest/*.go"
            - "!$test"
          allow:
            - $gostd
            - github.com/maccavelli/go-core-lib/selfupdate$
```

A strict rule's denial carries no message of its own. Each floor's comment
in the file names its record.

**Proofs.** Run on a scratch copy, one plant at a time, with `GOOS=linux`
golangci-lint and this config. The local modules live in the scratch space
and are reached by `replace`:

| ID | Plant | Must report |
| :--- | :--- | :--- |
| P0 | nothing | 0 issues |
| P1 | `github.com/maccavelli/mcplib/x` in `selfupdate/zz.go` | `forbidden`, with its message *(`banned` after D1)* |
| P2 | the same, in `selfupdate/zz_test.go` | `forbidden` |
| P3 | `github.com/modelcontextprotocol/go-sdk/mcp` in `buildinfo/zz.go` | `forbidden` |
| P4 | `github.com/maccavelli/go-llmprovider-sdk/x` in `selfupdate/cli/zz.go` | `forbidden` |
| P5 | `charm.land/lipgloss/v2` in `selfupdate/zz.go` | `forbidden` |
| P6 | `github.com/charmbracelet/x/ansi` in `selfupdate/zz.go` | `forbidden` |
| P7 | `github.com/example/newdep` in `buildinfo/zz_test.go` | `module` |
| P8 | `golang.org/x/term` in `buildinfo/zz.go` | `buildinfo` |
| P9 | `github.com/maccavelli/go-core-lib/selfupdate/selfupdatetest` in `selfupdate/cli/zz.go` | `selfupdate-cli` |
| P10 | `golang.org/x/term` in `selfupdate/selfupdatetest/zz.go` | `selfupdatetest` |
| P11 | `os/exec` in `buildinfo/zz.go` | 0 issues: an allowed import passes |

Each plant uses its import, so the file compiles. A proof passes only when
golangci-lint's output has the named rule's finding and no type or load
error.

### Step 3: documentation

* **`AGENTS.md`, Dependencies,** gains: "`depguard` in `.golangci.yml`
  enforces these rules. It refuses the forbidden modules in every file. It
  allows only the standard library, this module and the required modules
  anywhere. It allows each package only the imports its record names. A
  record that adds a module or a package amends those rules in the same
  commit."
* **`docs/architecture.md`, Tooling,** names the six rules and the MADR.

## Verification

* P0–P11 behave as the table says, and the output of each is quoted in the
  execution record.
* `make lint` reports 0 issues for `GOOS=linux`, `darwin` and `windows`.
* `make pre-add-check` passes on a Go file of each package, so the
  pre-add check runs the new rules.
* `go mod tidy -diff` is clean, and `go.mod` is unchanged.
* The disclosure guard passes on the outgoing commits.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.** The owner pushes Steps 1–3. Nothing else changes for
  consumers.
* **Rollback.** One commit reverts the config. No code depends on it.

## Execution Record

### Step 1: records (2026-10-02)

The owner said "i committed and pushed, proceed". The MADR is `accepted`,
with Q1 and Q2 answered as recommended, and this PLAN is `in-progress`.
Committed as `d7cd32f`.

### Step 2: `.golangci.yml` (2026-10-02)

* **The config.** `depguard` is enabled, and the block is the one above,
  with one change, D1.
* **The proofs.** Each ran on a fresh scratch copy. Local fixture modules,
  wired in by `replace`, made each import load. Each run was
  `golangci-lint --enable-only depguard` with `GOOS=linux`. The first run,
  under the PLAN's names, gave:
  * P0, P1, P2, P4–P11: as the table says;
  * **P3: not as planned.** `go-sdk/mcp` planted in `buildinfo/zz.go` was
    refused, but reported as `import 'github.com/modelcontextprotocol/go-sdk/mcp'
    is not allowed from list 'buildinfo'`, without the `forbidden` rule's
    reason.

**Deviation D1 (2026-10-02): `forbidden` becomes `banned`.**

* **Found.** depguard v2.2.1 reports every rule that refuses an import
  (`depguard.go`, `run`). golangci-lint keeps one finding per line, and
  the rules run in name order. `buildinfo` sorts before `forbidden`, so its
  finding was the one kept. In `selfupdate` and `cli`, `forbidden` sorted
  first, which is why P1 and P4–P6 showed the reason.
* **Decision.** The owner chose option 1: "option 1 follow recommendations.
  commit to main". The alternative was to turn off golangci-lint's
  one-finding-per-line rule for every linter.
* **Changed.**
  * The rule is named `banned`, which sorts before every other rule. A
    comment in `.golangci.yml` says why, and that new rules must sort
    after it.
  * The MADR gains amendment D1. The block and P1 above are annotated.
  * **P12 is added:** mcplib planted in `selfupdatetest`, so that a banned
    import is proven in every package, not only the ones that sort after
    it.

**The proofs after D1**, every one passing:

| ID | Plant | Finding |
| :--- | :--- | :--- |
| P0 | nothing | 0 issues |
| P1 | mcplib in `selfupdate/zz.go` | `… from list 'banned': this module never imports mcplib (AGENTS.md, Dependencies)` |
| P2 | mcplib in `selfupdate/zz_test.go` | the same, in the test file |
| P3 | MCP go-sdk in `buildinfo/zz.go` | `… from list 'banned': this module never imports the MCP go-sdk (AGENTS.md, Dependencies)` |
| P4 | go-llmprovider-sdk in `selfupdate/cli/zz.go` | `… from list 'banned': this module never imports go-llmprovider-sdk …` |
| P5 | `charm.land/lipgloss/v2` in `selfupdate/zz.go` | `… from list 'banned': Charm lives in go-tui-lib, not here (0004-MADR, Decision Outcome)` |
| P6 | `github.com/charmbracelet/x/ansi` in `selfupdate/zz.go` | the same message |
| P7 | `github.com/example/newdep` in `buildinfo/zz_test.go` | `… from list 'module'` |
| P8 | `golang.org/x/term` in `buildinfo/zz.go` | `… from list 'buildinfo'` |
| P9 | `selfupdatetest` in `selfupdate/cli/zz.go` | `… from list 'selfupdate-cli'` |
| P10 | `golang.org/x/term` in `selfupdate/selfupdatetest/zz.go` | `… from list 'selfupdatetest'` |
| P11 | `os/exec` in `buildinfo/zz.go` | 0 issues |
| P12 | mcplib in `selfupdate/selfupdatetest/zz.go` | `… from list 'banned': this module never imports mcplib …` |

**The pre-add check.**

* The planted mcplib import in `buildinfo/zz.go`, in a scratch clone, made
  `make pre-add-check FILES=buildinfo/zz.go` exit non-zero, with the
  `banned` finding.
* The first attempt ran in a copy without `.git`. `go-precheck.sh` stopped
  at `fatal: not a git repository` before linting, so that attempt proved
  nothing, and was redone in a clone.
* The run's `TestStampedBinary` failures are the plant's side effect: that
  test builds a binary offline, and the fixture module cannot be fetched.

**Checks on the real tree:**

| Check | Result |
| :--- | :--- |
| `make lint` | 0 issues for `GOOS=linux`, `darwin` and `windows` |
| `make pre-add-check` on one file of each package | `4 file(s) clean` |
| `go mod tidy -diff` | rc 0; `go.mod` unchanged |

### Step 3: documentation (2026-10-02)

* **`AGENTS.md`, Dependencies,** names Charm beside the three banned
  modules. It says that `depguard` enforces the rules, lists the six rules,
  and says that a record adding a module or a package amends them in the
  same commit, with new rule names sorting after `banned`.
* **`docs/architecture.md`, Tooling,** gains "Import rules": the six rules
  and the MADR.
* **Also fixed in the same section.** It said `make fuzz` "refuses fewer
  than four" targets. Since `0004-PLAN-v1-4-0-command-surface.md` Step 3
  the `Makefile` passes `-m 5`, so it now says five. That plan's Step 8
  updated the Go-code table but missed this line.
* markdownlint-cli2: 0 issues on both files. The link resolver: no broken
  link.

### Close-out (2026-10-02)

The owner pushed Steps 1–3. CI run `37043440716` on `bb1af9c` concluded
`success` on `ubuntu-24.04`, `macos-15` and `windows-2025`. Every item under
Verification holds, so this PLAN is `complete`. No tag was cut (MADR Q2).
