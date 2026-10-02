# Implementation navigation

Read the affected contracts in [SPEC.md](../SPEC.md) before changing behavior.
Workspace metadata and Windows entry are implemented. Personal URL records and
native launch adapters for other platforms remain planned.

| Area | Implementation | Verification |
|---|---|---|
| Workspace discovery, selectors, initialization and identity | [workspace/service.go](workspace/service.go) | [workspace/service_test.go](workspace/service_test.go) |
| Shared/device TOML and storage locations | [workspace/config.go](workspace/config.go) | [workspace/config_test.go](workspace/config_test.go) |
| Registry validation, backups and init recovery | [workspace/registry.go](workspace/registry.go) | Recovery/concurrency cases in workspace/service_test.go |
| Exact entry selectors, identity rebinding and recency transaction | [workspace/entry.go](workspace/entry.go) | [launch/launch_test.go](launch/launch_test.go) |
| Read-only launch planning, digest consent and dispatch results | [launch/launch.go](launch/launch.go) | [launch/launch_test.go](launch/launch_test.go) |
| Windows executable/URL dispatch and default launcher resolution | [platform/native_windows.go](platform/native_windows.go), [boundary](platform/platform.go) | [platform/native_windows_test.go](platform/native_windows_test.go), desktop smoke in [PILOT](../PILOT.md) |
| CLI arguments, JSON/exit codes and terminal-safe display | [cli/cli.go](cli/cli.go), called by [main](../cmd/desk/main.go) | [cli/cli_test.go](cli/cli_test.go), including separate-process writers |
| Atomic file publication and native locks | [fileio/files.go](fileio/files.go), adjacent OS adapters | [fileio/files_test.go](fileio/files_test.go) |

Workspace owns the registry and identity rules. Launch owns approval storage and
cross-feature entry orchestration; the platform adapter owns native dispatch.
CLI owns output and interaction, including console-only consent. Its Windows
[entry subprocess tests](cli/launch_windows_test.go) check the complete path.
Workspace does not print, prompt, or launch. File IO has no workspace policy.
Tests inject `workspace.Locations` so they never touch the user's actual registry.
The [verification runner](../tools/verify/main.go) owns the full check sequence;
the root README explains invocation. Add deeper guidance only as features grow.
