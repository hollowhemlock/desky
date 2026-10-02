# Desky

A cross-platform project-context launcher for fast re-entry into a working
directory and its editor, terminal, applications, and saved web resources.

Current state: implementation increment 1 is available. The Go CLI initializes
workspace configuration, inspects directory/workspace identity, lists registered
checkouts, and reads device configuration. Application launching, URL management
and the interactive picker remain planned.

## Start here

| Document | Role |
|---|---|
| [AGENTS.md](AGENTS.md) | Required reading order, maintenance rules, and the project-local autonomy pilot policy |
| [plan.md](plan.md) | Product intent, boundaries, identity, storage scopes, privacy, and requested next deliverable |
| [namespace.md](namespace.md) | Proposed terminal interface, command semantics, and first-version versus future scope |
| [SPEC.md](SPEC.md) | Selected release boundary, architecture, identity, schemas, CLI, trust, platform launch and browser seam |
| [IMPLEMENTATION.md](IMPLEMENTATION.md) | Incremental acceptance, planned verification, Git integration context and readiness review |
| [internal/README.md](internal/README.md) | Implemented capabilities, ownership, code entry points and tests |
| [PILOT.md](PILOT.md) | Ready-to-use first-task prompt and evaluation sequence |

The source documents mix firm constraints with examples and possible future
features. Preserve their distinctions when deriving the specification. Update
this entry point as concrete specifications and implementation become available;
point to their authoritative sources instead of maintaining parallel inventories.

## Build and use

Install the Go version specified by [go.mod](go.mod) or a compatible newer release.
From this repository, run `go run ./tools/verify`. The
[verification runner](tools/verify/main.go) checks formatting, vets and tests the
module, then builds `bin/desk.exe` on Windows or `bin/desk` on macOS/Linux. It is
also used by [CI](.github/workflows/verify.yml). With a supported C toolchain,
`go run ./tools/verify -race` additionally runs the race detector. Dependencies
are pinned in go.mod/go.sum; the first run may need network access to download them.

Run the resulting executable with `--help` for the implemented command catalog,
owned by [the CLI](internal/cli/cli.go). For example, run `desk init` in a project
directory, then `desk info --json` or `desk list`. Here `desk` means the built
executable's absolute path, or a binary you have placed on your PATH; the build
does not install it or change shell settings. Init creates a workspace.toml with
a stable UUID and editor resource, and registers the checkout outside the repo.
It never stages that file in Git. Repeating a completed init refuses to overwrite.

Use `desk config path` to locate device TOML, and `desk config get personal_data_dir`
to inspect the effective personal directory. Edit configuration using the schema
in SPEC.md. Metadata inspection does not create state or register a checkout;
unregistered directory identities have empty IDs with `registered: false`.
Cloned committed configs can be inspected now; automatic registration on entry
arrives with increment 2. Fuzzy/ambiguous inspection requires an exact selector
until the interactive picker is implemented. Trust is reported as not implemented.

## Recovery and verification limits

Init uses a device-local `pending-init.json` journal across config/registry writes.
If interrupted, rerun init: it completes only the exact prepared configuration,
preserving its original name and IDs. It never overwrites a differing config.
Until recovery completes, read-only commands expose only committed registry data.
If pending config and actual config differ, preserve both and restore the prepared
config (the journal's base64 `config` field) before retrying; do not discard personal
files. An unavailable pending checkout must be restored before another init.

Registry corruption fails explicitly. Preserve the damaged file, inspect
`registry.json.bak` in the device state directory, and restore a known-good backup
while no Desky commands are running. State locations and durability limits are
defined in SPEC.md. Initial filesystem support requires hard links for publishing
new config/journal files; unsupported filesystems fail without overwriting data.

Windows/amd64 metadata tests run locally, including process concurrency and
interruption recovery. macOS/Linux builds and CI are separate evidence; no desktop
launch behavior has been implemented or verified. The next work is increment 2
in IMPLEMENTATION.md. Keep these limits current as further increments land.
