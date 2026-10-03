---
status: complete
date: 2026-10-02
associated-madr: "0009-MADR-rename-to-go-selfupdate-lib.md"
---
# Rename go-core-lib to go-selfupdate-lib (`v1.4.1`, `v1.5.0`)

Associated MADR: [0009-MADR-rename-to-go-selfupdate-lib.md](0009-MADR-rename-to-go-selfupdate-lib.md)

## Goal

* **The old path gets a last release.** `github.com/maccavelli/go-core-lib`
  ends with `v1.4.1`, which `go` reports as deprecated, pointing to the
  new path.
* **The repository is renamed.** It is `maccavelli/go-selfupdate-lib`.
* **The new path starts at `v1.5.0`.** `github.com/maccavelli/go-selfupdate-lib`
  has the same API as `v1.4.1`, and the API gate proves it.
* **Everything current names the new path:** code, tests, config, scripts
  and current docs. Records 0001–0008 stay as written (MADR Q4).

Done means every item under Verification holds, CI is green on `main` and on
both tags, and the proxy serves both.

## Scope

### In scope

| Step | Who | Paths | What |
| :--- | :--- | :--- | :--- |
| 1 | agent | `docs/decisions/0009-*`, `docs/README.md` | records |
| 2 | agent | `go.mod`, `README.md` | the deprecation notice, on the old path |
| 3 | owner, then agent | — | push, tag `v1.4.1`, CI, and the proxy check |
| 4 | owner, then agent | — | the GitHub rename, and the redirect check |
| 5 | agent | `scripts/check-api-compat.sh`, `scripts/check-api-compat_test.sh` | the gate across a path change (MADR §4) |
| 6 | agent | every non-record file that names the old path or name | the rename commit (MADR §5, §6) |
| 7 | owner, then agent | — | push, tag `v1.5.0`, CI, proxy checks, and the old path's `@latest` |
| 8 | owner, with the agent | the local checkout and the agent's memory | local follow-up |
| 9 | agent | this PLAN, `docs/README.md` | release notes and close-out |

### Out of scope

* **Other repositories** (MADR §8): go-tui-lib's `AGENTS.md` and
  `0001-MADR` §3, and pi-go's proposed `0004-MADR`. Each changes under its
  own records. This PLAN lists them in its close-out.
* **Records 0001–0008, and the records of other repositories.**
* **Any API change.** The API gate must report none.
* **Push, tags and the GitHub rename,** which are the owner's.

## Rules for every step

1. **Order.** Steps run in order. Step 4 (the rename) waits until step 3 has
   shown the deprecation through the proxy (MADR §2).
2. **Commits.** Use `git commit --no-edit`, after the owner authorizes
   commits to `main` in that turn. The disclosure guard and the marker
   scan must pass in the same command first (`&&`).
3. **Checks before each commit that changes code or config:**
   * `make pre-add-check` on the changed Go files;
   * `make lint`, three `GOOS`;
   * `go test -race -count=1 ./...` and `go mod tidy -diff`;
   * the Windows test host.
4. **Proofs on scratch copies,** never in the tree. Each new check is seen
   failing on a planted break.
5. **No shell scripts of the agent's own.** Session tooling is Python. The
   repository's own scripts and `make` targets run as they always do.

## Implementation Steps

### Step 1: records

* **The MADR** is `accepted`. The owner's answers to Q1–Q4 are recorded
  there, and so is the corrected `v1.4.1` confirmation.
* **This PLAN** is `in-progress` once the owner approves it.
* **`docs/README.md`** indexes both. Committed alone.

### Step 2: the deprecation notice (on the old path)

* **`go.mod`** gains, immediately before the `module` directive:

  ```go
  // Deprecated: renamed to github.com/maccavelli/go-selfupdate-lib. Use that module.
  ```

* **`README.md`** gains, under its title, one paragraph saying the module
  is renamed, the new path, and that `v1.4.1` is the old path's last
  release.
* **Checks:**
  * `go mod tidy -diff` stays clean, because a comment is not a
    requirement;
  * `go build ./...`, `go test -count=1 ./...` and `make lint`;
  * on a scratch clone with the change committed, a module that requires
    the clone through `replace` builds. That shows the comment does not
    disturb the `module` directive.
* Committed as one commit.

### Step 3: `v1.4.1` (owner, then agent)

1. **The owner** pushes, waits for CI on `main` to pass, tags `v1.4.1`
   (annotated) on the step 2 commit, and pushes the tag.
2. **The agent** checks each of these, and records the output:
   * `gh run` for `main` and for the tag: `success` on all three
     operating systems;
   * `GOPROXY=https://proxy.golang.org go list -m github.com/maccavelli/go-core-lib@v1.4.1`
     prints `v1.4.1`;
   * **the deprecation, as `go` reports it.** In a scratch module (`go mod
     init example.com/probe`, `go get github.com/maccavelli/go-core-lib@v1.4.0`
     through the proxy), `go list -m -u all` must print the old path with
     `(deprecated)`. `go list -m -u -json github.com/maccavelli/go-core-lib`
     must carry the message `renamed to github.com/maccavelli/go-selfupdate-lib.
     Use that module.`;
   * `git diff v1.4.0 v1.4.1 -- . ':!docs' ':!*.md'` holds only `go.mod`'s
     comment and `.golangci.yml`, and no `.go` file changes.
3. **If the deprecation does not show,** stop. The rename waits, because
   after it the old path cannot be fixed cleanly.

### Step 4: the rename on GitHub (owner, then agent)

1. **The owner** renames the repository to `go-selfupdate-lib`, in its
   Settings or with `gh repo rename go-selfupdate-lib -R maccavelli/go-core-lib`.
2. **The agent** checks each of these, read-only:
   * `gh repo view maccavelli/go-selfupdate-lib` resolves, public, with the
     same default branch;
   * `git ls-remote https://github.com/maccavelli/go-core-lib HEAD` still
     answers, through GitHub's redirect, with the same commit as the new
     URL;
   * the tags `v1.0.0`–`v1.4.1` are listed at the new URL.
3. **The local `origin` keeps working through the redirect.** Changing it is
   step 8, so that nothing is pushed in between under an unchecked remote.

### Step 5: the API gate across a path change

* **`scripts/check-api-compat.sh`:**
  * `MODULE` becomes the working tree's path, read with
    `go list -m` in the repository root;
  * the base's path is read with `go list -m` in the base worktree;
  * when the two differ, every `.go` file and `go.mod` in the base worktree
    has the base path rewritten to the working tree's, with the
    standard library's `go mod edit -module` for `go.mod` and an exact
    string replacement for imports. Then the export data is written.

  The script's message names both paths when it rewrote one.
* **`scripts/check-api-compat_test.sh`** gains case 6, "a base under
  another path compares by API":
  * the test's clone commits a path change (the current path to
    `example.com/renamed`, in `go.mod` and every import);
  * the gate against `BASE` must exit 0;
  * then a planted change of `DefaultLockTimeout` must exit 1, with that
    identifier in the report.
* **Proofs** (scratch clones):
  * the test passes, 7 of 7 or more;
  * with the rewrite removed from the gate, case 6 fails. It fails as the
    trial did, on loading or on the qualified-type report;
  * cases 1–5 are unchanged.
* **Checks:** shellcheck on both scripts, then `make apicheck` on the real
  tree, still `compatible with v1.4.1`.
* Committed as one commit, still on the old path, so the gate change is
  proven before the rename needs it.

### Step 6: the rename commit (on the new path)

1. **Mechanical.** A Python script replaces `github.com/maccavelli/go-core-lib`
   with `github.com/maccavelli/go-selfupdate-lib` in every tracked file
   except:
   * records (`docs/decisions/`, `docs/reports/`);
   * the deprecation paragraph in `README.md`, which names the old path on
     purpose.

   It asserts each file's count before and after. The expected total is
   the 53 occurrences in 24 files outside the docs that the MADR measured,
   plus those in current docs, which it lists.
2. **`go.mod`.** The deprecation comment goes; the `module` directive is the
   new path.
3. **Prose that names the module:**
   * `Makefile` line 1 becomes "go-selfupdate-lib — the fleet's self-update
     library";
   * `selfupdate/doc.go` names "the exact go-selfupdate-lib module-tag
     commit";
   * `AGENTS.md`'s identity paragraph states the scope of MADR §1: a
     self-update library, one top-level directory per package, no root
     package;
   * `README.md`'s title, opening, Status and the `go get` line name
     `v1.5.0`, and say the module was `go-core-lib` up to `v1.4.1`;
   * `docs/README.md`'s title. It also gains one line: records 0001–0008
     use the old name, and the code they describe moved with the rename;
   * `docs/architecture.md`: the name, the module path, the current
     release (`v1.5.0`, set in step 9 once tagged);
   * `docs/guides/*.md`: the name, the import paths, `go get` lines, and
     the `-X` names in the `Makefile` example. The guide's §5 heading text
     says "`v1.5.0` (formerly `go-core-lib` `v1.4.0`)".
4. **The `uses:` lines** (`README.md`; the migration guide's §3 diff and its
   `git ls-remote` line):
   * they name `maccavelli/go-selfupdate-lib`;
   * they pin the full commit of the `v1.4.1` tag, known from step 3;
   * the release workflow is unchanged since `v1.3.0`. Step 6 checks that
     with `git diff v1.3.0 v1.4.1 -- .github/workflows/publish-selfupdate-release.yml`,
     which must be empty, so the pin carries the `prerelease-channels-json`
     input the README documents. The old pin, `v1.0.0`, did not.
5. **Checks:**
   * **the marker scan,** a session script: no tracked file outside the
     records contains `go-core-lib`, except the README's deprecation
     paragraph, the README's "formerly" sentence, the guide's "formerly"
     sentence and the `docs/README.md` note;
   * build, vet, `go test -race -count=1 ./...`, `go test -shuffle=on
     -count=2 ./...`, `go mod tidy -diff`, `make lint` (three `GOOS`), `make
     vuln`, `make fuzz` and `make apicheck`. The gate compares with
     `v1.4.1` across the path change and must report compatible;
   * `scripts/check-api-compat_test.sh` and every other script test CI runs;
   * markdownlint, the link resolver, and the Windows test host;
   * **`TestStampedBinary`** builds with the new `buildinfo.LDFlags`, which
     proves the new `-X` names stamp a real binary;
   * **the depguard proofs of
     [0008-PLAN-enforce-import-rules-with-depguard.md](0008-PLAN-enforce-import-rules-with-depguard.md)**,
     re-run against the new path. In particular P9 now plants
     `go-selfupdate-lib/selfupdate/selfupdatetest` in `cli`. Every one must
     behave as before.
6. Committed as one commit.

### Step 7: `v1.5.0` (owner, then agent)

1. **The owner** pushes, waits for CI on `main`, tags `v1.5.0` (annotated)
   on the step 6 commit, and pushes the tag.
2. **The agent** checks each of these, and records the output:
   * CI on `main` and on the tag: `success`;
   * `GOPROXY=https://proxy.golang.org go list -m github.com/maccavelli/go-selfupdate-lib@latest`
     prints `v1.5.0`;
   * in a scratch module, `go get github.com/maccavelli/go-selfupdate-lib@v1.5.0`
     and a build that imports `…/selfupdate`, `…/selfupdate/cli` and
     `…/buildinfo` succeed;
   * **the old path's `@latest`** (MADR §7): `go list -m
     github.com/maccavelli/go-core-lib@latest` through the proxy, and `go
     list -m -u all` in step 3's scratch module. Whatever they print is
     recorded. If `@latest` no longer resolves to `v1.4.1`, that is the
     risk the MADR named, and it is recorded, not worked around.

### Step 8: local follow-up (owner, with the agent)

* **The owner:**
  * renames the local working directory to `go-selfupdate-lib`;
  * runs `git remote set-url origin` with the new URL;
  * confirms `git fetch` works.
* **The agent's per-project memory** is keyed by the working directory's
  path. After the directory moves, the agent copies its memory files to the
  new directory's key, and keeps the old key until the owner confirms the
  new one is read. This writes outside the repository, so it waits for the
  owner's go-ahead in that turn.

### Step 9: release notes and close-out

* **The execution record** gains release notes for `v1.4.1` and `v1.5.0`.
* **`docs/architecture.md`** names `v1.5.0`'s commit.
* **This PLAN** is `complete`, and `docs/README.md` says so.
* **Follow-ups for other repositories** are listed (MADR §8).

## Verification

* **Step 3:** CI is green on `v1.4.1`; the proxy serves it; `go list -m -u`
  reports the old path deprecated, with the message; the non-docs diff from
  `v1.4.0` is as stated.
* **Step 4:** the new name resolves, the old one redirects, and every tag is
  present.
* **Step 5:** the gate's new case passes, and fails without the rewrite.
* **Step 6:**
  * the marker scan is clean;
  * every check passes, including `make apicheck` reporting compatible with
    `v1.4.1` across the path change;
  * the depguard proofs behave as in 0008;
  * the Windows test host passes;
  * the disclosure guard passes.
* **Step 7:** CI is green on `v1.5.0`; the proxy's `@latest` for the new
  path is `v1.5.0`; a scratch consumer builds; the old path's `@latest` is
  recorded.
* `go.mod`'s requirements and `go.sum` are unchanged throughout.

## Rollout and Rollback

* **Rollout.** Steps 1–9, in order. No consumer is affected, because none
  imports the module (MADR, Evidence).
* **Rollback, by step:**
  * **Before step 3,** each commit is local.
  * **Between steps 3 and 4,** the rename can be abandoned. `v1.4.1` stays
    as a deprecated release, and a later `v1.4.2` could drop the comment.
  * **After step 4,** the repository can be renamed back on GitHub, as long
    as no repository has been created under the old name.
  * **After step 7,** the new path's releases are permanent. A problem is
    fixed forward in `v1.5.1`.

## Execution Record

### Step 1: records (2026-10-02)

The owner approved the PLAN ("approved, commit to main"). The MADR is
`accepted` with Q1–Q4 answered, and this PLAN is `in-progress`.

### Step 2: the deprecation notice (2026-10-02)

* **`go.mod`** carries `// Deprecated: renamed to
  github.com/maccavelli/go-selfupdate-lib. Use that module.`, immediately
  before the `module` directive.
* **`README.md`** opens with a "Renamed" note: the new path and repository,
  `v1.4.1` as the old path's last release, and `v1.5.0` as the first under
  the new one.

**Checks:**

| Check | Result |
| :--- | :--- |
| `go list -m` | `github.com/maccavelli/go-core-lib`, unchanged |
| `go mod tidy -diff` | rc 0 |
| `go build ./...`, `go test -count=1 ./...` | rc 0 |
| `make lint` | 0 issues for `GOOS=linux`, `darwin` and `windows` |
| markdownlint on `README.md` | 0 issues |
| `go mod edit -json` on a scratch clone | `Module.Deprecated` is `renamed to github.com/maccavelli/go-selfupdate-lib. Use that module.` |
| the same, with the marker lowercased (`// deprecated:`) | no `Deprecated` field: `go` reads only the exact marker |
| a scratch consumer importing `buildinfo` and `selfupdate` through `replace` | builds, rc 0 |
| Windows test host | `go vet` and `go test -race` rc 0; every script test rc 0 |

Committed as `58411f1`.

### Step 3: `v1.4.1` (2026-10-02)

* **The owner** pushed `58411f1`, and tagged and pushed `v1.4.1` (annotated)
  on it.
* **CI.** Run `37053181466` on `main` and run `37058909932` on the tag both
  concluded `success`, on `ubuntu-24.04`, `macos-15` and `windows-2025`.
* **Through `proxy.golang.org`**, from a scratch module:

  ```text
  $ go list -m github.com/maccavelli/go-core-lib@v1.4.1
  github.com/maccavelli/go-core-lib v1.4.1
  $ go list -m github.com/maccavelli/go-core-lib@latest
  github.com/maccavelli/go-core-lib v1.4.1
  $ go get github.com/maccavelli/go-core-lib@v1.4.0
  go: module github.com/maccavelli/go-core-lib is deprecated: renamed to github.com/maccavelli/go-selfupdate-lib. Use that module.
  $ go list -m -u all
  github.com/maccavelli/go-core-lib v1.4.0 [v1.4.1] (deprecated)
  ```

  `go list -m -u -json` carries `Deprecated`: `renamed to
  github.com/maccavelli/go-selfupdate-lib. Use that module.`
* **The non-docs diff from `v1.4.0`** is `.golangci.yml` (0008's rules, 70
  lines) and `go.mod`'s one comment line. No `.go` file or `go.sum` line
  changes.

Every step 3 check holds, so the rename may go ahead.

### Step 4: the rename (2026-10-02)

* **The owner** renamed the repository on GitHub.
* **Checked, read-only:**
  * `gh repo view maccavelli/go-selfupdate-lib`: public, default branch
    `main`;
  * `git ls-remote` gives `HEAD` `58411f1f7b00` at the new URL and at the
    old one, through GitHub's redirect, which matches local `main`;
  * the tags at the new URL are `v1.0.0`, `v1.0.1`, `v1.1.0`, `v1.2.0`,
    `v1.3.0`, `v1.3.1`, `v1.4.0` and `v1.4.1`;
  * the local `origin` still names the old URL, as planned. Step 8
    changes it.

### Step 5: the API gate across a path change (2026-10-02)

* **`scripts/check-api-compat.sh`:**
  * `MODULE` is read with `GOWORK=off go list -m`, no longer written in;
  * the base's path is read the same way in its worktree;
  * when the two differ, `go mod edit -module` sets the base's path, and
    `perl` replaces the old path with the new in every `.go` file. The
    replacement is exact, and only where the path is followed by `"` or
    `/`;
  * the message `… declares <old>; comparing it as <new>` goes to stderr.
* **`scripts/check-api-compat_test.sh`** gains case 3b. The clone's tree
  moves to `example.com/renamed`, and the gate against `BASE` must exit 0.
  A planted `DefaultLockTimeout` change must then exit 1, and the report
  must name it. That is three new checks.

**Checks.**

| Check | Result |
| :--- | :--- |
| shellcheck on both scripts | rc 0 |
| `make apicheck` (same path) | `compatible with v1.4.1` |
| `scripts/check-api-compat_test.sh` | `9 passed, 0 failed` |
| the same, on a scratch clone with the rewrite disabled (`if false`) | `6 passed, 3 failed`: `a base under another path compares by API: want exit 0, got 2`, and the two following checks |

The PLAN asked for 7 checks or more; the test has 9.

Committed as `fb78a16`.

### Step 6: the rename commit (2026-10-02)

**What changed.**

* **The mechanical pass** replaced the module path in every tracked file
  outside the records and `README.md`: 66 occurrences in 27 files. That is
  the MADR's 53 in 24 code, test and config files, plus 13 in three current
  documents: `AGENTS.md`, `docs/architecture.md` and both guides. Each file
  was checked to hold none of the old path afterwards.
* **`go.mod`:** the deprecation comment is gone; `go list -m` prints
  `github.com/maccavelli/go-selfupdate-lib`.
* **Prose:**
  * the `Makefile`'s first line;
  * `selfupdate/doc.go`;
  * `AGENTS.md`'s identity paragraph, which now states MADR §1's scope;
  * `docs/README.md`'s title, and its note on records 0001–0008;
  * `docs/architecture.md`'s opening;
  * the migration guide's title, `go 1.27.1` line, `go get` line (now
    `@v1.5.0`), §3 pin text, `git ls-remote` line (now `v1.4.1`), and §5's
    opening and `go.mod` line.
* **`README.md`, rewritten where it was stale:**
  * the title;
  * a "Formerly `go-core-lib`" note;
  * the self-update scope;
  * the new module line;
  * Status names `v1.5.0`, with a package table that now lists
    `selfupdate/cli`, `buildinfo` and `selfupdatetest`;
  * the publish paragraph;
  * an "I want to…" row for the rename.
* **The `uses:` lines** (`README.md` and the guide) name
  `maccavelli/go-selfupdate-lib` and pin
  `58411f1f7b00b5c98391c0f94503d09e0687d078`, the `v1.4.1` commit.
  `git diff v1.3.0 HEAD -- .github/workflows/publish-selfupdate-release.yml`
  is empty, so the pin carries the `prerelease-channels-json` input the
  README documents. The `v1.0.0` pin it replaces did not.
* **A pattern miss, fixed before any write to the guide.** The script's
  guide edits stopped on an assert. The mechanical pass had already turned
  the `git ls-remote` URL to the new path, so the pattern for the old one
  matched nothing. The asserts run before the file is written, so the
  guide was untouched. The edits were then applied by a second script with
  the corrected pattern.

**Checks.**

| Check | Result |
| :--- | :--- |
| `gofmt -l .` | empty |
| `go build ./...`, `go vet ./...` | rc 0 |
| `go mod tidy -diff` | rc 0; requirements and `go.sum` unchanged |
| `go test -race -count=1 ./...` | every package ok, under the new path |
| `go test -shuffle=on -count=2 ./...` | rc 0 |
| `TestStampedBinary` | pass: the new `-X` names stamp a real binary |
| `make lint` | 0 issues for `GOOS=linux`, `darwin` and `windows` |
| `make apicheck` | `v1.4.1 declares github.com/maccavelli/go-core-lib; comparing it as github.com/maccavelli/go-selfupdate-lib`, then `compatible with v1.4.1` |
| `make vuln` | `No vulnerabilities found.` |
| `make fuzz` | `5 fuzz targets ran clean in ./selfupdate` |
| script tests | `check-api-compat_test` 9/9, `check-release-tag_test` 51/51, `verify-selfupdate-release_test` all, `refuse-existing-release_test` 8/8, `go-fuzz_test` 12/12 |
| shellcheck, markdownlint-cli2, actionlint v1.7.12 | rc 0 each |
| cross `go vet` (freebsd, openbsd, linux/386) | rc 0 |
| 0008's depguard proofs P0–P12, on the new path | all as in 0008; P9 now plants `go-selfupdate-lib/selfupdate/selfupdatetest` in `cli`, and is refused by `selfupdate-cli` |
| the marker scan | 0 unexpected mentions. Allowed: the "formerly" notes in `AGENTS.md`, `README.md`, `docs/README.md`, `docs/architecture.md` and the guide, and record titles in the index. With a stray mention planted in a scratch copy it reported `selfupdate/doc.go:132` and failed |
| the link resolver, on the six current documents | no broken link |
| Windows test host | `go vet` rc 0; `go test -race` rc 0 for all four packages, under the new path; every script test rc 0 |

**Next:** step 7 is the owner's: push, tag `v1.5.0` on this commit, and push
the tag.

### Step 7: `v1.5.0` (2026-10-02)

* **The owner** pushed `main` at `6deaa52`, and tagged and pushed `v1.5.0`
  (annotated, tag object `ce5cf7e`) on it. `git ls-remote origin` shows
  `refs/tags/v1.5.0^{}` at `6deaa524cfb28aad90bea97a6d9162e5b4257204`.
* **CI.** Run `37062108333` on `main` and run `37077092111` on the tag both
  concluded `success`, on `ubuntu-24.04`, `macos-15` and `windows-2025`.
* **The new path, through `proxy.golang.org`**, from a scratch module with
  its own module cache:

  ```text
  $ go get github.com/maccavelli/go-selfupdate-lib@latest
  go: added github.com/maccavelli/go-selfupdate-lib v1.5.0
  $ go list -m -versions github.com/maccavelli/go-selfupdate-lib
  github.com/maccavelli/go-selfupdate-lib v1.0.0 v1.0.1 v1.1.0 v1.2.0 v1.3.0 v1.3.1 v1.4.0 v1.4.1 v1.5.0
  $ go get github.com/maccavelli/go-selfupdate-lib@v1.4.0
  go: github.com/maccavelli/go-selfupdate-lib@v1.4.0 requires github.com/maccavelli/go-selfupdate-lib@v1.4.0: parsing go.mod:
      module declares its path as: github.com/maccavelli/go-core-lib
              but was required as: github.com/maccavelli/go-selfupdate-lib
  ```

  The proxy's `@latest` endpoint answers `v1.5.0`, origin
  `refs/tags/v1.5.0` at `6deaa52`. The version list also names the eight
  tags from before the rename, because they are tags of the same
  repository. As MADR §3 states, none of them resolves under this path.
* **The scratch consumer** imports `…/buildinfo`, `…/selfupdate` and
  `…/selfupdate/cli`, and calls `buildinfo.Identity`, `cli.Help` and
  names `selfupdate.Request`. `go mod tidy`, `go build` and `go vet` exit 0.
  The binary runs and prints `dev (local) | true`. Tidy pulls
  `golang.org/x/mod v0.40.0`, `golang.org/x/sys v0.47.0` and
  `golang.org/x/term v0.43.0`, the requirements in `go.mod`.
* **The old path's `@latest` (MADR §7): the named risk happened.** From a
  second scratch module with its own module cache:

  ```text
  $ go list -m github.com/maccavelli/go-core-lib@latest
  github.com/maccavelli/go-core-lib v1.5.0
  $ go get github.com/maccavelli/go-core-lib@v1.4.0
  go: added github.com/maccavelli/go-core-lib v1.4.0
  $ go list -m -u all
  github.com/maccavelli/go-core-lib v1.4.0 [v1.5.0]
  $ go get github.com/maccavelli/go-core-lib@latest
  go: github.com/maccavelli/go-core-lib@latest (v1.5.0) requires github.com/maccavelli/go-core-lib@v1.5.0: parsing go.mod:
      module declares its path as: github.com/maccavelli/go-selfupdate-lib
              but was required as: github.com/maccavelli/go-core-lib
  $ go get -u ./...
  go: github.com/maccavelli/go-core-lib@v1.5.0: parsing go.mod:
      module declares its path as: github.com/maccavelli/go-selfupdate-lib
              but was required as: github.com/maccavelli/go-core-lib
      trying github.com/maccavelli/go-core-lib@v1.4.1
  ...
  go: upgraded github.com/maccavelli/go-core-lib v1.4.0 => v1.4.1
  ```

  * The proxy's version list for the old path now includes `v1.5.0`,
    whose `.mod` declares the new path. `go` takes `@latest` from that
    list. The proxy's own `@latest` endpoint still answers `v1.4.1`, but
    `go` does not use it while the list is non-empty.
  * **The deprecation notice is lost.** Step 3 recorded it on `go get
    …@v1.4.0` and in `go list -m -u all` (`(deprecated)`). Neither shows it
    now, and `go list -m -u -json` has no `Deprecated` field, because `go`
    reads it from `v1.5.0`'s `go.mod`, which has none.
  * **What still works:**
    * pinned versions resolve, `v1.4.0` and `v1.4.1` included;
    * `go get -u` retries, and lands on `v1.4.1`.
  * **What fails:** `go get …@latest` on the old path. It fails loudly, with
    a message that names the new path.
  * As MADR §7 and step 7 direct, this is recorded and not worked around.
    No program depends on the old path (MADR, "Who depends on it").

**Next:**

* step 8, the owner's: rename the local directory and set the remote URL;
* then step 9: the release notes, which state the old-path behaviour above.

### Step 8: local follow-up (2026-10-02)

* **The working directory** had already been renamed to
  `go-selfupdate-lib` by the owner.
* **The remote.** `origin` still named `…/go-core-lib.git` for fetch and
  push, and worked only through GitHub's redirect. It now names
  `https://github.com/maccavelli/go-selfupdate-lib.git` for both:

  ```text
  $ git remote -v
  origin  https://github.com/maccavelli/go-selfupdate-lib.git (fetch)
  origin  https://github.com/maccavelli/go-selfupdate-lib.git (push)
  $ git fetch origin; echo $?
  0
  $ git rev-parse main origin/main
  2ab13aa45a46d575282200f5825a7ecbe9d7cd3e
  2ab13aa45a46d575282200f5825a7ecbe9d7cd3e
  ```

  `branch.main` still tracks `origin/main`. `gh repo view` resolves both the
  new and the old name to `maccavelli/go-selfupdate-lib`, not archived,
  default branch `main`.
* **Identity, checked, unchanged.** `user.name` and `user.email` are set in
  the repository's local config and agree with the global config. The last
  five commits carry that identity as author and committer. Hooks resolve
  to the global hooks directory.
* **The agent's per-project memory.** The old directory's key held
  `MEMORY.md` and one feedback memory; the new key's directory was empty.
  Both files were copied unchanged (`cmp` rc 0 for each). The old key is
  kept until the owner confirms that a session under the new directory
  reads them.
* **Deviation (2026-10-02).** The step names the owner as running
  `git remote set-url`. The agent ran it at the owner's request, in the
  turn the owner asked for the remote to be put right. The command and the
  check are as written; only who ran it changed.

**Next:**

* the owner: confirm that a new session here loads the copied memory; the
  old key is removed only on the owner's go-ahead;
* step 9: release notes and close-out.

### Step 9: release notes and close-out (2026-10-02)

The owner approved step 9 ("proceed").

* **`docs/architecture.md`** names `v1.5.0` as the current release, the
  annotated tag on `6deaa524cfb28aad90bea97a6d9162e5b4257204`.
* **This PLAN** is `complete`, and `docs/README.md` says so.
* **The per-project memory, under the new directory.** This session runs
  under the new directory's key. Its context loaded the copied `MEMORY.md`
  and the memory it indexes, so the new key is read. The old key is still
  on disk. Removing it writes outside the repository, so it waits for the
  owner's go-ahead.
* **CI on `main`.** Run `37080415438` on `2ab13aa` concluded `success`.
  Run `37084275297` on `cb902c4`, step 8's record, concluded `success` on
  `ubuntu-24.04`, `macos-15` and `windows-2025`.

### Release notes for `v1.4.1`

The last release under `github.com/maccavelli/go-core-lib`. It has the same
API and code as `v1.4.0`.

* **`go.mod`** marks the module deprecated: `renamed to
  github.com/maccavelli/go-selfupdate-lib. Use that module.`
* **`README.md`** opens with a "Renamed" note naming the new path, and
  `v1.5.0` as its first release.
* **`.golangci.yml`** carries the `depguard` import rules of
  [0008-MADR-enforce-import-rules-with-depguard.md](0008-MADR-enforce-import-rules-with-depguard.md).
  Nothing else outside the documentation changes: no `.go` file and no
  `go.sum` line.

**For consumers.**

* Pinned versions of the old path, `v1.0.0`–`v1.4.1`, keep resolving.
* Move to the new path at `v1.5.0` (below).
* **The deprecation notice is no longer shown** (step 7). Since `v1.5.0`
  was tagged, `go` resolves the old path's `@latest` to `v1.5.0`. Its
  `go.mod` declares the new path, so:
  * `go get github.com/maccavelli/go-core-lib@latest` fails, with a
    message that names the new path;
  * `go get -u` retries, and lands on `v1.4.1`;
  * `go list -m -u` no longer says `(deprecated)`.

### Release notes for `v1.5.0`

The first release under `github.com/maccavelli/go-selfupdate-lib`. The
repository was renamed from `maccavelli/go-core-lib`. The API is
`v1.4.1`'s, unchanged. `make apicheck` reports it compatible across the
path change.

* **The module path** is `github.com/maccavelli/go-selfupdate-lib`, for
  `selfupdate`, `selfupdate/cli`, `selfupdate/selfupdatetest` and
  `buildinfo`.
* **Requirements are unchanged.** Go 1.27.1, `golang.org/x/mod v0.40.0`,
  `golang.org/x/sys v0.47.0` and `golang.org/x/term v0.43.0`.
* **`buildinfo`'s `-X` symbols change with the path** (MADR §5).
  `VersionVar` and `KindVar` are now
  `github.com/maccavelli/go-selfupdate-lib/buildinfo.version` and
  `….kind`.
  * A program that stamps with `buildinfo.LDFlags` follows automatically.
  * **A program that writes the `-X` names out must change them.** If it
    does not, its release binary is stamped as a local build.
* **The reusable release workflow** is
  `maccavelli/go-selfupdate-lib/.github/workflows/publish-selfupdate-release.yml`.
  `README.md` and the migration guide pin it to `58411f1`, the `v1.4.1`
  commit.
* **Tooling.** `scripts/check-api-compat.sh` compares against a base
  release that declares another module path. It rewrites that path in its
  scratch worktree before comparing.

**For consumers.**

* Replace `github.com/maccavelli/go-core-lib` with
  `github.com/maccavelli/go-selfupdate-lib` in `go.mod` and in every
  import, then `go get github.com/maccavelli/go-selfupdate-lib@v1.5.0`.
* **The older tags are not versions of the new path.** `v1.0.0`–`v1.4.1`
  declare the old path, so `go get
  github.com/maccavelli/go-selfupdate-lib@v1.4.0` fails with a path
  mismatch (MADR §3). The proxy's version list still names them, because
  they are tags of the same repository.
* Apart from the `-X` names above, behaviour at run time does not change.

### Follow-ups in other repositories (MADR §8)

Each changes under its own records. None is blocked, since none imports
this module.

* **go-tui-lib.** `AGENTS.md`'s allowed-dependency list and `0001-MADR` §3
  name `github.com/maccavelli/go-core-lib`. Both must name
  `github.com/maccavelli/go-selfupdate-lib` before it adds `updatetea`.
* **pi-go.** Its proposed `0004-MADR` plans to depend on this module, and
  should name the new path, at `v1.5.0` or later.
* **Records elsewhere** that mention the old name stay as written.

### Close-out

Every item under Verification holds. CI is green on `v1.4.1`, on `v1.5.0`
and on `main` through `cb902c4`. The proxy serves both paths. Step 7
records the old path's `@latest` as MADR §7 asked. `go.mod`'s requirements
and `go.sum` did not change.

This PLAN is `complete`. These items stay open, and are the owner's:

* **Remove the old memory key,** now that the new one is read.
* **The go-tui-lib and pi-go follow-ups** above.
