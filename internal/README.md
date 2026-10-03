# Implementation navigation

Read the affected contracts in [SPEC.md](../SPEC.md) before changing behavior.
Workspace metadata, personal URL records, the MRU picker and Windows entry are
implemented. macOS/Linux native adapters are implemented with desktop qualification
still open; see the actual verification matrix in [PILOT](../PILOT.md).

| Area | Implementation | Verification |
|---|---|---|
| Workspace discovery, selectors, initialization and identity | [workspace/service.go](workspace/service.go) | [workspace/service_test.go](workspace/service_test.go) |
| Shared/device TOML and storage locations | [workspace/config.go](workspace/config.go) | [workspace/config_test.go](workspace/config_test.go) |
| Registry validation, backups and init recovery | [workspace/registry.go](workspace/registry.go) | Recovery/concurrency cases in workspace/service_test.go |
| Exact entry selectors, identity rebinding and recency transaction | [workspace/entry.go](workspace/entry.go) | [launch/launch_test.go](launch/launch_test.go) |
| Personal storage selection and save-time identity | [workspace/personal.go](workspace/personal.go) | [resources/resources_test.go](resources/resources_test.go) |
| Immutable URL revisions, conflicts and status operations | [resources/store.go](resources/store.go), [resources/service.go](resources/service.go) | [resources/resources_test.go](resources/resources_test.go), [failed-save recovery](resources/recovery_test.go) |
| MRU picker, filtering and URL CLI | [cli/picker.go](cli/picker.go), [cli/urls.go](cli/urls.go) | [cli/urls_test.go](cli/urls_test.go), terminal smoke in [PILOT](../PILOT.md) |
| Read-only launch planning, digest consent and dispatch results | [launch/launch.go](launch/launch.go) | [launch/launch_test.go](launch/launch_test.go) |
| Conflict isolation, pin merging and large-workspace latency | [launch/launch.go](launch/launch.go) | [launch/resources_test.go](launch/resources_test.go), [launch/benchmark_test.go](launch/benchmark_test.go) |
| Windows executable/URL dispatch and default launcher resolution | [platform/native_windows.go](platform/native_windows.go), [boundary](platform/platform.go) | [platform/native_windows_test.go](platform/native_windows_test.go), desktop smoke in [PILOT](../PILOT.md) |
| macOS/Linux profiles and URL helpers | [platform/native_unix.go](platform/native_unix.go) | [platform/native_unix_test.go](platform/native_unix_test.go) |
| Native desktop/remote-session preflight | [platform/session.go](platform/session.go), adjacent OS adapters | [platform/session_test.go](platform/session_test.go), [Windows station](platform/session_windows_test.go), [headless CLI](cli/launch_session_test.go) |
| Detached child survival, arguments/CWD and helper deadlines | [platform/process.go](platform/process.go), native dispatch adapters | [platform/native_process_test.go](platform/native_process_test.go) on each native OS |
| CLI arguments, JSON/exit codes and terminal-safe display | [cli/cli.go](cli/cli.go), called by [main](../cmd/desk/main.go) | [cli/cli_test.go](cli/cli_test.go), including separate-process writers |
| Atomic file publication and native locks | [fileio/files.go](fileio/files.go), adjacent OS adapters | [fileio/files_test.go](fileio/files_test.go), [Unix permissions](fileio/native_unix_test.go) |

Workspace owns the registry and identity rules. Launch owns approval storage and
cross-feature entry orchestration; the platform adapter owns native dispatch.
CLI owns output and interaction, including console-only consent. Its Windows
[entry subprocess tests](cli/launch_windows_test.go) check the complete path.
Workspace does not print, prompt, or launch. File IO has no workspace policy.
Resources owns complete immutable snapshots and revision graph interpretation.
Its service supplies the future browser seam without adding transport. Workspace
owns device locking/identity around mutations; personal revisions contain no
checkout paths or trust. Load reports resource-local problems without throwing
away healthy resources. Launch maps those problems to skipped results, hashes
only shared actions for approval, and attempts healthy pins in resource-ID order.
Resource reads use bounded concurrency with deterministic output; registry
validation reuses filesystem metadata while preserving path-alias checks.
Tests inject `workspace.Locations` so they never touch the user's actual registry.
The [verification runner](../tools/verify/main.go) owns the full check sequence;
the root README explains invocation. Add deeper guidance only as features grow.
