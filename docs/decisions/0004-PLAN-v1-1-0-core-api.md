---
status: in-progress
date: 2026-09-30
associated-madr: "0004-MADR-evolve-selfupdate-api-and-tui-support.md"
---
# Implement Phase 1: the `v1.1.0` core API

Associated MADR: [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md)

This PLAN implements the MADR's Decision Outcome §3, "Phase 1: core API
(`v1.1.0`)". It also covers these items from its §8 harness list:

* H3, the `selfupdatetest` package, which §8 says is exported in Phase 1;
* H5, golden reporter output;
* H7, the CI additions;
* the `apidiff` gate that the MADR's Confirmation names for Phase 1.

The previous PLAN under this number,
[0004-PLAN-v1-0-1-defect-release.md](0004-PLAN-v1-0-1-defect-release.md), is
complete, and `v1.0.1` is tagged at `2ec2c68`.

The MADR leaves names and shapes to "each phase's PLAN". This PLAN settles
them. Where a settlement changes something the MADR states, it is listed
under "Proposed MADR amendments", and Step 1 applies those amendments
before any code changes.

## Goal

`v1.1.0` ships every §3 item. Each item comes with:

* its API exactly as written in this PLAN;
* tests that fail when the behaviour they guard is broken, with every
  mutation killed;
* green checks on Linux, macOS and Windows;
* an `apidiff` report against `v1.0.1` that lists no incompatible change.

No new module is required, and `go.mod` does not change.

## Scope

### Item → step

| MADR §3 item | Step |
| :--- | :--- |
| records; MADR amendments | 1 |
| exported `ErrForceRequired` and `ErrLatestOlder` (G4); `ParseSHA256SUMS`, `ExactAssetName`, `AssetStateUploaded`; `Config.Versions` honoured (G2); function adapters, `DiscardReporter`, `MultiReporter`, `NonInteractiveConfirmer`, `NewPromptConfirmer` (G6) | 2 |
| `Checker`, `CheckRequest`, `Availability`, `Updater.Checker` (G3) | 3 |
| `CheckCached`, `CheckRecord`, `CheckStore`, `NewFileCheckStore` | 4 |
| `EventProgress`, `EventDeclined`, `EventFailed`, `EventRolledBack`, `Event.Total`, `Config.ProgressInterval` (G5); `NewJSONReporter`; `Result.Document` | 5 |
| `Credential`, `CredentialProvider`, `CredentialObserver`, `ChainCredentials`, `EnvCredential`; `GitHubOptions.Credentials` and `.Observer` (G10) | 6 |
| `ManifestVerifier`, `Config.ManifestVerifiers`, `Verification.OpenAsset`; `NewImageVerifier` (G9) | 7 |
| `Prober`, `NewVersionProber`, `Config.Probes`, runnable staging, `InstallOptions.PostInstall` (G9) | 8 |
| `StagingOwner`, `TwoPhaseSession`, `NewManagedInstallerFor`; `sessOwns` fails closed (G7) | 9 |
| `Request.DryRun`, `InstallOptions.KeepPrevious`, `(*StandaloneInstaller).CleanupPending` (G11) | 10 |
| H3 `selfupdatetest`; H5 golden reporter output | 11 |
| `apidiff` gate; H7 CI additions | 12 |
| documentation, examples and close-out | 13 |

### Out of scope

* MADR §4 (Phase 2): `RunWith`, `Stream`, `Interaction`, and the
  `updatetea` adapter. §5 (Phase 3): `selfupdate/cli`, `buildinfo` and
  `UserAgent`. §6: non-immutable sources, prerelease channels and signing.
  §7 (Phase 4): shared release tooling, installer templates, service
  adapters and codesign.
* H4, the end-to-end test that replaces a running copy of the test binary.
  It needs a helper-process harness of its own, and gets a separate PLAN.
* G12 (the split of `Limits`). No Phase 1 item needs it.
* Tagging `v1.1.0`, and any `git push`. Both need the owner's explicit ask.

### Fixed inputs

| Input | Value | Source |
| :--- | :--- | :--- |
| Baseline | `v1.0.1` = `2ec2c68` | `git rev-list -n1 v1.0.1` |
| Go | 1.27.1 | `go.mod` |
| Module requirements | `golang.org/x/mod v0.40.0`, `x/sys v0.47.0`, `x/term v0.43.0`; unchanged | `go.mod` |
| API diff tool | `golang.org/x/exp/cmd/apidiff@v0.0.0-20260908205506-85c1c2202aba`, run with `go run` and never required by `go.mod` | pkg.go.dev versions tab; x/exp has no tags |
| Windows test host | Git Bash, go1.27.1 with cgo (`-race` works) | as in the Phase 0 PLAN |

### Compatibility rules the design follows

These rules come from `apidiff`'s README and source (x/exp
`apidiff/README.md`, `compatibility.go`, `apidiff.go`) and from
<https://go.dev/blog/module-compatibility>:

* **Adding an exported struct field is compatible,** unless the struct
  stops being comparable (`old is comparable, new is not`). `Request`,
  `Result`, `Event`, `InstallRequest`, `InstallResult` and `GitHubOptions`
  are comparable today, so every field added to them has a comparable type
  (string, bool, int, `time.Duration`, an enum, or an interface).
  `Credential`, which holds a `[]byte`, is a new type and appears in none of
  them.
* **Adding a method to an exported interface is incompatible.** New
  behaviour arrives through new interfaces, discovered with a type
  assertion (`StagingOwner`, `TwoPhaseSession`), or through new `Config`
  fields.
* **Appending constants to a typed enum is compatible; changing a value is
  not.** New `EventKind`s are appended after `EventComplete` (value 9), and
  a test pins every numeric value.
* **Changing an exported signature is incompatible.** No existing exported
  function or method changes. `NewManagedInstaller(*StandaloneInstaller, …)`
  stays, and `NewManagedInstallerFor` is added.
* **Unkeyed composite literals of exported structs are the caller's
  risk,** and `apidiff` does not report them. New fields are appended last,
  so a mis-ordered literal fails to compile rather than silently
  mis-assigning a field.
* **Enum JSON encoding does not change.** No existing enum gains
  `MarshalText` (MADR §3). The JSON forms are separate document types.

### Rules for every step

1. **Order within a step.** Code and tests first. Then the step's mutation
   proofs, each on a scratch copy of the working tree. Then
   `make pre-add-check` on the changed files, a Windows test-host run
   (`go vet ./...` and `go test -race -count=1 ./...`) when the step
   touches code that runs on Windows, and `make apicheck` from Step 12 on.
   Then one `git commit --no-edit`.
2. **Mutation proofs.** Each named mutation replaces one exact string in
   one file, on a scratch copy, and the step's named tests must then fail.
   A surviving mutation stops the step. The runner is the Phase 0 script
   pattern (`mutation_proofs.py`): exact `old` → `new`, one test pattern,
   `go test -count=1 -timeout 60s`.
3. **Existing tests.** They change only where this PLAN says so: appended
   event expectations in Step 5, and a `StagingOwner` method on test
   sessions in Step 9. Any other change to an existing assertion stops
   the step.
4. **Deviations.** Anything outside this PLAN stops the step and is
   recorded as a dated deviation before work continues.
5. **Doc comments.** Every new exported identifier has one (revive
   `exported`), and cites the MADR item it implements.

## Proposed MADR amendments

Step 1 applies these to the MADR, each marked *(amended 2026-MM-DD,
0004-PLAN-v1-1-0-core-api)*. Approving this PLAN approves them.

| ID | MADR text today | Amendment | Why |
| :--- | :--- | :--- | :--- |
| A1 | §3: progress is emitted every `Config.ProgressInterval`, "default 100 ms" | A zero `ProgressInterval` emits **no** progress events; progress is opt-in. | A v1 consumer's reporter would otherwise start receiving about 10 events a second without asking for them. All six known consumers use `NewTextReporter`, but unknown reporters exist by definition. The go-tui-lib adapter (Phase 2) sets the interval itself. |
| A2 | §3: `Prober.Probe(ctx, path string, rel Release)` | `Probe(ctx context.Context, req ProbeRequest) error` | A request struct can gain fields compatibly; parameters cannot. |
| A3 | §3: `Availability{Operation, Current, Latest, ReleaseURL, Available}` | Named like `Result`: `Product`, `CurrentVersion`, `TargetVersion`, `ReleaseURL`, `AssetName`, `Operation`, `Available`, `ForceRequired`. | One vocabulary across the package. `ForceRequired` replaces reading `Event.Detail`. |
| A4 | §3: `TwoPhaseSession.Apply` returns an opaque `Applied` | `AppliedReplacement{Target, Backup string; State any}` | A third-party session has to carry its own state from `Apply` to `Commit`, and an opaque type with unexported fields would stop it. |
| A5 | §3 lists no result fields for rollback, dry run or keep-previous | Add `InstallResult.RolledBack`, `InstallResult.Previous`, `InstallRequest.TargetVersion`, `Result.DryRun` and `Result.Previous`. | The coordinator and the caller need these facts, and all five fields are comparable. |
| A6 | §3: `Credential{Header string; Value []byte; Source string}`, with the header semantics unstated | An empty `Header` means `Authorization: Bearer <Value>`. Otherwise `Header: Value` is sent verbatim. | This fits GitHub's scheme with no helper, and still covers `PRIVATE-TOKEN`-style headers. |
| A7 | §3: only progress events are "advisory" | `EventDeclined`, `EventFailed` and `EventRolledBack` are advisory too: a reporter error on them is ignored. | They report an outcome that has already happened, so a reporter failure must not change it. |
| A8 | §3 is silent on `NewTextReporter` | `NewTextReporter` skips `EventProgress`. It prints the other new kinds with their existing `String()` names. | This keeps existing text output free of progress spam, whatever the interval. |

## Implementation Steps

### Step 1: records

1. Apply amendments A1–A8 to the MADR.
2. Add this PLAN's row to `docs/README.md`, with status `in-progress`.
3. In [0004-PLAN-v1-0-1-defect-release.md](0004-PLAN-v1-0-1-defect-release.md),
   record that the owner tagged `v1.0.1` at `2ec2c68`, and that CI was
   green.

### Step 2: foundations (`errors.go`, `version.go`, `assets.go`, `checksums.go`, `github.go`, `adapters.go` (new), `confirmer.go`)

**API**

```go
// errors.go: the existing unexported values, renamed; messages unchanged.
var (
    ErrForceRequired = errors.New("selfupdate: replacing a local build requires --force")
    ErrLatestOlder   = errors.New("selfupdate: latest release is older than the running version")
)

// checksums.go
func ParseSHA256SUMS(data []byte) (map[string]string, error) // wraps parseSHA256SUMS; the map is the caller's

// assets.go
const AssetStateUploaded = "uploaded"
func ExactAssetName(product string, platform Platform) string // returns exactAssetName; validates nothing

// adapters.go
type ReporterFunc func(context.Context, Event) error
func (f ReporterFunc) Report(ctx context.Context, ev Event) error
type ConfirmerFunc func(context.Context, Prompt) (bool, error)
func (f ConfirmerFunc) Confirm(ctx context.Context, p Prompt) (bool, error)
type VerifierFunc func(context.Context, Verification) error
func (f VerifierFunc) Verify(ctx context.Context, v Verification) error
type TransformerFunc func(context.Context, TransformRequest) error
func (f TransformerFunc) Transform(ctx context.Context, r TransformRequest) error
func DiscardReporter() Reporter
func MultiReporter(reporters ...Reporter) Reporter
func NonInteractiveConfirmer() Confirmer

// confirmer.go
func NewPromptConfirmer(in io.Reader, out io.Writer, interactive bool) Confirmer
```

**Behaviour**

1. Every internal use of `errForceRequired` and `errLatestOlder` becomes
   the exported name. The error texts are byte-identical.
2. `validateRequest(req Request, versions VersionPolicy)` validates
   `CurrentVersion` (release builds) and `TargetVersion` with `versions`,
   not with `NewStrictVersionPolicy()` (`version.go:72,77` today). `New`
   passes `cfg.Versions`. With the strict policy, behaviour is unchanged.
3. `github.go` compares `a.State` with `AssetStateUploaded`.
4. `MultiReporter` drops nil and typed-nil entries (`isNil`) at
   construction.
   * `Report` calls every remaining reporter in order, even after one
     fails, and returns `errors.Join` of their errors.
   * With no reporters left it behaves like `DiscardReporter`.
5. `DiscardReporter().Report` returns nil.
6. `NonInteractiveConfirmer().Confirm` returns `false` and
   `selfupdate: pass --yes to apply without prompting: <ErrConfirmationRequired>`.
7. `NewPromptConfirmer(in, out, interactive)`:
   * **Errors:** a nil `in` gives `ErrConfirmationRequired`
     ("confirmation input is nil"). `interactive == false` gives
     `ErrConfirmationRequired`, as in rule 6. A nil `out` gives
     "confirmation output is nil".
   * **Prompt:** the prompt text and yes/no parsing are those of the
     terminal confirmer.
   * **Reads:** one outstanding read, byte by byte (`readLine` from Phase 0
     R2); a cancelled Confirm leaves its read for the next one (0003 C7).
8. `NewTerminalConfirmer(in *os.File, out)` keeps its behaviour and error
   texts.
   * It shares a single unexported line-confirmer type with
     `NewPromptConfirmer`.
   * Its interactivity check is `isTerminal(int(in.Fd()))`, made at every
     `Confirm`.
9. `New` already rejects typed-nil funcs through `isNil`
   (`reflect.Func`), so `ReporterFunc(nil)` is refused like any typed nil.

**Tests** (`foundations_test.go`)

* `TestExportedSentinels`:
  * a local-build `Run` without `Force` matches `ErrForceRequired`;
  * a latest release older than the running one matches `ErrLatestOlder`.
* `TestValidateRequestUsesConfiguredPolicy`: a test policy that accepts
  `v1.2.3-rc.1` lets `Request.TargetVersion = "v1.2.3-rc.1"` pass
  validation; with `NewStrictVersionPolicy` it fails.
* `TestParseSHA256SUMSExported`: for all 23 `testdata/manifest-parity`
  cases, the verdict and entries are equal to `parseSHA256SUMS`.
* `TestExactAssetName`: `linux/amd64` gives `demo-linux-amd64`;
  `windows/arm64` gives `demo-windows-arm64.exe`.
* `TestMultiReporter`:
  * order;
  * every reporter is called after an earlier one errors;
  * the errors are joined;
  * nil and typed-nil entries are dropped.
* `TestFuncAdapters`: each adapter forwards its arguments and its result.
* `TestNonInteractiveConfirmer`: the result matches
  `ErrConfirmationRequired`.
* `TestPromptConfirmer`:
  * `y`, `yes`, `n` and EOF;
  * `interactive=false`, nil `in` and nil `out`;
  * the host's next line is left unread (the Phase 0 R2 property).

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| `validateRequest` uses `NewStrictVersionPolicy()` again | `TestValidateRequestUsesConfiguredPolicy` |
| `MultiReporter` returns on the first error | `TestMultiReporter` |
| `MultiReporter` keeps nil entries | `TestMultiReporter` (panics) |
| `NewPromptConfirmer` ignores `interactive` | `TestPromptConfirmer` |
| `ParseSHA256SUMS` returns the internal map without the error check | `TestParseSHA256SUMSExported` |

### Step 3: `Checker` (`checker.go` (new), `updater.go`, `version.go`)

**API**

```go
type CheckerConfig struct {
    Source   ReleaseSource
    Versions VersionPolicy
    Assets   AssetSelector
    Limits   Limits
}
type Checker struct{ /* source, versions, assets, limits */ }
func NewChecker(cfg CheckerConfig) (*Checker, error)
func (u *Updater) Checker() *Checker

type CheckRequest struct {
    Product        string
    CurrentVersion string
    CurrentBuild   BuildKind
    TargetVersion  string
    Platform       Platform
}
type Availability struct {
    Product        string
    CurrentVersion string
    TargetVersion  string // the selected release tag
    ReleaseURL     string
    AssetName      string
    Operation      Operation
    Available      bool // Operation != OperationNone
    ForceRequired  bool // CurrentBuild == LocalBuild && Available
}
func (c *Checker) Check(ctx context.Context, req CheckRequest) (Availability, error)
```

**Behaviour**

1. `NewChecker` applies `New`'s rules to its four fields: required, not
   typed nil, and `Limits.valid()`. Error texts reuse `New`'s wording.
2. **Shared discovery.** The discovery sequence is factored out of
   `execute` into an unexported
   `discover(ctx, req Request) (Release, Selection, Operation, error)`,
   shared by `execute` and `Check`. It runs, in this order:
   1. `normalizePlatform`;
   2. `fetchRelease`;
   3. the immutable check, then the draft and prerelease check;
   4. `versions.Validate(tag)`;
   5. `Select`;
   6. `validateAssetMetadata`, for the binary and then the manifest;
   7. `classifyOperation`.
3. **What `Check` does.** It maps `CheckRequest` onto a `Request` with
   `CheckOnly: true`, validates it with rule 2 of Step 2, and calls
   `discover`.
   * It never calls an `Installer`, `Confirmer` or `Reporter`, and never
     downloads an asset body.
   * Its errors are wrapped as `Run` wraps them
     (`selfupdate: <product>: …`). An invalid product is returned
     unwrapped.
   * A latest release older than the running one is the error
     `ErrLatestOlder`, as in `Run --check`.
4. **`Run --check` is unchanged.** It resolves the target before discovery
   (MADR G3: "`Run --check` keeps its current behaviour"). Its events,
   `Result` and `ErrUpdateAvailable` stay exactly as they are.
5. **`Updater.Checker`.** It returns a `*Checker` that shares the updater's
   source, versions, assets and limits. `Checker` holds no mutable state
   and is safe for concurrent use.

**Tests** (`checker_test.go`)

* `TestCheckerNeedsNoInstaller`: `NewChecker` works with a fake source
  alone, and `Check` works with `userHomeDir` pointing nowhere valid.
* `TestCheckerAvailability`, a table of upgrade, up to date, local build
  (with `ForceRequired`), exact rollback tag, latest older
  (`ErrLatestOlder`), mutable release (`ErrMutableRelease`), prerelease,
  unsupported platform (`ErrUnsupportedPlatform`), and a digest-syntax
  failure.
* `TestCheckerAgreesWithRunCheck`: for every row above, `Run` with
  `CheckOnly` gives a `Result` whose `TargetVersion`, `ReleaseURL`,
  `AssetName` and `Operation` equal the `Availability` fields, and the
  error classes agree.
* `TestCheckerDownloadsNothing`: the fake source records no `OpenAsset`
  call.
* `TestUpdaterChecker`: `u.Checker().Check` uses the updater's source and
  policy.
* The existing `updater_contract_test.go` suite passes unchanged. That is
  the guard for rule 4.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| `Available` set to `true` unconditionally | `TestCheckerAvailability` (up-to-date row) |
| `ForceRequired` never set | `TestCheckerAvailability` (local row) |
| immutable check removed from `discover` | `TestCheckerAvailability` and `updater_contract_test.go` |
| `Check` validates with the strict policy | `TestCheckerAvailability` with a permissive-policy row |

### Step 4: cached checks (`checkcache.go` (new))

**API**

```go
type CheckRecord struct {
    Request      CheckRequest // normalised: Platform is never zero
    Availability Availability // meaningful only when CheckedAt is non-zero
    CheckedAt    time.Time
    NotBefore    time.Time // no network check before this instant
}
type CheckStore interface {
    Load(context.Context) (CheckRecord, error)
    Save(context.Context, CheckRecord) error
}
var ErrNoCheckRecord = errors.New("selfupdate: no cached check record")
var ErrCheckDeferred = fmt.Errorf("selfupdate: check deferred by an earlier rate limit: %w", ErrRateLimited)
func NewFileCheckStore(path string) (CheckStore, error)
func (c *Checker) CheckCached(ctx context.Context, req CheckRequest, store CheckStore, maxAge time.Duration) (CheckRecord, error)
```

**Behaviour** of `CheckCached`. `now` is `timeNow()`, the existing seam in
`github.go`.

1. A nil store, or `maxAge <= 0`, is an error. Validation and platform
   normalisation happen as in `Check`.
2. `rec, lerr := store.Load(ctx)`. The record *matches* when
   `lerr == nil && rec.Request == normalised req`. Any load error counts
   as no match.
3. **Back-off.** If it matches and `now.Before(rec.NotBefore)`, return
   `rec, ErrCheckDeferred` without touching the network.
4. **Fresh cache.** If it matches, `!rec.CheckedAt.IsZero()`,
   `!now.Before(rec.CheckedAt)` and `now.Sub(rec.CheckedAt) < maxAge`,
   return `rec, nil`.
5. **Otherwise, check.** Call `Check`. On success, save
   `{Request, Availability, CheckedAt: now}`. Return that record and nil;
   if the save failed, return the record and
   `selfupdate: save check record: <err>`.
6. **Rate limited.** If the check fails with a `*RateLimitError` (use
   `errors.As`):
   * `NotBefore` is the later of `Reset` and `now + RetryAfter`.
   * When both are zero, it is `now + 1m`: GitHub's guidance for a
     secondary limit with no `retry-after` is to "wait at least one
     minute"
     (<https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api>).
   * The saved record is the matched record with `NotBefore` updated, or
     `{Request, NotBefore}` when nothing matched.
   * Return the saved record and the check error.
7. **Any other failure.** Return the matched record (or the zero record)
   and the check error. Nothing is saved.
8. **Contract, stated in the doc comment.** When `CheckedAt` is non-zero,
   `Availability` is a real answer as of `CheckedAt`, whatever the error
   says.

**Behaviour** of the file store:

1. **The path.** `NewFileCheckStore` requires an absolute path. The caller
   chooses it, typically under `os.UserCacheDir()`:
   * Unix: `$XDG_CACHE_HOME`, otherwise `$HOME/.cache`;
   * macOS: `$HOME/Library/Caches`;
   * Windows: `%LocalAppData%`.

   Source: `os/file.go` in GOROOT.
2. **The document.** UTF-8 JSON, with these keys in this order:
   `schema_version` (1), `product`, `current_version`, `current_build`
   (`BuildKind.String()`), `target_version`, `platform{os,arch}`,
   `available`, `force_required`, `operation` (`Operation.String()`),
   `selected_version`, `release_url`, `asset_name`, `checked_at` and
   `not_before` (RFC 3339 with nanoseconds, UTC).
3. **`Load`.**
   * A missing file gives `ErrNoCheckRecord`.
   * The body is read with a 64 KiB cap.
   * Any other `schema_version`, an unknown enum string, or bad JSON gives
     an error wrapping `ErrNoCheckRecord`, so a corrupt cache is a miss.
4. **`Save`.**
   1. `MkdirAll(dir, 0o700)`.
   2. `os.CreateTemp(dir, "."+base+".tmp-*")`, then write, `Sync`,
      `Close` and `Chmod(0o600)`.
   3. `os.Rename(tmp, path)`, removing the temp file on any error.

   On Windows `os.Rename` replaces an existing file through
   `MoveFileEx(…, MOVEFILE_REPLACE_EXISTING)`
   (`internal/syscall/windows/syscall_windows.go`). It is not atomic
   there (`os/file.go`). A torn file decodes as a miss (rule 3), and that
   is acceptable for a cache.

**Tests** (`checkcache_test.go`). The clock comes from `setSeam(t, &timeNow, …)`.

* `TestCheckCachedFreshHit`: no source call.
* `TestCheckCachedStaleRefresh`.
* `TestCheckCachedKeyChange`: a changed `CurrentVersion` refreshes.
* `TestCheckCachedRateLimitReset`, `…RetryAfter` and `…DefaultMinute`.
* `TestCheckCachedDeferredNoNetwork`.
* `TestCheckCachedClockSkew`: a `CheckedAt` in the future refreshes.
* `TestCheckCachedSaveError`: the record is returned together with the
  error.
* `TestFileCheckStoreRoundTrip`, which also pins the byte-exact JSON.
* `TestFileCheckStoreCorruptIsMiss` and `TestFileCheckStoreSchemaMismatch`.
* `TestFileCheckStoreMode`: 0600 on Unix.
* `TestFileCheckStoreReplaces`: runs on the Windows host too.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| the `rec.Request ==` key check is dropped | `TestCheckCachedKeyChange` |
| `NotBefore` is ignored | `TestCheckCachedDeferredNoNetwork` |
| `<` becomes `>` in the age check | `TestCheckCachedFreshHit` |
| the one-minute default is removed | `TestCheckCachedRateLimitDefaultMinute` |
| `Save` renames without `Sync` and skips the temp file | `TestFileCheckStoreRoundTrip` (the temp-name assertion) |

### Step 5: events and structured output (`types.go`, `updater.go`, `download.go`, `reporter.go`, `jsonreporter.go` (new), `document.go` (new), `session.go`, `managed.go`)

**API**

```go
const (
    EventProgress   EventKind = iota + EventComplete + 1 // 10, "progress"
    EventDeclined                                        // 11, "declined"
    EventFailed                                          // 12, "failed"
    EventRolledBack                                      // 13, "rolled-back"
)
// Event, appended last:
//     Total int64 // EventProgress: advertised byte length; zero otherwise
// Config, appended last:
//     ProgressInterval time.Duration // 0: no progress events (A1); < 0: New fails
// InstallResult, appended last:
//     RolledBack bool // the installer restored the previous binary itself
func NewJSONReporter(w io.Writer) Reporter
type ResultDocument struct {
    SchemaVersion     int    `json:"schema_version"` // 1
    Product           string `json:"product"`
    CurrentVersion    string `json:"current_version"`
    TargetVersion     string `json:"target_version,omitempty"`
    ReleaseURL        string `json:"release_url,omitempty"`
    AssetName         string `json:"asset_name,omitempty"`
    Operation         string `json:"operation"` // Operation.String()
    Checked           bool   `json:"checked"`
    Applied           bool   `json:"applied"`
    Declined          bool   `json:"declined"`
    DryRun            bool   `json:"dry_run"`  // set from Step 10
    ReleaseDigest     string `json:"release_digest,omitempty"`
    InstalledDigest   string `json:"installed_digest,omitempty"`
    ServiceInstalled  bool   `json:"service_installed"`
    ServiceWasRunning bool   `json:"service_was_running"`
    PendingBackup     string `json:"pending_backup,omitempty"`
    Previous          string `json:"previous,omitempty"` // set from Step 10
}
func (r Result) Document() ResultDocument
```

**Behaviour**

1. **Progress.** When `ProgressInterval > 0`, the binary download (the
   manifest download is not tracked) emits `EventProgress` with `Product`,
   `Target`, `Asset`, `Bytes` (the count so far) and `Total`
   (`sel.Binary.Size`, always positive after `validateAssetMetadata`):
   * one event with `Bytes = 0` before the copy;
   * one event whenever `timeNow() - last >= ProgressInterval`;
   * one final event with `Bytes == Total` after a successful copy.

   It is implemented as a counting writer inside `copyLimited`'s
   `MultiWriter`, and the counting writer never fails the copy. Reporter
   errors on `EventProgress` are ignored (A1). For comparison,
   `cheggaaa/pb` refreshes every 200 ms by default and
   `schollz/progressbar` throttles at 65 ms, so a consumer choosing an
   interval has references.
2. **`EventDeclined`.** It is emitted after the confirmer returns false,
   with `Product`, `Current`, `Target` and `Asset`, and its reporter error
   is ignored (A7).
3. **`EventRolledBack`.** It is emitted after `Install` returns
   `RolledBack: true`, and its reporter error is ignored (A7). Two places
   set `RolledBack`:
   * the standalone R3 rollback in `installSession.Install`;
   * managed recovery, when its rollback succeeded.

   Step 8's post-install probe adds a third.
4. **`EventFailed`.** It is emitted exactly once, as the last event, when
   `execute` returns a non-nil error. There are three exceptions:
   * the error is `ErrUpdateAvailable`;
   * the product name is invalid;
   * the error is `Run`'s overlap `ErrConcurrentUpdate`, which is returned
     before `execute`.

   `Detail` is the first match, in this order:

   | Error | `Detail` |
   | :--- | :--- |
   | `context.Canceled` | `canceled` |
   | `context.DeadlineExceeded` | `deadline-exceeded` |
   | `ErrConfirmationRequired` | `confirmation-required` |
   | `ErrForceRequired` | `force-required` |
   | `ErrLatestOlder` | `latest-older` |
   | `ErrMutableRelease` | `mutable-release` |
   | `ErrRateLimited` | `rate-limited` |
   | `ErrUnsupportedPlatform` | `unsupported-platform` |
   | `ErrConcurrentUpdate` | `concurrent-update` |
   | `ErrManagedInstall` | `managed-install` |
   | `ErrIntegrity` | `integrity` |
   | anything else | `error` |

   Its reporter error is ignored (A7).
5. **`NewTextReporter`.**
   * It returns nil for `EventProgress` without writing (A8).
   * The new kinds print through the existing `formatEvent`, for example
     `selfupdate: failed product=demo integrity`.
   * `Total` is never printed.
6. **`NewJSONReporter(w)`.** It writes one JSON object and `\n` per event,
   in a single `Write` call, following JSON Lines (<https://jsonlines.org/>)
   and NDJSON 1.0.0 (UTF-8, one value per line, `\n` terminated):
   * The keys, in order, are `kind` (`EventKind.String()`), `product`,
     `current`, `target`, `asset`, `bytes`, `total` and `detail`.
   * Empty strings and zero numbers are omitted, except `kind`.
   * Every string goes through `sanitizeText`, and the encoder does not
     escape HTML.
   * A nil `w` gives an error at `Report`, like the text reporter.
   * It reports progress events: JSON consumers want them.
7. **`Document`.** It copies every field, with `Operation.String()`, and
   has no side effects. The enums keep their JSON encoding; `Document` is
   the string form.
8. **Existing tests.** In `updater_contract_test.go`, every failing-run
   expectation gains `EventFailed` as its final event, and every declined
   run gains `EventDeclined`. No other expectation changes (rule 3).

**Tests** (`events_test.go`, `jsonreporter_test.go`, `document_test.go`)

* `TestEventKindValues`: pins 0–13 numerically, and each `String()`.
* `TestProgressEvents`: a fake clock and a chunking source; asserts the
  first, throttled and final events and `Total`.
* `TestProgressDisabledByDefault`.
* `TestProgressIntervalNegativeRejected`.
* `TestProgressReporterErrorIgnored`.
* `TestDeclinedEvent`.
* `TestFailedEventClasses`: one row per table entry, plus
  `ErrUpdateAvailable`, which emits none.
* `TestFailedEventReporterErrorIgnored`.
* `TestRolledBackEvent`: through the R3 swap path on macOS and Linux, and
  through managed recovery.
* `TestTextReporterSkipsProgress`.
* `TestJSONReporterLines`: key order, omission, sanitising and exactly one
  `\n`.
* `TestResultDocumentJSON`: byte-exact against a fixture.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| progress is emitted when the interval is 0 | `TestProgressDisabledByDefault` |
| no final progress event | `TestProgressEvents` |
| `EventFailed` is emitted for `ErrUpdateAvailable` | `TestFailedEventClasses` |
| the integrity row moves above `canceled` | `TestFailedEventClasses` (a cancelled run whose error also wraps `ErrIntegrity`) |
| the text reporter prints progress | `TestTextReporterSkipsProgress` |
| the JSON reporter drops `kind` | `TestJSONReporterLines` |
| a progress reporter error is returned | `TestProgressReporterErrorIgnored` |
| `EventDeclined` is appended with the next kind's value | `TestEventKindValues` |

### Step 6: credentials (`credentials.go` (new), `github.go`, `types.go`)

**API**

```go
type Credential struct {
    Header string // "" means Authorization: Bearer <Value> (A6)
    Value  []byte
    Source string // a label for diagnostics; never the secret
}
type CredentialRequest struct {
    Origin      *url.URL // a copy of the API origin
    Cause       error    // non-nil on the retry after a 401
    Interactive bool     // false in Phase 1; set by Phase 2's Stream
}
type CredentialProvider interface {
    Credential(context.Context, CredentialRequest) (Credential, error)
}
type CredentialObserver interface {
    Accepted(context.Context, Credential)
}
var ErrNoCredential = errors.New("selfupdate: no credential")
func ChainCredentials(providers ...CredentialProvider) CredentialProvider
func EnvCredential(header string, names ...string) CredentialProvider
// GitHubOptions, appended last: Credentials CredentialProvider; Observer CredentialObserver
```

**Behaviour**

1. **Order.** The credential order is:
   1. `GitHubOptions.Token`;
   2. `GitHubOptions.Credentials`;
   3. `GH_TOKEN`;
   4. `GITHUB_TOKEN`.

   The environment is read at construction, as today (`resolveToken`).
   The provider is asked lazily, at the first API-origin request. The
   answer is cached per source under a mutex, so concurrent `Check` calls
   ask once.

   With `Credentials` nil, `NewGitHubSource` behaves exactly as in
   `v1.0.1`.
2. **Provider results.** `ErrNoCredential` falls through to the next link.
   Any other error fails the request that asked. When every link falls
   through, the request goes out anonymously.
3. **Validation.** A header name must be a non-empty RFC 7230 token, and a
   value must not contain CR, LF or NUL. Anything else is refused with
   `selfupdate: invalid credential from <Source>`. Error text never
   includes `Value`.
4. **Origin scoping.** The credential is attached only when the request's
   origin is the API base (`sameOrigin`); `normalizeAPIBase` already
   requires HTTPS or loopback. `checkRedirect` deletes `Authorization`, and
   the provider's header when it is non-empty, on every cross-origin hop.
5. **`Observer.Accepted`.** It is called once per source, synchronously,
   after the first 2xx response to an API-origin request that carried a
   credential.
6. **The 401 retry.** On a 401 from the API origin, with a credential from
   `Credentials` attached:
   * the provider is asked once more, with `Cause` set to the 401 error;
   * a credential whose header or value differs is sent in one retry of
     the same request;
   * anything else returns the 401 error.

   A credential from `Token` or the environment is never retried.
7. **`EnvCredential`.** It returns the first non-empty variable, read
   through `lookupEnv`, as `{Header: header, Value: v, Source: "env:<NAME>"}`;
   when none is set, `ErrNoCredential`.
8. **`ChainCredentials`.** It drops nil and typed-nil entries, and the
   first result other than `ErrNoCredential` wins.

**Tests** (`credentials_test.go`). They use two `httptest.NewTLSServer`
origins: the API origin, and an asset host it redirects to.

* `TestCredentialOrder`.
* `TestCredentialLazy`: the constructor does not call the provider.
* `TestCredentialFallThrough`.
* `TestCredentialProviderErrorFailsRequest`.
* `TestCredentialHeaderModes`.
* `TestCredentialInvalidRefused`.
* `TestCredentialStrippedCrossOrigin`: both header modes.
* `TestCredentialAcceptedOnce`: not after a 401.
* `TestCredential401RetryOnce`.
* `TestCredential401SameValueNoRetry`.
* `TestEnvCredentialSource`.
* `TestCredentialConcurrent`: eight concurrent `Check` calls under
  `-race` ask the provider once.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| the custom header is not stripped cross-origin | `TestCredentialStrippedCrossOrigin` |
| `Accepted` is called before the response | `TestCredentialAcceptedOnce` |
| the retry loops twice | `TestCredential401RetryOnce` |
| the provider is asked before `Token` | `TestCredentialOrder` |
| the per-source cache is dropped | `TestCredentialConcurrent` |

### Step 7: integrity seams (`manifestverify.go` (new), `imageverify.go` (new), `updater.go`, `types.go`)

**API**

```go
type ManifestVerification struct {
    Product   string
    Release   Release
    Selection Selection
    Manifest  []byte // a copy of the SHA256SUMS bytes
    OpenAsset func(ctx context.Context, name string, limit int64) (io.ReadCloser, error)
}
type ManifestVerifier interface {
    VerifyManifest(context.Context, ManifestVerification) error
}
type ManifestVerifierFunc func(context.Context, ManifestVerification) error
func (f ManifestVerifierFunc) VerifyManifest(ctx context.Context, v ManifestVerification) error
// Config, appended last:       ManifestVerifiers []ManifestVerifier
// Verification, appended last: OpenAsset func(ctx context.Context, name string, limit int64) (io.ReadCloser, error)
func NewImageVerifier(p Platform) (Verifier, error)
```

**Behaviour**

1. **When manifest verifiers run.** In order, after the manifest has been
   downloaded and parsed and the selected entry found, and before
   `EventDownloadingBinary` and `CreateStaging`.
   * A failure returns
     `selfupdate: manifest verification failed: <errors.Join(ErrIntegrity, err)>`,
     so `errors.Is(err, ErrIntegrity)` always holds.
   * No staging file exists yet, so the target is untouched.
   * `New` rejects a nil entry: `selfupdate: manifest verifier %d is nil`.
2. **`OpenAsset(ctx, name, limit)`.**
   * **Finding the asset.** It needs exactly one asset in `rel.Assets` with
     that exact name; otherwise
     `selfupdate: release <tag> has no asset "<name>"` or `… duplicate asset …`.
   * **The limit.** `limit` must be positive, and is capped at
     `Limits.Executable`. `validateAssetMetadata(asset, limit)` applies.
   * **The body.** It opens through `Source.OpenAsset`, wrapped in a reader
     that fails with `ErrIntegrity` on a short or long body, and checks the
     GitHub digest, when present, at EOF.
   * **Where it is available.** The same function is set on
     `Verification.OpenAsset` for binary verifiers.
3. **`NewImageVerifier(p)`.** A zero `p` means the runtime platform.

   | OS | Format | Check |
   | :--- | :--- | :--- |
   | `linux`, `freebsd`, `netbsd`, `openbsd`, `dragonfly` | ELF | class and `Machine` |
   | `darwin` | Mach-O | `Cpu`, thin or fat |
   | `windows` | PE | `Machine` |

   * The architectures are `amd64`, `arm64`, `386` and `arm`. Any other
     pair is a construction error wrapping `ErrUnsupportedPlatform`.
   * The machine constants, from `debug/elf`, `debug/macho` and `debug/pe`
     in GOROOT, are:
     * amd64: `EM_X86_64`, `CpuAmd64`, `IMAGE_FILE_MACHINE_AMD64`;
     * arm64: `EM_AARCH64`, `CpuArm64`, `IMAGE_FILE_MACHINE_ARM64`;
     * 386: `EM_386`, `Cpu386`, `IMAGE_FILE_MACHINE_I386`;
     * arm: `EM_ARM`, `CpuArm`, `IMAGE_FILE_MACHINE_ARMNT`.
   * **ELF.** ELF `OSABI` is not checked. Go's linker writes
     `ELFOSABI_NONE` for linux (`cmd/link/internal/ld/elf.go`), so it
     cannot tell the OS apart.
   * **Mach-O.** `macho.NewFatFile` is tried first; `macho.ErrNotFat` falls
     back to `macho.NewFile`. A fat file passes when any arch matches.
   * **Reading the file.** `Verify` needs `Open()` to return an
     `io.ReaderAt`; the staged `*os.File` does. Otherwise it fails with
     `selfupdate: image verifier needs a seekable staged file`.
   * **Failure.** A mismatch is
     `selfupdate: staged binary is not a <os>/<arch> executable: <ErrIntegrity>`.

**Tests** (`manifestverify_test.go`, `imageverify_test.go`)

* `TestManifestVerifierOrder`: runs before any binary `OpenAsset`.
* `TestManifestVerifierFailureIntegrity`: the target is byte-identical and
  no staging is left.
* `TestManifestVerifierReadsSibling`.
* `TestOpenAssetEnforcesSize`: short, long and over-limit bodies.
* `TestOpenAssetUnknownName`.
* `TestNewNilManifestVerifier`.
* `TestImageVerifier`. Its fixtures are built in the test:
  * `go build` of a two-line `main` into `t.TempDir()` for
    `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` and
    `windows/amd64` (with `CGO_ENABLED=0`, and the build cache keeps it
    fast);
  * a fat Mach-O written by the test from the two darwin builds, in the
    fat layout of `debug/macho/fat.go`;
  * every match and mismatch pair, plus a non-executable body.
* `TestImageVerifierUnsupportedPlatform`.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| manifest verifiers moved after the binary download | `TestManifestVerifierOrder` |
| the size check is dropped from `OpenAsset` | `TestOpenAssetEnforcesSize` |
| the image verifier ignores `Machine` | `TestImageVerifier` (mismatch rows) |
| the fat branch is removed | `TestImageVerifier` (fat row) |
| `ErrIntegrity` is not joined | `TestManifestVerifierFailureIntegrity` |

### Step 8: probes and runnable staging (`probe.go` (new), `session.go`, `updater.go`, `standalone.go`, `types.go`)

**API**

```go
type ProbePhase uint8
const (
    ProbeStaged    ProbePhase = iota + 1 // "staged"
    ProbeInstalled                       // "installed"
)
func (p ProbePhase) String() string
type ProbeRequest struct {
    Product       string
    TargetVersion string
    Path          string
    Phase         ProbePhase
}
type Prober interface {
    Probe(context.Context, ProbeRequest) error
}
type ProberFunc func(context.Context, ProbeRequest) error
func (f ProberFunc) Probe(ctx context.Context, r ProbeRequest) error
func NewVersionProber(args []string, want func(tag string) string, timeout time.Duration) (Prober, error)
// Config, appended last:          Probes []Prober
// InstallOptions, appended last:  PostInstall Prober
// InstallRequest, appended last:  TargetVersion string
```

**Behaviour**

1. **Runnable staging.** On Windows, `CreateStaging` uses the pattern
   `"."+base+".selfupdate-*.exe"`; elsewhere the pattern is unchanged.
   * `os.CreateTemp` replaces the last `*` (`os/tempfile.go`).
   * `exec` accepts an absolute path whose extension is in `PATHEXT` as it
     is (`os/exec/lp_windows.go:85-91`).
   * `CreateProcess` assumes no default extension for
     `lpApplicationName` (Microsoft `CreateProcessW` documentation).

   ~~Without `.exe`, whether the probe can run at all is unverified.~~
   *(Deviation D3, 2026-09-30: verified that it runs without `.exe`; the
   suffix is kept as a convention. See the execution record.)*
2. **Staged probes.** `Config.Probes` run in order after verification and
   the optional transform and its re-hash, and before `EventInstalling`.
   * Before the first probe the staged file is chmodded to `0o700`.
     `CreateTemp` creates it `0o600`; the chmod is harmless on Windows.
   * A failure returns `selfupdate: staged binary failed a probe: <err>`,
     with the target untouched.
   * `New` rejects a nil entry.
3. **The post-install probe.** `InstallOptions.PostInstall` runs inside the
   standalone session's replacement, after `replaceTarget` and `checkDir`,
   and before commit. It runs for both `Install` and the managed `Apply`
   (Step 9), with `Phase: ProbeInstalled` and `Path: target.Path`.
   * On failure, `rollbackReplacement` runs. The result is
     `Applied: false` and `RolledBack: true`, and the error is
     `selfupdate: installed binary failed its probe: <err>`.
   * If the rollback fails too, the result carries `Backup`, so Phase 0
     R1 reports it, and the rollback error is joined.
4. **`InstallRequest.TargetVersion`.** The coordinator sets it to
   `rel.Tag`.
5. **`NewVersionProber(args, want, timeout)`.**
   * **Construction.** An empty `args` or a non-positive `timeout` is an
     error; a nil `want` means the identity function.
   * **Running.** `Probe` runs
     `exec.CommandContext(ctx with timeout, req.Path, args...)` with no
     stdin, stdout captured up to 64 KiB (the rest discarded), stderr
     discarded, the environment inherited, and `WaitDelay = time.Second`.
   * **Success.** Exit 0, and stdout containing `want(req.TargetVersion)`.
   * **Failure.** It quotes at most 200 bytes of the first stdout line,
     sanitised.

**Tests** (`probe_test.go`). The package already has a `TestMain`, in
`replace_native_test.go`, with a helper-process branch for
`SELFUPDATE_NATIVE_HELPER`.

* It gains a second branch: when `SELFUPDATE_TEST_PRINT_VERSION` is set,
  the test binary prints that value and exits 0.
* A second `TestMain` would not compile, so none is added.
* The release bodies are the test binary's own bytes, and their digests
  are computed at run time.

* `TestProbesRunBeforeInstall`: the order, and the target is untouched on
  failure.
* `TestStagedProbeRunsRealBinary`: macOS, Linux and the Windows host,
  ~~where it proves the `.exe` rule~~ *(D3: it does not; the suffix is
  asserted by `TestProbeRequestFields`)*.
* `TestPostInstallRollback`.
* `TestPostInstallRollbackFailureReportsBackup`.
* `TestVersionProberMatch`, `…Mismatch`, `…NonZeroExit` and `…Timeout`.
* `TestProbeRequestFields`.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| no chmod before probes | `TestStagedProbeRunsRealBinary` (Unix) |
| no `.exe` in the Windows pattern | ~~`TestStagedProbeRunsRealBinary` (Windows host)~~ `TestProbeRequestFields` (Windows host), deviation D3 |
| `PostInstall` ignored | `TestPostInstallRollback` |
| probes run after `Install` | `TestProbesRunBeforeInstall` |
| `strings.Contains` becomes `==` | `TestVersionProberMatch` (the helper prints `demo v1.1.0`) |

### Step 9: first-class custom installers (`session.go`, `managed.go`, `updater.go`, `types.go`)

**API**

```go
type StagingOwner interface {
    Owns(path string) bool
}
type AppliedReplacement struct {
    Target string
    Backup string // "" when no backup exists
    State  any    // installer-private; returned to Commit or Rollback unchanged
}
type TwoPhaseSession interface {
    InstallSession
    StagingOwner
    Apply(context.Context, InstallRequest) (AppliedReplacement, error)
    Commit(context.Context, AppliedReplacement) (InstallResult, error)
    Rollback(context.Context, AppliedReplacement) error
}
func NewManagedInstallerFor(inner Installer, life Lifecycle, rec Reconciler) (*ManagedInstaller, error)
```

**Behaviour**

1. **`installSession`.** It implements `StagingOwner` and
   `TwoPhaseSession`:
   * `Apply` wraps `apply` (and the Step 8 post-install probe) into
     `AppliedReplacement{Target, Backup, State: applyResult}`.
   * `Commit` and `Rollback` type-assert `State`, failing with
     `selfupdate: replacement was not applied by this session` on a
     foreign value.

   `managedSession` implements `StagingOwner` by delegating.
2. **`sessOwns`** returns `StagingOwner.Owns` when the session has it, and
   `false` otherwise. A custom session paired with a `Transformer` must
   therefore implement `StagingOwner`. This is the MADR's stated
   behaviour change, and it goes in the release notes.
3. **`ManagedInstaller`.** Its unexported `inner` field becomes an
   `Installer`, and `NewManagedInstaller` calls `NewManagedInstallerFor`.
   * `Begin` requires the inner session to be a `TwoPhaseSession`.
     Otherwise it closes that session and fails with
     `selfupdate: managed installer requires a two-phase session`.
   * `managedSession` drives the session through `Apply`, `Commit` and
     `Rollback`.
   * The R1 kept-backup check uses `AppliedReplacement.Backup` and
     `os.Lstat`.
4. **Test sessions.** `logSession` and `captureSession` gain `Owns`, the
   only change to existing tests (rule 3).

**Tests** (`twophase_test.go`)

* `TestManagedDrivesCustomTwoPhaseSession`: the call order is
  `Stop`, `Apply`, `Reconcile`, `Start`, `WaitHealthy`, `Commit`.
* `TestManagedRollsBackCustomSession`.
* `TestManagedRejectsSingleStepSession`: the session is closed exactly
  once.
* `TestSessOwnsFailsClosed`.
* `TestCommitRejectsForeignState`.
* The existing `managed_test.go` passes unchanged.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| `sessOwns` returns true for an unknown session | `TestSessOwnsFailsClosed` |
| `Begin` accepts a plain `InstallSession` | `TestManagedRejectsSingleStepSession` |
| `Commit` skips the `State` assertion | `TestCommitRejectsForeignState` |

### Step 10: lifecycle (`types.go`, `version.go`, `updater.go`, `replace_unix.go`, `replace_windows.go`, `session.go`, `standalone.go`)

**API**

```go
// Request, appended last:         DryRun bool
// Result, appended last:          DryRun bool; Previous string
// InstallOptions, appended last:  KeepPrevious bool
// InstallResult, appended last:   Previous string
func (s *StandaloneInstaller) CleanupPending(ctx context.Context) error
```

**Behaviour**

1. **`DryRun`.**
   * **Validation.** `CheckOnly && DryRun` fails with
     `selfupdate: --check and --dry-run are contradictory`.
   * **Flow.** It skips confirmation, then runs `Begin`, the manifest,
     manifest verifiers, staging and the download, `verifyIntegrity`,
     verifiers, the transform and the Step 8 probes. It never calls
     `Install` and never emits `EventInstalling`.
   * **End of run.** It closes the session, which removes the staging
     file, and emits `EventComplete` with
     `Detail: "dry run: verified, nothing installed"`.
   * **Result.** `DryRun: true`, `Applied: false`, and both digests.
     `OperationNone` returns early, as today, with `DryRun` echoed.
2. **`KeepPrevious`.** At commit, standalone and managed, the backup is
   renamed rather than removed. The rename goes through `replacePath`, and
   replaces any existing file:
   * the new name is `Dir/"."+base+".previous"`;
   * the directory is then synced;
   * `InstallResult.Previous` and `Result.Previous` are set to that path.

   The name matches neither `backupPrefix` nor the lock or receipt names,
   so `validateReceiptBackup` never accepts it.
3. **`KeepPrevious` on Windows.** The backup is a hard link to the running
   image. Renaming the running image's primary name is documented to work
   (golang/go#21997). Renaming a *hard link* to it is **unverified**.
   * The step therefore starts with a Windows-host experiment: rename a
     hard link to a running test binary with `MoveFileEx`.
   * If it succeeds, the rule above applies on Windows too.
   * If it fails, the step stops for a deviation. The fallback on the
     table is to keep the backup under the cleanup receipt, as today, with
     `Previous` empty.
4. **`CleanupPending(ctx)`.** It runs `resolveTarget`, then `beginSession`
   (which processes the receipt), then `Close`, and returns any error.
   * When another update holds the lock, the error is
     `ErrConcurrentUpdate`; the doc comment says this is benign and to
     retry later.
   * It is useful on every OS: elsewhere it removes a stale receipt.

**Tests** (`lifecycle_test.go`)

* `TestDryRunLeavesTargetUntouched`: bytes, no staging left, no
  `EventInstalling`, probes ran, `Result.DryRun`.
* `TestDryRunNeedsNoConfirmation`.
* `TestCheckAndDryRunRejected`.
* `TestKeepPrevious`: old bytes at `.demo.previous`, and an older
  `.previous` replaced; on macOS, Linux and the Windows host.
* `TestKeepPreviousManaged`.
* `TestCleanupPendingProcessesReceipt`: a Windows receipt, or a stale
  receipt elsewhere.
* `TestCleanupPendingLocked`: `ErrConcurrentUpdate`.

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| dry run calls `Install` | `TestDryRunLeavesTargetUntouched` |
| `KeepPrevious` removes the backup | `TestKeepPrevious` |
| `CleanupPending` does not `Close` | `TestCleanupPendingLocked`, run twice in a row |

### Step 11: `selfupdatetest` and golden output (H3, H5) (`selfupdate/selfupdatetest/` (new), `testdata/golden/` (new))

**API** (package `github.com/maccavelli/go-core-lib/selfupdate/selfupdatetest`,
standard library only)

```go
type AssetSpec struct {
    Name       string
    Body       []byte
    State      string // "" means selfupdate.AssetStateUploaded
    OmitDigest bool   // leave Asset.Digest empty
}
type ReleaseSpec struct {
    Tag                          string
    Immutable, Draft, Prerelease bool
    Assets                       []AssetSpec
}
func NewRelease(product, tag string, platforms []selfupdate.Platform, body func(selfupdate.Platform) []byte) ReleaseSpec
type FakeSource struct{ /* … */ }
func NewFakeSource(latest string, releases ...ReleaseSpec) *FakeSource
func (s *FakeSource) Calls() []string
// FakeSource implements selfupdate.ReleaseSource.
type RecordingReporter struct{ /* … */ }
func (r *RecordingReporter) Report(context.Context, selfupdate.Event) error
func (r *RecordingReporter) Events() []selfupdate.Event
func (r *RecordingReporter) Kinds() []selfupdate.EventKind
type ScriptedConfirmer struct {
    Answers []bool
    Err     error
}
func (c *ScriptedConfirmer) Confirm(context.Context, selfupdate.Prompt) (bool, error)
func (c *ScriptedConfirmer) Prompts() []selfupdate.Prompt
type GitHubServer struct {
    APIBase *url.URL
    Client  *http.Client // trusts both test servers' certificates
    // unexported: releases, request log, knobs
}
func NewGitHubServer(t testing.TB, owner, repo string, releases ...ReleaseSpec) *GitHubServer
func (g *GitHubServer) RateLimit(status int, header http.Header)
func (g *GitHubServer) TruncateAssets(bool)
func (g *GitHubServer) Requests() []RecordedRequest
type RecordedRequest struct {
    Host, Path    string
    Authorization bool
}
```

**Behaviour**

1. **`NewRelease`.** It builds one binary per platform, named with
   `ExactAssetName`, and a matching `SHA256SUMS`. Digests are computed
   from the bodies.
2. **The two servers.** `GitHubServer` serves `releases/latest`,
   `releases/tags/{tag}` and `releases/assets/{id}` from one
   `httptest.NewTLSServer`. Each asset request is a 302 to a second TLS
   server, which serves the bytes. The knobs cover rate limits (403 or 429
   with headers) and truncated bodies.
3. **Log and cleanup.** The request log records the host, the path and
   whether `Authorization` was present. `t.Cleanup` closes both servers.
4. **Golden files.** `testdata/golden/{text,jsonl}-{upgrade,check,local,failed-integrity,declined,dry-run}.golden`
   are produced by a full `Run` through `FakeSource`, compared byte for
   byte, and rewritten only with `go test -run TestGolden -update`.

**Tests**

* `selfupdatetest/selfupdatetest_test.go`: the fakes' own contracts.
* `selfupdate/e2e_github_test.go` (package `selfupdate_test`):
  `Run` with `NewGitHubSource` against `GitHubServer`, a
  `NewStandaloneInstaller` whose target is a temporary file (not the
  running binary; that is H4), and a token. It asserts:
  * one cross-origin hop per asset;
  * no `Authorization` on the asset host;
  * the target's new bytes;
  * `ExitCode` 0.
* `TestGolden` (package `selfupdate_test`).

**Mutation proofs**

| Mutation | Must fail |
| :--- | :--- |
| the server serves assets directly (no 302) | `e2e_github_test.go`, the cross-origin assertion |
| one golden byte flipped | `TestGolden` |
| `NewRelease` writes a wrong digest | `selfupdatetest_test.go` |

### Step 12: API gate and CI (H7) (`scripts/check-api-compat.sh` (new), its `_test.sh`, `Makefile`, `.github/workflows/ci.yml`)

**Behaviour**

1. **`scripts/check-api-compat.sh [BASE]`.**
   1. `BASE` defaults to `git describe --tags --abbrev=0 --match 'v1.*'`.
   2. It adds a detached `git worktree` of `BASE` under a `mktemp -d`
      directory, and removes it on exit (`trap`).
   3. It writes the base's export data with
      `go run golang.org/x/exp/cmd/apidiff@v0.0.0-20260908205506-85c1c2202aba -m -w <tmp>/base.api github.com/maccavelli/go-core-lib`.
   4. From the working tree it runs the same tool with
      `-m -incompatible <tmp>/base.api github.com/maccavelli/go-core-lib`.
   5. It exits 1 when that output is non-empty. `apidiff` itself exits 0
      even when it finds incompatible changes (`cmd/apidiff/main.go`
      only writes the report), so the gate is the output.
2. **The gate's own test.** `check-api-compat_test.sh`, working on a
   scratch clone, removes an exported identifier and asserts that the gate
   exits 1, then asserts that the unchanged tree exits 0.
3. **Make.** A `Makefile` target, `apicheck`, runs the script.
4. **CI (Linux).**
   * `actions/checkout` gains `fetch-depth: 0` on the Linux leg only, so
     the tags are present.
   * A step "API compatibility" runs `make apicheck`. On a tag push the
     base can be the pushed tag itself. That case compares the tree with
     itself and passes, which is harmless.
5. **CI (H7).**
   * `go test -race -count=1 ./...` also runs on `macos-15`; the runner
     has a C toolchain.
   * A Linux step runs `go test -shuffle=on -count=2 ./...`.
   * A Linux step runs `CGO_ENABLED=0 go vet ./...` for `freebsd/amd64`,
     `openbsd/amd64` and `linux/386`.
6. **Actionlint and shellcheck** stay clean.

**Proofs**

* The gate's test shows it fails on the planted removal.
* The first push after approval shows the new CI steps run.

### Step 13: documentation and close-out (`doc.go`, `example_test.go`, `docs/architecture.md`, `docs/guides/extending-selfupdate.md` (new), `docs/README.md`, this PLAN)

1. **`doc.go`** describes the query API, the events (progress is opt-in),
   JSON output, credentials, manifest verifiers, probes, dry run,
   keep-previous and custom installers. It keeps the paragraph saying no
   publisher signature is verified by default, and names the
   `ManifestVerifier` hook.
2. **Runnable examples** (`// Output:` offline, through
   `selfupdatetest.FakeSource`):
   * `ExampleNewChecker`;
   * `ExampleChecker_CheckCached`;
   * `ExampleNewJSONReporter`;
   * `ExampleMultiReporter`;
   * `ExampleChainCredentials`;
   * `ExampleNewManagedInstallerFor`;
   * `ExampleResult_Document`.
3. **`docs/guides/extending-selfupdate.md`** has one section per seam,
   each with a runnable pointer. `docs/README.md` gains "I want to…"
   rows: "show an update banner", "read JSON output", "plug in a
   credential", "verify a signature later", "probe the new binary".
4. **`docs/architecture.md`** records the new files, the `selfupdatetest`
   package and the gate.
5. **Release notes for `v1.1.0`**, written into the execution record:
   * **Additions.**
   * **Behaviour changes:**
     * `NewTextReporter` prints `declined` and `failed` lines;
     * `sessOwns` fails closed;
     * Windows staging names end in `.exe`;
     * `Config.Versions` is honoured in request validation.
   * **Migration notes.**
6. The PLAN is marked `complete` only after CI is green on the pushed
   tree.

## Verification

* **Every step.**
  * Its mutations are all killed.
  * `make pre-add-check` passes, with `make lint` for three targets.
  * `go test -race -count=3 ./...` and `go test -shuffle=on -count=2 ./...`
    pass locally.
  * From Step 12 on, `make apicheck` against `v1.0.1` prints nothing.
  * Every step that changes Go code (2–11) also passes the Windows test
    host.
* **Before release.**
  * `go mod tidy -diff` is clean; `go.mod` is unchanged from `v1.0.1`.
  * `make vuln` passes.
  * `apidiff -m` (all changes) against `v1.0.1` is recorded in the
    execution record. It lists only additions.
* **After the owner's push.** CI is green on `ubuntu-24.04`, `macos-15`
  and `windows-2025`, including the new steps.

## Rollout and Rollback

* **Rollback.** Each step is one commit, so a step can be reverted alone.
  The later steps depend on the earlier ones in the order written.
* **Tagging.** `v1.1.0` is tagged only on the owner's ask, after Step 13.
* **Consumers.** They pick it up when they migrate (MADR owner decision
  5). None needs a code change: every behaviour change is opt-in or
  cosmetic, except fail-closed `sessOwns` for custom sessions with a
  `Transformer`, which no known consumer has.
* **Stopping partway.** If Phase 1 has to stop, the steps already on
  `main` remain additive and compatible (the `apidiff` gate is the check),
  and `v1.0.x` stays available.

## Execution Record

### Step 1: records (2026-09-30)

* The owner approved this PLAN ("approved, proceed"), together with
  amendments A1–A8.
* Before approving, the owner asked whether the download status is a
  progress event. It is not: `EventDownloadingBinary` is a one-off
  milestone carrying the advertised size, and `EventProgress` (Step 5)
  is new.
* The MADR's §3 gained the amendments block and inline markers.
* This PLAN was set to `in-progress`, and the index updated.
* [0004-PLAN-v1-0-1-defect-release.md](0004-PLAN-v1-0-1-defect-release.md)
  gained a release note for the `v1.0.1` tag.

### Step 2: foundations (2026-09-30)

**What changed.**

* **Sentinels.** `ErrForceRequired` and `ErrLatestOlder` are exported, with
  byte-identical texts.
* **Version policy.** `validateRequest(req, versions)` validates with
  `Config.Versions` (G2).
* **Asset helpers.** `AssetStateUploaded` is added, and `github.go` uses
  it.
* **Adapters.** The new `adapters.go` has the four function adapters,
  `DiscardReporter`, `MultiReporter` and `NonInteractiveConfirmer`.
* **Confirmers.** `NewPromptConfirmer` shares the terminal confirmer's
  type through an `allow` check.
  * A typed-nil `*os.File` passed as `in` is treated as nil.
  * `NewTerminalConfirmer`'s texts and its per-`Confirm` TTY check are
    unchanged.
* **`ParseSHA256SUMS` and `ExactAssetName` are the implementations
  themselves.** The PLAN had them wrapping the unexported functions, but
  revive's `confusing-naming` refuses two package functions whose names
  differ only in case.
  * The unexported names were removed, and every internal use renamed:
    16 for the parser, 6 for the asset name.
  * `TestParseSHA256SUMSExported` therefore checks each parity fixture's
    own `expect` verdict, rather than comparing the function with itself.
* **Existing tests.** They changed mechanically, with no assertion changed
  (rule 3):
  * `version_test.go`: the two sentinels renamed, and 10 `validateRequest`
    calls given `NewStrictVersionPolicy()`;
  * `checksums_test.go`, `manifest_parity_test.go`, `assets_test.go`,
    `updater_test.go` and `fuzz_test.go`: the two renames.

**Tests** (`foundations_test.go`): `TestExportedSentinels`,
`TestValidateRequestUsesConfiguredPolicy`, `TestParseSHA256SUMSExported`,
`TestExactAssetName`, `TestMultiReporter`, `TestFuncAdapters`,
`TestNonInteractiveConfirmer` and `TestPromptConfirmer`, which includes
the typed-nil `*os.File` case and the host-input property.

**Mutation proofs.** Six mutations, none survived:

| Mutation | Killed by |
| :--- | :--- |
| strict policy restored | `configured policy accepts the tag, yet: … "v1.2.3-rc.1" is not a strict stable tag` |
| `MultiReporter` returns on the first error | `order = [a]` |
| `MultiReporter` keeps nil entries | a nil-pointer panic |
| `NewPromptConfirmer` ignores `interactive` | `interactive=false: <nil>` |
| `ParseSHA256SUMS` ignores a bad line | `byte-order-mark: accepted=true (<nil>), expect "reject"` |
| `ErrLatestOlder` no longer wrapped | `latest older than running: … older` |

* **The runner now rejects mutations that do not compile.** Two first
  drafts failed only at the build stage: one used an unimported `errors`,
  and one targeted text the rename had removed. Such a mutation now
  reports `NO-BUILD` and counts as a failed proof. Both were rewritten.

**Checks.**

* `make pre-add-check` passed on the 16 files, after lint fixes:
  `errors.Is` in `TestFuncAdapters`, and the rename above.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`.

### Step 3: `Checker` (2026-09-30)

**What changed.**

* `checker.go` adds `CheckerConfig`, `Checker`, `CheckRequest`,
  `Availability`, `NewChecker`, `Updater.Checker` and `Checker.Check`.
* The discovery sequence moved from `execute` into `Checker.discover`:
  fetch, then immutable, then draft and prerelease, then the tag, `Select`,
  both assets' metadata, and `classifyOperation`. `fetchRelease` moved with
  it.
* `execute` calls `u.Checker().discover` after its `EventFetchingRelease`,
  so `Run --check` still resolves the target first, and its events and
  results are unchanged. The existing suite passed unchanged after the
  refactor; that is rule 4's guard.

**Tests** (`checker_test.go`).

* `TestCheckerAvailability` has 10 rows: upgrade, up to date, local build,
  exact rollback, latest older, mutable, prerelease, unsupported platform,
  bad digest syntax, and a configured permissive policy.
* `TestCheckerAgreesWithRunCheck` covers the same 10 rows.
* `TestCheckerNeedsNoInstaller`: `userHomeDir` and `osExecutable` fail,
  and the check still succeeds. It also covers nil and typed-nil sources.
* `TestCheckerDownloadsNothing` and `TestUpdaterChecker`.

**Mutation proofs.** Five mutations, none survived:

| Mutation | Killed by |
| :--- | :--- |
| `Available` set unconditionally | the up-to-date row: `… Available:true …, want … available=false` |
| `ForceRequired` never set | the local row: `… ForceRequired:false}, want … force=true` |
| immutable check removed | `err = <nil>, want selfupdate: release is not immutable` |
| `Check` validates with the strict policy | `… "v1.2.3-rc.1" is not a strict stable tag` |
| `AssetName` taken from the manifest | `Check {… AssetName:SHA256SUMS …} disagrees with Run {… AssetName:demo-darwin-arm64 …}` |

**Checks.**

* `make pre-add-check` passed on the three files.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`, including the five new tests.

### Step 4: cached checks (2026-09-30)

**What changed.**

* `checkcache.go` adds `CheckRecord`, `CheckStore`, `ErrNoCheckRecord`,
  `ErrCheckDeferred` (which wraps `ErrRateLimited`), `Checker.CheckCached`
  and `NewFileCheckStore`. They follow the PLAN's rules: key match,
  back-off, freshness, and `NotBefore` from `Reset` and `RetryAfter`, with
  a one-minute default.
* `Check` was split into `prepare`, which validates and normalizes, and
  `checkPrepared`, so `CheckCached` shares them.
* Load errors wrap both the cause and `ErrNoCheckRecord` with two `%w`
  verbs (Go 1.20). errorlint refused the first draft's `%v`.

**Tests** (`checkcache_test.go`). Ten `CheckCached` tests cover:

* fresh hit, stale refresh, key change and clock skew;
* rate limit with Reset, with RetryAfter (including "the later of the
  two"), and with the default minute;
* deferred with no network access;
* a save error;
* bad arguments.

Six file-store tests: a byte-exact round trip with no temporary files left
behind, corrupt as a miss, schema mismatch, mode 0600, replace, and
absolute-path validation.

**Mutation proofs.** Eight mutations, none survived:

| Mutation | Killed by |
| :--- | :--- |
| the key check is dropped | `a record for another running version was reused` |
| `NotBefore` is ignored | `TestCheckCachedDeferredNoNetwork`: `… calls=[Latest]` |
| the age comparison is inverted | `TestCheckCachedFreshHit`: `… calls=[Latest]` |
| the one-minute default is removed | `NotBefore = 2026-09-30 12:00:00 +0000 UTC, want one minute ahead` |
| the saved file is 0644 | `mode = -rw-r--r--, want 0600` |
| the schema is not checked | `a schema-2 record loaded: <nil>` |
| malformed JSON is not a miss | `body "not json": invalid character 'o' …` |
| a fresh answer is not saved | `saves = []` |

**Deviation D1 (2026-09-30): Step 4 mutation list.**

* **Found.** The PLAN listed the mutation "`Save` renames without `Sync`
  and skips the temp file", to be killed by a temp-name assertion. A
  missing `fsync` leaves no observable trace in a test, so that mutation
  cannot be killed.
* **Decision.** It was replaced by two observable mutations, "the saved
  file is 0644" and "malformed JSON is not a miss", and one more was added,
  "a fresh answer is not saved". The temp file is still asserted: the round
  trip checks that no temporary file is left.
* **MADR.** No MADR change: nothing it states is affected.

**Checks.**

* `make pre-add-check` passed on the three files.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`, including `TestFileCheckStoreReplaces`
  (rename over an existing file).

### Step 5: events and structured output (2026-09-30)

**What changed.**

* **`types.go`.** `EventProgress` (10), `EventDeclined` (11), `EventFailed`
  (12) and `EventRolledBack` (13) are appended in a second const block,
  with their names. `Event.Total`, `Config.ProgressInterval` and
  `InstallResult.RolledBack` are appended last.
* **`updater.go`.**
  * `New` refuses a negative interval.
  * `execute`'s deferred `EventFailed` carries the class from
    `failureClasses`.
  * `EventDeclined` is reported on a decline, and `EventRolledBack` when
    `Install` reports it.
  * Progress runs through `downloadProgress`: a first event at 0, events
    throttled by `timeNow`, and a final event at `Total`. A
    `progressWriter` wraps only the staging file, so the manifest download
    and `copyLimited`/`downloadAsset` are unchanged.
* **`session.go`** (the R3 rollback) and **`managed.go`** (a recovery
  rollback that succeeded) set `RolledBack`.
* **Reporters.** `NewTextReporter` skips `EventProgress` (A8). The new
  `jsonreporter.go` and `document.go` add `NewJSONReporter` and
  `Result.Document`.
* **Two refinements the PLAN did not spell out, both within A7.**
  * Outcome events are reported on `context.WithoutCancel`, so a
    cancelled run still delivers its `EventFailed`.
  * Advisory reporter errors go to `advisory(error)`, a documented sink.
    The repository's errcheck sets `check-blank: true`, which refuses
    `_ =`, and the setting stays.
* **Kept private.** The first draft exported a `ResultDocumentSchema`
  constant that the PLAN's API does not list. It is now unexported, so
  the surface is exactly as approved.
* **Existing tests.** None changed. `calls()` filters events out, and no
  existing test asserted a failure-path event sequence, so the appended
  events broke nothing (rule 3).

**Tests** (`events_test.go`).

* `TestEventKindValues` pins 0–13.
* `TestProgressEvents` uses a one-byte source and a stepping clock. It
  expects 10 events for 9 bytes when every write passes the interval, and
  exactly the first and final events when throttled. It also checks the
  events sit between downloading-binary and verified.
* `TestProgressDisabledByDefault`, `TestProgressIntervalNegativeRejected`
  and `TestProgressReporterErrorIgnored`.
* `TestDeclinedEvent`, where the reporter errors on declined.
* `TestFailureClass` covers every row, and cancellation wrapping
  integrity.
* `TestFailedEventClasses` covers a mutable release, check mode finding
  an update (no event), and an unsafe product (no events).
* `TestFailedEventReporterErrorIgnored` and `TestRolledBackEvent`
  (rolled-back, then failed).
* `TestInstallersReportRolledBack` covers managed recovery and the
  standalone swap; the swap is refused and logged on Windows.
* `TestTextReporterSkipsProgress`, `TestJSONReporterLines` (byte-exact, one
  `Write` per event) and `TestResultDocumentJSON` (byte-exact).

**Mutation proofs.** Fifteen mutations, none survived:

| Mutation | Killed by |
| :--- | :--- |
| progress with a zero interval | `progress reported with a zero interval: […]` |
| no final progress | `9 progress events for 9 bytes …; want 10` |
| progress not throttled | `throttled progress = [ten events], want only the first and final events` |
| `EventFailed` on update-available | `check mode reported failed: [… failed]` |
| integrity above canceled | `TestFailureClass`: `context canceled` classed wrongly |
| text reporter prints progress | `text = "selfupdate: progress product=demo bytes=5\n…"` |
| JSON drops the kind value | `TestJSONReporterLines` mismatch |
| a progress reporter error surfaces | panic: `report failed` |
| declined renumbered (a reserved constant inserted) | `kind 11: value 12, name "declined"` |
| no `EventDeclined` | `last event = verified, want declined` |
| an outcome reporter error surfaces | panic: `report failed` |
| no `EventRolledBack` | `events = […installing failed], want rolled-back then failed` |
| managed recovery without `RolledBack` | `managed: res={… RolledBack:false}` |
| standalone swap without `RolledBack` | `standalone swap rollback: {… RolledBack:false}` |
| `Document` drops `Applied` | `TestResultDocumentJSON` mismatch |

Two first drafts were invalid mutations and were rewritten:

* **"JSON drops kind via `omitempty`"** survived, because a kind name is
  never empty, so the mutation changed nothing observable. It now removes
  the value.
* **"Declined renumbered via `= 20`"** did not compile: the explicit value
  repeats down the const block and duplicates `switch` cases. It now
  inserts a reserved constant.

**Checks.**

* `make pre-add-check` passed on the eight files.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`.

### Step 6: credentials (2026-09-30)

**What changed.**

* **`credentials.go`** adds `Credential`, `CredentialRequest`,
  `CredentialProvider`, `CredentialObserver`, `ErrNoCredential`,
  `ChainCredentials` and `EnvCredential`, with `validateCredential`.
* **`types.go`** appends `GitHubOptions.Credentials` and `.Observer`.
* **`github.go`.**
  * `GitHubSource` keeps `token` with its v1.0 meaning (explicit, else
    `GH_TOKEN`, else `GITHUB_TOKEN`), so the existing `TestGitHubTokenOrder`
    passes unchanged. It adds `explicit` and `envName`, and a mutex-guarded
    `credentialState` resolved on the first API request.
  * Both request sites go through `send`, which:
    * attaches the credential only on the API origin;
    * after a 401 asks the provider once more and retries once;
    * tells `Observer` once, after a 2xx.
  * `checkRedirect` also deletes a provider's own header on a cross-origin
    hop.
* **The retry is once per source**, not once per request, so a provider
  that prompts can never be asked in a loop. The PLAN said "once more"
  without saying per what. The doc comment states the choice, and a test
  pins it.
* **`drainClose`** routes its discarded errors through `advisory`, whose
  doc now covers both uses; errcheck's `check-blank` stays on.
* **The test servers** are the two plain-HTTP loopback servers the
  existing GitHub tests use: two ports are two origins, and loopback is
  exempt from the HTTPS rule. The TLS pair arrives in Step 11, as planned.

**Tests** (`credentials_test.go`). Twelve tests:

* order: explicit token, then provider, then environment;
* lazy resolution;
* fall-through, including to a chain's second link;
* a provider error fails the request before any request is sent;
* the two header modes;
* invalid credentials refused without echoing the secret;
* stripping across origins in both modes;
* `Accepted` once, and never for a refused credential;
* the 401 retry once, and once per source;
* no resend of the same credential, and no retry for an explicit token;
* `EnvCredential`'s source label;
* eight concurrent first requests share one provider call.

**Mutation proofs.** Nine mutations, none survived:

| Mutation | Killed by |
| :--- | :--- |
| the custom header not stripped | `credential "X-Api-Key" reached the asset host` |
| `Accepted` before the response | `Accepted called for a refused credential` |
| the retry may repeat | `the provider was asked 3 times; the retry is once per source` |
| the provider before `Token` | `request 0: Authorization "Bearer prov", want "Bearer explicit"` |
| the per-source cache dropped | `provider asked 2 times for two requests` |
| the same credential resent | `2 requests; the same credential must not be resent` |
| an invalid header accepted | `credential "Bad Header": err = … invalid header field name` |
| the chain stops at `ErrNoCredential` | `the chain did not fall through to its second link` |
| the env label wrong | `… Source:env …` |

Two mutations survived the first run: the retry mutation and the chain
mutation. Each exposed a missing case, not a defect: nothing sent a
second request after the one retry, and no chain had a later link that
supplied a credential. Both cases were added.

**Checks.**

* `make pre-add-check` passed on the five files, after two lint fixes:
  `bytes.Equal`, and the `drainClose` sink.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`.

### Step 7: integrity seams (2026-09-30)

**What changed.**

* **`manifestverify.go`** adds `ManifestVerification`, `ManifestVerifier`,
  `ManifestVerifierFunc`, `runManifestVerifiers` (which joins `ErrIntegrity`
  into every failure), and `openAsset`.
  * `openAsset` requires an exact, unique name, and a positive limit capped
    at `Limits.Executable`, then runs `validateAssetMetadata`.
  * Its `checkedAsset` reader fails with `ErrIntegrity` on a long body, a
    short body, or a GitHub-digest mismatch.
* **`types.go`** appends `Config.ManifestVerifiers` and
  `Verification.OpenAsset`.
* **`updater.go`.** `New` refuses nil and typed-nil manifest verifiers. The
  verifiers run right after `checksumFor`, before
  `EventDownloadingBinary` and `CreateStaging`, and binary verifiers get
  `OpenAsset`.
* **`imageverify.go`** adds `NewImageVerifier`, using the GOROOT constants
  in the PLAN's table.
  * An ELF passes on class, machine, and `ET_EXEC` or `ET_DYN` (PIE).
  * A Mach-O passes as a thin `TypeExec` with a matching CPU, or as a fat
    file with any matching `TypeExec` arch; a `macho.ErrNotFat` result
    falls back to a thin file.
  * A PE passes on machine and `IMAGE_FILE_EXECUTABLE_IMAGE`.

**Tests.**

* **`manifestverify_test.go`:**
  * order: the manifest before the verifier, and the binary after it; a
    failure stops before the binary;
  * a failure matches `ErrIntegrity`, and leaves the target byte-identical
    with no staging;
  * both kinds of verifier can read a sibling asset;
  * `openAsset` rules, with pinned diagnostics: "shorter than advertised",
    "longer than advertised", "github digest", limit, missing, duplicate
    and zero limit;
  * nil and typed-nil verifiers refused.
* **`imageverify_test.go`:**
  * The fixtures are real executables, cross-built in the test for
    `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` and
    `windows/amd64`, plus a fat Mach-O the test assembles from the two
    darwin builds.
  * There are 15 match and mismatch rows, including non-executable
    bodies.
  * It also covers unsupported platforms and a non-seekable body.

**Mutation proofs.** Ten mutations, none survived:

| Mutation | Killed by |
| :--- | :--- |
| manifest verifiers removed from their place | `calls before the manifest verifier = []` |
| `OpenAsset` drops the size check | `long: … is shorter than advertised …, want … "longer than advertised"` |
| the digest check dropped | `digest: <nil>, want ErrIntegrity` |
| duplicate names accepted | `dup with limit 1024 was opened` |
| `ErrIntegrity` not joined | `… manifest verification failed: untrusted manifest, want ErrIntegrity` |
| binary verifiers without `OpenAsset` | a nil-function panic |
| ELF machine ignored | `linux-amd64 as linux/arm64: <nil>, want ErrIntegrity` |
| the fat branch removed | `darwin-fat as darwin/amd64: … not a darwin/amd64 executable` |
| PE machine ignored | `windows-amd64 as windows/arm64: <nil>, want ErrIntegrity` |
| nil manifest verifier accepted | `manifest verifier <nil> accepted` |

**The size-check mutation survived the first run.** A long body is still
caught at EOF, because the reader lets one extra byte through, but it was
reported as "shorter than advertised". The test now pins the right
diagnostic for each case.

**Checks.**

* `make pre-add-check` passed on the six files, after one lint fix: an
  `unconvert`, because `macho.MagicFat` is already `uint32`.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`, including the cross-built fixtures.

### Step 8: probes and runnable staging (2026-09-30)

**What changed.**

* **`probe.go`** (new) adds `ProbePhase` (`ProbeStaged`, `ProbeInstalled`),
  `ProbeRequest`, `Prober`, `ProberFunc` and `NewVersionProber`.
  * `versionProber.Probe` runs the binary under a timeout with
    `WaitDelay = time.Second`, keeps at most 64 KiB of stdout, and passes
    when the exit is 0 and stdout contains `want(TargetVersion)`.
  * A failure says "timed out", "failed", or "printed %q, want %q", with the
    first stdout line sanitised.
* **`types.go`** appends `Config.Probes`, `InstallOptions.PostInstall` and
  `InstallRequest.TargetVersion`.
* **`updater.go`.** `New` refuses a nil probe. `runProbes` chmods the
  staging file `0o700` and runs each probe with `ProbeStaged`, after the
  transform and before `EventInstalling`. The coordinator sets
  `InstallRequest.TargetVersion` to `rel.Tag`.
* **`session.go`.**
  * `stagingSuffix()` is `.exe` on Windows and empty elsewhere, and is
    appended to the `CreateStaging` pattern.
  * `probeInstalled` runs `PostInstall` on `target.Path` with
    `ProbeInstalled`. On failure it rolls back on
    `context.WithoutCancel` with the retry budget, and joins
    `errProbeRolledBack`, or the rollback error when rollback fails.
  * `Install` runs it after `checkDir` and before commit, setting
    `RolledBack` or, when rollback failed, `Backup`. `apply` runs it after
    `replaceLocked`.
* **`standalone.go`** carries `PostInstall` into the session; **`managed.go`**
  reports `RolledBack` when the probe's rollback succeeded.
* **`replace_native_test.go`.** The existing `TestMain` gains the
  `SELFUPDATE_TEST_PRINT_VERSION` branch, with optional
  `SELFUPDATE_TEST_SLEEP` and `SELFUPDATE_TEST_EXIT`.

**Tests** (`probe_test.go`, 11 tests). The release bodies are the test
binary itself.

* `TestStagedProbeRunsRealBinary` and `TestProbesRunBeforeInstall`.
* `TestPostInstallRollback`, `TestPostInstallRollbackFailureReportsBackup`
  and `TestPostInstallManaged`.
* `TestVersionProberMatch`, `…Mismatch`, `…NonZeroExit` and `…Timeout`.
* `TestProbeRequestFields`. It compares against the `EvalSymlinks` form of
  the target, because the macOS temporary directory resolves through the
  `/var` symlink, and on Windows it asserts the `.exe` suffix.
* `TestNewVersionProberArguments`.

**Mutation proofs.** Seven local mutations, none survived:

| Mutation | Killed by |
| :--- | :--- |
| no chmod before probes | `TestStagedProbeRunsRealBinary` (`probe_test.go:81`): the run failed, `res={… Applied:false …}` |
| `PostInstall` ignored | `TestPostInstallRollback` (`probe_test.go:148`): `res={… Applied:true …}` |
| probes run after `Install` | `a mismatched version was installed` |
| `strings.Contains` becomes `==` | `printed "demo v1.1.0 (linux/amd64)", want "v1.1.0"` |
| post-install rollback not attempted | `TestPostInstallRollback` (`probe_test.go:148`) |
| the timeout is not applied | `err=<nil> after 10.015076792s` |
| `InstallRequest` carries no tag | `installed request {Product:demo TargetVersion: …}` |

On the Windows test host, `stagingSuffix` returning `""` was killed only by
`TestProbeRequestFields`: `Windows staging "<path>" does not end in .exe`.
`TestStagedProbeRunsRealBinary` passed. See D3.

**Deviation D2 (2026-09-30): two line-scoped `gosec` exemptions.**

* **Found.** `make pre-add-check` failed on two new lines, neither
  pre-existing:
  * `selfupdate/probe.go`, `exec.CommandContext(ctx, r.Path, p.args...)`:
    G204, "Subprocess launched with a potential tainted input or cmd
    arguments". Running the binary is the probe's purpose.
  * `selfupdate/updater.go`, `os.Chmod(stagedPath, 0o700)`: G302, "Expect
    file permissions to be 0600 or less". The staged binary must be
    executable to be probed.
  * The repository had no `//nolint`, and `.golangci.yml` excludes only
    G104.
* **Decision.** The owner chose a `//nolint:gosec` on exactly those two
  lines, each with its rule and reason. `.golangci.yml` is unchanged, and
  these are the repository's first `nolint` directives.
* **Scope proof.** On a scratch copy, `gosec` over `./selfupdate/...`
  reported 0 issues. Adding one unexempted `os.Chmod(p, 0o755)` and one
  `exec.CommandContext(ctx, r.Path, args...)` elsewhere in `probe.go` made it
  exit 1 with exactly those two, G302 and G204.
* **MADR.** No MADR change: nothing it states is affected.

**Deviation D3 (2026-09-30): the `.exe` suffix is not needed to run staging.**

* **Found.** On the Windows test host, with `stagingSuffix` returning `""`,
  `TestStagedProbeRunsRealBinary` passed: an extensionless staging file
  ran. A staging name always contains a dot (`.demo.selfupdate-NNN`), so
  `os/exec` on Windows treats it as having an extension and passes the path
  to `CreateProcess` as it is. Behaviour rule 1's "unverified" and the
  planned mutation row were wrong.
* **Decision.** The owner kept the suffix, as a convention for tools that
  key on the extension. The `stagingSuffix` comment states that reason
  instead of claiming `exec` needs it. The `.exe` mutation is killed by
  `TestProbeRequestFields`'s suffix assertion.
* **MADR.** Annotated where it says staging lacked `.exe` (the capability
  table and the §3 sketch), and noted in the §3 amendment block.

**Checks.**

* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...` on the Step 8 tree before D2 and D3. Those
  two change only comments; the `windows` lint target runs in
  `make pre-add-check`.

### Step 9: first-class custom installers (2026-09-30)

**What changed.**

* **`types.go`** adds `StagingOwner`, `AppliedReplacement` and
  `TwoPhaseSession`, after `InstallSession`.
* **`session.go`.** `installSession` implements both interfaces.
  * The exported methods are the implementations. The unexported `apply`,
    `commit` and `rollback` are gone, and `owns` became `ownsLocked`,
    because `revive`'s `confusing-naming` refuses two methods that differ
    only in case.
  * `Apply` returns `AppliedReplacement{Target, Backup, State: applyResult}`.
    When the post-install probe rolled back, `Backup` is empty.
  * `Commit` and `Rollback` fail with
    `selfupdate: replacement was not applied by this session` when `State`
    is not an `applyResult`.
  * A commit refused by the directory check still reports `Applied: true`
    with its `Backup`, as the old `commit` path did: the replacement is
    live.
* **`updater.go`.** `sessOwns` is `ok && o.Owns(path)` on `StagingOwner`,
  and false for any other session.
* **`managed.go`.** `ManagedInstaller.inner` is an `Installer`.
  * `NewManagedInstaller` keeps its nil check and calls
    `NewManagedInstallerFor`, which refuses a nil or typed-nil installer.
  * `Begin` refuses a session that is not a `TwoPhaseSession`, closing it.
  * `managedSession` drives `Apply`, `Commit` and `Rollback`, delegates
    `Owns`, and adds `ServiceInstalled` and `ServiceWasRunning` to
    `Commit`'s result.
  * Recovery rolls back when `AppliedReplacement.Backup` is set. When that
    rollback fails, it reports the backup if `os.Lstat` still finds it
    (Phase 0 R1).
* **`updater_contract_test.go`.** `logSession` gains `Owns`, for the files
  its `CreateStaging` makes. `captureSession` embeds `*logSession`, so it
  gets `Owns` with no edit. This is the only change to existing tests.

**Tests** (`twophase_test.go`):

* `TestManagedDrivesCustomTwoPhaseSession`: one log records the session,
  lifecycle and reconciler calls, and must read `Stop`, `Apply`,
  `Reconcile`, `Start`, `WaitHealthy`, `Commit`. `Commit` gets the
  session's own `State`.
* `TestManagedRollsBackCustomSession`: a health failure gives `Restore`,
  `Rollback`, `Start`, `WaitHealthy` after the forward calls, with the
  session's `State`, and `RolledBack: true`.
* `TestManagedRejectsSingleStepSession`: the refusal message, and the
  session closed exactly once.
* `TestSessOwnsFailsClosed`: a plain session owns nothing; a
  `StagingOwner` is asked; the standalone session owns its staging and not
  the target.
* `TestCommitRejectsForeignState`: a nil and a foreign `State`, for both
  `Commit` and `Rollback`, with the target unchanged.

Three tests beyond the PLAN's list:

* `TestTransformRequiresStagingOwner`: the behaviour change end to end. A
  custom session without `Owns` plus a `Transformer` stops with
  `transformed staging is not owned by the session`, before `Install`.
* `TestNewManagedInstallerForRejectsNil`.
* `TestStandaloneApplyCommit`: the standalone two-phase path keeps the
  backup until `Commit`, then removes it.

**Mutation proofs.** The PLAN's three plus six more; none survived:

| Mutation | Killed by |
| :--- | :--- |
| `sessOwns` returns true for an unknown session | `a session that is not a StagingOwner owns a path` |
| `Begin` accepts a plain `InstallSession` | `Begin = {…}, <nil>; want the two-phase refusal` |
| the refused session is not closed | `the refused session was closed 0 times, want 1` |
| `Commit` skips the `State` assertion | `Commit(nil) = <nil>, want "selfupdate: replacement was not applied by this session"` |
| `Rollback` skips the `State` assertion | `Rollback(nil) = selfupdate: no backup to restore, want …` |
| the managed `Commit` drops the session's `State` | `Commit got State <nil>, want the value Apply returned` |
| recovery does not roll back the custom session | `calls = [Stop Apply Reconcile Start WaitHealthy Restore Start WaitHealthy], want […Restore Rollback Start WaitHealthy]` |
| a refused commit reports not applied | `TestManagedCommitRefusesMovedDirectory`: `Applied=false …; want an applied install whose commit was refused` |
| `NewManagedInstallerFor` accepts a typed nil | `typed nil installer accepted` |

The "refused commit" mutation reproduces a regression that the first draft
of `Commit` actually had: the existing `TestManagedCommitRefusesMovedDirectory`
caught it before the tests above were written.

**Checks.**

* `make pre-add-check` passed on the six files.
* The Windows test host passed `go vet ./...` and
  `go test -race -count=1 ./...`.
* No deviation: the API is the PLAN's, and nothing the MADR states changed.
