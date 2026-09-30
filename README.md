# go-core-lib

A Go library of general-purpose packages shared by the fleet's programs:
code that is neither MCP-specific (that stays in `mcplib`) nor LLM-provider
access (that lives in `go-llmprovider-sdk`). Each capability is its own
package in its own top-level directory. It ships no binary, and it depends on
none of the fleet's other libraries.

Module: `github.com/maccavelli/go-core-lib`

**Documentation:** [docs/](docs/README.md)

## Status

Not usable yet. There is no package and no release. The module requires Go
1.27.1.

- The repository is scaffolded: agent rules, lint and pre-add checks, CI and
  the documentation tree
  ([0001-MADR](docs/decisions/0001-MADR-scaffold-shared-go-library.md)).
- The first package will be `selfupdate`, re-homed from `mcplib` under its
  own record. Until it lands, `make test`, `make vet`, `make lint` and
  `make vuln` fail with "no packages". That is expected.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](docs/architecture.md) |
| know why the repository is set up the way it is | [0001-MADR](docs/decisions/0001-MADR-scaffold-shared-go-library.md) |
| contribute: checks, records and commit rules | [AGENTS.md](AGENTS.md) |

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
