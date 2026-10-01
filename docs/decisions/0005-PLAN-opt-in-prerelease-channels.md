---
status: in-progress
date: 2026-10-01
associated-madr: "0005-MADR-opt-in-prerelease-channels.md"
---
# Implement opt-in prerelease channels (`v1.3.0`)

Associated MADR: [0005-MADR-opt-in-prerelease-channels.md](0005-MADR-opt-in-prerelease-channels.md)

This PLAN implements the MADR's Decision Outcome §1–§5. The owner accepted it
on 2026-10-01 with the four recommended answers:

* named channels;
* the restricted `-NAME.N` grammar;
* publication in this record;
* a pinned prerelease needs a channel that admits it.

The MADR leaves names and shapes to the PLAN. Where this PLAN settles
something differently from the MADR's text, the change is under "Proposed
MADR amendments", and Step 1 applies it.

## Goal

`v1.3.0` lets a program offer channels such as `rc` and `beta`:

* a stable user never sees a prerelease, whether attended or unattended;
* a channel user gets the newest admissible build, under every stable
  check;
* the reusable workflow can publish a prerelease that never becomes
  "latest".

Each item comes with:

* tests that fail when the behaviour they guard is broken, with every
  mutation killed;
* green CI on Linux, macOS and Windows;
* `make apicheck` against `v1.2.0` reporting no incompatible change.

`go.mod` does not change.

## Scope

### Code facts at `v1.2.0`

| Fact | Evidence |
| :--- | :--- |
| `VersionPolicy` has only `Validate` and `Compare`, so a new method would break custom policies. | `selfupdate/types.go`, `type VersionPolicy` |
| The strict policy is a regular expression plus `semver.IsValid`, and compares with `semver.Compare`. | `selfupdate/version.go` |
| `validateRequest(req, versions)` validates `CurrentVersion` and `TargetVersion` with the configured policy (Phase 1, G2). | `selfupdate/version.go` |
| `Checker.prepare` turns a `CheckRequest` into a check-only `Request`. `CheckCached` keys on the `CheckRequest`, compared with `==`. | `selfupdate/checker.go`; `selfupdate/checkcache.go` |
| The cache file is schema 1, and `Load` refuses any other schema as `ErrNoCheckRecord`. | `selfupdate/checkcache.go`, `fileCheckRecord`, `Load` |
| `GitHubSource.getRelease` calls `validateFetchedRelease`, which refuses drafts, prereleases and mutable releases, for `Latest` and `ByTag` alike. | `selfupdate/github.go` |
| `Checker.discover` refuses `Draft \|\| Prerelease` again, then validates the tag. | `selfupdate/checker.go` |
| The default `Limits.ReleaseJSON` is 2 MiB, which bounds one response body. | `selfupdate/types.go`, `DefaultLimits` |
| The workflow's tag gate is an inline `grep -Eq` on the strict pattern, and the release is created with `gh release create "$TAG" --draft --verify-tag --generate-notes`. | `.github/workflows/publish-selfupdate-release.yml`, "Require a strict stable tag" and the create step |
| `selfupdatetest.FakeSource` keeps its releases in a map, and `GitHubServer` serves `latest`, `tags/{tag}` and `assets/{id}`, but no list. | `selfupdate/selfupdatetest/` |
| Existing tests that assert today's refusal: `github_test.go:219-239`, where the source refuses a prerelease, and `checker_test.go:40`, where discovery refuses one. | as named |

### Item → step

| Item | Step |
| :--- | :--- |
| records; MADR amendments E1–E5 | 1 |
| `NewSemverPolicy`, `SemverOptions`, `ChannelPolicy` | 2 |
| `Request.Channel`, `CheckRequest.Channel`; validation; the cache key and file schema 2 | 3 |
| `ReleaseLister`, `ListOptions`, `GitHubSource.ListReleases`; the source stops refusing prereleases; the `selfupdatetest` list support | 4 |
| channel discovery | 5 |
| publication: `scripts/check-release-tag.sh`, the workflow, the verifier | 6 |
| the end-to-end channel case; docs, examples and close-out | 7 |

### Out of scope

* The `--channel` flag (Phase 3, `selfupdate/cli`).
* Mutable hosts (0004-MADR §6).
* Automatic channel switching (MADR §6).
* Any `git push` or tag.

### Fixed inputs

| Input | Value |
| :--- | :--- |
| Baseline | `v1.2.0` = `cfc95c8` |
| Go | 1.27.1; `golang.org/x/mod v0.40.0` (`semver`), already required |
| GitHub list | `GET /repos/{o}/{r}/releases`, `per_page` maximum 100, with no documented ordering (MADR, Evidence) |

### Rules for every step

These are the Phase 2 PLAN's rules ([0004-PLAN-v1-2-0-interaction-stream.md](0004-PLAN-v1-2-0-interaction-stream.md)),
with `make apicheck` in every step. Rule 3's list of existing tests that may
change is this PLAN's own:

* `github_test.go:219-239`, in Step 4, now expects the source to return a
  prerelease;
* in `checkcache_test.go`, in Step 3:
  * `wantCheckRecordJSON` (line 167) becomes the schema-2 document, with
    `channel`;
  * the unknown-schema test (lines 244-248), which uses `schema_version: 2`
    as its unknown example, moves to `3`;
  * a schema-1 document is kept as a read-only fixture.

## Proposed MADR amendments

Step 1 applies these to the MADR, each marked *(amended 2026-MM-DD,
0005-PLAN-opt-in-prerelease-channels)*. Approving this PLAN approves them.

| ID | MADR text today | Amendment | Why |
| :--- | :--- | :--- | :--- |
| E1 | §2–§3 use "the policy's `Channels`" without saying how discovery reaches them | Add the optional `ChannelPolicy interface { VersionPolicy; ValidChannel(name string) error; Admits(channel, tag string) bool }`. `NewSemverPolicy`'s value implements it, and so may any custom policy. With any other policy, every non-empty channel is refused. | `VersionPolicy` cannot gain a method in v1. An optional interface also lets a program define its own channels without this package. |
| E2 | §1: `Channels` "most stable first" | Each name must match `^[a-z][a-z0-9]{0,15}$`, and the list must be in strictly descending ASCII order (`["rc","beta","alpha"]` passes; `["beta","nightly"]` is refused). | SemVer orders prerelease names lexically (§11). Requiring stability order to equal ASCII order makes "the most stable admissible build" and "the highest version" the same thing. Otherwise `v1.3.0-nightly.1` would outrank `v1.3.0-beta.9`. |
| E3 | §3: "`per_page` at its maximum … `Limit` defaults to 100 and is capped at 300" | `per_page` is 30. `Limit` defaults to 90 and is capped at 300. | Each page is one body under `Limits.ReleaseJSON` (2 MiB by default), and list entries carry full release notes, so 100 per page could exceed it on a real repository. |
| E4 | §5: the workflow "accepts a tag of the prerelease grammar" | It does so only when the caller passes a new input, `prerelease-channels-json`, naming the admitted names. The default is `[]`, which keeps today's behaviour. | This removes the MADR's last "Bad" consequence. An existing caller can never publish a prerelease because a tag looks like one. |
| E5 | §3: discovery "picks the highest" admissible release | The winner is chosen on its tag and flags alone. Then it, and only it, is validated in full: structure, immutability, assets. Any failure is an error. A release elsewhere in the list that fails structure checks does not fail the list. | There is no silent fallback anywhere: a bad newest release is an error, never a reason to install an older one. A malformed old release cannot block a channel either. |

**An assumption the cap rests on.** GitHub lists releases newest first in
practice, but does not document it. If that ever changed, a channel could
miss an admissible release beyond the cap. The failure is a missed update,
never a wrong one, because the winner is still checked in full. The doc
comment for `ListOptions` says this.

## Implementation Steps

### Step 1: records

1. Apply amendments E1–E5 to the MADR.
2. Add this PLAN's row to `docs/README.md` (`in-progress`).
3. One commit; docs only.

### Step 2: the policy (`selfupdate/version.go`, `selfupdate/types.go`)

**API**

```go
type SemverOptions struct {
    AllowPrerelease bool
    Channels        []string
}
func NewSemverPolicy(o SemverOptions) (VersionPolicy, error)
type ChannelPolicy interface {
    VersionPolicy
    ValidChannel(name string) error
    Admits(channel, tag string) bool
}
```

**Behaviour**

1. **Construction.**
   * With `AllowPrerelease`, `Channels` must be non-empty and follow E2.
     Without it, `Channels` must be empty.
   * The options are copied, so a later change to the caller's slice has no
     effect.
   * Errors name the offending channel name.
2. **`Validate(tag)`.**
   * The strict core is required, as in `NewStrictVersionPolicy`.
   * The only suffix is `-NAME.N`, with `NAME` in `Channels` and `N` matching
     `^(0|[1-9][0-9]*)$`. No build metadata is accepted.
   * Without `AllowPrerelease`, the policy accepts exactly the strict tags.
3. **`Compare(a, b)`** validates both, then returns `semver.Compare`.
4. **`ValidChannel(name)`.** The empty name (stable) is always valid. Any
   other name must be in `Channels`.
5. **`Admits(channel, tag)`** assumes `tag` is valid.
   * A stable tag is admitted on every channel.
   * A prerelease tag is admitted when its `NAME`'s index in `Channels` is at
     most the channel's index. `"beta"` admits `rc` and `beta`, not `alpha`.
   * The stable channel admits no prerelease.

**Tests** (`semver_policy_test.go`):

* construction: valid lists; and refusals for an empty list with
  `AllowPrerelease`, a list without it, the wrong order, a duplicate, a bad
  name, and a defensive copy;
* `Validate`: a table of accepted and refused tags, including `v1.2.3+meta`,
  `v1.2`, `v1.2.3-rc`, `v1.2.3-rc.01`, `v1.2.3-RC.1`, an unknown name, and a
  name with a hyphen;
* `Compare` order: `rc.10 > rc.2`, `rc.1 > beta.9`, and `v1.3.0 > v1.3.0-rc.1`;
* `Admits` across the channels.

**Mutation proofs:**

| Mutation | Must fail |
| :--- | :--- |
| the order check removed | the `["beta","nightly"]` refusal |
| build metadata accepted | `v1.2.3+meta` |
| `N` allows a leading zero | `rc.01` |
| `Admits` compares indexes the wrong way | `"beta"` admitting `alpha` |
| the options slice is not copied | the defensive-copy test |

### Step 3: the request field and the cache (`selfupdate/types.go`, `version.go`, `checker.go`, `checkcache.go`)

**API.** `Request` and `CheckRequest` gain `Channel string`, appended last.

**Behaviour**

1. **`validateRequest`.**
   * A non-empty `Channel` needs the policy to be a `ChannelPolicy` whose
     `ValidChannel` accepts it. Otherwise the error is
     `selfupdate: channel %q is not offered by the version policy`.
   * A prerelease `TargetVersion` needs a `Channel` that `Admits` it
     (MADR Q4). Otherwise:
     `selfupdate: %s is a prerelease; request a channel that admits it`.
   * `CurrentVersion` may be any valid tag under the policy, so a user
     running an rc can be checked on the stable channel.
2. **`Checker.prepare`** copies `Channel` into the `Request`.
3. **The cache.**
   * `CheckRequest` equality now includes `Channel`.
   * The file schema becomes 2, with a `channel` field. `Load` reads schema 1
     as an empty `Channel`, and refuses anything else, as today. `Save`
     writes schema 2.
   * An older library reading a schema-2 file gets `ErrNoCheckRecord`, and
     simply checks again.

**Tests**:

* `validateRequest`: each refusal and each acceptance;
* `CheckCached`: a record saved for channel `rc` is not reused for the
  stable channel, and the reverse;
* schema 1 is still read, and schema 2 is written. The schema test changes,
  as rule 3 allows.

**Mutation proofs:**

| Mutation | Must fail |
| :--- | :--- |
| `Channel` dropped from the file record | the channel-isolation test (a reload loses the channel) |
| a schema-1 file refused | the schema-1 read test |
| the prerelease-pin check removed | the Q4 refusal |

### Step 4: listing (`selfupdate/types.go`, `selfupdate/github.go`, `selfupdate/selfupdatetest/`)

**API**

```go
type ListOptions struct {
    Limit int // releases to consider: 0 means 90; more than 300 is refused
}
type ReleaseLister interface {
    ListReleases(ctx context.Context, o ListOptions) ([]Release, error)
}
func (s *GitHubSource) ListReleases(ctx context.Context, o ListOptions) ([]Release, error)
// selfupdatetest: FakeSource implements ReleaseLister; GitHubServer serves the list.
```

**Behaviour**

1. **`GitHubSource.ListReleases`.**
   * It requests `releases?per_page=30&page=N` for `N = 1, 2, …`. It stops
     after `Limit` releases, on a short page, or after `ceil(Limit/30)`
     pages.
   * Each body is bounded by `Limits.ReleaseJSON`, and goes through `send`:
     the same credential, header scrubbing, 401 retry and rate-limit
     mapping as `Latest`.
   * Each entry is decoded and mapped with its flags and tag. A release whose
     assets fail `validateAssetStructure` is kept, but marked, so discovery
     can refuse it if it wins (E5).
   * Drafts are returned. Discovery drops them.
2. **The source stops refusing prereleases.** `validateFetchedRelease` keeps
   its draft and immutability refusals and drops the prerelease one.
   `Checker.discover` still refuses a prerelease when the request has no
   channel, so a stable run is unchanged. The release notes record this
   for direct callers of `ByTag` (MADR, Consequences).
3. **`FakeSource`** records its releases in declared order, and implements
   `ListReleases`, returning them in that order up to `Limit`.
4. **`GitHubServer`** serves `GET …/releases` with `page` and `per_page`, in
   declared order, applying the same `RateLimit` and `RequireToken` rules.

**Tests**:

* `TestGitHubSourceListReleases`: 70 releases over three pages, a `Limit`
  that stops early, a limit over 300 refused, an oversized page refused, a
  429 on page two mapped to `RateLimitError`, and the token sent on every
  page;
* `github_test.go:219-239` now expects the prerelease to be returned
  (rule 3);
* `selfupdatetest`: the list's order, pagination and limits.

**Mutation proofs:**

| Mutation | Must fail |
| :--- | :--- |
| no stop on a short page | the request count |
| `Limit` not applied | the early-stop case |
| a page body not bounded | the oversized page |
| `ListReleases` bypasses `send` (no credential) | the token check |

### Step 5: channel discovery (`selfupdate/checker.go`)

**Behaviour.** `discover` with a non-empty `Channel` and no `TargetVersion`:

1. The source must be a `ReleaseLister`. Otherwise:
   `selfupdate: channel %q needs a source that lists releases`.
2. **Candidates.** Each listed release is dropped when:
   * it is a draft;
   * `Validate(tag)` fails;
   * its `prerelease` flag disagrees with its tag (MADR §3);
   * `Admits(channel, tag)` is false.
3. **The winner** is the highest candidate by `Compare`. With none, the error
   is `selfupdate: no release on channel %q`.
4. **The winner is then checked in full** (E5): structure, immutability
   (`ErrMutableRelease`), then the same asset selection and metadata checks
   as today. Any failure returns the error.
5. **Classification** uses `fromLatest = true`, so no channel ever silently
   downgrades (`ErrLatestOlder`).
6. **With a channel and a `TargetVersion`,** `ByTag` is used, then the same
   flag/tag agreement and admission checks, then today's path.
7. **Without a channel** nothing changes: `Latest`, and a prerelease is
   refused.

**Tests** (`channel_test.go`), through `FakeSource` and through
`GitHubServer`:

* out-of-order releases: the highest admissible version wins;
* `"beta"` takes `rc.1` over `beta.3` of the same core, and a newer stable
  over both;
* a mutable winner is an error, not a fallback;
* a malformed winner is an error, and a malformed older release is
  harmless;
* a flag/tag mismatch is dropped, in both directions;
* a channel on a source with no list is refused;
* a stable request ignores prereleases: `Latest` is called, not the list;
* the moves of MADR §4: rc to stable upgrades, and stable-older gives
  `ErrLatestOlder`;
* a pinned prerelease, with and without a channel that admits it.

**Mutation proofs:**

| Mutation | Must fail |
| :--- | :--- |
| the first candidate wins, not the highest | out-of-order |
| a mutable winner falls back to the next | the mutable-winner test |
| flag/tag agreement not checked | the mismatch tests |
| a stable request uses the list | `Latest` not called |
| `fromLatest` false on a channel | the downgrade test |

### Step 6: publication (`scripts/check-release-tag.sh` (new) and its test, the workflow, the verifier)

1. **`scripts/check-release-tag.sh TAG [CHANNELS-JSON]`.** It exits 0 for a
   strict tag, or for `-NAME.N` with `NAME` in the JSON array, and checks
   that array against E2. It exits 1 otherwise, and 2 on a usage error. Its
   test covers the same tag table as Step 2.
2. **The workflow** gains the input
   `prerelease-channels-json` (string, default `[]`):
   * "Require a strict stable tag" becomes "Require an admitted tag", which
     calls the script with the tag and the input through `env:`, with no
     `${{ }}` in `run`;
   * the create step adds `--prerelease --latest=false` when the tag has a
     suffix;
   * the immutable-and-verified wait is unchanged.
3. **The verifier's `--tag` check** takes an optional `--channels JSON` and
   applies the same rule. Its tests gain prerelease cases.
4. **`check-workflows.sh`** passes on the changed workflow (both rules), and
   `actionlint` is clean.

**Mutation proofs:**

| Mutation | Must fail |
| :--- | :--- |
| the script accepts any suffix | its unknown-name case |
| the workflow omits `--latest=false` | a `check-workflows`-style assertion added to the script's test, which greps the workflow's create step |
| the default input admits a channel | the default-refusal case |

### Step 7: the end-to-end case, documentation and close-out

1. **`e2e_running_test.go`** gains `TestE2EUpdateRunningCopyOnChannel`. It
   builds the helper as `v1.3.0-rc.1` and serves it as a prerelease beside a
   stable `v1.2.0`.
   * The channel is `rc`, under `NewSemverPolicy`, with the version probes
     on. A running v1 is replaced by `v1.3.0-rc.1`.
   * The same release, on the stable channel, installs `v1.2.0`.
2. **Docs:**
   * `doc.go`: a "Channels" section;
   * `ExampleNewSemverPolicy`;
   * the guide's "Offer a beta channel";
   * `architecture.md`;
   * the README row "offer a beta or rc channel";
   * the workflow's input in the migration and publishing notes.
3. **Release notes for `v1.3.0`**, in the execution record, including the
   `ByTag` behaviour change.
4. **Verification** is the Phase 2 PLAN's list, plus `apidiff -m` against
   `v1.2.0` listing only additions.
5. **Status.** Marked `complete` after CI is green on the pushed tree. The
   owner decides the `v1.3.0` tag.

## Verification

* Every step's mutations are killed.
* `make pre-add-check`, `make lint` and `make apicheck` pass in every step.
* `SELFUPDATE_REQUIRE_PYTHON=1 go test -race -count=1 ./...`,
  `go test -shuffle=on -count=2 ./...` and `make fuzz` pass.
* The script tests pass, and so do `shellcheck`, `actionlint` and
  `check-workflows.sh`.
* `make vuln` passes, `go mod tidy -diff` is clean, and `go.mod` is
  unchanged.
* The Windows test host passes every step that changes Go code.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollback.** Each step is one commit. Step 6 can be reverted alone. That
  keeps the client able to read prereleases while the workflow stays unable
  to publish them.
* **Consumers.** Nothing changes for a program that sets no `Channel` and
  keeps `NewStrictVersionPolicy`. The exception is direct callers of
  `GitHubSource.ByTag`, as the release notes say.
* **Workflow callers.** Nothing changes unless they pass
  `prerelease-channels-json` (E4).
* **Tagging.** `v1.3.0` is tagged on the owner's ask.

## Execution Record

### Approval (2026-10-01)

The owner approved this PLAN and amendments E1–E5 ("proceed, approved").

### Step 1: records (2026-10-01)

* The MADR's Decision Outcome gained the "Amended 2026-10-01" block, with
  E1–E5 and inline marks in §1, §3 and §5.
* This PLAN was indexed as `in-progress`.
