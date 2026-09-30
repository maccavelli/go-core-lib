---
status: accepted
date: 2026-09-29
decision-makers: go-core-lib maintainers
consulted: mcplib maintainers; owners of prepare-commit-msg, magic-cli-remote, mcp-server-recall, mcp-server-magictools, mcp-server-socratic-thinker, mcp-server-duckduckgo
informed: fleet consumers of github.com/maccavelli/mcplib/selfupdate
---
# Re-home `selfupdate` and its release tooling from mcplib into go-core-lib as v1.0.0, without history

## Context and Problem Statement

`mcplib`'s `selfupdate` package is the fleet's CLI self-update
implementation. Six programs import it, and the same six publish their
releases through `mcplib`'s reusable workflow
`.github/workflows/publish-selfupdate-release.yml`. It is not MCP code.
`go-llmprovider-sdk`
`docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md`, sixth
amendment, records the owner's decision that it moves to this repository,
"extracted under that repository's own records". That move is also what lets
`prepare-commit-msg` drop `mcplib` entirely: `selfupdate` is its only
remaining `mcplib` import.
[0001-MADR-scaffold-shared-go-library.md](0001-MADR-scaffold-shared-go-library.md)
scaffolded this repository for it.

On 2026-09-29 the owner decided three things for this record:

> History does not migrate. Any reusable tooling or workflow that makes sense
> to use should be used. The first version tag will be v1.0.0

This record decides what moves, what changes in transit, and how the result
is proven and released. It does not migrate any consumer, and it does not
change `mcplib`.

Evidence gathered for this record (read-only, 2026-09-29, `mcplib` at
`2069912`):

* **The source is `mcplib` `v1.6.0` (`4e1f9a53e265`).**
  `git diff --stat v1.6.0 HEAD` over `selfupdate/`, `scripts/` and the
  reusable workflow is empty.
* **The package.** `selfupdate/` has 49 files: 26 non-test, 21 test, and
  `testdata/SHA256SUMS.valid` and `SHA256SUMS.invalid`. About 6,100 lines.
  * No other `mcplib` package imports it. It imports no `mcplib` package.
  * Its only `mcplib` references are the import in `example_test.go:10` and
    the text of `doc.go` ("for mcplib consumers"; the workflow "at the exact
    mcplib module-tag commit").
  * Nothing `mcplib`-specific reaches the wire. The consumer supplies the
    `User-Agent` (`GitHubOptions.UserAgent`, required and validated), and
    the token comes from `GH_TOKEN` / `GITHUB_TOKEN`.
  * The exported API, from `go doc -all`, is 612 lines: `Updater` / `New` /
    `Run`, `Config`, `Request`, `Result`, `ExitCode`, `GitHubSource`,
    `NewExactAssetSelector`, `NewStrictVersionPolicy`,
    `StandaloneInstaller`, `ManagedInstaller`, `NewTextReporter`,
    `NewTerminalConfirmer`, `DefaultLimits`, `RateLimitError`, and the
    `Lifecycle`, `Reconciler`, `Transformer`, `Verifier`, `Installer`,
    `InstallSession` and `ReleaseSource` seams.
* **Its non-standard imports, across darwin, linux and windows,** are
  `golang.org/x/mod/semver`, `golang.org/x/sys/unix`,
  `golang.org/x/sys/windows` and `golang.org/x/term`. `mcplib` pins
  `x/mod v0.40.0`, `x/sys v0.47.0` and `x/term v0.43.0`.
* **The release tooling** is five scripts and one workflow:
  * `scripts/verify-selfupdate-release.sh` (228 lines; validates a staged
    release against the product/platform/extra matrix) and its offline
    fixture test `verify-selfupdate-release_test.sh`;
  * `scripts/refuse-existing-release.sh` (refuses to publish over an
    existing release or draft, and fails closed when gh cannot tell) and
    its stubbed test `refuse-existing-release_test.sh`;
  * `scripts/check-workflow-gh-repo.sh` (asserts that every
    repository-scoped `gh` step in the workflow sets `GH_REPO`);
  * `.github/workflows/publish-selfupdate-release.yml`. It checks the tools
    out of the *called* workflow's own commit, at the hard-coded path
    `.mcplib-release-tools`. Its comments cite "MADR 0007", which is
    `mcplib` `docs/0007-MADR-restore-repository-context-in-the-reusable-release-workflow.md`.

  `mcplib`'s CI runs the verifier test on Linux, and the refuse test and
  the workflow check on Linux, macOS and Windows.
* **How consumers call it.** All six pass `artifact-name`, `products-json`,
  `platforms-json` and `extra-assets-json`. Five pin `mcplib` `v1.4.1`
  (`8d4d89ff`) and `prepare-commit-msg` pins `v1.6.0` (`4e1f9a53`). Five
  pass `bridge-release: false`. `magic-cli-remote` passes
  `bridge-release: ${{ github.ref_name == 'v0.16.0' }}`.
* **`bridge-release` can no longer succeed.** The verifier allows it only
  for `maccavelli/magic-cli-remote` tag `v0.16.0`
  (`verify-selfupdate-release.sh:136-137`). That release was published on
  2026-09-02 and is immutable, and `refuse-existing-release.sh` refuses any
  tag that already has a release. The bridge code is about 40 lines of the
  verifier, the input, one `env` line, one argument, and four of the
  fixture test's cases.
* **A scratch re-home passes this repository's gates.** On a scratch clone
  of this repository at `0d795fa`, the probe copied `selfupdate/` from
  `mcplib` `v1.6.0` and rewrote only the import path. `go get` of the three
  pins, then `go mod tidy`, gave exactly those three requirements. The six
  `go.sum` `h1:` lines are identical to `mcplib`'s. With Go 1.27.1:
  * `make lint` (this repository's stricter `.golangci.yml`): `0 issues.`;
  * `go test -race -count=1 ./...`: pass, coverage 75.5 %, the same as
    `mcplib`'s own run of the same package;
  * `govulncheck ./...`: `No vulnerabilities found.`;
  * `make pre-add-check`: `47 file(s) clean`. `go mod tidy -diff`: exit 0;
  * the three release-script tests and `check-workflow-gh-repo.sh` pass,
    with the tools path renamed, and `shellcheck` is clean on all five
    scripts.
* **Four lint findings exist only on Windows, and are pre-existing.**
  `CGO_ENABLED=0 GOOS=windows golangci-lint run` reports them with this
  repository's configuration *and* with `mcplib`'s. `mcplib`'s CI lints
  on Linux only, so nothing has ever reported them:
  1. `cleanup_windows.go:54` G304: the receipt is `Lstat`ed through an
     `os.Root`, then read by joined path with `os.ReadFile`.
  2. `cleanup_windows.go:123` SA1019: `windows.OpenCurrentProcessToken` is
     deprecated. It is `OpenProcessToken(CurrentProcess(), TOKEN_QUERY, …)`.
  3. `replace.go:11` unused on Windows: `osRename` is used only by
     `replace_unix.go`.
  4. `replace_windows.go:16` unused: `applyResult.pendingBackup` is never
     written or read. The pending backup travels as `commitReplacement`'s
     return value instead.

  With `CGO_ENABLED=0`, linux and darwin lint and `go vet` are clean on all
  three targets. The Windows test binary compiles.
* **Cross-OS runs need `CGO_ENABLED=0` explicitly.** This host's `go env`
  sets `CGO_ENABLED=1`. `GOOS=linux go vet` then fails in `runtime/cgo`
  against the macOS SDK, identically on `mcplib`'s own tree.
* **Open work in `mcplib` about this code.**
  `docs/0005-PLAN-canonicalize-cli-self-update-in-mcplib.md` is
  `in-progress`, with three open items: G2 §18.3 native update smoke, the
  Phase 12 deduplication audit, and a deferred Magic bridge cleanup (no
  earlier than 90 days after `magic-cli-remote` `v0.16.0`).
  `docs/0007-MADR-…` is `proposed` although its PLAN is `complete`.

## Decision Drivers

* The owner's three decisions: no history, reuse the tooling that makes
  sense, and `v1.0.0` first.
* Consumers change as little as possible: an import path, and the one
  `uses:` line of their release job.
* The moved code is proven to be the same code. Every difference from
  `mcplib` `v1.6.0` is listed and checked, not asserted.
* This repository's gates hold on every target the code builds for, not
  only on the host.
* No dead contract moves: nothing that can only fail.

## Considered Options

* **A. Fresh copy of the package and its release tooling; the API and workflow contract kept, except the dead `bridge-release` input.**
* **B. As A, but keep `bridge-release`,** so that every consumer's workflow call is unchanged apart from `uses:`.
* **C. Package only.** The workflow and scripts stay in `mcplib`, and consumers keep pinning `mcplib` for publication.
* **D. Import with history** (`git filter-repo`, as `go-llmprovider-sdk` did).

## Decision Outcome

Chosen option: "A", because it is the only option that puts the updater and
its one supported publication path in the same repository, under one tag.
It follows the owner's no-history decision, and it moves nothing that
cannot run. ~~The cost is one deleted line in one consumer.~~ *Corrected
2026-09-29 (0003-MADR D2): every one of the six consumers passes
`bridge-release`, five as `false`, and each deletes that line.*

### 1. What moves

| `mcplib` `v1.6.0` path | Path here |
|---|---|
| `selfupdate/` (49 files) | `selfupdate/` |
| `scripts/verify-selfupdate-release.sh`, `…_test.sh` | same |
| `scripts/refuse-existing-release.sh`, `…_test.sh` | same |
| `scripts/check-workflow-gh-repo.sh` | same |
| `.github/workflows/publish-selfupdate-release.yml` | same |

* Package name `selfupdate`, import path
  `github.com/maccavelli/go-core-lib/selfupdate`.
* Nothing else moves. `mcplib`'s CI, Makefile and README keep their own
  content.

### 2. No history, and no records

* Files are copied from `mcplib` at `v1.6.0` (`4e1f9a53e265`), not
  imported with history. ~~The first commit that adds them names that tag
  and commit.~~ *Amended 2026-09-29 (PLAN deviation D1): the global hook
  writes commit messages, and it did not name them. The tag and commit are
  recorded in the PLAN's execution record, which is committed with the
  copy, and in `docs/architecture.md`.*
* `mcplib`'s records stay in `mcplib`. That covers
  `docs/0005-MADR-canonicalize-cli-self-update-in-mcplib.md` and its PLAN,
  `docs/0006-MADR-raise-go-toolchain-floor-to-1-26-6.md` and
  `docs/0007-MADR-restore-repository-context-in-the-reusable-release-workflow.md`
  and its PLAN. Moved code and comments cite them by repository and
  filename. "MADR 0007" in the workflow and scripts becomes
  "mcplib `docs/0007-MADR-restore-repository-context-in-the-reusable-release-workflow.md`".
* `mcplib` 0005's open items are not transferred. They are about consumer
  releases and an audit across `mcplib`'s consumers, and they stay open
  there. When a consumer migrates here, its own record says what it does
  about them.

### 3. What changes in transit, and nothing else

A normalised `diff -r` against `mcplib` `v1.6.0` shows only these changes.

**In `selfupdate/`:**

* the `example_test.go` import path;
* `doc.go`: "for mcplib consumers" becomes "for fleet programs", and the
  workflow sentence names this repository's tag commit;
* the four Windows lint fixes, each behaviour-preserving:
  1. `cleanup_windows.go`: read the receipt with `root.ReadFile(name)`
     through the `os.Root` that already `Lstat`ed it, instead of
     `os.ReadFile` on a joined path. A receipt the old code accepted is
     read identically. Only a path that escapes the root, which the old
     code would have followed, is now refused;
  2. `cleanup_windows.go`: `windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)`,
     which is the deprecated helper's own body;
  3. `osRename` moves from `replace.go` to `replace_unix.go`, its only user;
  4. the unused `applyResult.pendingBackup` field is deleted.

**In the release tooling:**

* the tools checkout path becomes `.core-lib-release-tools`;
* record citations become repository-named;
* `bridge-release` is removed from the workflow inputs, from the verifier
  (`--bridge`, the compatibility set and `SHA256SUMS-0.16.0`), and from the
  fixture test's four bridge cases.

The exported API is unchanged. A normalised `go doc -all` of the package,
diffed against `mcplib` `v1.6.0`'s, shows only the `doc.go` package comment.

### 4. Dependencies

These are the three modules
[0001-MADR-scaffold-shared-go-library.md](0001-MADR-scaffold-shared-go-library.md)
§3 left for this record to name, at `mcplib`'s pins:

* `golang.org/x/mod v0.40.0`
* `golang.org/x/sys v0.47.0`
* `golang.org/x/term v0.43.0`

They are all direct requirements, in `go mod tidy`'s layout. `go.sum` is
what tidy writes. Its `h1:` lines are `mcplib`'s. No version moves in this
record.

### 5. Lint covers every target the code builds for

`make lint`, `scripts/go-precheck.sh` step 2, and therefore CI run
`golangci-lint run -c .golangci.yml ./...` three times: for `GOOS=linux`,
`darwin` and `windows`, each with `CGO_ENABLED=0`. `govet` runs inside
golangci-lint, so this also vets every target. This supersedes
[0001-MADR-scaffold-shared-go-library.md](0001-MADR-scaffold-shared-go-library.md)
§4 for those two files: the `Makefile`'s `lint` target and the script's
lint step no longer match `go-llmprovider-sdk`'s. Everything else in them
still does.

### 6. CI runs the release tooling's own tests

As in `mcplib`'s CI: `verify-selfupdate-release_test.sh` on Linux, and
`refuse-existing-release_test.sh` and `check-workflow-gh-repo.sh` on Linux,
macOS and Windows.

### 7. Documentation

* `README.md` and `docs/architecture.md` describe `selfupdate` and the
  reusable workflow, taking over the substance of `mcplib`'s README
  "Self-update" section.
* `docs/guides/migrating-from-mcplib-selfupdate.md` gives consumers the
  steps: the import path, `go get github.com/maccavelli/go-core-lib@v1.0.0`,
  the `uses:` line at the `v1.0.0` commit SHA, removing `bridge-release`,
  and the Go 1.27.1 floor that the requirement brings.

### 8. Release: `v1.0.0`

* The first tag is `v1.0.0`, on the last code-and-docs commit of this
  record's PLAN (its Phase 5). The PLAN's closing execution-record commit
  follows the tag and changes only records.
* Push and tag need the owner's explicit ask in the same turn. Under 0001
  §6, that push is also the scaffold's first push.
* `v1.0.0` is tagged only after the push's CI is green on Linux, macOS and
  Windows. The tag's own CI must be green, and
  `GOPROXY=https://proxy.golang.org go list -m github.com/maccavelli/go-core-lib@v1.0.0`
  must resolve.
* Consumers pin the reusable workflow by the `v1.0.0` commit's full SHA,
  never by the tag, as they do today.

### 9. Not decided here

* **Each consumer's migration.** `prepare-commit-msg` goes first. Its move
  also completes `go-llmprovider-sdk`
  `docs/decisions/0002-PLAN-migrate-llmprovider-from-mcplib.md` Phase 10.
  Then `magic-cli-remote`, `mcp-server-recall`, `mcp-server-magictools`,
  `mcp-server-socratic-thinker` and `mcp-server-duckduckgo`, each under its
  own record in its own repository. Each moves its `go` directive to
  1.27.1.
* **`mcplib`'s side:** deprecating, then removing, `selfupdate` and the
  workflow, and closing or re-scoping its 0005 open items. That needs a
  record in `mcplib`. Removing the workflow file there does not break a
  caller pinned to an existing commit SHA.

### Consequences

* Good, because the updater and its publication workflow version together,
  in a repository whose only dependencies are three `golang.org/x` modules.
* Good, because consumers change one import path and one `uses:` line.
  ~~`magic-cli-remote` also deletes its `bridge-release` line.~~
  *Corrected 2026-09-29 (0003-MADR D2): all six consumers delete their
  `bridge-release` line.*
* Good, because four Windows-only defects that `mcplib` never linted are
  fixed, and cross-target lint keeps new ones out.
* Good, because the same-code claim is checked: a normalised diff, a
  `go doc` diff, and identical coverage.
* Neutral, because `go.sum` and the three pins are exactly `mcplib`'s.
* Bad, because without history, `git blame` here starts at the copy. The
  history stays readable in `mcplib` at `v1.6.0`, ~~and the first commit
  names it~~ *and the PLAN's execution record and `docs/architecture.md`
  name it (corrected 2026-09-29, 0003-MADR D12; see the §2 amendment)*.
* Bad, because a module requiring `go-core-lib` must be at `go 1.27.1`.
  Every consumer is at 1.26.6 and must move, as each already must for
  `go-llmprovider-sdk` under `magic-cli-remote` 0169 D2.
* Bad, because until `mcplib` records its side, the updater exists in two
  places, and a fix made here does not reach consumers still on `mcplib`.
* Bad, because tripling the lint run makes `make lint` and the pre-add gate
  slower. The PLAN measures by how much.

### Confirmation

* The normalised `diff -r` against `mcplib` `v1.6.0` shows only §3's
  changes, and the `go doc -all` diff shows only the package comment.
  Both outputs are recorded in the PLAN.
* `go test -race` passes, with coverage recorded and compared to 75.5 %.
  Cross-target lint is clean. `govulncheck`, `go mod tidy -diff` and the
  release-script tests pass.
* The cross-target lint is seen to fail on a scratch copy. With each of the
  four fixes reverted, it reports that finding under `GOOS=windows`.
* Fixes 1 and 2 run under the existing `TestWindowsCleanupReceiptRoundTrip`
  (`cleanup_windows_test.go`), which reads a receipt through
  `processCleanupReceipt` and writes one through `restrictToCurrentUser`.
  `TestWindowsCleanupReceiptDigestMismatch` also covers fix 1. Fixes 3 and 4
  change declarations only, so compiling the package and its tests for all
  three targets proves them. Windows tests run only on a Windows host, and
  here that means the Windows CI leg. Their first run is therefore the
  release push, and `v1.0.0` waits for it to pass.
* CI is green on all three operating systems before the tag, and on the
  tag. The module proxy resolves `v1.0.0`.

## Pros and Cons of the Options

### A. Fresh copy, contract kept, bridge dropped

* Good, because nothing moves that can only fail.
* Good, because the package and its publication path share one tag and one
  CI.
* Bad, because ~~`magic-cli-remote` must delete its `bridge-release`
  line~~ *every consumer must delete its `bridge-release` line (corrected
  2026-09-29, 0003-MADR D2)*. Passing an input a called workflow does not
  define is an error.

### B. Keep `bridge-release`

* Good, because every consumer's `with:` block is unchanged.
* Bad, because it carries a verifier branch, an input and four test cases
  whose only permitted use has already happened and cannot recur.
* Bad, because `magic-cli-remote`'s deferred bridge cleanup (`mcplib` 0005
  §19) would later have to reach into this repository to remove it.

### C. Package only; workflow stays in `mcplib`

* Good, because the workflow is not touched.
* Bad, because consumers keep a pin on `mcplib` for publication, and
  `mcplib` keeps non-MCP tooling it cannot drop.
* Bad, because the package and the release contract it verifies would be
  versioned in different repositories.

### D. Import with history

* Good, because blame would continue.
* Bad, because the owner decided against it.

## More Information

* `mcplib` `docs/0005-MADR-canonicalize-cli-self-update-in-mcplib.md`: the
  design of the package and the asset contract.
* `mcplib` `docs/0007-MADR-restore-repository-context-in-the-reusable-release-workflow.md`:
  why the workflow sets `GH_REPO` and why the guard fails closed.
* `go-llmprovider-sdk`
  `docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md`, sixth
  amendment, "The owner's further decisions (2026-09-29)".
* `magic-cli-remote`
  `docs/decisions/0169-MADR-standardize-toolchains-on-current-supported-advisory-free-releases.md`
  D2: Go 1.27.1 across the fleet.
* Amendment 2026-09-29: `v1.0.0` includes the fixes of
  [0003-MADR-remediate-debugging-pass-findings.md](0003-MADR-remediate-debugging-pass-findings.md).
  The owner decided to fix every debugging-pass finding before the first
  tag. `v1.0.0` is therefore `mcplib` `v1.6.0` plus §3 plus 0003's fixes,
  not `v1.6.0` plus §3 alone.
  * The exported API is still unchanged: G-api may differ only in doc
    comments.
  * G-diff's baseline grows by every 0003 change.
  * §8's tag waits for 0003's PLAN to be `complete`.
* Owner's decision (2026-09-29): "Proceed, drop bridge release and fix
  windows findings". Option A is accepted as written, including §3's
  `bridge-release` removal and the four Windows fixes.
