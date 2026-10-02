---
status: accepted
date: 2026-10-01
decision-makers: owner
---
# Adopt govulncheck v1.8.0 in CI, in the install hints, and on every development host

## Context and Problem Statement

CI pins `govulncheck` at `v1.7.0` (`.github/workflows/ci.yml`; the pin came
from [0001-MADR-scaffold-shared-go-library.md](0001-MADR-scaffold-shared-go-library.md)
§5). The install hints in `Makefile` and `scripts/go-precheck.sh` name
`@latest`. `docs/architecture.md` names `v1.7.0`.

On 2026-10-01, while scaffolding go-tui-lib, the owner chose v1.8.0, the
current release, and widened the choice: "use 1.8.0 across the entire go
environment. ensure it is in place here, on [the Windows test host], the WSL,
and [the Linux host]. pin it in go-core-lib for parity." (Host names are
replaced by roles, as AGENTS.md "Identifiers" requires.)

Evidence (read-only, 2026-10-01):

* **The release.** `golang.org/x/vuln` v1.8.0 is the newest version on the
  module proxy. Its `go.mod` says `go 1.26.0`, below every host's Go 1.27.1.
* **v1.8.0 on this repository.** Built into a scratch directory and run with
  `CGO_ENABLED=0` for `GOOS=linux`, `darwin` and `windows`, it reports
  `No vulnerabilities found.` on each, exit 0. Nothing in the tree needs to
  change for the new pin.
* **The hosts.** Each runs Go 1.27.1 and has v1.7.0 on `PATH`:

  | Host | Where `govulncheck` resolves |
  | :--- | :--- |
  | the macOS development host | `$HOME/go/bin` |
  | the Linux host | `$HOME/.local/bin`, not `$HOME/go/bin` |
  | the Windows test host | `%USERPROFILE%\go\bin` |
  | that host's default WSL distribution (Ubuntu 24.04) | `$HOME/go/bin` |

  The Windows test host has three more WSL distributions. None has Go
  installed, so none runs these gates.
* **Elsewhere in the fleet.** `magic-cli-remote` also pins v1.7.0, and
  ocp-login pins its own version. Neither is named by the owner's request.

## Decision Drivers

* **One version everywhere the gate runs.** The pre-add gate runs
  `govulncheck` from `PATH`, and CI from its pin. Two versions mean a commit
  can pass one and fail the other, which is the drift 0006-MADR closed for
  golangci-lint.
* **Current tooling.** The fleet rule is current, supported, advisory-free
  releases (magic-cli-remote
  `docs/decisions/0169-MADR-standardize-toolchains-on-current-supported-advisory-free-releases.md`).
* **A hint is an instruction.** An install hint of `@latest` installs
  whatever is newest on the day, not the version CI runs.

## Considered Options

* **A. Pin v1.8.0 in CI, pin the install hints to v1.8.0, and install v1.8.0
  on every host where the gate runs.**
* **B. Pin CI only,** and leave the hosts to update themselves.
* **C. Keep v1.7.0** for parity with the other fleet repositories.

## Decision Outcome

Chosen option: **"A"** (the owner, 2026-10-01, quoted above), because only it
makes the local gate and CI run the same scanner, on every host named.

* **CI.** `.github/workflows/ci.yml` installs `govulncheck@v1.8.0`.
* **Hints.** `Makefile` (`vuln`) and `scripts/go-precheck.sh` name
  `govulncheck@v1.8.0` instead of `@latest`.
* **Docs.** `docs/architecture.md` names v1.8.0.
* **Hosts.** Each of the four hosts above gets v1.8.0 through
  `go install golang.org/x/vuln/cmd/govulncheck@v1.8.0`, with `GOBIN` set
  to the directory where `govulncheck` resolves today. Every copy on `PATH`
  is then v1.8.0, so none is shadowed by an old one.
* **go-tui-lib** adopts the same pin in its scaffold
  (go-tui-lib `docs/decisions/0001-MADR-scaffold-charm-tui-library.md` §5).

### Consequences

* Good, because the gate, CI and go-tui-lib run one scanner version.
* Good, because the hints now install what CI runs.
* Neutral, because v1.8.0 reports nothing new here, so no code changes.
* Bad, because magic-cli-remote and ocp-login keep their own pins until
  their records move them. This record does not change them.
* Bad, because a later release must be adopted on four hosts as well as in
  the files. The PLAN's host step is the procedure.

### Confirmation

* `govulncheck -version` reports `Scanner: govulncheck@v1.8.0` on each of
  the four hosts, for every copy on `PATH`.
* `make vuln` and `make pre-add-check` pass on the macOS development host
  with the installed v1.8.0.
* CI is green on the pushed change, and its govulncheck step installs
  v1.8.0.
* `git grep` finds no `govulncheck@v1.7.0` and no `govulncheck@latest`
  outside historical records.

## More Information

* [0006-MADR-adopt-golangci-lint-v2-14.md](0006-MADR-adopt-golangci-lint-v2-14.md):
  the same drift, closed for golangci-lint.
* go-tui-lib `docs/decisions/0001-MADR-scaffold-charm-tui-library.md`: the
  scaffold during which the owner made this decision.
