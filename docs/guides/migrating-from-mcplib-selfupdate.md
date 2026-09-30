# Migrating from `mcplib/selfupdate` to go-core-lib

For a program that imports `github.com/maccavelli/mcplib/selfupdate`, or that
publishes its releases through `mcplib`'s `publish-selfupdate-release.yml`.
The package and workflow here are `mcplib` `v1.6.0`'s, with the same Go API,
so the move is mechanical. Why they moved, and exactly what changed, is in
[0002-MADR](../decisions/0002-MADR-rehome-selfupdate-from-mcplib.md).

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
`github.com/maccavelli/go-core-lib/selfupdate`. No identifier, signature or
behaviour your program can observe changes. The package name is still
`selfupdate`, so no call site changes. Then:

```bash
go mod tidy
```

If `selfupdate` was your last `mcplib` import, `mcplib` and everything it
brought in (the MCP go-sdk among them) leave your `go.mod`.

## 3. The release workflow

In the job that publishes your release, change the `uses:` line and delete
`bridge-release`:

```diff
-    uses: maccavelli/mcplib/.github/workflows/publish-selfupdate-release.yml@<mcplib SHA> # mcplib v1.x.y
+    uses: maccavelli/go-core-lib/.github/workflows/publish-selfupdate-release.yml@<v1.0.0 commit SHA> # go-core-lib v1.0.0
     with:
       artifact-name: …
       products-json: …
       platforms-json: …
       extra-assets-json: …
-      bridge-release: false
```

- **Pin the full commit SHA of the `v1.0.0` tag, never the tag name.**
  Resolve it with
  `git ls-remote https://github.com/maccavelli/go-core-lib refs/tags/v1.0.0`.
- **`bridge-release` must go,** even when it is `false`. The workflow no
  longer declares it, and GitHub rejects an input the called workflow does
  not define. It only ever permitted `magic-cli-remote` `v0.16.0`, which is
  already published.
- `artifact-name`, `products-json`, `platforms-json` and
  `extra-assets-json` mean exactly what they did. The job still needs
  `contents: write`, `id-token: write` and `attestations: write`.

## 4. Check

- `go build ./...`, `go vet ./...` and your full `go test ./...` pass.
- `grep -rn 'mcplib/selfupdate' --include='*.go' .` finds nothing.
- `grep -rn 'maccavelli/mcplib/.github/workflows' .github` finds nothing.
- Your release job's `with:` block has no `bridge-release`.
- The first tag you push after the change produces a complete, immutable
  release. A tag-only job cannot be tested before a tag, so plan that tag
  as the check.
