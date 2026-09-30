# Migrating from `mcplib/selfupdate` to go-core-lib

For a program that imports `github.com/maccavelli/mcplib/selfupdate`, or that
publishes its releases through `mcplib`'s `publish-selfupdate-release.yml`.
The package and workflow here are `mcplib` `v1.6.0`'s, with the same Go API
and the fixes of a debugging pass, so the move is mechanical. Why they moved
is in [0002-MADR](../decisions/0002-MADR-rehome-selfupdate-from-mcplib.md);
the fixes, and the behaviour they change, are in
[0003-MADR](../decisions/0003-MADR-remediate-debugging-pass-findings.md).

Do the migration under your own repository's records. This guide is the
checklist, not the authorization.

## 1. Go 1.27.1

go-core-lib requires `go 1.27.1`, so `go get` raises your `go` directive to
at least that. Move it deliberately first, and run your full test suite at
1.27.1 before changing anything else.

## 2. The Go import

```bash
go get github.com/maccavelli/go-core-lib@v1.0.0
```

Replace every `github.com/maccavelli/mcplib/selfupdate` import with
`github.com/maccavelli/go-core-lib/selfupdate`. No identifier or signature
changes, and the package name is still `selfupdate`, so no call site
changes. Then:

```bash
go mod tidy
```

If `selfupdate` was your last `mcplib` import, `mcplib` and everything it
brought in (the MCP go-sdk among them) leave your `go.mod`.

### Behaviour you may notice

Each of these is a fix. None needs a code change unless your program relied
on the old behaviour.

- **Updating a running program on Windows works reliably.** A replacement
  that Windows refused with "Access is denied" while the old image was
  running is now retried.
- **`--version` is pinned.** If the release source returns a different tag
  than the one requested, `Run` fails with `ErrIntegrity` instead of
  installing it.
- **End of input at the `[y/N]` prompt is "no".** It used to be an `io.EOF`
  error and exit 1; now it is a decline, `Declined` with a nil error, exit
  0.
- **An installer that commits nothing is an error.** A custom `Installer`
  returning `Applied: false` with no error used to count as success.
- **`New` rejects nil and typed-nil collaborators,** including a nil element
  in `Verifiers`, which used to panic mid-update.
- **Unrelated release assets no longer matter.** Only the selected binary
  and `SHA256SUMS` are checked, so an extra asset that is large,
  still uploading, or zero-sized no longer blocks updates.
- **Redirects must stay on HTTPS,** except to a loopback host.
- **Events arrive in order:** `verified` before `transforming`, and
  `complete` only after the session (and its lock) is released. A local
  build's check mode says that apply needs `--force`.
- **Every error names the product** (`selfupdate: <product>: …`), and
  `errors.Is` still matches the sentinels.
- **`PendingBackup` is a path.** On Windows it was always the backup's full
  path; the documentation said "basename" and now says "path".

## 3. The release workflow

In the job that publishes your release, change the `uses:` line and delete
`bridge-release`:

```diff
-    uses: maccavelli/mcplib/.github/workflows/publish-selfupdate-release.yml@<mcplib SHA> # mcplib v1.x.y
+    uses: maccavelli/go-core-lib/.github/workflows/publish-selfupdate-release.yml@b36ca4494b86e52cf1b4a315554603f1c6ee3a21 # go-core-lib v1.0.0
     with:
       artifact-name: …
       products-json: …
       platforms-json: …
       extra-assets-json: …
-      bridge-release: false
```

- **Pin the full commit SHA of the `v1.0.0` tag, never the tag name.**
  Tags are annotated, so the tag ref names a tag object, not the commit
  `uses:` needs. Resolve the commit with the peeled ref:
  `git ls-remote https://github.com/maccavelli/go-core-lib 'refs/tags/v1.0.0^{}'`.
- **`bridge-release` must go,** even when it is `false`. The workflow no
  longer declares it, and GitHub rejects an input the called workflow does
  not define. It only ever permitted `magic-cli-remote` `v0.16.0`, which is
  already published.
- `artifact-name`, `products-json`, `platforms-json` and
  `extra-assets-json` mean what they did. The job still needs
  `contents: write`, `id-token: write` and `attestations: write`. Two
  checks are stricter:
  - every extra asset name must match `^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`.
    `install.sh`, `install.ps1` and `<product>-vX.Y.Z-arm64.apk` do; a name
    with a space, a glob character or a leading `-` does not;
  - `SHA256SUMS` must parse exactly as the client parses it: `\n` or
    `\r\n` line endings, and each line under 4096 bytes. `sha256sum`
    output always does.
- The staging artifact must hold regular files only; a symlink is refused.

## 4. Check

- `go build ./...`, `go vet ./...` and your full `go test ./...` pass.
- `grep -rn 'mcplib/selfupdate' --include='*.go' .` finds nothing.
- `grep -rn 'maccavelli/mcplib/.github/workflows' .github` finds nothing.
- Your release job's `with:` block has no `bridge-release`.
- The first tag you push after the change produces a complete, immutable
  release. A tag-only job cannot be tested before a tag, so plan that tag
  as the check.
