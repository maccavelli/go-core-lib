# go-core-lib documentation

## Records

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | MADR | [Scaffold go-core-lib as a Go 1.27.1 shared library](decisions/0001-MADR-scaffold-shared-go-library.md) | accepted |
| 0001 | PLAN | [Implement the library scaffold](decisions/0001-PLAN-scaffold-shared-go-library.md) | complete |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](architecture.md) |
| contribute: checks, records and commit rules | [AGENTS.md](../AGENTS.md) |
| know what may be imported here, and what never may | [0001-MADR, §3](decisions/0001-MADR-scaffold-shared-go-library.md#3-toolchain-and-dependencies) |
| know why CI and `make lint` fail until the first package lands | [0001-MADR, §6](decisions/0001-MADR-scaffold-shared-go-library.md#6-gates-are-not-taught-to-pass-on-nothing) |
| know how this repository's tooling differs from `go-llmprovider-sdk`'s | [0001-MADR, §5](decisions/0001-MADR-scaffold-shared-go-library.md#5-deliberate-differences-from-go-llmprovider-sdk) |
| see how the scaffold's checks were proven | [0001-PLAN, execution record](decisions/0001-PLAN-scaffold-shared-go-library.md#execution-record) |
