---
status: accepted
date: 2026-10-01
decision-makers: owner
consulted: []
informed: []
---
# Offer opt-in prerelease channels without weakening the stable path

## Context and Problem Statement

`selfupdate` installs only stable releases. The owner wants prerelease
channels, so that a program can offer its users early builds such as
`v1.3.0-rc.1`
([0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md),
owner decision 3). That record left prereleases to a record of their own,
because they change what an unattended `update` may install (§6). It
scheduled this record for after Phase 1, which added the `Config.Versions`
fix (G2) that prereleases depend on. Phase 1 shipped in `v1.1.0`.

### Where prereleases are refused today

Three independent layers in the client refuse a prerelease, and the release
workflow refuses to publish one:

| Layer | What it does | Evidence |
| :--- | :--- | :--- |
| The GitHub source | `validateFetchedRelease` rejects any release whose `prerelease` flag is set, for `Latest` and for `ByTag` alike, before the coordinator sees it. | `selfupdate/github.go`, `validateFetchedRelease` |
| Discovery | `Checker.discover` rejects `rel.Draft \|\| rel.Prerelease` again, for any source. | `selfupdate/checker.go`, `discover` |
| The version policy | `NewStrictVersionPolicy` accepts only `vMAJOR.MINOR.PATCH`, with no prerelease or build suffix. It compares with `golang.org/x/mod/semver`, which already orders prereleases. | `selfupdate/version.go` |
| Publication | The reusable workflow's "Require a strict stable tag" step, and the verifier's `tag_re`, refuse a tag like `v1.3.0-rc.1`. | `.github/workflows/publish-selfupdate-release.yml`, "Require a strict stable tag"; `scripts/verify-selfupdate-release.sh` |

### What "latest" cannot do

* **`Latest` never sees a prerelease.** `ReleaseSource.Latest` maps to
  GitHub's `GET /repos/{owner}/{repo}/releases/latest`, which serves only the
  newest release that is neither a draft nor a prerelease. Finding the
  newest prerelease needs the release *list*.
* **GitHub documents no order for the release list,** so the client must
  choose by version precedence itself. `latest` is defined by creation
  time, not version. The citations are in More Information.
* **A release's `prerelease` flag is mutable even when the release is
  immutable.** GitHub locks an immutable release's tag and assets, but
  still lets its owner "change whether it is marked as a pre-release". The
  flag alone cannot decide what a channel admits.
* **The cache key.** `Checker.CheckCached` keys its record on
  `CheckRequest`, compared with `==`. A field added to `CheckRequest` keeps
  the channels' cached answers apart in memory. The file store's schema v1
  has no such field, so it would need one (`selfupdate/checkcache.go`).

### The guarantees a prerelease must keep

Everything that protects a stable install must apply unchanged:

* an immutable release;
* `SHA256SUMS`, the GitHub digest and the advertised size;
* the image check and the probes;
* the lock, the backup and the rollback.

## Decision Drivers

* **The default does not move.** A program that changes nothing never
  installs a prerelease, whether its run is attended or unattended. That
  includes `CheckCached` banners and any background check.
* **The same integrity, or none.** A prerelease is installed only under every
  guarantee a stable release gets. Being a prerelease never relaxes a
  check.
* **v1 compatibility,** checked by `make apicheck`:
  * only new types, functions and fields;
  * a new capability arrives as an optional interface, never as a method on
    an existing one;
  * new fields have comparable types.
* **Deterministic selection.** The same set of releases gives the same
  answer, whatever order GitHub lists them in.
* **Bounded cost.** Listing must have a hard cap on requests and bodies, and
  respect rate limits as `Latest` does.
* **Safe tags.** A tag reaches file names, shell arguments and messages.
  Today's tag grammar is strict for that reason (0003-MADR D5, 0004-MADR R6),
  and a prerelease grammar must be no looser than it needs to be.
* **Extensibility.** The owner prefers sensible, future-proof API (2026-10-01).
  The mechanism should leave room for named channels and other sources,
  without a later redesign.
* **A path to publish.** A channel nobody can publish to is not a channel,
  so the release workflow is in scope.

## Considered Options

* **A. Opt-in prerelease selection in the client.** A semver policy that
  admits prereleases, a per-request opt-in, and an optional
  `ReleaseLister` that sources implement for discovery.
* **B. A separate repository per prerelease channel** (for example
  `owner/tool-beta`), published as ordinary stable releases.
* **C. Pinning only.** Allow `--version v1.3.0-rc.1` to install a named
  prerelease, with no "newest prerelease" discovery.
* **D. Keep the refusal** (mcplib 0005-MADR's original position).

## Decision Outcome

Chosen option: **"A. Opt-in prerelease selection in the client"**, because:

* it is the only option that gives a real channel (the newest admissible
  build) without a second repository per program;
* it keeps every integrity check;
* it is purely additive in v1.

C is subsumed: a pinned prerelease is one case of A.

**Amended 2026-10-01 by [0005-PLAN-opt-in-prerelease-channels.md](0005-PLAN-opt-in-prerelease-channels.md).**
The owner approved five amendments with the PLAN. Each is marked where it
applies:

* **E1.** An optional `ChannelPolicy` (`VersionPolicy` plus
  `ValidChannel` and `Admits`) carries the channel operations.
  `NewSemverPolicy`'s value implements it, and any other policy refuses
  every non-empty channel.
* **E2.** Channel names match `^[a-z][a-z0-9]{0,15}$`, in strictly
  descending ASCII order, so that stability order equals SemVer
  precedence.
* **E3.** `per_page` is 30. `Limit` defaults to 90 and is capped at 300,
  because each page is one body under `Limits.ReleaseJSON`.
* **E4.** The workflow publishes a prerelease only when the caller
  passes `prerelease-channels-json`. The default `[]` keeps today's
  behaviour.
* **E5.** The winner is chosen on its tag and flags, then validated in
  full. A failure is an error, never a fallback, and a malformed release
  elsewhere in the list does not fail the list.

### 1. The version grammar: `NewSemverPolicy`

```go
type SemverOptions struct {
    // AllowPrerelease admits vMAJOR.MINOR.PATCH-PRERELEASE tags.
    AllowPrerelease bool
    // Channels lists the admitted prerelease names, most stable first,
    // for example []string{"rc", "beta", "alpha"} (decision Q1).
    // Amended E2: strictly descending ASCII order; amended E1: discovery
    // reaches channels through the optional ChannelPolicy.
    Channels []string
}
func NewSemverPolicy(o SemverOptions) (VersionPolicy, error)
```

* **Validate.**
  * The core stays strict: `vMAJOR.MINOR.PATCH`, with no leading zeroes.
  * The only suffix allowed is `-NAME.N` (decision Q2):
    * `NAME` is one of `Channels`;
    * `N` is a decimal with no leading zeroes.
  * Build metadata (`+…`) is refused. SemVer ignores it for precedence, so
    two tags differing only in it would compare equal.
* **Compare** is `semver.Compare`, so `v1.3.0-rc.1 < v1.3.0`, and
  `-rc.2 > -rc.1`.
* **`NewStrictVersionPolicy` is unchanged,** and stays the default.

### 2. The per-run opt-in: `Request.Channel`

* **The fields.** `Request` and `CheckRequest` gain `Channel string`. The
  empty string means stable, today's behaviour. A name from the policy's
  `Channels` admits stable releases plus prereleases at least that stable:
  `"beta"` admits `rc` and `beta`, but not `alpha` (decision Q1).
* **Validation.** A channel the policy does not list is refused, as is any
  channel with the strict policy.
* **Cache.** `Channel` is part of the cache key, and the file store's schema
  gains it.
* **Pinning.** `TargetVersion` can name a prerelease only when the request
  names a channel that admits it (decision Q4).

### 3. Discovery: `ReleaseLister`

```go
type ListOptions struct{ Limit int } // releases to consider; capped
type ReleaseLister interface {
    ListReleases(ctx context.Context, o ListOptions) ([]Release, error)
}
```

* **The optional interface.** It is found by a type assertion, never added
  to `ReleaseSource`.
* **The GitHub source** pages `GET /repos/{owner}/{repo}/releases`, with
  `per_page` at its maximum. It stops at `Limit`, which defaults to 100 and
  is capped at 300 *(amended E3: `per_page` 30, `Limit` 90 by default)*. Every body is bounded by `Limits.ReleaseJSON`, and rate
  limits are mapped as `Latest`'s are.
* **Without a channel,** discovery uses `Latest`, exactly as today.
* **With a channel,** the source must be a `ReleaseLister`; otherwise the
  run fails, naming the missing capability. Discovery then:
  1. drops drafts and releases whose tag the policy refuses or the channel
     does not admit;
  2. picks the highest by `Compare` *(amended E5: on tag and flags alone,
     then validated in full)*;
  3. requires that one to be immutable. A mutable winner is an error
     (`ErrMutableRelease`), never a reason to fall back to an older
     release.
* **The tag decides, and the flag must agree.** A release flagged
  `prerelease` must have a prerelease tag, and the reverse, or it is
  dropped.
  * The tag is locked by immutability and the flag is not, so the tag is
    the authority.
  * A disagreement means someone flipped the flag after publication, and
    the release is excluded on every channel.
* **The source no longer refuses prereleases outright.**
  `validateFetchedRelease`'s prerelease refusal moves into discovery,
  where the request's channel is known. Without a channel, the behaviour is
  unchanged.

### 4. Moving between channels

These follow from version precedence, with no special case:

* On `v1.3.0-rc.2`, a stable `v1.3.0` is an upgrade on every channel.
* On `v1.3.0-rc.2`, with the stable channel at `v1.2.0`, the answer is
  `ErrLatestOlder`, as today. Leaving a channel never installs an older
  build unasked; `--version v1.2.0` is the explicit way back.

### 5. Publishing prereleases

The reusable workflow accepts a tag of the prerelease grammar *(amended E4:
only for the names the caller lists in `prerelease-channels-json`)*. It creates
the release with `--prerelease` and marks it so it can never become "latest".
Everything else stays the same:

* `refuse-existing-release`, the verifier and the attestation;
* the requirement that the release be immutable and verified.

The verifier's tag rule mirrors §1 (decision Q3).

### 6. What this record does not decide

* **The command-line surface** (`--channel`) is Phase 3's
  (0004-MADR §5).
* **Mutable hosts** (GitLab and the like) stay with the record on
  non-immutable releases (0004-MADR §6).
* **Automatic channel switching,** such as a stable build that subscribes
  itself to `rc`.

## Owner questions

**Decided 2026-10-01:** the owner accepted the four recommendations
("accept the recommendations"):

* Q1: named channels;
* Q2: the restricted `-NAME.N` grammar;
* Q3: publication in this record;
* Q4: a pinned prerelease needs a channel that admits it.

The questions as asked:

1. **Q1, named channels or a switch.**
   * **Named channels, recommended:** `rc`, `beta`, `alpha`, with "at
     least this stable" admission.
   * **A switch:** a single `AllowPrerelease` that admits any prerelease.

   Named channels cost one string list and give a program "beta testers get
   rc and beta builds" with no API change later. A switch is smaller, but
   is replaced as soon as two channels are needed.
2. **Q2, the prerelease grammar.**
   * **The restricted `-NAME.N` form, recommended.**
   * **Full SemVer prerelease syntax:** any dot-separated identifiers,
     hyphens included.

   The restricted form keeps tags as safe as today's in file names and
   shell arguments, and makes channel admission a lookup.
3. **Q3, publication in this record, or a follow-on.** Publishing changes
   the reusable workflow's contract, which 0002-MADR made the only supported
   publication path. Recommended: in this record, so the feature is usable
   end to end.
4. **Q4, pinning a prerelease.** Should `--version v1.3.0-rc.1` work without
   a channel?
   * **Recommended: no.** The request must name a channel that admits it,
     so a prerelease is never installed without an explicit channel.
   * **Alternative:** the pin itself counts as consent.

### Consequences

* Good, because a program can offer `rc` or `beta` builds with no change for
  its stable users.
* Good, because a prerelease is installed under exactly the stable checks.
  The only relaxation is in what the version grammar admits.
* Good, because `ReleaseLister` is a general capability. A future source
  (GitLab, a signed index) can implement it, and named channels need no API
  change.
* Neutral, because a channel run costs one to three list requests instead of
  one `latest` request, within the same rate-limit handling.
* Bad, because the prerelease refusal moves from the source to discovery. A
  custom consumer that called `GitHubSource.ByTag` directly would now
  receive a prerelease where it used to get an error. The release notes
  must say so.
* Bad, because the reusable workflow's contract grows (Q3). A consumer that
  pushes a prerelease-shaped tag gets a published prerelease, where today it
  gets a refusal.

### Confirmation

* Every refusal that protects the stable path is shown by a test to still
  hold with `Channel` empty. That includes a prerelease tag and a
  prerelease-flagged release under the strict policy, and the `Latest` path
  untouched.
* A test serves out-of-order releases through `selfupdatetest.GitHubServer`:
  * the highest admissible version wins;
  * a mutable winner fails rather than falling back;
  * a flag/tag mismatch is dropped.
* The H4 end-to-end test gains a channel case that installs `v1.3.0-rc.1`
  over a running copy.
* `make apicheck` reports only compatible changes.
* The workflow's tests cover the prerelease tag grammar, if Q3 is accepted.

## Pros and Cons of the Options

### A. Opt-in prerelease selection in the client

* Good, because there is one repository and one release history per program.
* Good, because every existing check applies unchanged.
* Neutral, because it needs the list API and its pagination.
* Bad, because the client must own "which is newest", which `latest` did
  for it before.

### B. A separate repository per channel

* Good, because there is no client change at all.
* Bad, because every program needs a second repository, a second release
  pipeline and a second token scope, per channel.
* Bad, because a stable release must be published twice to reach beta users.

### C. Pinning only

* Good, because it is the smallest change.
* Bad, because there is no channel: users must learn each tag by hand, and
  `--check` can never say "a newer beta exists".

### D. Keep the refusal

* Good, because nothing changes.
* Bad, because it goes against the owner's stated want (0004-MADR, owner
  decision 3).

## More Information

* [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md):
  owner decision 3, §6, and G2.
* mcplib `docs/0005-MADR-canonicalize-cli-self-update-in-mcplib.md`, the
  original rejection of prereleases, quoted under "Evidence".
* The code facts above, at `v1.2.0`.

### Evidence

* **mcplib's rejection was policy, not a finding.**
  * Its 0005-MADR requires "an uploaded, non-draft, non-prerelease
    release; the initial canonical CLI accepts stable complete tags only
    and rejects prerelease targets" (lines 262-263).
  * Its PLAN lists "prerelease installation in v1 of the package" as out
    of scope (line 216).
  * The reasoning it does give is about grammar. The old parser "does not
    validate empty or illegal prerelease/build identifiers, discards build
    metadata, and orders arbitrary invalid strings lexically"
    (lines 149-152). §1's strict prerelease grammar answers that.
  * Its rule "latest cannot silently downgrade" (PLAN line 428) is kept
    by §4.
* **GitHub REST, Releases**
  (<https://docs.github.com/en/rest/releases/releases>):
  * "The latest release is the most recent non-prerelease, non-draft
    release, sorted by the created_at attribute."
  * The list's `per_page` has a maximum of 100, and the page states no
    ordering. "Only users with push access will receive listings for
    draft releases."
  * Get by tag: "Get a published release with the specified tag."
  * On create, `make_latest`: "Drafts and prereleases cannot be set as
    latest."
* **GitHub immutable releases**
  (<https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases>):
  * "Once an immutable release is published, its associated Git tag is
    locked to a specific commit."
  * "You can still edit the title and release notes of a published
    immutable release, and change whether it is marked as a pre-release
    or as the latest release."
  * "If you delete the immutable release, you can delete the tag, but you
    cannot reuse the same tag name."
* **SemVer 2.0.0** (<https://semver.org/spec/v2.0.0.html>):
  * §9: prerelease identifiers "MUST comprise only ASCII alphanumerics
    and hyphens", and "Numeric identifiers MUST NOT include leading
    zeroes".
  * §10: build metadata "MUST be ignored when determining version
    precedence".
  * §11: "1.0.0-alpha < 1.0.0-alpha.1 < 1.0.0-alpha.beta < 1.0.0-beta <
    1.0.0-beta.2 < 1.0.0-beta.11 < 1.0.0-rc.1 < 1.0.0".
* **`golang.org/x/mod/semver`:**
  * it "follows Semantic Versioning 2.0.0 … with two exceptions": the `v`
    prefix is required, and `vMAJOR` and `vMAJOR.MINOR` are accepted as
    shorthands;
  * `Compare` never looks at build metadata.

  So `IsValid("v1.2")` is true, and a strict wrapper stays necessary.
* **`gh release create`** (<https://cli.github.com/manual/gh_release_create>):
  `--prerelease` ("Mark the release as a prerelease") and `--latest=false`
  ("explicitly NOT set as latest").
