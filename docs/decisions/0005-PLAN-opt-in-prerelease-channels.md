---
status: complete
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
| ~~`N` allows a leading zero~~ *(D2: equivalent; replaced by "`N` need not be numeric")* | ~~`rc.01`~~ `rc.x` |
| `Admits` compares indexes the wrong way | `"beta"` admitting `alpha` |
| ~~the options slice is not copied~~ *(D1: not expressible; the policy keeps no slice)* | the defensive-copy test |

### Step 3: the request field and the cache (`selfupdate/types.go`, `version.go`, `checker.go`, `checkcache.go`)

**API.** `Request` and `CheckRequest` gain `Channel string`, appended last.

**Behaviour**

1. **`validateRequest`.**
   * A non-empty `Channel` needs the policy to be a `ChannelPolicy` whose
     `ValidChannel` accepts it. Otherwise the error is
     `selfupdate: channel %q is not offered by the version policy`.
   * A prerelease `TargetVersion` needs a `Channel` that `Admits` it
     (MADR Q4). *(D3: under a `ChannelPolicy` only.)* Otherwise:
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
   refused. *(D4: under a `ChannelPolicy`, the flag/tag
   agreement check applies here too.)*

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

### Step 2: the policy (2026-10-01)

The code is commit `4c271e0`, which the owner made and pushed from the
working tree this step left. Its content is what is described and tested
here. Work paused after the tests and before the commit, while
[0006-PLAN-adopt-golangci-lint-v2-14.md](0006-PLAN-adopt-golangci-lint-v2-14.md)
cleared a lint upgrade that had blocked every Go commit.

**What changed.**

* **`selfupdate/types.go`** adds `ChannelPolicy` (E1).
* **`selfupdate/version.go`** adds:
  * `channelNameRe` (`^[a-z][a-z0-9]{0,15}$`) and `prereleaseNumRe`
    (`^(0|[1-9]\d*)$`; v2.14.0's gocritic `regexpSimplify` asked for `\d`);
  * `SemverOptions` and `NewSemverPolicy`. Construction checks names,
    duplicates and strictly descending ASCII order (E2). The ranks are
    built into a map, so the caller's slice is never kept;
  * `Validate`: the strict core plus an optional `-NAME.N`, with no build
    metadata, then `semver.IsValid`;
  * `Compare`, which is `semver.Compare` after validation;
  * `ValidChannel`, and `Admits`, which uses rank order.
* **`selfupdate/semver_policy_test.go`** (new) has five tests:
  * the option refusals, each with its message;
  * the defensive copy;
  * a 19-row `Validate` table for the channel and stable policies;
  * `Compare` order, both ways;
  * `Admits` across every channel and tag.

**Mutation proofs:**

| Mutation | Killed by |
| :--- | :--- |
| the order check removed | `stability not in ASCII order: err = <nil>` |
| build metadata accepted (the core cut before `+`) | `Validate("v1.2.3+meta") accepted=true` |
| `Admits` compares ranks the wrong way | `Admits("beta", "v1.0.0-rc.1") = false, want true` |
| an unknown prerelease name accepted | `Validate("v1.2.3-RC.1") accepted=true` |
| `N` need not be numeric (`^[0-9a-z]+$`) | `Validate("v1.2.3-rc.x") accepted=true` |

**Deviation D1 (2026-10-01).** The PLAN's "the options slice is not copied"
cannot be written as one mutation: the policy never stores the slice, only
a rank map built during construction. `TestSemverPolicyDoesNotShareChannels`
pins the property the mutation was meant to guard. Editing the caller's
slice after construction neither breaks `rc` nor admits a new name.

**Deviation D2 (2026-10-01).** "`N` allows a leading zero" survived: the
test passed. `semver.IsValid` already refuses `rc.01`, because SemVer §9
forbids leading zeroes in numeric identifiers, so the regex's leading-zero
rule is a second layer. What only the regex enforces is that `N` is
numeric: SemVer allows `rc.x`. That mutation replaced it, and was killed.

**Checks.**

* The tests passed locally, and on the Windows test host, before the regex
  was simplified. The simplification is equivalent, and the tests passed
  again afterwards.
* With v2.14.0, `make lint` reports `0 issues` on all three targets.
* `make apicheck`: `compatible with v1.2.0`.

### Step 3: the request field and the cache (2026-10-01)

**What changed.**

* **`Request.Channel` and `CheckRequest.Channel`** are appended last.
  `Checker.prepare` copies the channel into the `Request`.
* **`validateChannel`** (`version.go`) is called at the end of
  `validateRequest`:
  * a non-empty channel needs a `ChannelPolicy` whose `ValidChannel`
    accepts it;
  * under a `ChannelPolicy`, a prerelease `TargetVersion` needs a channel
    that `Admits` it.
* **The cache file** is schema 2, with `channel` after `platform`. `Load`
  accepts schemas 1 and 2, and a schema-1 document reads as the stable
  channel. `Save` writes 2, and `CheckCached`'s key includes the channel.

**Deviation D3 (2026-10-01): the scope of Q4.**

* **Found.** Applying Q4 under every policy failed two existing tests:
  `TestCheckerAvailability/configured_policy` and
  `TestValidateRequestUsesConfiguredPolicy`. They are Phase 1's G2 proofs: a
  plain custom policy that accepts prereleases pins `v1.2.3-rc.1`. A plain
  policy has no channel, so Q4 refused the pin, overriding the policy's own
  decision.
* **Decision.** The owner chose "ChannelPolicy only".
  * Q4 applies when the policy offers channels.
  * A plain `VersionPolicy` keeps full control, so the G2 tests are
    unchanged.
  * The built-in path is unaffected, because `NewStrictVersionPolicy`
    refuses prerelease tags.
* **MADR.** §2's pinning bullet carries the amendment.

**Tests.**

* **`request_channel_test.go`** (new):
  * `TestValidateRequestChannel`, eleven cases: known, unknown and stable
    channels; a channel under the strict policy and under a plain policy;
    the Q4 refusals and acceptances; running an rc on the stable channel;
    and D3's plain-policy pin;
  * `TestCheckerPrepareKeepsChannel`.
* **`checkcache_test.go`.** These changes are the ones rule 3 lists:
  * `wantCheckRecordJSON` is schema 2, with `"channel":""`;
  * the unknown-schema test now uses schemas 3 and 0.

  It also gains `schema1CheckRecordJSON` with `TestFileCheckStoreReadsSchema1`,
  `TestFileCheckStoreKeepsChannel`, and `TestCheckCachedChannelIsolation`
  (saved on `rc` and asked on stable, the reverse, and `rc` against `beta`).

**Mutation proofs.** The PLAN's three plus three more; none survived:

| Mutation | Killed by |
| :--- | :--- |
| `Channel` dropped from the file record | `channel after a round trip = ""` |
| a schema-1 file refused | `a schema-1 record did not load: … check record schema 1 …` |
| the prerelease-pin check removed | `a pinned prerelease with no channel: err = <nil>` |
| the channel's validity not checked | `an unknown channel: err = <nil>` |
| `prepare` drops the channel | `prepare = {… Channel: …}; want the channel carried` |
| Q4 applied to plain policies (D3 reverted) | `a plain custom policy pins a prerelease: … request a channel that admits it` |

**Checks.**

* `make pre-add-check` passed on the six files (with v2.14.0), and
  `make apicheck` reported `compatible with v1.2.0`.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`.

### Step 4: listing (2026-10-01)

**What changed.**

* **`types.go`** adds `ListOptions` and `ReleaseLister`. The `Limit` doc
  carries the PLAN's assumption: GitHub's order is undocumented, so a missed
  release costs an update, never a wrong one.
* **`github.go`.**
  * `GitHubSource.ListReleases` pages
    `releases?per_page=30&page=N` through `send`, so each page gets the
    credential, redirect scrubbing, 401 retry and rate-limit mapping.
    * It stops at the limit or on a short page.
    * Each body is bounded by `Limits.ReleaseJSON`.
    * `listLimit` maps 0 to 90 and refuses anything outside 1..300.
  * `validateFetchedRelease` no longer refuses prereleases. Discovery still
    does, when no channel is named.
* **"Kept, but marked" (behaviour 1), done without a marker.** Listed
  entries are mapped by `mapReleaseUnchecked`, which skips the per-asset
  structure check that `mapRelease` applies to `Latest` and `ByTag`.
  Step 5's discovery runs that check on the winner only (E5). The behaviour
  is the PLAN's, with no new field on `Release`. The check's helper,
  `validateReleaseStructure`, lands in Step 5 with its first caller, so
  `unused` stays clean.
* **`selfupdatetest`.**
  * `FakeSource` records declared order, and its `ListReleases` returns
    every release, drafts and prereleases included, in that order, up to a
    positive `Limit`.
  * `GitHubServer` serves `GET …/releases` with `page` and `per_page`
    (default 30, at most 100), in declared order, under the same
    `RateLimit` and `RequireToken` rules. `serveRelease` now shares
    `releaseDoc` with it.

**Tests.**

* **`github_list_test.go`** (new):
  * `TestGitHubSourceListReleases`: 70 releases from pages 1, 2 and 3, the
    third short, in order and with the token on every page; `Limit 40`
    stops after two pages; the default gives 90 from three pages; `-1` and
    `301` are refused;
  * `TestGitHubSourceListReleasesBounds`: a page over `Limits.ReleaseJSON`,
    and a 429 on page 2 mapped to `RateLimitError`.
* **`github_test.go`.** `ByTag` of a prerelease now expects the release,
  flagged (rule 3).
* **`selfupdatetest`:** `TestFakeSourceListReleases`, and
  `TestGitHubServerList`, which covers paging, a page beyond the end, and
  `RequireToken`.

**Mutation proofs.** The PLAN's four plus two more; none survived:

| Mutation | Killed by |
| :--- | :--- |
| no stop on a short page | `panic: test timed out after 1m30s`: the source would request pages forever |
| `Limit` not applied | `Limit 40: 60 releases from pages [1 2]` |
| a page body not bounded | `a page larger than Limits.ReleaseJSON was accepted` |
| `ListReleases` bypasses `send` | `page 1 sent Authorization ""` |
| the source still refuses prereleases | `prerelease: … is a prerelease; want it returned, flagged` |
| `FakeSource` lists in reverse | `ListReleases = [v1.3.0 v1.2.0-rc.1 v1.1.0], want … declared order` |

The first spec for the last mutation used `slices`, which that file does not
import, and did not compile. It was rewritten as a reversal, then killed.

**Checks.**

* `make pre-add-check` passed on the seven files, and `make apicheck`
  reported `compatible with v1.2.0`.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`.

### Step 5: channel discovery (2026-10-01)

**What changed.**

* **`checker.go`.**
  * `fetchRelease` sends a request with a channel and no `TargetVersion` to
    `channelRelease`, and reports it as `fromLatest`, so a channel never
    silently downgrades (rule 5).
  * `channelRelease` requires a `ReleaseLister` (rule 1). It lists with the
    default `ListOptions` and drops drafts, tags `Validate` refuses, and
    releases `checkChannel` refuses (rule 2). The highest by `Compare` wins
    (rule 3), and only then is its structure checked (E5).
  * `discover` then applies today's checks to the winner, in today's order:
    immutability (`ErrMutableRelease`), draft, `Validate`, and the asset
    checks (rule 4). It refuses a flagged prerelease only when no channel is
    named (rule 7), and calls `checkChannel` on every path, so a pinned
    `TargetVersion` on a channel gets the same agreement and admission checks
    (rule 6).
  * `checkChannel` is new. Under a `ChannelPolicy` it requires the flag to
    agree with the tag, and a named channel to admit the tag. Under a plain
    policy it refuses any named channel (unreachable after
    `validateRequest`) and checks nothing else (D4).
* **`github.go`.** `validateReleaseStructure` lands with its first caller,
  as Step 4 recorded. It checks the release's identity and every asset's
  structure, the checks `mapRelease` applies to `Latest` and `ByTag`.
* **Test fixtures.** The internal `scriptSource` (`updater_test.go`) gains
  `ListReleases`, and `networkCalls` (`checkcache_test.go`) counts it.
  * Step 3's `TestCheckCachedChannelIsolation` asks for a channel, which now
    reaches discovery and needs a lister.
  * These files are outside the PLAN's Step 5 list, but the change follows
    from rule 1 and decides nothing.

**Deviation D4 (2026-10-01): the flag/tag check without a channel.**

* **Found.** MADR §3 excludes a release whose flag disagrees with its tag
  "on every channel". Rule 7 says nothing changes without a channel.
  * Under `NewSemverPolicy` with `AllowPrerelease`, an rc whose flag was
    cleared after publication can be GitHub's `Latest`.
  * It passes `Validate`, so a stable request would have installed it.
  * Applying the check to every request would also refuse such tags under a
    plain custom policy, changing v1.2.0's G2 behaviour.
* **Decision.** The owner chose "ChannelPolicy only".
  * Under a `ChannelPolicy`, the agreement check runs on every request, the
    stable channel included.
  * A plain `VersionPolicy` is unchanged, as in D3.
  * `NewStrictVersionPolicy` refuses prerelease tags before the check is
    reached, so the built-in path is unchanged.
* **Records.** MADR §3's agreement bullet carries the amendment, and rule 7
  above is annotated.

**Tests** (`channel_test.go`, new). `TestChannelDiscovery` runs 20 cases on
both `FakeSource` and `GitHubServer`:

* out-of-order releases, where the highest admissible wins;
* `rc` admits stable but not `beta`;
* `beta` takes `rc.1` over `beta.3`, and a newer stable beats both;
* a draft is dropped;
* a mutable winner is `ErrMutableRelease`, not a fallback;
* a malformed winner is an error, and a malformed older release is harmless;
* a flag/tag mismatch is dropped, in both directions;
* nothing on the channel;
* MADR §4: rc to stable upgrades, on a channel and on stable;
  leaving a channel and staying on one both give `ErrLatestOlder`;
* pinned versions:
  * a prerelease without a channel is refused;
  * one on a channel that admits it is an upgrade;
  * an older one is a rollback;
  * one whose flag disagrees is refused;
* D4: a stable request refuses an unflagged rc served as `Latest`.

`TestChannelSourceCalls` shows a stable request calls `Latest` and never
lists, and a channel request lists and never calls `Latest`, on both
backends. `TestChannelNeedsLister` shows a source without `ListReleases` is
refused by name, with no fallback to `Latest`.

**Mutation proofs.** The PLAN's five plus two more; none survived:

| Mutation | Killed by |
| :--- | :--- |
| the first candidate wins, not the highest | `selected v1.1.0 (upgrade), want v1.3.0-beta.1 (upgrade)` |
| a mutable winner falls back to the next | `err = <nil>, want selfupdate: release is not immutable` |
| flag/tag agreement not checked | `selected v1.4.0-rc.1 (upgrade), want v1.2.0 (upgrade)` |
| a stable request uses the list | `stable = , … release "v1.3.0-rc.1" is not a stable published release; want v1.2.0` |
| `fromLatest` false on a channel | `err = <nil>, want selfupdate: latest release is older than the running version` |
| D4: the stable channel skips the flag check | `err = <nil>, want one containing "prerelease flag that disagrees with its tag"` |
| the winner's structure not checked | `err = <nil>, want one containing "asset name \"bad/name\" is not a basename"` |

The first spec for the `fromLatest` mutation matched two `return rel, true,
err` lines, and the runner refused it. It was narrowed to the channel
branch, then killed.

**Checks.**

* `make pre-add-check` passed on the five files, and `make apicheck`
  reported `compatible with v1.2.0`.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`.

### Step 6: publication (2026-10-01)

**What changed.**

* **`scripts/check-release-tag.sh`** (new) holds the tag rule, once.
  * It admits a strict `vX.Y.Z`, or `vX.Y.Z-NAME.N` when `NAME` is in
    `CHANNELS-JSON` and `N` has no leading zero, as `NewSemverPolicy` does.
  * Every match is a `fullmatch`, so a trailing newline is refused.
  * `CHANNELS-JSON` defaults to `[]` and is checked against E2. A bad array
    is a caller's configuration error, so it exits 2 with the usage errors;
    a tag that is not admitted exits 1.
* **The workflow.**
  * The input `prerelease-channels-json` is optional, defaulting to `'[]'`.
  * "Require a strict stable tag" is now "Require an admitted tag".
    * It keeps the tag-ref check and calls the script with `$TAG` and
      `$CHANNELS_JSON` from `env:`.
    * It moved after "Check out the called workflow commit", which holds
      the script. It still runs before the artifact is read and before any
      release is created; the checkout it now follows mutates nothing.
  * "Validate the staged file set" passes `--channels`.
  * "Create a draft release" adds `--prerelease --latest=false` when the tag
    contains `-`, a tag the step above has already admitted.
  * The immutability and attestation wait is unchanged.
* **The verifier** takes `--channels JSON` (default `[]`) and applies the
  rule by calling `check-release-tag.sh`, so the two checks cannot drift.
  Its own strict-tag regex is gone, and the script's exit 2 stays a usage
  error.
* **`ci.yml`** gains "Verify the release tag rule", Linux only like the
  verifier's fixtures, running `scripts/check-release-tag_test.sh`. The
  file is outside Step 6's list, but a test that CI does not run is unused;
  the step decides nothing.

**Tests.**

* **`scripts/check-release-tag_test.sh`** (new), 51 cases:
  * Step 2's tag table, on `["rc","beta","alpha"]` and on the default;
  * the default and `[]` refusing a prerelease, and a single channel
    admitting only itself;
  * seven bad channel arrays and two usage errors, each exit 2;
  * one workflow assertion, read from the workflow text as its steps run:
    * the input defaults to `'[]'` and is optional;
    * the tag step reads the channels through `env:` and calls the script
      with both;
    * the verifier gets `--channels`;
    * only a `*-*` tag gets `--prerelease --latest=false`, passed to
      `gh release create`.
* **`verify-selfupdate-release_test.sh`** gains four cases:
  * a prerelease tag without channels is refused;
  * one on a named channel is accepted;
  * one on another channel is refused;
  * channels out of order are a usage error.

**Mutation proofs.** The PLAN's three, plus five more, run on scratch
copies by a script-test variant of the mutation runner; none survived:

| Mutation | Killed by |
| :--- | :--- |
| the script accepts any suffix | `FAIL [v1.2.3-gamma.1] on ["rc","beta","alpha"]: want exit 1, got 0` (the runner showed the first failure, `[v1.2.3-rc.1] stable only`; the unknown-name case was confirmed separately) |
| the workflow omits `--latest=false` | `FAIL workflow: the create step must add --prerelease --latest=false for a suffixed tag` |
| the script's default admits a channel | `FAIL [v1.2.3-rc.1] stable only: want exit 1, got 0` |
| the workflow input's default admits a channel | `FAIL workflow: prerelease-channels-json must default to '[]'` |
| the workflow's tag step drops the channels | `FAIL workflow: the tag step must call check-release-tag.sh with the tag and channels` |
| the create step marks every tag a prerelease | `FAIL workflow: the create step must add --prerelease --latest=false for a suffixed tag` |
| the verifier ignores `--channels` | `not ok - prerelease tag on a named channel` |
| the verifier skips the tag rule | `not ok - non-strict tag rejected (expected failure)` |

**Checks.**

* Clean: `shellcheck scripts/*.sh`, `check-workflows.sh` (both rules),
  `check-workflows.sh --rule expressions` on `ci.yml`, and
  `check-workflows_test.sh` (24 passed).
* Also clean: actionlint v1.7.12, and markdownlint-cli2 0.23.2.
* The Windows test host passed `go vet ./...`, `go test -race -count=1 ./...`
  and the script tests it runs, the verifier's four new cases among them.

### Step 7: the end-to-end case and documentation (2026-10-01)

**What changed.**

* **`e2e_running_test.go`** gains `TestE2EUpdateRunningCopyOnChannel`.
  * It builds the helper as `v1.0.0`, `v1.2.0` and `v1.3.0-rc.1`, and serves
    a stable `v1.2.0` beside a flagged prerelease `v1.3.0-rc.1` through
    `GitHubServer`, under `NewSemverPolicy` with the channel `rc`.
  * The image verifier and both version probes run, as in
    `TestE2EUpdateRunningCopy`.
  * On `rc` the running v1 becomes the rc build, and reports
    `demo v1.3.0-rc.1`. On the stable channel, from the same releases, it
    becomes `v1.2.0`.
  * Each case stops the old process, runs `CleanupPending`, and requires no
    leftovers and no receipt, on every OS.
  * `e2eOptions` gains `releases` and `versions`, and `run` becomes
    `runOn("")`; the existing cases are unchanged.
* **Docs:**
  * `doc.go`: a "Channels" section, and the workflow input in the
    publishing paragraph;
  * `ExampleNewSemverPolicy`: one checker, three channels (stable, `rc`,
    `beta`) and their answers;
  * the extending guide: "Offer a beta channel", covering the policy, the
    choice, moving between channels and publishing;
  * `architecture.md`:
    * the channel API by file;
    * the new end-to-end case;
    * the workflow's order, with the tag rule second;
    * `check-release-tag.sh` in the tree and in CI;
    * the test-file count, now 56;
  * the "offer a beta or rc channel" row in `README.md` and
    `docs/README.md`;
  * the optional input in `README.md`'s publishing notes and in the
    migration guide's workflow section.

**Mutation proofs** for the new end-to-end case; neither survived:

| Mutation | Killed by |
| :--- | :--- |
| the request drops the channel | `ExitCode = 0, res = {… TargetVersion:v1.2.0 …}` on the `rc` case |
| discovery ignores the channel | `ExitCode = 0, res = {… TargetVersion:v1.2.0 …}` on the `rc` case |

**Link check.** Nothing validates links in guides and READMEs, so a
throwaway resolver checked every relative link and `#anchor` in the five
changed documents: 0 broken. A scratch copy with one bad anchor and one
missing file planted reported both.

**Verification** (the list above), on the macOS development host:

| Check | Result |
| :--- | :--- |
| `make pre-add-check` (every tracked Go file) | rc=0 |
| `make lint` (three targets) | rc=0 |
| `make apicheck` | `compatible with v1.2.0` |
| `SELFUPDATE_REQUIRE_PYTHON=1 go test -race -count=1 ./...` | rc=0 |
| `go test -shuffle=on -count=2 ./...` | rc=0 |
| `make fuzz` | rc=0 |
| `make vuln` | rc=0 |
| `go mod tidy -diff` | rc=0 |
| `git diff --exit-code v1.2.0 -- go.mod go.sum` | rc=0: unchanged |
| the six script tests | rc=0 each |
| `shellcheck scripts/*.sh`, actionlint v1.7.12 | rc=0 |
| `check-workflows.sh` (both rules), on `ci.yml` (expressions), and its test | rc=0 |
| markdownlint-cli2 0.23.2 | 0 issues |

`apidiff -m` against `v1.2.0`, run on a `git archive` export of the tag,
lists only additions:

```text
Compatible changes:
- ./selfupdate.(*GitHubSource).ListReleases: added
- ./selfupdate.ChannelPolicy: added
- ./selfupdate.CheckRequest.Channel: added
- ./selfupdate.ListOptions: added
- ./selfupdate.NewSemverPolicy: added
- ./selfupdate.ReleaseLister: added
- ./selfupdate.Request.Channel: added
- ./selfupdate.SemverOptions: added
- ./selfupdate/selfupdatetest.(*FakeSource).ListReleases: added
```

The Windows test host passed `go vet ./...`, `go test -race -count=1 ./...`
and the script tests. A run of the new case alone showed
`--- PASS: TestE2EUpdateRunningCopyOnChannel` with both subtests.

### Release notes for `v1.3.0`

**Additions.** Everything is additive; `apidiff` lists only the additions
above.

* **Prerelease grammar:** `NewSemverPolicy(SemverOptions{AllowPrerelease,
  Channels})`.
  * It accepts strict `vX.Y.Z` and `vX.Y.Z-NAME.N` for each listed channel.
  * Channels are listed most stable first, in descending ASCII order.
  * Build metadata is never accepted.
* **Channels:** `ChannelPolicy` (`ValidChannel`, `Admits`), the policy
  `NewSemverPolicy` returns; `Request.Channel` and `CheckRequest.Channel`.
* **Listing:** `ReleaseLister` and `ListOptions`, implemented by
  `GitHubSource.ListReleases` (90 releases by default, 300 at most) and by
  `selfupdatetest.FakeSource`. `GitHubServer` serves the paged list.
* **Publishing:** the release workflow's optional
  `prerelease-channels-json` input, and `scripts/check-release-tag.sh`.

**Behaviour changes.**

* **`GitHubSource.ByTag` and `Latest` return prereleases**, flagged with
  `Release.Prerelease`; before, they refused them.
  * `Updater.Run` and `Checker` are unaffected: without a channel,
    discovery still refuses a prerelease.
  * A program that calls `ByTag` directly and relied on the refusal must
    check `Release.Prerelease` itself.
* **The check-cache file is schema 2**, which adds `channel`.
  * `v1.3.0` reads schema 1 as the stable channel.
  * A program downgraded to `v1.2.x` reads a schema-2 file as
    `ErrNoCheckRecord`, a cache miss: it checks once and rewrites the file.
* **The release workflow** checks the tag after checking out its own
  tools, still before it reads the artifact or creates a release. A
  refused tag's message now comes from `check-release-tag.sh`. With the
  default input, the tags it accepts are unchanged.

**Migration notes.**

* No code change is needed to upgrade from `v1.2.x`, and `go.mod` requires
  nothing new.
* **To offer a channel:**
  * replace `NewStrictVersionPolicy` with `NewSemverPolicy`;
  * set `Request.Channel` from a flag or a setting;
  * pass `prerelease-channels-json` to the release workflow.

  The extending guide's "Offer a beta channel" has the details.
* **A custom `ReleaseSource` used with a channel** must implement
  `ReleaseLister`; without it, a channel request fails and names the
  missing capability.
* **Workflow callers** change nothing unless they publish prereleases. Pin
  the new release's commit as before.

### Close-out (2026-10-01)

The owner pushed Steps 2–7 (`1d5470e`..`f200c51`; Step 2's code is
`4c271e0`, pushed earlier). CI run `36939185878` on `f200c5193a2c`
concluded `success` on `ubuntu-24.04`, `windows-2025` and `macos-15`.
Every acceptance criterion in Verification is met, so this PLAN is
`complete`. The owner decides the `v1.3.0` tag.
