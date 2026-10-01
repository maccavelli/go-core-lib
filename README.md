# go-core-lib

A Go library of general-purpose packages shared by the fleet's programs:
code that is neither MCP-specific (that stays in `mcplib`) nor LLM-provider
access (that lives in `go-llmprovider-sdk`). Each capability is its own
package in its own top-level directory. It ships no binary, and it depends on
none of the fleet's other libraries: only the standard library and
`golang.org/x/mod`, `golang.org/x/sys` and `golang.org/x/term`.

Module: `github.com/maccavelli/go-core-lib`

**Documentation:** [docs/](docs/README.md)

## Status

The module requires Go 1.27.1. The current release is `v1.0.0` (commit
`b36ca4494b86e52cf1b4a315554603f1c6ee3a21`):

```bash
go get github.com/maccavelli/go-core-lib@v1.0.0
```

| Package | What it does |
| :--- | :--- |
| [`selfupdate`](selfupdate/) | GitHub Releases discovery, exact assets, SHA-256 integrity, locked replacement of the running binary |

`selfupdate` and its release workflow come from `mcplib` `v1.6.0`, with the
same API
([0002-MADR](docs/decisions/0002-MADR-rehome-selfupdate-from-mcplib.md)).
Before the first release, a debugging pass fixed 42 findings in both
([0003-MADR](docs/decisions/0003-MADR-remediate-debugging-pass-findings.md)).
Among them: updating a running Windows program no longer fails with
"Access is denied", and a failed rollback is no longer reported as a clean
failure. The behaviour a consumer can see is listed in the
[migration guide](docs/guides/migrating-from-mcplib-selfupdate.md#behaviour-you-may-notice).

## Self-update

`selfupdate` discovers GitHub Releases, selects the exact asset
`<product>-<goos>-<goarch>[.exe]`, checks it against `SHA256SUMS` (and the
GitHub digest when present), and replaces the running binary under a lock.
It proves release-asset integrity, not publisher signature authenticity.

The library never reads flags or calls `os.Exit`. The program binds:

- a **source** (`NewGitHubSource`, with a required product/version
  `User-Agent`; `GH_TOKEN` or `GITHUB_TOKEN` is used when set),
- a **version policy** (`NewStrictVersionPolicy`: strict `vMAJOR.MINOR.PATCH`;
  `NewSemverPolicy` adds opt-in prerelease channels),
- an **asset selector** (`NewExactAssetSelector`),
- an **installer** (`StandaloneInstaller` for a plain binary;
  `ManagedInstaller` when a service must stop and start around the replace),
- a **reporter** and a **confirmer** (`NewTextReporter`,
  `NewTerminalConfirmer`).

`ExitCode` maps a run to the process status: 0 when there is no error
(already current, declined, or applied), 10 when a check finds an update,
and 1 for any other error. `selfupdate/example_test.go` shows a complete
standalone binding, a managed-service binding, and `ExitCode`.

An exact `--version` is pinned: a source that returns another tag is an
`ErrIntegrity` failure. Redirects must stay on HTTPS unless they go to a
loopback host. End of input at the confirmation prompt is a decline.

### Publishing releases

Programs publish through the reusable workflow
`.github/workflows/publish-selfupdate-release.yml`, pinned to the full SHA of
a go-core-lib tag commit. It accepts only a complete staged set that matches
the declared products, platforms and extras, refuses a tag that already has a
release (drafts included), attests the files, and publishes an immutable
release. It never `--clobber`s. It is the only supported publication path for
the asset contract. Its `SHA256SUMS` check is the client's own parser,
ported, so it never publishes a manifest the client cannot read. Extra asset
names must match `^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`, and staged entries
must be regular files. The optional `prerelease-channels-json` names the
channels it may publish `vX.Y.Z-NAME.N` prereleases for, which never become
the latest release; the default `[]` publishes stable tags only.

```yaml
release:
  permissions:
    contents: write
    id-token: write
    attestations: write
  uses: maccavelli/go-core-lib/.github/workflows/publish-selfupdate-release.yml@b36ca4494b86e52cf1b4a315554603f1c6ee3a21 # go-core-lib v1.0.0
  with:
    artifact-name: <the uploaded artifact holding the staged release>
    products-json: '["<product>"]'
    platforms-json: '[{"os":"linux","arch":"amd64"},{"os":"windows","arch":"amd64"}]'
    extra-assets-json: '[]'
    # Optional; for prereleases tagged vX.Y.Z-rc.N or vX.Y.Z-beta.N:
    # prerelease-channels-json: '["rc","beta"]'
```

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](docs/architecture.md) |
| move a program from `mcplib/selfupdate` to this module | [the migration guide](docs/guides/migrating-from-mcplib-selfupdate.md) |
| offer a beta or rc channel | [the extending guide](docs/guides/extending-selfupdate.md#offer-a-beta-channel) |
| know why `selfupdate` moved here, and what changed on the way | [0002-MADR](docs/decisions/0002-MADR-rehome-selfupdate-from-mcplib.md) |
| know why the repository is set up the way it is | [0001-MADR](docs/decisions/0001-MADR-scaffold-shared-go-library.md) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
