---
status: in-progress
date: 2026-10-01
associated-madr: "0004-MADR-evolve-selfupdate-api-and-tui-support.md"
---
# Implement Phase 2 core: per-run options and the interaction `Stream` (`v1.2.0`)

Associated MADR: [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md)

This PLAN implements the core half of the MADR's Decision Outcome §4, "Phase 2:
framework-neutral interaction, and the TUI adapter". That half is everything
in package `selfupdate`, standard library only. It also covers H6 from §8 for
the code it adds.

The adapter half, `go-tui-lib/updatetea`, belongs to another repository, which
records it there (MADR §1 table). That repository is still empty: one README
commit, with no `go.mod`, no records and no `AGENTS.md`. So the adapter waits
for a bootstrap of go-tui-lib under its own records, and the MADR's Phase 2
Confirmation (an example Bubble Tea program in go-tui-lib) is met there, not
here.

The previous PLAN under this number,
[0004-PLAN-v1-1-0-core-api.md](0004-PLAN-v1-1-0-core-api.md), is complete. The
owner tagged `v1.1.0` on `96b3096`, and CI was green on `main` and on the tag.

The MADR leaves two things to "the phase's PLAN": how a per-run credential
provider reaches a source that was built before the run, and the names and
shapes of the API. This PLAN settles both. Where a settlement changes something
the MADR states, it is listed under "Proposed MADR amendments", and Step 1
applies those amendments before any code changes.

## Goal

`v1.2.0` ships, in `selfupdate`:

* `Updater.RunWith`, with per-run reporter, confirmer, credentials and
  progress interval;
* `Start` and `Stream`: a pull-based interaction queue that any event loop can
  drive, with ordered lifecycle delivery and latest-wins progress;
* `ConfirmNeeded`, `CredentialNeeded` and `Finished`;
* `Stream.Cancel`, which cancels the run and still delivers `Finished`. That
  fixes the O6 class of defect by construction.

Each item comes with:

* its API exactly as written in this PLAN;
* tests that fail when the behaviour they guard is broken, with every mutation
  killed;
* a goroutine-leak check on every `Stream` and confirmer test (H6);
* green checks on Linux, macOS and Windows;
* `make apicheck` against `v1.1.0` reporting no incompatible change.

No new module is required, and `go.mod` does not change.

## Scope

### Item → step

| MADR §4 item | Step |
| :--- | :--- |
| records; MADR amendments; the `v1.1.0` release entry | 1 |
| `RunOption`, `RunWith`, `WithReporter`, `WithConfirmer`, `WithProgressInterval` | 2 |
| `WithCredentials`, `CredentialedSource`, `GitHubSource.WithCredentials`; `selfupdatetest.GitHubServer.RequireToken` | 3 |
| `Start`, `Stream`, `Interaction`, `Progressed`, `ConfirmNeeded`, `Finished`; the leak-check helper (H6) | 4 |
| `CredentialNeeded`, `PromptCredential` | 5 |
| H6 for the existing confirmer tests | 6 |
| documentation, examples and close-out | 7 |

### Out of scope

* **`go-tui-lib/updatetea`** (MADR §4, "Adapter"), and the bootstrap of
  go-tui-lib that it needs. Both are recorded in go-tui-lib. Facts for that
  work:
  * the only TUI consumer, ocp-login, uses Bubble Tea v2
    (`charm.land/bubbletea/v2 v2.0.9`, `charm.land/bubbles/v2 v2.2.1`);
  * its reusable pieces are unexported, in `internal/ui`.
* **ocp-login's adoption.** The owner deferred it (MADR, owner decision 2).
* **MADR §5 (Phase 3), §6 records and §7 (Phase 4).** That includes the
  prerelease-channel record, which the owner wants and which is scheduled
  after Phase 1. It is a separate MADR, and nothing here depends on it.
* **The remaining harness items:**
  * H2: the CI fuzz step and the Python/Go manifest differential;
  * H4: the end-to-end test that replaces a running copy of the test binary.

  They are unrelated to Phase 2's API and get their own PLAN.
* Tagging `v1.2.0`, and any `git push`. Both need the owner's explicit ask.

### Fixed inputs

| Input | Value | Source |
| :--- | :--- | :--- |
| Baseline | `v1.1.0` = `96b3096` | `git rev-list -n1 v1.1.0` |
| Go | 1.27.1; `iter.Seq` is in the standard library | `go.mod` |
| Module requirements | `golang.org/x/mod v0.40.0`, `x/sys v0.47.0`, `x/term v0.43.0`; unchanged | `go.mod` |
| API gate | `make apicheck`, which compares with the newest `v1.*` tag, `v1.1.0` | `scripts/check-api-compat.sh` |
| Windows test host | Git Bash, go1.27.1 with cgo | as in the Phase 1 PLAN |

### Code facts the design rests on

* **`Updater` cannot be copied per run.** It holds `running atomic.Bool`
  (`types.go`, `type Updater`), and `go vet`'s `copylocks` refuses a copy.
* **Nine methods read the run's collaborators through the `Updater`:**
  * `execute`, `apply`, `report`, `reportOutcome` and `newProgress` in
    `updater.go`;
  * `runManifestVerifiers` and `openAsset` in `manifestverify.go`;
  * `downloadProgress.emit` and `downloadProgress.written`.

  Between them they read `u.source`, `u.reporter`, `u.confirmer` and
  `u.progress`. `execute` discovers through `u.Checker()`, which copies
  `u.source`.
* **`GitHubSource` resolves its credential once per source.** It does so on
  the first API request, under `credentialState.mu`, in the order explicit
  `Token`, then `Credentials` provider, then `GH_TOKEN` / `GITHUB_TOKEN`
  (`github.go`, `credential` and `resolveLocked`).
  * A 401 to a provider's credential asks the provider once more, with
    `CredentialRequest.Cause` set (`refresh`).
  * The provider is called with the request's context, which is the run's
    context.
* **The precedent.** ocp-login runs its update inside Bubble Tea with a
  64-slot channel per step and a self-reissuing `tea.Cmd`
  (`internal/ui/steps.go:200-216`).
  * Its ctrl+c handler sets `context.Canceled` and quits without cancelling
    anything (`:225-231`). That is O6. After the quit nothing drains the
    channel, and the producer can block forever on a full buffer.
  * Its token prompt and confirmation both happen outside the run
    (`cmd/update.go:126-259`), which is how O3 arises.

### Compatibility rules the design follows

The rules are the Phase 1 PLAN's ("Compatibility rules the design follows"),
unchanged. They come down to four:

* only new types, functions and methods;
* no method added to an existing interface;
* no field added to a comparable struct unless its type is comparable;
* no signature changed.

`Run` keeps its signature and becomes `RunWith` with no options.

### Rules for every step

The Phase 1 PLAN's five rules apply, with two changes:

* `make apicheck` runs in every step;
* rule 3's list of existing tests that may change is this PLAN's own:
  * Step 6 adds the leak check to confirmer tests;
  * no other existing assertion changes.

## Proposed MADR amendments

Step 1 applies these to the MADR, each marked *(amended 2026-MM-DD,
0004-PLAN-v1-2-0-interaction-stream)*. Approving this PLAN approves them.

| ID | MADR text today | Amendment | Why |
| :--- | :--- | :--- | :--- |
| B1 | §4: "How that provider reaches a source constructed before the run is left to the phase's PLAN. The two candidates are a request-scoped context value, and an optional `ReleaseSource` interface…" | Both are used, each for one job. **`WithCredentials(p)`** needs the source to implement a new optional interface, `CredentialedSource`, whose `WithCredentials(p)` returns a per-run copy of the source with `p` in its provider slot and fresh credential state. `RunWith` refuses the option for any other source. **The prompt** is a provider value, `PromptCredential()`. It finds the run's `Stream` through the run's context, and outside a `Stream` returns `ErrNoCredential`. | A per-run copy keeps one run's credential out of the next. A context value for optional parameters is what the `context` package documentation advises against. The prompt is different: the run's interaction queue is request-scoped data, which is exactly what context values are for. Because it is a plain provider, it composes with `ChainCredentials` (environment first, prompt last). It also works when set once in `GitHubOptions.Credentials`, with no per-run option at all. |
| B2 | §4: `func Start(ctx context.Context, u *Updater, req Request) *Stream` | `func Start(ctx context.Context, u *Updater, req Request, opts ...RunOption) *Stream`. `WithReporter` here *adds* a reporter: events go to the `Stream` first, then to it. `WithConfirmer` here *replaces* `ConfirmNeeded`. The `Updater`'s own reporter and confirmer are not used. | A TUI still wants a log file, or can auto-approve. A variadic tail is source-compatible with the MADR's call shape. |
| B3 | §4 has no per-run progress setting; amendment A1's reason says "the go-tui-lib adapter (Phase 2) sets the interval itself" | Add `WithProgressInterval(d time.Duration) RunOption`. Zero turns progress off; a negative value is refused. | A1's promise needs a per-run way to keep it. `Config.ProgressInterval` stays the default. |
| B4 | §4: "Progress is latest-wins" and `Progressed struct{ Event Event }` | `Progressed` carries every reported `Event`, in order. Only `EventProgress` is coalesced: a new progress event replaces a progress event that is still the newest undelivered item, and never any other item. | Ordering is preserved, and lifecycle events can never be dropped. The queue holds at most one progress item between two lifecycle items. |
| B5 | §4: `Cancel(err error)` on both requests, with nil unstated | `ConfirmNeeded.Cancel(nil)` fails the confirmation with `context.Canceled`. `CredentialNeeded.Cancel(nil)` returns `ErrNoCredential`, so a chain moves on or the request goes anonymous. A non-nil error is returned as given. The first call to `Answer`, `Supply` or `Cancel` wins, and later calls do nothing. | "Dismiss" means different things for the two requests. An idempotent reply makes a double key-press harmless. |
| B6 | §4: "`Next` … `io.EOF` after `Finished`" | `Next(ctx)` returns `io.EOF` only once `Finished` has been returned. When `ctx` ends first it returns `ctx.Err()` and consumes nothing. `All(ctx)` yields up to and including `Finished`, and stops early when `ctx` ends or the loop breaks. Breaking out of `All` does not cancel the run; `Cancel` does. | This makes `Next` safe to call from a UI command with its own deadline. |

## Implementation Steps

### Step 1: records

1. Apply amendments B1–B6 to the MADR, in §4.
2. Add this PLAN's row to `docs/README.md`, with status `in-progress`.
3. In [0004-PLAN-v1-1-0-core-api.md](0004-PLAN-v1-1-0-core-api.md), add
   "Release (2026-10-01)": the owner tagged `v1.1.0` on `96b3096`, and CI run
   `36802910523` (`main`) and run `36802931390` (the tag) concluded `success`.
4. In `docs/architecture.md`, change the current release to `v1.1.0` on
   `96b30961180671ab3697585951219001ecbb1c90`.
5. One commit; docs only.

### Step 2: per-run options (`runoptions.go` (new), `updater.go`, `manifestverify.go`, `checker.go`)

**API**

```go
type RunOption interface{ applyRun(*runScope) error }
func WithReporter(r Reporter) RunOption
func WithConfirmer(c Confirmer) RunOption
func WithProgressInterval(d time.Duration) RunOption
func (u *Updater) RunWith(ctx context.Context, req Request, opts ...RunOption) (Result, error)
```

**Behaviour**

1. **The run scope.**
   * An unexported `type run struct { *Updater; source ReleaseSource;
     reporter Reporter; confirmer Confirmer; progress time.Duration }` holds
     one run's collaborators.
   * The nine methods under "Code facts" take `*run` as their receiver.
     Their bodies keep `u.reporter`, `u.source`, `u.confirmer` and
     `u.progress`, which now resolve to the run's fields, because a
     shallower field wins over the embedded one.
   * `downloadProgress.u` becomes `*run`.
   * Discovery uses a `Checker` built from the run's source.
2. **`RunWith`.**
   * It takes the same `running` guard as `Run` (`ErrConcurrentUpdate`).
   * It applies the options in order, so the last one of a kind wins, then
     runs `execute`.
   * A nil or typed-nil reporter or confirmer, or a negative interval, fails
     before anything else, with `selfupdate: WithReporter: reporter is nil`
     or the matching message. No event is reported for it.
   * `Run(ctx, req)` is `RunWith(ctx, req)`.
3. **Nothing about the `Updater` changes.** A later `Run` uses the
   configured collaborators again.

**Tests** (`runoptions_test.go`)

* `TestRunWithOverridesReporter`: the override gets every event, and the
  configured reporter gets none. A following `Run` reports to the
  configured one again.
* `TestRunWithOverridesConfirmer`.
* `TestRunWithProgressInterval`: a 1 ns override yields `EventProgress` on an
  `Updater` configured with zero, and `WithProgressInterval(0)` turns progress
  off on one configured with 1 ns.
* `TestRunWithLastOptionWins`.
* `TestRunWithRejectsInvalidOptions`: nil, typed nil and negative, with
  their messages.
* `TestRunWithSharesRunGuard`: a `RunWith` during a `Run` is
  `ErrConcurrentUpdate`.
* The existing suite passes unchanged, which covers `Run` as `RunWith` with
  no options.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| `report` reads the `Updater`'s reporter | `TestRunWithOverridesReporter` |
| the confirm path reads the `Updater`'s confirmer | `TestRunWithOverridesConfirmer` |
| `newProgress` reads `Config.ProgressInterval` | `TestRunWithProgressInterval` |
| options applied in reverse | `TestRunWithLastOptionWins` |
| the nil-reporter check removed | `TestRunWithRejectsInvalidOptions` |
| `RunWith` skips the guard | `TestRunWithSharesRunGuard` |

### Step 3: per-run credentials (`runoptions.go`, `credentials.go`, `github.go`, `selfupdatetest/githubserver.go`)

**API**

```go
func WithCredentials(p CredentialProvider) RunOption
type CredentialedSource interface {
    ReleaseSource
    WithCredentials(CredentialProvider) ReleaseSource
}
func (s *GitHubSource) WithCredentials(p CredentialProvider) ReleaseSource
// selfupdatetest:
func (g *GitHubServer) RequireToken(token string) // "" serves anonymously again
```

**Behaviour**

1. **`GitHubSource.WithCredentials(p)`** returns a new `*GitHubSource`.
   * It shares the repository, client, API base, user agent, explicit token,
     environment token, observer, limits and clock.
   * Its provider is `p`, or none when `p` is nil or typed nil.
   * Its `credentialState` is new and zero. The struct is built field by
     field, because a copy would copy the mutex.
   * The receiver is never changed.
2. **`WithCredentials(p)` in `RunWith`.**
   * When the configured source is a `CredentialedSource`, the run uses
     `source.WithCredentials(p)`.
   * Otherwise the run fails with
     `selfupdate: WithCredentials: the source does not accept per-run credentials`.
   * A nil or typed-nil `p` is refused, as in Step 2.
   * The credential order for the run is the explicit `Token`, then `p`, then
     the environment. That is `GitHubOptions.Credentials`' order, with `p`
     in its slot.
3. **`GitHubServer.RequireToken(token)`.** API requests whose
   `Authorization` is not `Bearer <token>` get a 401 with GitHub's JSON
   message. Asset downloads on the second origin are not affected.

**Tests**

* `TestRunWithCredentialsIsPerRun` (`e2e_github_test.go`, package
  `selfupdate_test`):
  * The server requires a token, and the source is anonymous.
  * `RunWith(WithCredentials(static))` applies.
  * A following plain `Run` on the same `Updater` fails with the 401.
  * No `Authorization` reaches the download origin.
* `TestGitHubSourceWithCredentials` (`credentials_test.go`): the receiver's
  provider and state are unchanged; the copy resolves `p`; the explicit
  `Token` still wins; and a nil `p` gives a copy with no provider.
* `TestRunWithCredentialsNeedsCredentialedSource` (package
  `selfupdate_test`): with `FakeSource`, the refusal message, and no
  `Latest` call.
* `selfupdatetest`: `TestGitHubServer` gains `RequireToken` cases.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| `WithCredentials` returns the receiver with its provider changed | `TestRunWithCredentialsIsPerRun` (the plain `Run` succeeds) |
| the copy keeps the receiver's resolved state | `TestGitHubSourceWithCredentials` |
| `RunWith` ignores the option for a source that is not a `CredentialedSource` | `TestRunWithCredentialsNeedsCredentialedSource` |
| `RequireToken` ignored | `TestGitHubServer` |

### Step 4: the `Stream` (`stream.go` (new))

**API**

```go
type Stream struct{ /* unexported */ }
func Start(ctx context.Context, u *Updater, req Request, opts ...RunOption) *Stream
func (s *Stream) Next(ctx context.Context) (Interaction, error)
func (s *Stream) All(ctx context.Context) iter.Seq[Interaction]
func (s *Stream) Cancel()

type Interaction interface{ interaction() }
type Progressed struct{ Event Event }
type ConfirmNeeded struct {
    Prompt Prompt
    // unexported reply slot
}
func (c *ConfirmNeeded) Answer(ok bool)
func (c *ConfirmNeeded) Cancel(err error)
type Finished struct {
    Result Result
    Err    error
}
```

`Progressed` and `Finished` are delivered as values, and `*ConfirmNeeded` as
a pointer.

**Behaviour**

1. **`Start`.**
   * It derives the run context with `context.WithCancel(ctx)` and stores the
     `Stream` in it under an unexported key, which Step 5 uses.
   * It then calls the `run` path of Step 2 in one goroutine, with the
     `Stream`'s reporter and confirmer as amendment B2 describes.
   * When the run returns it enqueues `Finished`, then releases the context.
   * A nil `u`, or an invalid option, gives a `Stream` whose only
     interaction is `Finished` with that error. So `Start` never returns nil.
2. **The queue.** One mutex guards a slice and a one-slot wake channel.
   * The `Stream`'s reporter enqueues `Progressed{ev}` and always returns nil.
   * An `EventProgress` replaces the newest undelivered item when that item
     is also an `EventProgress` (amendment B4). Nothing else is ever
     replaced or dropped.
   * Enqueueing never blocks, so the run never waits for the UI.
3. **The confirmer** enqueues `&ConfirmNeeded{Prompt: p}`, then waits for the
   reply or for the run context to end.
   * `Answer(ok)` returns `ok, nil`.
   * `Cancel(err)` returns `false` and `err`, or `context.Canceled` when
     `err` is nil (amendment B5).
   * The reply slot has one place and a `sync.Once`, so later replies do
     nothing and never block.
   * `Request.Yes` and `Request.DryRun` skip the confirmer, as today, so no
     `ConfirmNeeded` appears.
4. **`Next` and `All`** follow amendment B6. `Next` is safe from several
   goroutines; each interaction is returned once.
5. **`Cancel`** cancels the run context, and may be called any number of
   times, from any goroutine.
   * The run returns through its usual paths: the installer's recovery, the
     session close, and an `EventFailed` with detail `canceled`. Then
     `Finished` arrives with an error matching `context.Canceled`.
   * A waiting confirmer returns as soon as the context ends.
6. **No goroutine outlives the run.** The one run goroutine ends when the run
   returns, and the run can wait only on a reply or the context. A host must
   answer each request or call `Cancel`, and the doc comment says so.

**Tests**

There are two files:

* **`stream_test.go`** (package `selfupdate`) covers the queue itself.
* **`stream_run_test.go`** (package `selfupdate_test`) covers whole runs
  through `selfupdatetest.FakeSource`. The package's internal tests cannot
  import `selfupdatetest`, because it imports `selfupdate`.

The leak helper is `checkNoLeak(t)`. It records `runtime.NumGoroutine()`, and
in `t.Cleanup` polls for up to 2 s until the count is back at or below it,
failing with both numbers. It moves the loop out of
`TestConfirmLeaksNoReader`, and every test below calls it. `export_test.go`
(package `selfupdate`) exposes it to the external tests as
`selfupdate.CheckNoLeak`. That is the standard `export_test.go` pattern,
and it adds nothing to the package's API.

* `TestStreamDeliversRunInOrder`. With `FakeSource`, a standalone installer
  on a temporary target and `Yes: true`:
  * the kinds of the `Progressed` events equal `jsonl-upgrade.golden`'s
    kinds;
  * then `Finished` with `Applied`;
  * then `io.EOF`.
* `TestStreamConfirm`. A `ConfirmNeeded` carries the `Prompt`, and:
  * `Answer(true)` applies;
  * `Answer(false)` gives `Declined`;
  * a second `Answer` neither blocks nor changes anything.
* `TestStreamCancelWhileConfirming`. `Cancel` with a `ConfirmNeeded`
  pending gives `Finished` with `context.Canceled`, and the target is
  byte-identical.
* `TestStreamCancelDuringDownload` (O6):
  * The binary body blocks until its context ends, and `Cancel` is called
    while it is downloading.
  * `Finished` arrives with `context.Canceled`, and only after the session
    has closed: no staging file is left when `Finished` is read.
  * The target is byte-identical.
* `TestStreamProgressLatestWins`. Directly on the queue:
  * 1,000 progress events, then a lifecycle event, then 1,000 more;
  * with no reader running, that gives exactly three items: the last of the
    first batch, the lifecycle event, and the last of the second batch, in
    that order.
* `TestStreamNextContext`. `Next` on an ended context returns `ctx.Err()`,
  and the next `Next` still returns the item.
* `TestStreamAll`: `All` yields up to and including `Finished`. A `break`
  leaves the run going, and `Cancel` then finishes it.
* `TestStartInvalid`: a nil `Updater`, and `WithReporter(nil)`, each give a
  lone `Finished` with the error.
* `TestStreamConcurrentRun`: `Start` during a `Run` gives `Finished` with
  `ErrConcurrentUpdate`.
* `TestStartReporterFansOut` and `TestStartConfirmerReplaces` (amendment B2).

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| `Cancel` does nothing | `TestStreamCancelDuringDownload` (times out) |
| `Finished` is enqueued before the run returns | `TestStreamCancelDuringDownload` (staging still present) |
| progress replaces any newest item | `TestStreamProgressLatestWins` |
| progress is never coalesced | `TestStreamProgressLatestWins` |
| the confirmer ignores the context | `TestStreamCancelWhileConfirming` |
| `Next` consumes before the context check | `TestStreamNextContext` |
| `Answer` without `sync.Once` (sends twice) | `TestStreamConfirm` (blocks) |

### Step 5: credential prompts (`stream.go`, `credentials.go`)

**API**

```go
type CredentialNeeded struct {
    Request CredentialRequest
    // unexported reply slot
}
func (c *CredentialNeeded) Supply(cred Credential)
func (c *CredentialNeeded) Cancel(err error)
func PromptCredential() CredentialProvider
```

**Behaviour**

1. **`PromptCredential().Credential(ctx, r)`.**
   * When `ctx` carries a `Stream` (Step 4, behaviour 1), it enqueues
     `&CredentialNeeded{Request: r}` and waits for the reply or for `ctx` to
     end.
   * `Supply(c)` returns `c`. The source still validates it, as for any
     provider.
   * `Cancel(err)` returns `err`, or `ErrNoCredential` when `err` is nil
     (amendment B5).
   * Without a `Stream` it returns `ErrNoCredential` at once, so the same
     chain works under `Run` without a prompt.
2. **The refusal.** A 401 to a prompted credential prompts once more, with
   `Request.Cause` set, through the existing `refresh`. A second refusal is
   the run's error.
3. **`CredentialNeeded.Request.Origin`** is a copy. Mutating it changes
   nothing.

**Tests**

* `TestStreamCredentialPrompt` (`e2e_github_test.go`):
  * The `GitHubServer` requires a token, and the source is built with
    `Credentials: ChainCredentials(EnvCredential("", "SELFUPDATE_TEST_UNSET"), PromptCredential())`.
  * The first `CredentialNeeded` has a nil `Cause`. `Supply` gives the wrong
    token.
  * The second has a non-nil `Cause`. `Supply` gives the right one, and the
    run applies.
  * The `CredentialObserver` is told once, with the right one.
* `TestStreamCredentialPerRunOption`: the same, with the prompt passed as
  `Start(…, WithCredentials(PromptCredential()))` to an anonymous source.
* `TestPromptCredentialWithoutStream`: `ErrNoCredential`, at once.
* `TestCredentialNeededCancel`:
  * `Cancel(nil)` goes anonymous and ends in the 401;
  * `Cancel(err)` ends the run with an error matching `err`;
  * `Stream.Cancel` while one is pending gives `context.Canceled`.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| `PromptCredential` ignores the context's `Stream` | `TestStreamCredentialPrompt` |
| `Cancel(nil)` returns `context.Canceled` | `TestCredentialNeededCancel` |
| the prompt ignores `ctx.Done()` | `TestCredentialNeededCancel` (`Stream.Cancel` case) |
| `Supply` without `sync.Once` | `TestStreamCredentialPrompt` (a repeated `Supply` blocks) |

### Step 6: H6 for the confirmer tests (`confirmer_test.go`, `confirmer_read_test.go`)

1. Every test that calls `Confirm` on a terminal or prompt confirmer calls
   `checkNoLeak(t)` first. `TestConfirmLeaksNoReader` keeps its assertion,
   through the helper.
2. **The three `time.Sleep` calls stay.** They are not ordering
   handshakes. Each one gives a reader wrongly left behind time to block,
   so the failure it would cause is observed rather than raced. Two already
   say so in a comment (`confirmer_test.go:105-107`,
   `confirmer_read_test.go:67-69`), and the third gains one
   (`confirmer_read_test.go:99`, inside the leak poll, which moves into the
   helper). Replacing them with a channel would need a hook inside the reader
   that the defect itself removes.
3. **Proof.** On a scratch copy, a confirmer that leaves its reader running
   after an answer must fail `checkNoLeak` in at least two tests:
   `TestConfirmersShareInput` and `TestConfirmLeaksNoReader`.

### Step 7: documentation and close-out (`doc.go`, `example_test.go`, `docs/guides/extending-selfupdate.md`, `docs/architecture.md`, `docs/README.md`, this PLAN)

1. **`doc.go`.** A section "Driving an update from an event loop": what
   `RunWith`, `Start`, `Stream` and `PromptCredential` do, and the host's
   duty to answer or cancel.
2. **Runnable examples** (offline, `// Output:`):
   * `ExampleUpdater_RunWith`;
   * `ExampleStart`: a plain loop over `Next` that answers `ConfirmNeeded`;
   * `ExampleStream_All`;
   * `ExamplePromptCredential`.
3. **The guide.** It gains "Drive an update from a TUI": the `Next` loop,
   the Bubble Tea "wait for activity" pattern, cancellation, and a pointer
   to the planned `go-tui-lib/updatetea`. `docs/README.md` gains the row
   "drive an update from a TUI or event loop".
4. **`docs/architecture.md`** records `runoptions.go`, `stream.go` and the
   `run` scope.
5. **Release notes for `v1.2.0`**, in the execution record. They contain:
   * the additions;
   * one behaviour note: `Run` is now `RunWith` with no options, and
     behaves the same;
   * migration notes.
6. **Verification before close-out** is the Phase 1 PLAN's list, plus a full
   `apidiff -m` against `v1.1.0` that lists only additions.
7. **Status.** The PLAN is marked `complete` only after CI is green on the
   pushed tree.

## Verification

* **Every step.**
  * Its mutations are all killed.
  * `make pre-add-check` passes, and so does `make apicheck`.
  * Every step that changes Go code (2–6) also passes the Windows test host.
* **Before release.**
  * `go test -race -count=3 ./...` and `go test -shuffle=on -count=2 ./...`
    pass.
  * `make vuln` passes.
  * `go mod tidy -diff` is clean, and `go.mod` is unchanged from `v1.1.0`.
  * `apidiff -m` against `v1.1.0` lists only additions.
* **After the owner's push.** CI is green on `ubuntu-24.04`, `macos-15` and
  `windows-2025`.
* **Not here.** The MADR's Phase 2 Confirmation (a Bubble Tea example in
  go-tui-lib against the H3 server, where ctrl+c leaves the target unchanged)
  is met by go-tui-lib's own records. This PLAN's
  `TestStreamCancelDuringDownload` proves the property it relies on.

## Rollout and Rollback

* **Rollback.** Each step is one commit, and a step can be reverted alone,
  in reverse order. Step 2's refactor is the only change to existing code
  paths; the existing suite passing unchanged is its check.
* **Tagging.** `v1.2.0` is tagged only on the owner's ask, after Step 7.
* **Consumers.** Nothing changes for a consumer that does not call the new
  API.
* **Stopping partway.** Steps 2 and 3 are useful without the `Stream`. If
  the work stops after Step 3, `v1.2.0` can ship `RunWith` alone.

## Execution Record

### Approval (2026-10-01)

The owner approved this PLAN and its amendments B1–B6. They kept
`CredentialNeeded` and `PromptCredential`, which have no consumer today,
with this standing direction: "i do want to build speculative API, when
sensible, to future-proof as we build this out. extensibility,
flexibility, and idiomatic/modular design is optimal."

### Step 1: records (2026-10-01)

* The MADR §4 gained the "Amended 2026-10-01" block, with B1–B6, and
  three inline marks: on `Start`, on `Progressed`, and on the mid-run
  credentials paragraph.
* This PLAN was indexed in `docs/README.md` as `in-progress`.
* [0004-PLAN-v1-1-0-core-api.md](0004-PLAN-v1-1-0-core-api.md) gained its
  "Release (2026-10-01)" entry.
* `docs/architecture.md` names `v1.1.0` on `96b3096` as the current
  release.
