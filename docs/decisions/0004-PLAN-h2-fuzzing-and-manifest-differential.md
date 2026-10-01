---
status: in-progress
date: 2026-10-01
associated-madr: "0004-MADR-evolve-selfupdate-api-and-tui-support.md"
---
# Implement harness item H2: CI fuzzing and the manifest differential

Associated MADR: [0004-MADR-evolve-selfupdate-api-and-tui-support.md](0004-MADR-evolve-selfupdate-api-and-tui-support.md)

This PLAN implements item H2 of the MADR's §8, "Harness, across every phase":

> Commit the four fuzz targets. Run them for 20 s each in a Linux CI step, and
> add the Python/Go manifest differential (N=5000) to the verifier test.

Phase 0 ([0004-PLAN-v1-0-1-defect-release.md](0004-PLAN-v1-0-1-defect-release.md),
Step 6) did the first sentence, so what is left is the CI fuzz step and the
differential. It also settles H7's last bullet, "the fuzz seed corpus", which
is already met (see "What is already in place").

H4, the end-to-end test against a running copy of the binary, is the other
open harness item. It needs a helper-process harness of its own and gets its
own PLAN.

## Goal

Two defences against a class of defect the current suite cannot see:

* **The fuzz step.** Every pushed tree fuzzes each `selfupdate` fuzz target
  for 20 s on Linux. A crash fails CI and leaves the failing input as a
  downloadable artifact.
* **The manifest differential.** Every `go test` compares the client's
  `ParseSHA256SUMS` with the release verifier's own Python parser, on 5,000
  generated manifests. They must agree on accept or reject, and on the
  entries.

  The cost of disagreeing is high and permanent. A manifest the verifier
  accepts but the client rejects becomes an immutable release nobody can
  install (0003-MADR D1). The 23 parity fixtures pin known cases; the
  differential looks for unknown ones.

Each piece comes with:

* a gate that has been seen to fail on a deliberately broken input;
* green CI on Linux, macOS and Windows;
* no change to `go.mod`, and no change to the exported Go API
  (`make apicheck`).

## Scope

### What is already in place

| Fact | Evidence |
| :--- | :--- |
| Four fuzz targets exist: `FuzzParseSHA256SUMS` (seeded with all 23 parity manifests plus 8 hand seeds), `FuzzGitHubReleaseJSON`, `FuzzSanitize` and `FuzzVersionPolicy`. | `selfupdate/fuzz_test.go`; Phase 0 Step 6 record |
| `go test` runs every seed on every platform, so H7's "fuzz seed corpus" bullet is met. | the `testing` package: with fuzzing off, the target runs on the seeds from `F.Add` and `testdata/fuzz/<Name>` |
| No CI step fuzzes, and no corpus is committed. | `.github/workflows/ci.yml`; there is no `selfupdate/testdata/fuzz/` |
| The verifier's parser is `parse_sums` in a `python3 - <<'PY'` heredoc inside `scripts/verify-selfupdate-release.sh`. It shares one scope with the release checks, so nothing can call it alone. | `scripts/verify-selfupdate-release.sh:53-234` |
| Parity today is 23 fixtures. The Go test checks them in `TestManifestParityFixtures`, and the shell test runs them through the whole verifier. | `selfupdate/manifest_parity_test.go`; `scripts/verify-selfupdate-release_test.sh`, "parity" loop |
| No fixture contains U+2028, U+2029, U+3000 or U+1680, although Go's `unicode.IsSpace` and the Python mirror `go_isspace` both treat them as separators. So the parsers' agreement on Unicode separators beyond NBSP and NEL is untested. | a byte scan of the 23 `SHA256SUMS` files |
| The reusable release workflow runs the verifier from a full checkout of its own commit, at `.core-lib-release-tools`. | `.github/workflows/publish-selfupdate-release.yml:63-72,110-111` |

### Item → step

| Item | Step |
| :--- | :--- |
| records; MADR amendments C1–C2 | 1 |
| the verifier's parser as a module that can be called alone | 2 |
| the manifest differential | 3 |
| the CI fuzz step | 4 |
| documentation and close-out | 5 |

### Out of scope

* **H4**, which gets its own PLAN.
* **Continuous fuzzing services** (OSS-Fuzz, ClusterFuzzLite). Both run Go
  native fuzz targets, but need a Docker build definition and an external
  service or a scheduled workflow. The MADR asks for 20 s per target per
  push, and nothing more.
* **Persisting the fuzz cache across CI runs.** Each run starts from the
  seeds. Carrying the generated corpus forward would need a cache step whose
  key and eviction must then be reasoned about. A crasher, by contrast, is
  kept: as an artifact, and then as a committed seed.
* **Any change to `ParseSHA256SUMS` or `parse_sums` behaviour.** If the
  differential finds a disagreement, the step stops for a deviation; the fix
  is decided then.
* Any `git push` or tag.

### Fixed inputs

| Input | Value | Source |
| :--- | :--- | :--- |
| Go | 1.27.1 | `go.mod` |
| Fuzz flags | `-fuzz` must match exactly one fuzz test in exactly one package. `-fuzztime` bounds a run (the default is forever). A failing input is written to `testdata/fuzz/<Name>/` in the package directory, and is a seed from then on. | `go help testflag`; `go doc testing`, "Fuzzing" |
| Python on CI | `ubuntu-24.04`: 3.12.3. `macos-15`: 3.14.7. `windows-2025`: 3.12.10, where the command is `python`, so `python3` may be absent. | `actions/runner-images` READMEs (`Ubuntu2404`, `macos-15`, `Windows2025`) |
| Artifact upload | `actions/upload-artifact` v7.0.1 = `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a` | `gh api repos/actions/upload-artifact/releases/latest` and its tag ref |
| Differential size | N = 5000 (MADR H2), with a fixed seed | MADR §8 |

### Rules for every step

1. **Order.** Code and tests first. Then each step's proofs, on a scratch
   copy. Then `make pre-add-check` on changed Go files, `shellcheck` on
   changed scripts, `make apicheck`, and a Windows test-host run when the
   step changes something that runs on Windows. Then one
   `git commit --no-edit`.
2. **Proofs.** Every new gate is seen to fail on a deliberately broken
   input, on a scratch copy. Every proof records its failure line.
3. **Existing tests** do not change, except the CI environment of the
   `go test -race` step (Step 3).
4. **Deviations** stop the step and are recorded before work continues.
   That includes the differential finding a real disagreement.

## Proposed MADR amendments

Step 1 applies these to the MADR's §8, each marked *(amended 2026-MM-DD,
0004-PLAN-h2-fuzzing-and-manifest-differential)*. Approving this PLAN
approves them.

| ID | MADR text today | Amendment | Why |
| :--- | :--- | :--- | :--- |
| C1 | H2: "add the Python/Go manifest differential (N=5000) to the verifier test" | The differential is a Go test, `TestManifestDifferential`, run with the package's tests. It drives the verifier's own parser, which moves out of the heredoc into `scripts/selfupdate_manifest.py`. The verifier imports it, unchanged in behaviour and messages. CI requires `python3` for it on Linux and macOS; elsewhere it skips when `python3` is absent. | A differential must test the code that ships, not a copy, and the heredoc cannot be called alone. Driving it from Go puts both parsers in one process run per suite: one Python process for all 5,000 cases, not 5,000 runs of the verifier. |
| C2 | H2: "Run them for 20 s each in a Linux CI step" | `scripts/go-fuzz.sh` fuzzes each fuzz target it finds in `selfupdate` for `FUZZTIME` (default 20s). It fails when it finds fewer than four. CI runs it on Linux through `make fuzz`, and a failure uploads `selfupdate/testdata/fuzz/` as an artifact. | `-fuzz` takes one target per run, so a loop is needed. Discovering the targets means a fifth is fuzzed without a CI edit, and the floor means a loop that finds nothing cannot pass (0001-MADR §6). |

## Implementation Steps

### Step 1: records

1. Apply amendments C1–C2 to the MADR's §8.
2. Add this PLAN's row to `docs/README.md`, with status `in-progress`.
3. *(Added 2026-10-01, at approval.)* Close
   [0004-PLAN-v1-2-0-interaction-stream.md](0004-PLAN-v1-2-0-interaction-stream.md),
   whose CI on the pushed tree was green. Mark it `complete`, with a
   close-out entry and a release entry for the owner's `v1.2.0` tag.
   Update its index row, and `docs/architecture.md`'s current release.
4. One commit; docs only.

### Step 2: the verifier's parser as a module (`scripts/selfupdate_manifest.py` (new), `scripts/verify-selfupdate-release.sh`, `.gitignore`)

**The module.** `scripts/selfupdate_manifest.py` is standard library only
and has no top-level side effects.

* **It holds, moved verbatim:** `MAX_CHECKSUM_LINE`, the digest regular
  expression, `go_isspace`, `go_trim_space` and `go_fields`.
* **`class ManifestError(ValueError)`.**
* **`parse_manifest(raw: bytes, label: str) -> dict[str, str]`** is the old
  `parse_sums`, reading from bytes instead of a path. It raises
  `ManifestError` with the same messages `fail()` printed before, for
  example `SHA256SUMS line 3: malformed digest`.
* **`python3 selfupdate_manifest.py parse DIR`** (Step 3's driver) parses
  every regular file in `DIR`, in sorted order, and prints one JSON object
  per line:
  * `{"case": NAME, "ok": true, "entries": {HEXNAME: DIGEST}}`, or
  * `{"case": NAME, "ok": false, "error": MESSAGE}`.

  Names are hex-encoded through `surrogateescape`, so names that are not
  valid UTF-8 compare byte for byte. A usage error exits 2.

**The verifier.**

* It computes its own directory and runs `python3 -B -`, passing that
  directory as the first argument. `-B` writes no `__pycache__` into the
  tools checkout.
* The heredoc puts that directory on `sys.path`, imports `ManifestError`
  and `parse_manifest`, and replaces its `parse_sums` block with:

  ```python
  try:
      sums = parse_manifest(raw, "SHA256SUMS")
  except ManifestError as e:
      fail(str(e))
  ```

* Nothing else in the heredoc changes.

**`.gitignore`** gains `__pycache__/` and `*.py[cod]`. The global rule is
that a Python file in a repository in another language brings its ignore
rules with it.

**Tests.** `scripts/verify-selfupdate-release_test.sh` runs unchanged and
must pass: 23 parity cases, the structure cases and the name cases.
`TestManifestParityFixtures` is unchanged.

**Proofs** (scratch copies):

| Mutation | Must fail |
| :--- | :--- |
| `parse_manifest` stops stripping a trailing `\r` | the shell test's `parity crlf` |
| `parse_manifest` raises on every input (a planted `raise ManifestError`) | the shell test, on every accepting case: this proves the verifier calls the module, not a copy |

**Checks.** `shellcheck scripts/*.sh`. The Windows test host runs the shell
test (it has Python), and must pass.

### Step 3: the manifest differential (`selfupdate/manifest_differential_test.go` (new), `.github/workflows/ci.yml`)

**The test** is `TestManifestDifferential`, in package `selfupdate`.

1. **Python.**
   * It finds `python3` with `exec.LookPath`.
   * When it is absent, the test skips, unless
     `SELFUPDATE_REQUIRE_PYTHON=1`, in which case it fails.
   * The CI `go test -race` step, which runs on Linux and macOS, sets that
     variable. Windows may skip, because its runner names Python `python`.
2. **Inputs.**
   * `N` is 5000 and the seed is a fixed constant.
   * `SELFUPDATE_DIFFERENTIAL_N` and `SELFUPDATE_DIFFERENTIAL_SEED` override
     them for a longer local hunt, and the test logs both.
   * The generator is `math/rand/v2` PCG with that seed, so a failure
     reproduces exactly.
3. **The generator** builds manifests from parts, chosen at random per
   line. About one case in ten is raw random bytes instead.
   * **Digests:** valid lowercase, valid uppercase, mixed case, 63 and 65
     characters, a non-hex character, and full-width digits.
   * **Separators:** space, two spaces, tab, `\v`, `\f`, U+0085, U+00A0,
     U+1680, U+2028, U+2029, U+3000, U+001C to U+001F (Python's
     `isspace` counts these and Go's does not), and none.
   * **Names:** a valid name, a `*` marker, a doubled marker, `/` and `\`,
     `.` and `..`, a leading or trailing space, invalid UTF-8, a name that
     duplicates an earlier one, and a long name.
   * **Lines:** blank, `#` comments with leading whitespace, three fields,
     CRLF, a lone CR, `\r\r\n`, a BOM at the start, and no final newline.
   * **Length:** lines of 4094 to 4097 bytes, around `maxChecksumLine`.
4. **Running.** The cases are written as files in `t.TempDir()`. One
   `python3 -B ../scripts/selfupdate_manifest.py parse DIR` run parses them
   all, and each is compared with `ParseSHA256SUMS`.
5. **Agreement means:**
   * both accept with identical entries (hex names, digests), or both
     reject;
   * an exact result count, so a case Python dropped fails;
   * error messages are not compared, because they differ by design.
6. **Floors.** At least 15% of cases must be accepted and at least 15%
   rejected. This stops a generator that drifts to all-garbage from passing
   trivially.
7. **Reporting.** It fails with at most ten disagreements, each printing the
   input (`%q`), both verdicts and the case's index, so the case can be
   added as a parity fixture.

**CI.** The `go test -race` step's `env:` gains
`SELFUPDATE_REQUIRE_PYTHON: "1"`.

**Proofs** (scratch copies):

| Mutation | Must fail |
| :--- | :--- |
| `go_isspace` drops the `Zl` and `Zp` categories | `TestManifestDifferential`, a U+2028 or U+2029 case. **No parity fixture catches this.** |
| Python's line limit `>=` becomes `>` | `TestManifestDifferential`, a 4096-byte line |
| the driver drops its last case | the count check |
| the generator emits only random bytes | the accept floor |
| `python3` missing, with `SELFUPDATE_REQUIRE_PYTHON=1` (a PATH with no `python3`, under `env -u BASH_ENV`) | a failure, not a skip |

**If the unmutated test fails,** the parsers disagree today. That is a
release-blocking defect, and a deviation: the step stops with the
disagreeing inputs. Each one becomes a parity fixture once the owner has
decided which parser is right.

### Step 4: the CI fuzz step (`scripts/go-fuzz.sh` (new), `scripts/go-fuzz_test.sh` (new), `Makefile`, `.github/workflows/ci.yml`)

**The script.** `scripts/go-fuzz.sh [-t FUZZTIME] [-m MIN] PKG`:

1. It lists the package's fuzz targets with
   `go test -list '^Fuzz' PKG`, keeping lines that start with `Fuzz`.
2. Fewer than `MIN` (default 4) exits 1, naming the count.
3. For each target, in order, it runs
   `go test -run '^$' -fuzz "^NAME\$" -fuzztime FUZZTIME PKG`. The default
   time is `20s`. The first failure stops the loop with that exit status,
   after printing where Go wrote the input
   (`PKG/testdata/fuzz/NAME/`).
4. A usage error exits 2.

The status of every command is captured before anything filters its
output.

**The script's test.** `scripts/go-fuzz_test.sh` builds throwaway modules in
a temporary directory, and runs the script on each with `-t 2s`:

* one with two passing targets and `-m 2`: exits 0 and fuzzes both;
* the same with `-m 3`: exits 1, with the count;
* one with a target that fails on a seed-reachable input: exits non-zero,
  and `testdata/fuzz/FuzzBoom/` holds a file;
* bad arguments: exits 2.

**`Makefile`.** `FUZZTIME ?= 20s`, and a `fuzz` target runs
`./scripts/go-fuzz.sh -t $(FUZZTIME) ./selfupdate`.

**CI** (Linux only), after "go test -shuffle":

* `fuzz`: `make fuzz`;
* `fuzz corpus` runs `if: failure()`, using `actions/upload-artifact` at
  `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1`, with
  `path: selfupdate/testdata/fuzz/`, `if-no-files-found: ignore` and
  `retention-days: 14`;
* the "API compatibility" step also runs `./scripts/go-fuzz_test.sh`.

The time cost is about 4 × 20 s plus one instrumented build per target.

**Proofs** (scratch copies):

| Mutation | Must fail |
| :--- | :--- |
| the script ignores `go test`'s status | `go-fuzz_test.sh`, the failing-target case |
| the floor check removed | `go-fuzz_test.sh`, the `-m 3` case |
| the target list is always empty | `go-fuzz_test.sh`, the passing case (fuzzed nothing), and the floor |
| a planted `t.Fatal` on a seed in `FuzzSanitize` | `make fuzz` exits non-zero, and `selfupdate/testdata/fuzz/FuzzSanitize/` gains the input |

**Checks.** `shellcheck`, `actionlint`, and `check-workflows.sh` with
`--rule expressions` on `ci.yml`. The script runs no `${{ }}`.

### Step 5: documentation and close-out (`docs/architecture.md`, `docs/README.md`, `AGENTS.md`, this PLAN)

1. **`docs/architecture.md`, "Tooling":** `make fuzz`, `scripts/go-fuzz.sh`,
   `scripts/selfupdate_manifest.py`, the differential, and the two new CI
   steps.
2. **`docs/README.md`** gains "I want to…" rows:
   * "fuzz locally";
   * "know what to do when CI finds a crasher".

   The answer to the second: download the artifact, copy the file into
   `selfupdate/testdata/fuzz/<Name>/`, fix the defect, and commit the file
   as a regression seed.
3. **`AGENTS.md`, "Pre-add checks":** one sentence noting that `go test`
   runs the differential when `python3` is present.
4. **Verification** is the Phase 2 PLAN's list.
5. **Status.** This PLAN is marked `complete` only after CI is green on the
   pushed tree, with the fuzz step and the differential (no skip) visible
   in the Linux log.

## Verification

* Every step's proofs fail as listed.
* `make pre-add-check`, `make lint` and `make apicheck` pass.
* `go test -race -count=1 ./...` passes, with
  `SELFUPDATE_REQUIRE_PYTHON=1`, on macOS.
* `go test -shuffle=on -count=2 ./...` passes.
* `make fuzz` passes locally.
* `scripts/verify-selfupdate-release_test.sh` and
  `scripts/go-fuzz_test.sh` pass.
* The Windows test host passes `go vet ./...`, `go test -race -count=1 ./...`
  and the verifier's shell test.
* **After the owner's push:** CI is green on all three operating systems.
  The Linux log shows four fuzz runs and the differential's logged seed.

## Rollout and Rollback

* **Rollback.** Each step is one commit and can be reverted alone. Reverting
  Step 2 restores the heredoc parser, and Step 3 must then be reverted too.
* **Release tooling.** Step 2 changes how the verifier is laid out, not
  what it does. Consumers pick it up with the next go-core-lib tag, because
  the reusable workflow runs from the tag's commit. The shell test is the
  proof that behaviour and messages are unchanged.
* **CI time.** About two minutes more on the Linux leg.
* **No tag is required.** Nothing in the module's Go API changes.

## Execution Record

### Approval (2026-10-01)

The owner approved this PLAN and amendments C1–C2 ("proceed"), after
pushing the `v1.2.0` tag. Step 1 gained item 3, the Phase 2 PLAN's
close-out, at that point.

### Step 1: records (2026-10-01)

* The MADR's H2 bullet gained amendments C1 and C2.
* This PLAN was indexed as `in-progress`.
* [0004-PLAN-v1-2-0-interaction-stream.md](0004-PLAN-v1-2-0-interaction-stream.md)
  was closed. It is marked `complete`, citing CI run `36882798775` on
  `cfc95c8`, with a release entry for the `v1.2.0` tag: run `36884014511`,
  `success`. Its index row reads `complete`.
* `docs/architecture.md` names `v1.2.0` on `cfc95c8` as the current
  release.
