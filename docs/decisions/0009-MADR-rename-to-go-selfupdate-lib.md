---
status: accepted
date: 2026-10-02
decision-makers: owner
consulted: GitHub's "Renaming a repository" documentation; the Go modules reference (go.dev/ref/mod); a rename trial on a scratch clone of this repository
informed: go-tui-lib, pi-go and the six programs on selfupdate (none of which depends on this module yet)
---
# Rename go-core-lib to go-selfupdate-lib: rename the repository in place, deprecate the old module path first, and publish the new path from v1.5.0

## Context and Problem Statement

On 2026-10-02 the owner wrote:

> I need to rename this project. This includes the repository, all code
> references, pathing, and docs. The new name is go-selfupdate-lib. Determine
> the best way to accomplish this. This shared library is going to stay limited
> to the self update functionality and naming it core is just not right.

[0001-MADR-scaffold-shared-go-library.md](0001-MADR-scaffold-shared-go-library.md)
§1 set the scope as "general-purpose Go packages shared by fleet programs". In
practice the module holds one capability, self-update, and the owner now
limits it to that. The name should say so.

A Go module's name is not just a label. It is the import path of every package,
the path the module proxy and checksum database have recorded for seven
releases, part of the linker symbols that stamp a release build, and the
repository name that other workflows call. This record decides how to change all
of these, and in what order.

### Evidence

Gathered read-only on 2026-10-02, at `7bc496f` (clean, level with
`origin/main`). The trials ran on scratch clones, never in the tree.

**Where the name appears in this repository.**

* `go-core-lib` appears 165 times in 48 tracked files, not counting this
  record. The counts are exact, from `namecount.py`, not `git grep -c`,
  which counts lines.
* **Outside the docs and Markdown: 55 times in 26 files.**
  * 53 are the module path `github.com/maccavelli/go-core-lib`, in 24 files:
    * `go.mod`;
    * the import lines of `selfupdate/cli` and `selfupdate/selfupdatetest`;
    * 15 test files: import lines, and in `buildinfo`'s tests the expected
      `-X` strings;
    * `buildinfo`'s `VersionVar` and `KindVar` constants, and its package
      comment;
    * the three `depguard` allow entries in `.golangci.yml`;
    * `MODULE` in `scripts/check-api-compat.sh`.
  * 2 are prose: the `Makefile`'s first line ("shared Go library of
    general-purpose packages") and a sentence in `selfupdate/doc.go`.
* **Current-state documents:**

  | File | Uses | Note |
  | :--- | :--- | :--- |
  | `README.md` | 6 | Also stale: it still calls `v1.0.0` the current release |
  | `AGENTS.md` | 2 | Its identity paragraph also describes a multi-capability library |
  | `docs/architecture.md` | 2 | |
  | `docs/guides/migrating-from-mcplib-selfupdate.md` | 17 | |
  | `docs/guides/extending-selfupdate.md` | 1 | |
  | `docs/README.md` | 2 | |

* **Records:** 80 uses in 16 MADR, PLAN and REPORT files. They describe
  what was done under the old name.

**A trial rename** (`rename_trial.py`, on a scratch clone).

* It rewrote the module path in every file outside `docs/` and `*.md`: 53
  occurrences in 24 files.
* After that, every check passed:
  * `go build ./...`, `go vet ./...` and `go test -count=1 ./...`;
  * `go mod tidy -diff`;
  * `golangci-lint run` with this repository's config, including the
    `depguard` rules of
    [0008-MADR-enforce-import-rules-with-depguard.md](0008-MADR-enforce-import-rules-with-depguard.md):
    0 issues.
* The only mentions left outside the docs were the two prose lines above.

The code change is mechanical.

**The API gate does not survive the rename as written.** It compares the working
tree with the newest `v1.*` tag through one `MODULE` path.

* **One path for both sides.** In the renamed clone the gate failed:
  `loading github.com/maccavelli/go-core-lib: found no packages for module
  github.com/maccavelli/go-core-lib`.
* **Each side under its own path.** apidiff then reported false
  incompatibilities, because it qualifies types by their full path:

  ```text
  - ./selfupdate/selfupdatetest.NewRelease: changed from func(string, string,
    []github.com/maccavelli/go-core-lib/selfupdate.Platform, …) ReleaseSpec
    to func(string, string, []github.com/maccavelli/go-selfupdate-lib/selfupdate.Platform, …) ReleaseSpec
  ```

* **The old tag's path rewritten in its scratch checkout first**
  (`rename_trial2.py`):
  * `v1.4.0` against the renamed tree reported nothing;
  * with `DefaultLockTimeout` changed from 5 s to 6 s, it reported
    `- ./selfupdate.DefaultLockTimeout: value changed from 5000000000 to
    6000000000`.

  So that comparison is real, and it still catches an incompatible change.

**The release workflow adapts.**
`.github/workflows/publish-selfupdate-release.yml` takes its repository
from `${{ github.repository }}` (lines 93, 134, 163 and 173), and names
no repository itself.

**What GitHub does on a rename** ("Renaming a repository", GitHub Docs, read
2026-10-02):

* "all `git clone`, `git fetch`, or `git push` operations targeting the
  previous location will continue to function as if made on the new
  location", and issues, wikis, stars and followers redirect.
* "GitHub will not redirect calls to an action hosted by a renamed
  repository". Workflows that use the old name "will fail with the error
  `repository not found`".
* Do not create a new repository under the old name, or "redirects to the
  renamed repository will no longer work".

So the documented `uses: maccavelli/go-core-lib/.github/workflows/publish-selfupdate-release.yml@…`
lines must change (`README.md` line 86; the migration guide line 75). The
pinned commit hashes stay valid, because a rename does not change commits.

Not re-checked: the docs say "action". That reusable workflows behave the
same is inferred, not tested. No caller exists today (below), so nothing
breaks either way.

**What Go does** (the Go modules reference, go.dev/ref/mod, read 2026-10-02):

* A module is deprecated by a comment beginning `Deprecated:` immediately
  before the `module` directive, or after it on the same line.
* "When the go command retrieves deprecation information for a module, it
  loads the go.mod file from the version matching the @latest version
  query". `go get` and `go list -m -u` report it (Go 1.17 and later).
* A module's `module` directive must match the path it is fetched by.

So a deprecation notice reaches users of the old path only through a release
whose `go.mod` still declares the old path. Any release after the rename
declares the new one.

**What is published.**

* Tags `v1.0.0` to `v1.4.0` (seven) are published under
  `github.com/maccavelli/go-core-lib`. The proxy serves their archives
  permanently, and the checksum database records them.
* The new name is free:
  * `gh repo view maccavelli/go-selfupdate-lib` reports "Could not
    resolve to a Repository";
  * `proxy.golang.org/github.com/maccavelli/go-selfupdate-lib/@v/list`
    returns HTTP 404.

**The repository on GitHub:** public, 0 stars, 0 forks, no Pages site, no
rulesets, `main` not protected, wiki enabled, no description or topics.

**Who depends on it.** Searched in every repository under the fleet's local
checkout:

* **No repository requires the module or calls its workflow.** The six
  `selfupdate` programs are still on `mcplib`'s copy.
* **Documents in other repositories name it:**

  | Repository | Where |
  | :--- | :--- |
  | go-tui-lib | `AGENTS.md`, its allowed-dependency list; `0001-MADR` §3; comments in `ci.yml`, `go-precheck.sh` and `go-fuzz.sh`; records |
  | pi-go | its proposed records `0003`, `0004` and `0005`, and `README.md` |
  | go-llmprovider-sdk, mcplib, magic-cli-remote | records only |

**On this machine:**

* the working directory is named `go-core-lib`;
* `origin` points at the old URL;
* the agent's per-project memory is stored under a key derived from the
  directory path.

## Decision Drivers

* **The name states the scope.** The owner limits the library to
  self-update, and "core" promises more.
* **Nothing a consumer has pinned may break.** Seven releases are
  immutable under the old path.
* **A consumer of the old path should be told where to go**, by the Go
  tooling itself.
* **History stays whole:** commits, tags, CI runs, records and redirects.
* **The gates keep working across the rename,** above all the API gate,
  which must prove the API did not change.
* **Every step is ordered,** because a rename cannot be undone cleanly
  once another repository takes the old name.

## Considered Options

* **A. Rename the repository in place. Publish a deprecation release on the old path first, then the new path from `v1.5.0`, with the package layout unchanged.**
* **B. Create a new repository** for the new name, push the history to it, and archive `go-core-lib`.
* **C. As A, and also flatten the layout,** putting package `selfupdate` at the module root.
* **D. Rename the repository and keep the module path `github.com/maccavelli/go-core-lib`.**
* **E. Keep the name.**

## Decision Outcome

Chosen option: **"A"**, because:

* it renames everything the owner named, in one repository, with GitHub's
  redirects for clones and links;
* it keeps every published release valid;
* it tells old-path users, through `go`, where the module went;
* the trial shows the change is mechanical, and the API gate can prove
  `v1.5.0`'s API identical to `v1.4.0`'s.

### 1. Scope

* **The library is limited to self-update.** That means `selfupdate`, its
  `cli` and `selfupdatetest` subpackages, and `buildinfo`, which stamps the
  identity that `selfupdate` decides on.
* **This supersedes, in part,
  [0001-MADR-scaffold-shared-go-library.md](0001-MADR-scaffold-shared-go-library.md):**
  * its §1 scope, "general-purpose Go packages shared by fleet programs";
  * the identity paragraphs in `README.md` and `AGENTS.md`.
* **Its §2 layout stays,** one top-level directory per package.
* **A new package must serve self-update.** Anything else belongs in
  another module.

### 2. The order

Each step is done, and checked, before the next starts.

1. **The deprecation release, on the old path.** A commit adds this to
   `go.mod`:

   ```go
   // Deprecated: renamed to github.com/maccavelli/go-selfupdate-lib. Use that module.
   module github.com/maccavelli/go-core-lib
   ```

   * The README says the same.
   * The owner tags it `v1.4.1`. Its Go code, `go.mod` requirements and
     `go.sum` are `v1.4.0`'s. It also carries
     [0008-MADR-enforce-import-rules-with-depguard.md](0008-MADR-enforce-import-rules-with-depguard.md)'s
     lint rules, which landed after `v1.4.0`.
   * Before going on, the proxy must serve it: `go list -m -u` against the
     proxy reports the old path `(deprecated)`.
2. **The owner renames the repository on GitHub,** to `go-selfupdate-lib`.
   This is the owner's action, with the owner's credentials.
3. **The rename commit, on the new path.** It changes:
   * the module path in `go.mod` and every import;
   * `buildinfo`'s symbols;
   * the `depguard` allow entries;
   * the API gate (§4);
   * the current-state docs (§6).

   The owner tags it `v1.5.0`.
4. **Local follow-up, by the owner:**
   * rename the working directory;
   * `git remote set-url origin` to the new URL. The old one redirects, so
     this is tidiness, not repair;
   * move the agent's per-project memory to the new directory's key.
5. **Other repositories** amend their own documents, under their own
   records (§8).

Steps 1 and 2 cannot be swapped. Once the repository is renamed, a new
release on the old path would be fetched through GitHub's redirect from a
repository whose newer tags declare the new path. Whether the proxy would
still publish it cleanly is not something to find out after the fact.

### 3. Versions

* **The new path starts at `v1.5.0`.**
  * Tags are shared by one repository, so `v1.0.0` and the others cannot
    be reused for the new path.
  * Continuing the sequence says truthfully that this is the same API, one
    minor release on.
  * `v2.0.0` would claim a breaking API change that does not exist, and
    would require a `/v2` path suffix.
* **The old path's tags `v1.0.0`–`v1.4.0` stay valid** for anyone who
  pinned them. `v1.4.1` is their last release.
* **Under the new path, the older tags are not valid versions.** Their
  `go.mod` declares the old path, so `go get
  github.com/maccavelli/go-selfupdate-lib@v1.4.0` fails with a path
  mismatch. `@latest` is `v1.5.0`. The release notes say so.

### 4. The API gate across the rename

`scripts/check-api-compat.sh` learns two paths:

* it reads the base's module path from the base checkout's `go.mod`, and
  the working tree's from its own;
* when they differ, it rewrites the base's module path in its scratch
  worktree, in `.go`, `go.mod` and scripts, before writing the export data.

This is exactly what `rename_trial2.py` did. It made the comparison clean,
and still caught a real change. The gate's own test,
`scripts/check-api-compat_test.sh`, gains a case for a base under a
different path. That case, and the rest of the gate, is proven in the
PLAN on a scratch clone.

### 5. `buildinfo`'s symbols

* `VersionVar` and `KindVar` become
  `github.com/maccavelli/go-selfupdate-lib/buildinfo.version` and
  `….kind`.
* A program that stamps with `buildinfo.LDFlags` follows automatically. One
  that writes the `-X` names out, as the migration guide's `Makefile` does,
  must change them, or its release binary is stamped as a local build. That
  is C2 of
  [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md),
  and the release notes say so prominently.
* No program has adopted `buildinfo` yet, so nothing in the field changes.

### 6. Files

| Area | Change |
| :--- | :--- |
| `go.mod` | the module path; the deprecation comment (step 1 only) |
| Go code and tests | import paths; `buildinfo`'s constants and package comment |
| `.golangci.yml` | three `depguard` allow entries; the `module` rule's own-module entry |
| `scripts/check-api-compat.sh` and its test | §4 |
| `Makefile` | its first line, now "self-update library" |
| `README.md` | the name, the scope, the current release (`v1.5.0`), the import path, the `uses:` line |
| `AGENTS.md` | the identity paragraph: one capability, self-update |
| `docs/architecture.md`, `docs/guides/*.md` | the name, paths and `uses:` lines |
| `docs/README.md` | the title, and a line saying the records before 0009 use the old name |

**Records 0001–0008 are not rewritten.** They describe what was done,
under the name the module had then. `docs/README.md` points this out
instead.

### 7. After the rename: the old path's `@latest`

* **Unknown.** After step 2, the proxy may see the renamed repository's
  newer tags when asked about the old path. If it then resolves the old
  path's `@latest` to `v1.5.0`, whose `go.mod` declares the new path, then
  `go get github.com/maccavelli/go-core-lib@latest` fails with a path
  mismatch. The deprecation message would not be shown, since Go reads it
  from `@latest`.
* **Why it is acceptable.** This cannot be tested before the rename, and
  this record does not claim either outcome. No program depends on the old
  path today.
* **What the PLAN does.** It checks the old path's `@latest` after step 3,
  and records what it finds. A pinned version, such as `@v1.4.0`, is
  unaffected either way.

### 8. Other repositories

Each changes under its own records. None is blocked, since none imports the
module.

* **go-tui-lib.** Its `AGENTS.md` allowed-dependency list and `0001-MADR`
  §3 name `github.com/maccavelli/go-core-lib`. Both must name the new path
  before go-tui-lib adds `updatetea`.
* **pi-go.** Its proposed `0004-MADR` plans to depend on this module.
* **Records elsewhere** that mention the old name stay as written.

### Consequences

* Good, because the name says what the module is, and the scope is written
  down.
* Good, because every published release keeps working, and old-path users
  get a deprecation notice from `go` itself, with the caveat in §7.
* Good, because the trial shows the code change is mechanical: build, vet,
  tests, tidy and lint all passed after it.
* Good, because the API gate proves `v1.5.0` changes nothing but the path.
* Neutral, because the old path keeps working forever in the proxy, frozen
  at `v1.4.1`.
* Bad, because every consumer's import path changes. None has adopted the
  module yet, so the cost is lowest now.
* Bad, because a program that writes `buildinfo`'s `-X` names out must
  change them, or its releases look local.
* Bad, because the old path's `@latest` after the rename is not known in
  advance (§7).

### Confirmation

* **Step 1:**
  * `go list -m -u github.com/maccavelli/go-core-lib` against the proxy
    reports `(deprecated)` with the message;
  * `v1.4.1`'s non-docs diff from `v1.4.0` is the `go.mod` deprecation
    comment, plus `.golangci.yml` from 0008. No `.go` file, requirement
    or checksum changes. *(Corrected 2026-10-02, before acceptance: the
    draft said `go.mod` alone, overlooking 0008's commits since `v1.4.0`.)*
* **Step 3:**
  * no file outside `docs/decisions/` and `docs/reports/` contains
    `go-core-lib`, except the deprecation notes and the migration notes
    that name it on purpose;
  * build, vet, test with `-race`, tidy, lint on three operating systems,
    govulncheck and fuzz all pass;
  * the API gate against `v1.4.1` (the newest tag) reports compatible, and
    against a planted change reports it;
  * the gate's new test case fails on a scratch copy without the §4 change;
  * the disclosure guard passes;
  * CI is green on `main` and on the `v1.5.0` tag.
* **After the release:**
  * `go list -m github.com/maccavelli/go-selfupdate-lib@latest` reports
    `v1.5.0`;
  * the old path's `@latest` is recorded, whatever it is (§7).

## Pros and Cons of the Options

### A. Rename in place, deprecate first, new path from v1.5.0

* Good, because it is one repository, with GitHub's redirects for clones,
  links, issues and the wiki.
* Good, because old-path users get a deprecation notice through `go`.
* Good, because the API gate proves continuity.
* Bad, because the old path's `@latest` after the rename is uncertain (§7).

### B. A new repository, the old one archived

* Good, because the old path stays a clean, archived repository whose
  `@latest` is certainly `v1.4.1`, deprecation included.
* Bad, because there are two repositories. Issues, the wiki, the CI
  history and the repository URL do not move.
* Bad, because pushing the history means pushing the old tags, which are
  invalid versions under the new path anyway, or a history without them.
  Either way it duplicates what A gets for free.
* Bad, because the owner asked to rename the repository.

### C. As A, and flatten the layout

* Good, because the import path loses its stutter:
  `github.com/maccavelli/go-selfupdate-lib` instead of `…/selfupdate`.
* Good, because every import path changes anyway, so consumers pay the
  move once.
* Bad, because about 100 files move:
  * the scripts, the `Makefile`'s fuzz target, the `depguard` globs,
    `ci.yml` and the golden and fixture paths all name `selfupdate/`;
  * every record's file citation goes stale.
* Bad, because the API gate compares by path within the module
  (`./selfupdate.X`). Moving the package to the root makes every
  identifier look removed. The gate would need a second translation, or a
  reset, in the very release that is meant to prove nothing changed.
* Bad, because the package name `selfupdate` would no longer match the
  last element of its import path, which Go style discourages.

### D. Rename the repository, keep the module path

* Good, because nothing in the code changes.
* Bad, because the import path, and so every consumer's code, would say
  `go-core-lib` forever, the name the owner rejects.
* Bad, because `go get` fetches by the path, through GitHub's redirect.
  The mismatch between the repository name and the module would confuse
  every reader.

### E. Keep the name

* Good, because there is no work.
* Bad, because it is what the owner asked to change.

## Owner questions

*Answered 2026-10-02: "q1. v1.5.0 q2. keep the layout. q3. yes q4.
yes". Each is the recommendation: `v1.5.0`; the layout kept; the
deprecation release `v1.4.1` before the rename; records 0001–0008 left as
written.*

* **Q1. The first version under the new path.** Recommended: `v1.5.0`, so
  the tag sequence and the API continue. The alternative is `v2.0.0` with
  a `/v2` path, which claims a breaking change that does not exist.
* **Q2. The layout.** Recommended: keep it (option A), with import paths
  `…/go-selfupdate-lib/selfupdate`, `…/selfupdate/cli` and `…/buildinfo`.
  The alternative is flattening (option C), which, if it is wanted, is
  cheapest now, at the costs listed above.
* **Q3. The deprecation release `v1.4.1` before the rename.** Recommended:
  yes. The alternative is to skip it: old-path users then get no notice
  from `go`, only a stale module.
* **Q4. The records.** Recommended: leave 0001–0008 as written, with a note
  in `docs/README.md`. The alternative, rewriting them, would make them
  say things that were not true when they were written.

## More Information

* GitHub Docs, "Renaming a repository":
  <https://docs.github.com/en/repositories/creating-and-managing-repositories/renaming-a-repository>
* The Go modules reference, `go.mod` `module` directive and deprecation:
  <https://go.dev/ref/mod>
* [0001-MADR-scaffold-shared-go-library.md](0001-MADR-scaffold-shared-go-library.md):
  §1 scope and §2 layout.
* [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md):
  the API this rename must not change, and C2.
* [0008-MADR-enforce-import-rules-with-depguard.md](0008-MADR-enforce-import-rules-with-depguard.md):
  the rules whose allow entries change.
* The trial scripts `rename_trial.py` and `rename_trial2.py` live in the
  session's scratch space. Their results are quoted above.
