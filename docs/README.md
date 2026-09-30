# go-core-lib documentation

## Records

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | MADR | [Scaffold go-core-lib as a Go 1.27.1 shared library](decisions/0001-MADR-scaffold-shared-go-library.md) | accepted |
| 0001 | PLAN | [Implement the library scaffold](decisions/0001-PLAN-scaffold-shared-go-library.md) | complete |
| 0002 | MADR | [Re-home `selfupdate` and its release tooling from mcplib as v1.0.0](decisions/0002-MADR-rehome-selfupdate-from-mcplib.md) | accepted |
| 0002 | PLAN | [Implement the `selfupdate` re-home](decisions/0002-PLAN-rehome-selfupdate-from-mcplib.md) | in-progress |
| 0003 | MADR | [Fix every debugging-pass finding before v1.0.0](decisions/0003-MADR-remediate-debugging-pass-findings.md) | accepted |
| 0003 | PLAN | [Implement the debugging-pass fixes](decisions/0003-PLAN-remediate-debugging-pass-findings.md) | in-progress |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| see what is in this repository today | [architecture.md](architecture.md) |
| move a program from `mcplib/selfupdate` to this module | [guides/migrating-from-mcplib-selfupdate.md](guides/migrating-from-mcplib-selfupdate.md) |
| know why `selfupdate` moved here, and what changed on the way | [0002-MADR](decisions/0002-MADR-rehome-selfupdate-from-mcplib.md) |
| know why `bridge-release` is gone | [0002-MADR, §3](decisions/0002-MADR-rehome-selfupdate-from-mcplib.md#3-what-changes-in-transit-and-nothing-else) |
| know why lint runs three times | [0002-MADR, §5](decisions/0002-MADR-rehome-selfupdate-from-mcplib.md#5-lint-covers-every-target-the-code-builds-for) |
| contribute: checks, records and commit rules | [AGENTS.md](../AGENTS.md) |
| know what may be imported here, and what never may | [0001-MADR, §3](decisions/0001-MADR-scaffold-shared-go-library.md#3-toolchain-and-dependencies) |
| know why the gates were never taught to pass on an empty module | [0001-MADR, §6](decisions/0001-MADR-scaffold-shared-go-library.md#6-gates-are-not-taught-to-pass-on-nothing) |
| see what the debugging pass found, and how each finding is fixed | [0003-MADR](decisions/0003-MADR-remediate-debugging-pass-findings.md) |
| know how this repository's tooling differs from `go-llmprovider-sdk`'s | [0001-MADR, §5](decisions/0001-MADR-scaffold-shared-go-library.md#5-deliberate-differences-from-go-llmprovider-sdk) |
| see how the scaffold's checks were proven | [0001-PLAN, execution record](decisions/0001-PLAN-scaffold-shared-go-library.md#execution-record) |
