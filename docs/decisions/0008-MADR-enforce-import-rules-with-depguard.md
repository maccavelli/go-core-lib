---
status: accepted
date: 2026-10-02
decision-makers: go-core-lib maintainers
consulted: depguard v2.2.1 source (the version golangci-lint v2.14.0 builds); go-tui-lib's .golangci.yml
informed: consumers of go-core-lib
---
# Enforce the module's import rules with depguard: forbidden modules, a module floor, and a floor per package

## Context and Problem Statement

On 2026-10-02 the owner asked: "this project should have absolutely zero
dependencies on mcplib, is that not the case?" It is the case. The owner then
said: "fix the lint gaps."

The gap is that the rules are stated, and nothing enforces them. A stray
import today is caught only by a reviewer who remembers the rule.

Evidence (read-only, 2026-10-02, at `4d7b053`):

* **No mcplib today.**
  * `go.mod` and `go.sum` never mention it, and `go mod graph` has no
    mcplib edge.
  * `go list -m all` is this module, `golang.org/x/mod v0.40.0`, `x/sys
    v0.47.0`, `x/term v0.43.0` and `x/tools v0.49.0`, the last in the graph
    only.
  * No Go file imports it.
* **The rules, and where they are stated:**

  | Rule | Where |
  | :--- | :--- |
  | Never import `github.com/maccavelli/mcplib`, `github.com/modelcontextprotocol/go-sdk` or `github.com/maccavelli/go-llmprovider-sdk` | `AGENTS.md`, Dependencies |
  | No module without a MADR that names it | `AGENTS.md`, Dependencies |
  | Charm stays out of this module; it lives in go-tui-lib | [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md), Decision Outcome and §1 |
  | `buildinfo` is standard library only | 0004-MADR §5, amendment F2 |
  | `selfupdate/cli` imports the standard library, `x/term`, `selfupdate` and `buildinfo` | 0004-MADR §5, amendment F6 |
  | `selfupdate` imports the standard library, `x/mod`, `x/sys` and `x/term` | 0004-MADR §1 table |

* **The lint config enforces none of them.** `.golangci.yml` does not
  enable `depguard`. go-tui-lib's `.golangci.yml` is this repository's
  plus one `depguard` block, which already refuses mcplib, the MCP go-sdk,
  go-llmprovider-sdk and the Charm v1 paths there.
* **The measured import sets.** `go list` for `GOOS=linux`, `darwin` and
  `windows` gives these imports beyond the standard library:

  | Package | Non-test files | Test files add |
  | :--- | :--- | :--- |
  | `buildinfo` | none | `buildinfo` |
  | `selfupdate` | `x/mod/semver`, `x/sys/unix` or `x/sys/windows`, `x/term` | `selfupdate`, `selfupdatetest` |
  | `selfupdate/cli` | `buildinfo`, `selfupdate`, `x/term` | `cli`, `selfupdatetest` |
  | `selfupdate/selfupdatetest` | `selfupdate` | `selfupdatetest` |

  Each matches what the records and `docs/architecture.md` state.
* **How depguard v2.2.1 decides.** Read in its source, `settings.go`:
  * each rule has file globs (`$all`, `$test`, `!`-negation, `**/` paths)
    and allow or deny lists;
  * allow and deny entries match by prefix, or exactly with a trailing
    `$`. `$gostd` is the standard library;
  * `strict` refuses anything not allowed, and `lax` allows anything not
    denied;
  * every rule whose files match a file is applied to it.
* **A typed-check caveat, seen in go-tui-lib.** golangci-lint must load a
  package before depguard runs. An import that does not resolve fails
  loading, and hides the depguard finding. go-tui-lib's proof therefore
  used small local modules through `replace` (go-tui-lib
  `0002-PLAN-multi-pane-workspace-layouts.md`, Verification).
* **A trial of §1–§3, on a scratch copy.** It used the block exactly as
  the PLAN writes it:
  * the current tree had 0 issues for `GOOS=linux`, `darwin` and
    `windows`;
  * planting `golang.org/x/term` in `buildinfo` reported
    `import 'golang.org/x/term' is not allowed from list 'buildinfo'`;
  * planting `selfupdatetest` in `cli`'s non-test code reported `… is not
    allowed from list 'selfupdate-cli'`.

  So the globs match, and the `$` makes the match exact. The PLAN's
  Step 2 proves the other rules.

## Decision Drivers

* **A rule nobody enforces is a rule somebody breaks.** Each stated rule
  above should fail `make lint`, the pre-add check and CI.
* **The dependency floor is the module's promise to consumers.** A
  consumer imports go-core-lib to avoid mcplib, the MCP SDK and Charm.
* **"No module without a MADR" should be mechanical.** A new module should
  fail lint until a record amends the allowed list.
* **No new module.** depguard ships inside golangci-lint, already pinned at
  v2.14.0.
* **Findings must say why.** Each message cites the record behind the rule.

## Considered Options

* **A. depguard: a forbidden list, a module floor, and a floor per package.**
* **B. depguard: the forbidden list only,** as go-tui-lib has.
* **C. A Go test** that runs `go list` and checks every package's imports.
* **D. Leave it to review.**

## Decision Outcome

Chosen option: **"A"**, because:

* it enforces every rule in the table, with the tool every commit and CI
  already run;
* it makes the module list exact, so a new dependency cannot arrive
  without a record;
* it adds no module.

### 1. Forbidden modules, in every file

A `lax` rule named `forbidden`, for `$all` files, tests included, denies:

* `github.com/maccavelli/mcplib`;
* `github.com/modelcontextprotocol/go-sdk`;
* `github.com/maccavelli/go-llmprovider-sdk`;
* `charm.land` and `github.com/charmbracelet`. Charm's v2 path and its v1
  path, and its `x` modules, all stay in go-tui-lib.

Each message names the record: `AGENTS.md` for the first three, and
0004-MADR for Charm.

### 2. The module floor, in every file

A `strict` rule named `module`, for `$all` files, allows only:

* `$gostd`;
* this module, `github.com/maccavelli/go-core-lib`;
* the three required modules, `golang.org/x/mod`, `golang.org/x/sys` and
  `golang.org/x/term`.

Any other import fails lint. A record that adds a module amends this list in
the same commit as the module's first import, which is what `AGENTS.md`
already requires of `go.mod`.

### 3. A floor per package, in non-test files

Each is `strict`, and covers the package's own directory, `!$test`:

| Rule | Files | Allows |
| :--- | :--- | :--- |
| `buildinfo` | `**/buildinfo/*.go` | `$gostd` |
| `selfupdate` | `**/selfupdate/*.go` | `$gostd`, `golang.org/x/mod/semver`, `golang.org/x/sys`, `golang.org/x/term` |
| `selfupdate-cli` | `**/selfupdate/cli/*.go` | `$gostd`, `golang.org/x/term`, `github.com/maccavelli/go-core-lib/selfupdate$`, `github.com/maccavelli/go-core-lib/buildinfo$` |
| `selfupdatetest` | `**/selfupdate/selfupdatetest/*.go` | `$gostd`, `github.com/maccavelli/go-core-lib/selfupdate$` |

* The `$` makes `selfupdate$` exact, so that `cli` cannot import
  `selfupdatetest` through the prefix `selfupdate`.
* Test files are covered by §1 and §2 only. A test may import the module's
  test doubles, and that is all it may add.
* A new package gets its floor in its own record.

### 4. What changes besides the config

* `AGENTS.md`, Dependencies, says that `depguard` enforces these rules and
  where they live.
* `docs/architecture.md`, Tooling, says the same.
* No Go code, `go.mod`, CI file or Makefile target changes. `make lint`
  already runs golangci-lint with `.golangci.yml`, once per `GOOS`.
* No release is needed. The lint configuration is not part of what a
  consumer builds.

### Consequences

* Good, because every stated import rule now fails `make lint`, the
  pre-add check and CI.
* Good, because a new module cannot arrive without its record amending
  §2's list.
* Good, because the package floors keep `buildinfo` importable by any
  program and keep `cli` light.
* Neutral, because `golang.org/x/tools`, in the graph through apidiff's
  tooling, stays unimportable by code here, which is the point.
* Bad, because adding a module, or a package, now edits `.golangci.yml`
  too. That is the intended friction, and the message says where.

### Confirmation

* `make lint` reports 0 issues for `GOOS=linux`, `darwin` and `windows` on
  the current tree.
* On a scratch copy, a planted import makes the named rule fire, for each
  of:
  * mcplib, in non-test code and in a test;
  * the MCP go-sdk, go-llmprovider-sdk, `charm.land/…` and
    `github.com/charmbracelet/…`;
  * a new module, in a test file;
  * `x/term` in `buildinfo`;
  * `selfupdatetest` in `cli`'s non-test code.
* A planted standard-library import in `buildinfo` raises nothing, so the
  floors do not refuse what they allow.
* The forbidden and new modules are small local modules reached through
  `replace` in the scratch copy, so that loading succeeds and depguard is
  what reports.
* CI is green on the pushed commit.

## Pros and Cons of the Options

### A. Forbidden list, module floor, package floors

* Good, because it enforces every rule in the evidence table.
* Good, because it uses the tool that already gates every commit.
* Bad, because there are six rules to keep in step with the code.

### B. Forbidden list only

* Good, because it is one rule, as in go-tui-lib.
* Bad, because a new module, or a heavier import in `buildinfo` or
  `cli`, still passes.

### C. A Go test over `go list`

* Good, because it could check the module graph as well as imports.
* Bad, because it duplicates a linter the repository already runs, and
  runs later than lint in the pre-add check.

### D. Review only

* Good, because there is nothing to build.
* Bad, because it is the gap the owner asked to close.

## Owner questions

*Answered 2026-10-02: the owner said "i committed and pushed, proceed",
approving the PLAN. Q1 and Q2 are taken as recommended: test files get
§1 and §2 only, and no tag is cut.*

* **Q1. Package floors on test files.** Recommended: no. Tests get §1 and
  §2 only, so a test can use any package of this module. The alternative
  is a floor per package for tests too, which would list
  `selfupdatetest` in each.
* **Q2. A tag.** Recommended: none. Nothing a consumer builds changes. The
  alternative is a `v1.4.1` that carries only lint configuration.

## More Information

* `AGENTS.md`, Dependencies.
* [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md):
  §1 (the package table), §5 and amendments F2 and F6.
* go-tui-lib `.golangci.yml`, the `forbidden` rule this record extends.
* depguard v2.2.1 `settings.go` and `README.md`.
