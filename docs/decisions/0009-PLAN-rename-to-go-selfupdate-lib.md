---
status: in-progress
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
