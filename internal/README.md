# Implementation navigation

Read the affected contracts in [SPEC.md](../SPEC.md) before changing behavior.
Only workspace metadata is implemented. No package currently launches applications
or reads/writes personal URL records.

| Area | Implementation | Verification |
|---|---|---|
| Workspace discovery, selectors, initialization and identity | [workspace/service.go](workspace/service.go) | [workspace/service_test.go](workspace/service_test.go) |
| Shared/device TOML and storage locations | [workspace/config.go](workspace/config.go) | [workspace/config_test.go](workspace/config_test.go) |
| Registry validation, backups and init recovery | [workspace/registry.go](workspace/registry.go) | Recovery/concurrency cases in workspace/service_test.go |
| CLI arguments, JSON/exit codes and terminal-safe display | [cli/cli.go](cli/cli.go), called by [main](../cmd/desk/main.go) | [cli/cli_test.go](cli/cli_test.go), including separate-process writers |
| Atomic file publication and native locks | [fileio/files.go](fileio/files.go), adjacent OS adapters | [fileio/files_test.go](fileio/files_test.go) |

Workspace owns the registry and identity rules. CLI owns output and interaction;
workspace does not print, prompt, or launch. File IO has no workspace policy.
Tests inject `workspace.Locations` so they never touch the user's actual registry.
The [verification runner](../tools/verify/main.go) owns the full check sequence;
the root README explains invocation. Add deeper guidance only as features grow.
