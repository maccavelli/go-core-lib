---
status: in-progress
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
        forbidden:
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
| P1 | `github.com/maccavelli/mcplib/x` in `selfupdate/zz.go` | `forbidden`, with its message |
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

None yet.
