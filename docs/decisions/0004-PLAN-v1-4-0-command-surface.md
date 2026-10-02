---
status: in-progress
date: 2026-10-02
associated-madr: "0004-MADR-evolve-selfupdate-api-and-tui-support.md"
---
# Implement Phase 3: the canonical command surface (`v1.4.0`)

Associated MADR: [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md)

This PLAN implements the MADR's Decision Outcome §5, "Phase 3: the canonical
command surface":

* `buildinfo`, the library-owned build stamps;
* `selfupdate/cli`, one update command for every program;
* `selfupdate.UserAgent`;
* the cobra recipe.

It also takes the `--channel` flag, which
[0005-MADR-opt-in-prerelease-channels.md](0005-MADR-opt-in-prerelease-channels.md)
§6 leaves to this phase.

The MADR's Confirmation for Phase 3 is met here: "the migration guide shows
one consumer (prepare-commit-msg) moved to `cli` and `buildinfo`, with a
byte-for-byte check of its `--check` output and exit codes." The migration
is proven on a scratch copy of prepare-commit-msg. The real migration, and
the fixes for C1–C6, belong to each consumer's own records (MADR §4).

The previous release, `v1.3.1`, is recorded in
[0004-PLAN-v1-3-1-ocp-login-pattern-rewording.md](0004-PLAN-v1-3-1-ocp-login-pattern-rewording.md).
Step 1 closes it out.

The MADR leaves "the phase's PLAN" to settle names and shapes. Where this PLAN
changes something §5 states, it is listed under "Proposed MADR amendments",
and Step 1 applies those amendments before any code changes. **Approving this
PLAN approves them.**

## Goal

`v1.4.0` ships:

* **`buildinfo`.** `Identity()` decides release or local from the library's
  own stamp variables, never from version text alone. A test proves that
  the documented `-ldflags` string really stamps them.
* **`selfupdate.UserAgent`,** whose output always passes `NewGitHubSource`'s
  User-Agent check.
* **`selfupdate/cli`,** with:
  * the canonical flags;
  * a request builder that refuses contradictions before any network
    work;
  * `Run`, with signal cancellation, a 15-minute default timeout and the
    stream rule;
  * `Exit`, with one canonical error line;
  * `Command`, the whole command in one call.
* **The stream rule, as tests:** without `--json` stdout stays empty, and
  under `--json` every line on stdout is one JSON object, ending with
  exactly one `"kind":"result"` object.
* **The migration proof.** A scratch copy of prepare-commit-msg, moved to
  `cli` and `buildinfo`, matches this repository's golden files byte for
  byte and exits 0, 10 and 1 where it should.

Done means:

* every row of the assertion register below has its test, and that test
  failed under its named mutation;
* `make pre-add-check`, `make lint`, `make vuln` and `make apicheck` pass;
* CI is green on Linux, macOS and Windows;
* `go.mod` is unchanged.

## Scope

### Item → step

| Item | Step |
| :--- | :--- |
| records: amendments F1–F9, the `v1.3.1` close-out, the index | 1 |
| `buildinfo` | 2 |
| `selfupdate.UserAgent`; `Makefile` (`fuzz -m 5`) | 3 |
| `cli`: `Flags`, `Bind`, `Parse`, `Request`, `Help` | 4 |
| `cli`: `Options`, `StdioOptions`, `Run`, `Exit`, `Summary` | 5 |
| `cli.Command` and the runnable examples | 6 |
| the migration proof, the migration guide and the cobra recipe | 7 |
| documentation and close-out | 8 |

### Out of scope

* **Phase 4** (MADR §7): the build-and-stage workflow, the installer
  templates, `selfupdate/service` and `selfupdate/codesign`.
* **Any change in a consumer repository,** prepare-commit-msg included. The
  proof uses a scratch copy that is never committed there.
* **A cobra or pflag dependency.** The recipe is documentation, and pflag's
  shape is proven with a test double.
* **`go-tui-lib/updatetea`,** which go-tui-lib records.
* **Pushes and tags,** which are the owner's.

### Fixed inputs

| Input | Value | Source |
| :--- | :--- | :--- |
| Baseline | `v1.3.1` = `351bd6a` | `git rev-parse v1.3.1^{commit}` |
| Go | 1.27.1 | `go.mod` |
| Module requirements | `golang.org/x/mod v0.40.0`, `x/sys v0.47.0`, `x/term v0.43.0`, unchanged | `go.mod` |
| API gate | `make apicheck`, against the newest `v1.*` tag, `v1.3.1` | `scripts/check-api-compat.sh` |
| Consumer for the proof | prepare-commit-msg at `cfada6e`, on `mcplib v1.6.0` | `git -C … log -1`; its `go.mod` |
| Windows test host | Git Bash, go1.27.1 with cgo | as in the Phase 1 PLAN |

### Code facts the design rests on

Each was read on 2026-10-02 at `351bd6a`.

* **Exit codes already exist.** `selfupdate.ExitCode(res, err)` returns 0
  for nil, 10 when `ErrUpdateAvailable` is in the chain, and 1 otherwise
  (`errors.go`).
* **The build kinds.** `BuildUnknown` is invalid in a request,
  `ReleaseBuild` is a stamped tag, and `LocalBuild` is anything else
  (`types.go`).
* **Contradictions are already refused.** `validateRequest` refuses
  `CheckOnly` with `Yes`, `Force` or `DryRun`, with the messages
  `selfupdate: --check and --yes are contradictory` (and `--force`,
  `--dry-run`). It validates the current version with the configured
  policy, so under `NewSemverPolicy` an `-rc.N` binary is a valid
  `ReleaseBuild` (`version.go`, `validateRequest`).
* **A check-only run** emits `resolving-target`, `fetching-release` and
  `selected`. It returns `ErrUpdateAvailable` when the operation is not
  `OperationNone`, and the result has `Checked` set. `selected` carries
  `local build: apply requires --force` for a local build (`updater.go`,
  `execute`).
* **Run options.** `RunWith`'s `WithReporter` and `WithConfirmer` replace
  `Config.Reporter` and `Config.Confirmer` for that run (`runoptions.go`).
* **The text reporter** writes `selfupdate: <kind> product=… current=…
  target=… asset=… bytes=… <detail>` and skips progress. The JSON reporter
  writes one object per line, with the keys `kind`, `product`, `current`,
  `target`, `asset`, `bytes`, `total` and `detail` (`reporter.go`,
  `jsonreporter.go`).
* **`NewGitHubSource`** refuses a User-Agent that is blank or contains a
  control character (`github.go`, `validateUserAgent`).
* **`NewPromptConfirmer(in, out, interactive)`** prompts on `out`, and
  refuses with `ErrConfirmationRequired` when `interactive` is false
  (`confirmer.go`).
* **prepare-commit-msg today:**
  * it stamps `main.RawVersion` and `main.RawBuildKind=release`;
  * it writes the text reporter and the prompt to **stdout**, against MADR
    §5;
  * it prints `Update failed: %v` on stderr;
  * it wires `signal.NotifyContext` and a 15-minute timeout in `main`;
  * it refuses positional arguments and the two `--check` contradictions
    itself;
  * its tests replace the updater through `var newUpdateUpdater`
    (`update.go`, `main.go`, `update_test.go`).

### Compatibility rules the design follows

The Phase 1 PLAN's rules, unchanged:

* only new types, functions and methods;
* no method added to an existing interface;
* no field added to a comparable struct unless its type is comparable;
* no signature changed.

Every new name in `selfupdate` and every name in the two new packages is an
addition, so `make apicheck` against `v1.3.1` must report none incompatible.

### Rules for every step

The Phase 1 PLAN's five rules apply:

* order within a step;
* mutation proofs on a scratch copy, with exact `old` → `new` replacements;
* existing tests change only where this PLAN says so;
* deviations stop the step;
* every new exported identifier has a doc comment citing its MADR item.

Four rules are added:

* **Rule 6, Determinism.**
  * No test sleeps. No test reaches the network beyond loopback.
  * No test asserts on wall-clock time, except Step 5's default-timeout
    check, which reads a context deadline with a one-second tolerance.
  * A test that must wait for another process waits for a marker line,
    never a duration.
* **Rule 7, Golden files.**
  * Goldens live under `selfupdate/cli/testdata/golden/`. They are
    rewritten only with `-update`, and every rewritten file is read before
    it is committed.
  * The only substitution allowed when comparing is `{{asset}}`, which
    becomes `selfupdate.ExactAssetName(product, runtime platform)`. That
    keeps each golden identical on all three operating systems.
* **Rule 8, `make apicheck`** runs in every step from Step 2 on.
* **Rule 9, The register is the contract.** A row with no test or no killed
  mutation means the step is not done. A new behaviour found while
  coding gets a register row before the step ends, or stops the step as a
  deviation.

## Proposed MADR amendments

Step 1 applies these to the MADR's §5, each marked *(amended 2026-MM-DD,
0004-PLAN-v1-4-0-command-surface)*.

| ID | MADR text today | Amendment | Why |
| :--- | :--- | :--- | :--- |
| F1 | `Identity()` is `ReleaseBuild` "only when the stamped kind is `release` and the version is a strict tag" | A release identity needs the stamped kind to be exactly `release`, and the stamped version to be either `vMAJOR.MINOR.PATCH` or `vMAJOR.MINOR.PATCH-NAME.N`, the 0005 grammar (NAME `^[a-z][a-z0-9]{0,15}$`, N a decimal with no leading zero). Anything else is local, and `Info.Reason` says why a `release` stamp was refused. Whether a prerelease is allowed stays the version policy's decision in `selfupdate`. | Since 0005, an `-rc.N` binary is a real release. Calling it local would make every rc need `--force`, which is C2 again. |
| F2 | `buildinfo` returns an `Info` with a kind | `buildinfo` is standard library only and does not import `selfupdate`. It has its own `Kind` (`KindUnknown`, `KindRelease`, `KindLocal`). `cli` maps `KindRelease` to `selfupdate.ReleaseBuild` and the others to `LocalBuild`. It exports `VersionVar` and `KindVar`, the two full `-X` symbol names, and `LDFlags(version)`, which returns the canonical `-X` pair. | Keeping `buildinfo` below `selfupdate` lets any program, or Phase 4's workflow, read build stamps without the updater. The exported names are tested against a real linked binary, so the documented string cannot drift. |
| F3 | `Flags{Check, Yes, Force, DryRun, JSON bool; Version string}` | Add `Channel string`, bound to `--channel`. | 0005-MADR §6 gives this flag to Phase 3. |
| F4 | `Flags.Bind`, `Flags.Request` and `Run` are the whole entry point | Add `(*Flags).Parse(args []string, stderr io.Writer) error` for the standard `flag` package, which also refuses positional arguments. Add `Command(ctx, args, product string, id buildinfo.Info, newUpdater func() (*selfupdate.Updater, error), o Options) int`: parse, request, build the updater lazily, run, report and return the exit code. | Six consumers would otherwise repeat the same thirty lines, and repetition is how C3 and C4 happened. Building the updater lazily keeps a flag error away from the network, as prepare-commit-msg's test requires today. |
| F5 | `const HelpText = "…"` | `const HelpText` is the flag list and the exit-status paragraph. `Help(prog string) string` returns `Usage: <prog> update [flags]`, a blank line, then `HelpText`. | A constant cannot name the program. |
| F6 | `Run(ctx, u, req, o Options)` with "signal ctx, 15 m timeout, streams" | `Options{Stdout, Stderr io.Writer; Stdin io.Reader; Interactive, JSON bool; Confirmer selfupdate.Confirmer; Timeout time.Duration; Signals []os.Signal}`, rules in Step 5. `StdioOptions()` fills the process's streams, with `Interactive` from `golang.org/x/term`, a module already required. | The MADR's package table lists `cli` as standard library only. `x/term` is already in `go.mod` through `selfupdate`, so no requirement changes. |
| F7 | `Exit` prints "one canonical error line" | The line is `update failed: <message>`, where the message is the error text made into one line with control characters removed. `ErrUpdateAvailable` prints no error line. `Exit` writes only to `stderr`. | One fixed format makes the output testable byte for byte. A spurious error on exit 10 is C3. |
| F8 | "under `--json`, a JSONL stream … then a final object with `"kind":"result"` that carries the `ResultDocument`" | The final object is `{"kind":"result","exit_code":N,"error":"…","result":{…}}`, with the keys in that order. `error` is omitted when there is no error, and `result` is the `ResultDocument`. Under `--json` every invocation that gets past flag parsing ends with exactly one such line, including a refused request or an updater that fails to build. `-h` writes help to stderr and nothing to stdout. | Nesting keeps the result's `product` apart from the event keys. One closing line on every path means a reader never waits for one that will not come. |
| F9 | Exit codes "0 … 10 … 1 otherwise" | Usage errors (an unknown flag, a positional argument) exit 1, as every other error does. `-h` and `--help` exit 0. | The MADR fixes three codes. `flag`'s conventional 2 would be a fourth. |

### Decisions to confirm

Approving the PLAN confirms each. Say otherwise to change one.

* **D-Q1, the error prefix.** `update failed: …` (F7), instead of
  `<product>: update failed: …`. The product is not known on every path,
  such as an updater that fails to build.
* **D-Q2, the usage exit code.** 1 (F9), instead of 2.
* **D-Q3, help under `--json`.** Help goes to stderr, and no result object
  is written (F8). `--help` is a request for text.

## Assertion register

Every assertion this PLAN makes, the test that proves it, and the mutation
that test must kill. Tests are named in the step that adds them.

| # | Assertion | Test | Killing mutation |
| :--- | :--- | :--- | :--- |
| A1 | A `release` stamp with a strict or 0005-prerelease tag is `KindRelease` | `TestIdentityKinds` | accept only strict tags |
| A2 | A `release` stamp with any other version is `KindLocal`, with a `Reason` | `TestIdentityKinds` | treat a stamped `release` as release whatever the version |
| A3 | Any stamp but exactly `release` is `KindLocal` | `TestIdentityKinds` | compare `kind` case-insensitively |
| A4 | Build info from `debug.ReadBuildInfo` never decides the kind | `TestIdentityIgnoresModuleVersion` | fall back to the module version when unstamped |
| A5 | `VersionVar`, `KindVar` and `LDFlags` stamp a real binary | `TestStampedBinary` | rename the `version` variable |
| A6 | `Info.Current()` is the stamped version, else the module version (not `(devel)`), else `dev` | `TestInfoCurrent` | return the module version first |
| A7 | `UserAgent` is `product/version (goos/goarch)`, and passes `NewGitHubSource` for any input | `TestUserAgent`, `FuzzUserAgent` | keep control characters |
| A8 | `Bind` registers exactly the eight canonical flag names, and uses `BoolVarP` for `-y` when the flag set offers it | `TestBindStdlib`, `TestBindShorthand` | skip the `BoolVarP` branch |
| A9 | `Parse` refuses positional arguments and unknown flags, and returns `flag.ErrHelp` for `-h` | `TestParse` | accept positional arguments |
| A10 | `Request` refuses the three contradictions with `validateRequest`'s own text, before any updater exists | `TestRequestContradictions` | drop the `--dry-run` pair |
| A11 | `Request` maps each flag and the identity to the request | `TestRequestMapping` | drop `Channel` |
| A12 | Without `--json`, nothing is written to stdout, on any path | `TestStdoutEmptyWithoutJSON` (every scenario) | write the summary to stdout |
| A13 | Under `--json`, every stdout line is one JSON object, and the last is the only `"kind":"result"` | `TestJSONStream` (every scenario) | write the result object twice on error |
| A14 | The result object matches F8 byte for byte | goldens `*.json.stdout` | rename `exit_code` |
| A15 | The summary line for each outcome matches Step 5's table | goldens `*.text.stderr` | swap the up-to-date and available lines |
| A16 | The prompt goes to stderr; `y` applies, `n` declines with exit 0, and a non-interactive run without `--yes` exits 1 with `ErrConfirmationRequired` | `TestConfirmation` | prompt on stdout |
| A17 | A signal cancels the run: exit 1, target byte-identical | `TestSignalCancels` (Unix, real `SIGTERM` to a child); `TestSignalsWired` (all OS, seam) | drop `SIGTERM` from the default list |
| A18 | `Timeout` 0 means 15 minutes; a negative one is refused; an expired one exits 1 with the target unchanged | `TestTimeout` | default to no timeout |
| A19 | `Exit` writes `update failed: <one line>` for errors, nothing for `ErrUpdateAvailable` or nil, never to stdout, and returns `ExitCode` | `TestExit` | print the line for `ErrUpdateAvailable` |
| A20 | `Command` builds no updater when parsing or the request fails | `TestCommandLazyUpdater` | build the updater before parsing |
| A21 | `Command`'s exit codes are 0, 10, 1, and 0 for `-h` | `TestCommandExitCodes` | return 2 for usage errors |
| A22 | The cli package's non-test code names `os.Stdout` only inside `StdioOptions` | `TestNoStdoutOutsideStdio` (a `go/ast` scan) | write a banner with `fmt.Println` |
| A23 | The migrated scratch copy of prepare-commit-msg matches the goldens byte for byte, with exit codes 0, 10 and 1 | Step 7's scratch test | change one summary word in the copy |
| A24 | No exported API changes incompatibly against `v1.3.1` | `make apicheck` | remove `Result.DryRun` (a scratch copy only) |
| A25 | Every example in the package docs compiles and runs offline | `Example*` with `// Output:` | change an example's expected output |

## Implementation Steps

### Step 1: records

1. Apply amendments F1–F9 to the MADR's §5, each marked.
2. In
   [0004-PLAN-v1-3-1-ocp-login-pattern-rewording.md](0004-PLAN-v1-3-1-ocp-login-pattern-rewording.md),
   add "Release (2026-10-02)":
   * the owner tagged `v1.3.1` on `351bd6a`;
   * CI run `37017346855` on `main` and run `37018271601` on the tag both
     concluded `success`;
   * `GOPROXY=https://proxy.golang.org go list -m
     github.com/maccavelli/go-core-lib@latest` output, quoted.

   Set that PLAN `complete` only if that output is `v1.3.1`. If it is not
   yet, record the output and leave the PLAN `in-progress`.
3. In `docs/architecture.md`, change the current release to `v1.3.1` on
   `351bd6a5ffbd4b352cacc5234f4a8c504c3c1b69`.
4. In `docs/README.md`, set this PLAN `in-progress`.
5. One commit, docs only.

### Step 2: `buildinfo` (`buildinfo/buildinfo.go`, new)

**API**

```go
package buildinfo

// Set with -ldflags "-X <VersionVar>=v1.2.3 -X <KindVar>=release".
var version, kind string

const (
    VersionVar = "github.com/maccavelli/go-core-lib/buildinfo.version"
    KindVar    = "github.com/maccavelli/go-core-lib/buildinfo.kind"
)

type Kind uint8

const (
    KindUnknown Kind = iota // zero value; Identity never returns it
    KindRelease
    KindLocal
)

func (k Kind) String() string // "unknown", "release", "local"

type Info struct {
    Version       string // the stamped version, verbatim; may be empty
    Kind          Kind
    Reason        string // why a "release" stamp was refused; empty otherwise
    Module        string // main module path, from debug.ReadBuildInfo
    ModuleVersion string // e.g. "(devel)" or "v1.2.3"
    GoVersion     string
    Revision      string // vcs.revision
    Time          string // vcs.time, verbatim
    Modified      bool   // vcs.modified == "true"
}

func Identity() Info
func (i Info) Current() string // F1 and A6
func (i Info) String() string  // "<Current> (<kind>)", plus " <revision[:12]>[-dirty]" when known
func LDFlags(version string) string // "-X <VersionVar>=<version> -X <KindVar>=release"
```

* **Decision rules.** These are F1, A1–A4 and A6:
  * `KindRelease` iff `kind == "release"` exactly, and `version` matches
    `^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[a-z][a-z0-9]{0,15}\.(0|[1-9][0-9]*))?$`;
  * otherwise `KindLocal`. When `kind == "release"`, `Reason` is
    `buildinfo: stamped version "<v>" is not a release tag`, with the
    version quoted by `strconv.Quote`;
  * `ReadBuildInfo` fills only the display fields.
* **No other import.** `buildinfo` imports only the standard library:
  `regexp`, `runtime/debug`, `strconv` and `strings`.
* **Tests** (`buildinfo/buildinfo_test.go`, package `buildinfo`, setting the
  two variables through a `t.Cleanup`-restored helper):
  * `TestIdentityKinds`, a table of at least these cases:
    * `v1.2.3` and `v1.4.0-rc.1` with `release` are release;
    * `v1.2`, `1.2.3`, `v1.2.3-rc.01`, `v1.2.3-RC.1`, `v1.2.3+meta` and
      the empty string with `release` are local, each with its exact
      `Reason`;
    * `v1.2.3` with `Release`, `local` or the empty string is local, with
      no `Reason`.
  * `TestIdentityIgnoresModuleVersion`: unstamped, `Kind` is local
    whatever `ModuleVersion` holds.
  * `TestInfoCurrent`: the three fallbacks in order, and `(devel)` is
    skipped.
  * `TestKindString`, and `TestInfoString`, a table with and without a
    revision, and with `Modified`.
* **`TestStampedBinary`** (`buildinfo/stamp_test.go`, package
  `buildinfo_test`):
  * it writes a `main` package into `t.TempDir()` that prints
    `Identity()` as JSON. Its `go.mod` requires this module through a
    `replace` to the working tree, and it copies this module's `go.sum`;
  * every `go` command runs with `GOFLAGS=-mod=mod`, `GOPROXY=off` and
    `GOWORK=off`, so the build is offline and cannot pick up a workspace.
    `buildinfo` imports only the standard library, so nothing needs
    downloading;
  * binaries take the platform's executable suffix, so the test runs on
    Windows too;
  * it builds it with `go build -ldflags "<LDFlags("v1.2.3")>"`, then
    unstamped, then with `-X <KindVar>=release -X <VersionVar>=v1.2`;
  * it runs each binary and asserts release, local and local with
    `Reason`;
  * it is skipped under `-short`.
* **Mutations:** A1–A6's mutations, applied one at a time.

### Step 3: `selfupdate.UserAgent` (`selfupdate/useragent.go`, new)

```go
// UserAgent returns "product/version (goos/goarch)" for GitHubOptions.UserAgent
// (0004-MADR §5).
func UserAgent(product, version string) string
```

* **Sanitising.** Each of `product` and `version`:
  * drops runes that are control characters, or invalid UTF-8;
  * replaces each run of space, `/`, `(` or `)` with one `-`;
  * trims `-` from both ends;
  * becomes `unknown` if nothing is left.

  The platform part is `runtime.GOOS` and `runtime.GOARCH`, unchanged.
* **Tests:**
  * `TestUserAgent`, a table of at least: `("demo", "v1.2.3")`, an empty
    product, a version with spaces and slashes, a product with `\n`, `\x7f`
    and `\u0085`, and invalid UTF-8;
  * `FuzzUserAgent`. For any input, the output passes
    `validateUserAgent`, matches
    `^[^\s/()]+/[^\s/()]+ \([a-z0-9]+/[a-z0-9]+\)$`, and is unchanged by a
    second sanitising pass.
  * **The fuzz step.** `make fuzz` already fuzzes every target in
    `./selfupdate` for 20 s, so `FuzzUserAgent` joins CI's fuzz step
    with no workflow change. The `Makefile`'s `fuzz` target gains
    `-m 5`, up from the script's default of 4, the count of today's
    targets. A lost target then fails the step. `make fuzz` must
    report five targets run.
* **Mutations:** keep control characters; skip the `unknown` fallback;
  rename `FuzzUserAgent` to `fuzzUserAgent`, which `make fuzz -m 5` must
  refuse.

### Step 4: `cli` flags and requests (`selfupdate/cli/flags.go`, new)

**API**

```go
package cli

type FlagSet interface {
    BoolVar(p *bool, name string, value bool, usage string)
    StringVar(p *string, name string, value string, usage string)
}

// boolVarP is pflag's shorthand method; Bind uses it when present.
type boolVarP interface {
    BoolVarP(p *bool, name, shorthand string, value bool, usage string)
}

type Flags struct {
    Check, Yes, Force, DryRun, JSON bool
    Version, Channel                string
}

func (f *Flags) Bind(fs FlagSet)
func (f *Flags) Parse(args []string, stderr io.Writer) error
func (f Flags) Request(product string, id buildinfo.Info) (selfupdate.Request, error)

const HelpText = "…" // below
func Help(prog string) string
```

* **`Bind`** registers `check`, `yes`, `force`, `dry-run`, `json`, `version`
  and `channel`. `-y` is registered with `BoolVarP("yes", "y")` when `fs`
  has it; otherwise as a second bool flag `y` that sets the same field.
* **`Parse`.**
  * It builds a `flag.FlagSet` named `update`, with `ContinueOnError` and
    output `stderr`, and a usage function that writes `HelpText`.
  * It returns `flag.ErrHelp` for `-h` and `--help`.
  * A positional argument returns
    `cli: positional arguments are not accepted`.
* **`Request`.**
  * It refuses the three contradictions with exactly `validateRequest`'s
    messages, unwrapped (for example
    `selfupdate: --check and --yes are contradictory`), before anything
    else.
  * Then it maps the fields: `Product`; `CurrentVersion` from
    `id.Current()`; `CurrentBuild` from `id.Kind`; then `TargetVersion`,
    `Channel`, `CheckOnly`, `Yes`, `Force` and `DryRun`.
* **`HelpText`**, exactly:

  ```text
  Flags:
    --check            report whether an update is available; install nothing
    --yes, -y          install without asking
    --force            replace a local build, or reinstall the same version
    --dry-run          download and verify the release; install nothing
    --json             write JSON Lines to stdout; everything else to stderr
    --version vX.Y.Z   install exactly this release
    --channel NAME     follow a prerelease channel, such as rc

  Exit status: 0 up to date, declined or installed; 10 an update is
  available (with --check); 1 any error.
  ```

* **Tests** (`selfupdate/cli/flags_test.go`):
  * `TestBindStdlib`: `VisitAll` on a `flag.FlagSet` sees exactly the
    eight names.
  * `TestBindShorthand`: a test double with `BoolVarP` records the pair
    `("yes", "y")`, and no separate `y`.
  * `TestParse`: each flag; `-y`; `-h`; `--help`; `--bogus`; a positional
    argument. Nothing but usage is written to `stderr`.
  * `TestRequestContradictions`: for each pair, `Updater.Run` on the same
    request returns `selfupdate: demo:`, a space, and then exactly the text of
    `Request`'s error (`wrapRun`'s prefix), so the two messages cannot
    drift. A spy proves `Request` asked no updater.
  * `TestRequestMapping`: every field, for a release and a local identity.
  * `TestHelpText`: `Help("demo")` equals the golden `help.txt`.
* **Mutations:** A8–A11's mutations; and `HelpText` loses the `--channel`
  line, which `TestHelpText` must kill.

### Step 5: `cli` run and exit (`selfupdate/cli/run.go`, new)

**API**

```go
type Options struct {
    Stdout      io.Writer           // protocol output; written only when JSON
    Stderr      io.Writer           // everything else; required
    Stdin       io.Reader           // confirmation input
    Interactive bool                // Stdin is a terminal
    JSON        bool                // --json
    Confirmer   selfupdate.Confirmer // nil: NewPromptConfirmer(Stdin, Stderr, Interactive)
    Timeout     time.Duration       // 0: DefaultTimeout; negative: refused
    Signals     []os.Signal         // nil: os.Interrupt and syscall.SIGTERM; empty: none
}

const DefaultTimeout = 15 * time.Minute

func StdioOptions() Options
func Run(ctx context.Context, u *selfupdate.Updater, req selfupdate.Request, o Options) (selfupdate.Result, error)
func Exit(stderr io.Writer, res selfupdate.Result, err error) int
func Summary(res selfupdate.Result, err error) string
```

* **Option errors.** `Run` refuses these with an error and does nothing
  else:
  * a nil `Updater` or `Stderr`;
  * `JSON` with a nil `Stdout`;
  * a negative `Timeout`.
* **Order.**
  1. `signal.NotifyContext` on the signals.
  2. `context.WithTimeout`.
  3. `u.RunWith(ctx, req, WithReporter(r), WithConfirmer(c))`. Under
     `JSON`, `r` is `NewJSONReporter(Stdout)`; otherwise
     `NewTextReporter(Stderr)`.
  4. Under `JSON`, one `Write` of the result object (F8). Otherwise one
     `Write` of `Summary(res, err)` plus `\n` to `Stderr`, when
     `Summary` is not empty.
  5. Return `res, err`.
* **The signal seam.** `notifyContext` is a package variable, defaulting
  to `signal.NotifyContext`, so a test can observe the signals passed.
* **`Summary`.** `<p>` is the product, and the versions are the result's,
  each sanitised to one line:

  | Outcome | Line |
  | :--- | :--- |
  | `ErrUpdateAvailable` | `<p>: update available: <current> -> <target>` |
  | nil, `OperationNone` | `<p>: up to date (<current>)` |
  | nil, `DryRun` | `<p>: dry run: <target> verified; nothing installed` |
  | nil, `Declined` | `<p>: update declined` |
  | nil, `Applied` | `<p>: updated <current> -> <target>` |
  | any other error | empty (the error line is `Exit`'s) |

  A `selected` event with a local build is already explained by its
  detail, so no line is added for it.
* **`Exit`.** It writes `update failed: <message>\n` to `stderr` for any
  error but `ErrUpdateAvailable`, and nothing otherwise. It returns
  `selfupdate.ExitCode(res, err)`.
* **`StdioOptions`** returns `os.Stdout`, `os.Stderr` and `os.Stdin`, with
  `Interactive: term.IsTerminal(int(os.Stdin.Fd()))`.
* **Scenarios.** One fixture each, on `selfupdatetest.NewFakeSource` and a
  `StandaloneInstaller` over a temporary target, product `demo`, current
  `v1.0.0`:
  * `up-to-date`: latest is `v1.0.0`, `--check`;
  * `available`: latest is `v1.1.0`, `--check`;
  * `local-available`: a local identity, `--check`;
  * `applied`: latest is `v1.1.0`, `--yes`;
  * `declined`: latest is `v1.1.0`, interactive, stdin `n\n`;
  * `dry-run`: latest is `v1.1.0`, `--dry-run`;
  * `no-confirm`: latest is `v1.1.0`, not interactive, no `--yes`;
  * `failed`: a source whose `Latest` returns
    `errors.New("fixture: source unavailable")`;
  * `contradiction`: `--check --yes`. Through `Run`, the updater's
    `validateRequest` refuses it before any network call. Through
    `Command`, `Request` refuses it first, with the same text but without
    `wrapRun`'s `selfupdate: demo:` prefix, so `Command` has its own golden
    for it, `contradiction.command.*`.

  Each runs in text and in JSON mode, giving `<scenario>.text.stderr`,
  `<scenario>.json.stdout`, `<scenario>.json.stderr` and `<scenario>.code`.
* **Tests** (`selfupdate/cli/run_test.go`):
  * `TestGolden`: every scenario × mode, byte for byte (rule 7).
  * `TestStdoutEmptyWithoutJSON`: every text-mode scenario leaves `Stdout`
    empty. Its buffer is passed even in text mode.
  * `TestJSONStream`: every JSON line decodes with `json.Unmarshal`, every
    line but the last has a `kind` other than `result`, and the last
    is `result`.
  * `TestConfirmation`: the prompt is on `Stderr`, `y` applies, `n`
    declines with exit 0, and a non-interactive run returns
    `ErrConfirmationRequired` with exit 1.
  * `TestSignalsWired`: with the seam, the default list is exactly
    `{os.Interrupt, syscall.SIGTERM}`, an empty slice passes none, and a
    seam that cancels at once makes `Run` return `context.Canceled` with
    the target byte-identical.
  * `TestSignalCancels` (Unix only; Windows cannot signal a child this
    way):
    * the test binary re-runs itself as a helper, which runs a real `Run`
      against a source that blocks in `OpenAsset` until its context ends;
    * the helper prints `READY` to stderr just before blocking;
    * the parent reads lines until `READY` and sends `SIGTERM`;
    * the helper must exit 1 with `update failed:` and
      `context canceled` on stderr, and the target is byte-identical.
  * `TestTimeout`:
    * with `Timeout` 0, a source that records `ctx.Deadline()` sees a
      deadline within one second of 15 minutes from the call;
    * `-1ns` is refused;
    * `1ns`, against a source that blocks until its context ends, returns
      `context.DeadlineExceeded`, exit 1, and the target is unchanged.
  * `TestExit`: a table over nil, `ErrUpdateAvailable`, a wrapped error,
    and a two-line error, with an `io.Writer` that fails the test if
    `Exit` writes to anything but its argument.
  * `TestNoStdoutOutsideStdio`: a `go/ast` scan of the package's non-test
    files finds `os.Stdout` only in the body of `StdioOptions`, and no call
    to `fmt.Print*`, `print` or `println`.
* **Mutations:** A12–A19 and A22's mutations; `Run` passes the Updater's
  own reporter (dropping `WithReporter`); `Exit` writes to `os.Stderr`
  instead of its argument.

### Step 6: `cli.Command` and examples (`selfupdate/cli/command.go`, new; `selfupdate/cli/example_test.go`)

```go
func Command(ctx context.Context, args []string, product string, id buildinfo.Info,
    newUpdater func() (*selfupdate.Updater, error), o Options) int
```

* **Paths.**
  1. `Parse`. `flag.ErrHelp`: write `Help(product)` to `Stderr` and return
     0. Another error: report it and return 1.
  2. `Request`. An error: report it and return 1.
  3. `newUpdater()`. An error: report it and return 1.
  4. `Run`, then `Exit`.

  "Report" means: under `JSON` (as parsed so far), the result object for
  `Result{Product, CurrentVersion: id.Current()}`, with exit code 1 and
  the error; then `Exit`'s error line on `Stderr`.
* **`Options.JSON`** is set from the parsed flag. A caller's own value is
  overwritten.
* **Tests** (`selfupdate/cli/command_test.go`):
  * `TestCommandLazyUpdater`: on each of the three early paths, the
    `newUpdater` spy is never called.
  * `TestCommandExitCodes`: 0 for up to date, 10 for available, 1 for
    failed, 1 for `--bogus`, 1 for a positional argument, and 0 for `-h`.
  * `TestCommandJSONEarlyError`: `--json --check --yes` writes exactly one
    result object, with exit code 1.
  * Every Step 5 scenario but `contradiction`, run again through
    `Command`, gives Step 5's goldens byte for byte. `contradiction` gives
    `contradiction.command.*`.
* **Examples** (A25), offline on `FakeSource`, each with `// Output:`:
  * `ExampleCommand`;
  * `ExampleRun`;
  * `ExampleExit`;
  * `ExampleFlags_Bind`.

  An example writes to its own buffers and prints them, so the output is
  checked.
* **Mutations:** A20 and A21's mutations; `Command` keeps the caller's
  `JSON`.

### Step 7: the migration proof (`selfupdate/cli/testdata/migration/`, `docs/guides/migrating-from-mcplib-selfupdate.md`)

1. **Fixtures here.** `TestMigrationFixture` runs the `up-to-date`,
   `available` and `failed` scenarios through `Command`, with product
   `prepare-commit-msg`, in text mode. It writes the goldens
   `migration/<scenario>.{stdout,stderr,code}`.
2. **The scratch copy.** `git clone` prepare-commit-msg at `cfada6e` into
   the session's scratch space, on the macOS development host.
   * **`go.mod`.** Require `go-core-lib`, with a `replace` to this working
     tree. The three `mcplib/selfupdate` imports move to go-core-lib.
     `mcplib` stays required, because `llmprovider` and `wizard` still
     import it.
   * **`update.go`.** `RawVersion`, `RawBuildKind` and `runUpdate` give way
     to `buildinfo` and `cli`. `newUpdateUpdater` stays, and the updater it
     builds reports nothing itself, because `cli.Run` supplies the
     reporter and the confirmer.
   * **`main.go`.** The `update` case becomes
     `osExit(cli.Command(ctx, args[1:], AppTitle, buildinfo.Identity(), newUpdateUpdater, updateOptions()))`,
     where `updateOptions` is a package variable defaulting to
     `cli.StdioOptions`. The test replaces it with buffers. The `version`
     subcommand prints `buildinfo.Identity().String()`.
   * **`Makefile`.** `RELEASE_LDFLAGS` uses the two `-X` names that
     `buildinfo.VersionVar` and `buildinfo.KindVar` hold, written out,
     because make cannot read a Go constant. A test in the copy runs
     `make -n build-all` and asserts that both names appear.
   * **A test in the copy** points `newUpdateUpdater` at the same
     `FakeSource` fixtures. It runs `update --check` for each scenario and
     compares stdout, stderr and the exit code byte for byte with the
     fixture files, applying only rule 7's `{{asset}}` substitution.
   * **The proof of the proof.** One word of a summary line is changed in
     the copy. The test must then fail on that scenario (A23).
3. **The migration guide** gains "Adopt the canonical update command":
   * the copy's diff of `update.go`, `main.go` and `Makefile`, after the
     identifier check;
   * the behaviour changes a consumer will see:
     * the reporter and the prompt move from stdout to stderr (MADR owner
       decision 1);
     * `Update failed:` becomes `update failed:`;
     * `--dry-run`, `--json` and `--channel` are new.
   * the cobra recipe, as text:
     * `cmd.Flags()` is pflag's flag set, which `Flags.Bind` accepts;
     * `SilenceErrors` and `SilenceUsage` are set inside `RunE`;
     * `RunE` calls `cli.Run` and `cli.Exit`, then `os.Exit` with the code
       when it is not 0. That fixes C3, and C4 through `Run`'s signals.
4. **Nothing is committed to prepare-commit-msg.** The scratch copy's test
   output and the copy's `git diff --stat` are quoted in the execution
   record.

### Step 8: documentation and close-out

* **`selfupdate/doc.go`:** a section "The canonical update command",
  pointing to `cli` and `buildinfo`.
* **`selfupdate/cli/doc.go` and `buildinfo`'s package comment:** what each
  package is, the stream rule, and the exit codes.
* **`docs/architecture.md`:** the two packages, and their imports
  (`buildinfo` → standard library; `cli` → `selfupdate`, `buildinfo`,
  `x/term`).
* **`docs/README.md`:** rows for "add the update command to my program" and
  "stamp a release build".
* **Release notes for `v1.4.0`** in the execution record.
* **Verification** as below. Mark this PLAN `complete` after CI is green on
  the pushed tree. The owner tags.

## Verification

* Every register row A1–A25 has its test, and its mutation was killed. The
  execution record quotes each failure.
* The scratch copy of prepare-commit-msg passes its byte-for-byte test, and
  fails it under the planted change (A23).
* On the macOS development host and the Windows test host, all pass:
  * `make pre-add-check`, `make lint`, `make vuln` and `make apicheck`;
  * `go test -race -count=1 ./...` and `go test -shuffle=on -count=2 ./...`;
  * `make fuzz`, including `FuzzUserAgent`.
* `TestSignalCancels` ran on macOS and on Linux in CI, and was skipped,
  with its reason, on Windows.
* `go mod tidy -diff` is clean, and `go.mod` is unchanged.
* The identifier scan finds nothing, and the disclosure guard passes on the
  outgoing commits.
* After the owner's push, CI is green on all three operating systems.

## Rollout and Rollback

* **Rollout.**
  * The owner pushes Steps 1–8 and tags `v1.4.0`.
  * Consumers adopt `cli` and `buildinfo` under their own records, in any
    order. Until they do, nothing changes for them: every addition is
    new API.
* **Rollback.**
  * Before the push, each step is one local commit.
  * After the tag, a consumer that finds a problem stays on its current
    command code, which keeps working against `v1.4.0`.
  * A defect is fixed forward in `v1.4.1`. No history is rewritten.

## Execution Record

### Approval (2026-10-02)

* **The owner:** "approve, commit to main". That approves the PLAN and
  amendments F1–F9.
* **Decisions D-Q1 to D-Q3** are confirmed as recommended:
  * the error prefix is `update failed:`;
  * usage errors exit 1;
  * `--help` writes text only.
* Commits to `main` were authorized for the turn.

### Step 1: records (2026-10-02)

* **The MADR.** Amendments F1–F9 are applied: a block at the head of §5,
  and a marker at each sketch they change.
* **The `v1.3.1` PLAN.** It gains "Release (2026-10-02)" and is
  `complete`. The proxy's `@latest` printed `github.com/maccavelli/go-core-lib v1.3.1`.
* **`docs/architecture.md`.** The current release is `v1.3.1` on
  `351bd6a`.
* **`docs/README.md`.** The `v1.3.1` PLAN is `complete`, and this PLAN is
  `in-progress`.

### Step 2: `buildinfo` (2026-10-02)

**What changed.**

* **`buildinfo/buildinfo.go`** (new) has the PLAN's API, in Go:
  * the stamp variables, `VersionVar` and `KindVar`;
  * `Kind` and `Info`, `Identity`, `Info.Current`, `Info.String` and
    `LDFlags`.

  It imports only `regexp`, `runtime/debug`, `strconv` and `strings`.
  `readBuildInfo` is a package variable, so a test can supply build
  information.
* **`LDFlags`' parameter is named `tag`,** not `version` as the PLAN's
  sketch has it, so that it does not shadow the package's `version`
  variable. The signature is otherwise as planned.
* **`buildinfo/buildinfo_test.go`** has `TestIdentityKinds` (19 cases),
  `TestIdentityIgnoresModuleVersion`, `TestInfoCurrent`, `TestKindString`,
  `TestInfoString` and `TestLDFlags`.
* **`buildinfo/stamp_test.go`** has `TestStampedBinary`. It builds three
  binaries with `GOFLAGS=-mod=mod GOPROXY=off GOWORK=off`: release,
  unstamped, and a `release` stamp on `v1.2`. It runs each one.

**Mutation proofs** (`mut.py`, on a scratch copy; each baseline passed first):

| ID | Mutation | Result |
| :--- | :--- | :--- |
| A1 | the prerelease group is dropped from the tag pattern | killed: `TestIdentityKinds` |
| A2 | the tag check is bypassed | killed: `TestIdentityKinds` |
| A3 | `kind` is compared case-insensitively | killed: `TestIdentityKinds` |
| A4 | an unstamped binary with a module version becomes release | killed: `TestIdentityIgnoresModuleVersion` |
| A5 | the linker variable is renamed (`stampedVersion`) | killed: `TestStampedBinary` |
| A5b | `VersionVar` names `buildinfo.Version` | killed: `TestStampedBinary` |
| A6 | `Current` prefers the module version | killed: `TestInfoCurrent` |

**Checks.**

| Check | Result |
| :--- | :--- |
| `make pre-add-check` on the three files | `3 file(s) clean (gofmt, golangci-lint, go vet, go test, govulncheck)` |
| `make apicheck` | `compatible with v1.3.1` |
| `go test -race -count=1 ./...` | rc 0 |
| `go mod tidy -diff` | rc 0; `go.mod` unchanged |
| Windows test host | `go vet` rc 0; `go test -race` rc 0, with `buildinfo` 3.3 s, so `TestStampedBinary` built its `.exe` binaries; the script tests rc 0 |

**Deviation D1 (2026-10-02): the guard did not run for Step 2's commit.**

* **Found.**
  * The command that committed Step 2 ran the disclosure guard and the
    commit with `;` between them. So `bf7221a` was made although the guard
    exited 1.
  * The exit came from the session's wrapper script, not from a finding.
    The wrapper fetched `origin/main` into a scratch clone, and once local
    `main` was ahead that fetch was a rewind, which git refuses. Step 1's
    run passed only because `main` still equalled `origin/main`.
  * Debugging the wrapper with shell tracing echoed the owner's shell
    profile into the session, credentials included. The trace file was
    deleted, and the owner was told to rotate the exposed token.
* **Decision.** The owner said "proceed".
  * The wrapper is replaced by a Python script. It runs `git` and the
    guard with a minimal environment, and no local shell script is run
    again.
  * The guard was proven on a planted home path, and rejected it (rc 1).
  * It then passed over `351bd6a..bf7221a`, Steps 1 and 2 (rc 0).
  * From Step 3 on, each commit runs only if the guard has passed, in
    the same command (`&&`).
* **Not changed.** No file was added to any step's scope. `bf7221a` itself
  needed no change.

### Step 3: `selfupdate.UserAgent` (2026-10-02)

**What changed.**

* **`selfupdate/useragent.go`** (new) has `UserAgent` and the token
  sanitiser, as planned. The empty token is the constant `uaUnknown`,
  which `goconst` asked for.
* **`selfupdate/useragent_test.go`** (new) has `TestUserAgent`, a table of
  nine cases that each check the exact output, `validateUserAgent` and the
  shape. It also has `FuzzUserAgent`, which checks the same shape and
  check, and that a second pass changes nothing.
* **`Makefile`.** `fuzz` passes `-m 5`.

**Checks.**

| Check | Result |
| :--- | :--- |
| `go test -fuzz FuzzUserAgent -fuzztime 20s` | 751,374 executions, no failure |
| `make fuzz FUZZTIME=2s` | `5 fuzz targets ran clean in ./selfupdate` |
| `make pre-add-check` on the two Go files | first run: `goconst` (`"unknown"` three times) and `gocritic` (`len(s) > 0`); both fixed in the code; second run: `2 file(s) clean` |
| `make apicheck` | `compatible with v1.3.1` |
| `go test -race -count=1 ./...` | rc 0 |
| `go mod tidy -diff` | rc 0 |
| Windows test host, on the final code | first run: every Go check passed; `scripts/check-workflows_test.sh` case R8 got exit 139, a Git Bash segfault, where it wants 1. This step does not touch that script, and its first run had passed. Re-run: every check rc 0, R8 included |

**Mutation proofs:**

| ID | Mutation | Result |
| :--- | :--- | :--- |
| A7 | control characters are kept | killed: `TestUserAgent` |
| A7b | the `unknown` fallback is removed | killed: `TestUserAgent` |
| A7c | `FuzzUserAgent` is renamed `fuzzUserAgent` | killed: `make fuzz` exits 1, finding 4 targets against `-m 5` |
