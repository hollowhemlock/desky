# Desky

A cross-platform project-context launcher for fast re-entry into a working
directory and its editor, terminal, applications, and saved web resources.

Current state: increments 1 and 2 are available. The Go CLI initializes and
inspects workspaces and opens editor, terminal, named application and shared URL
resources on Windows after recipe approval. Saved URL management, the interactive
picker, and native launch adapters for macOS/Linux remain planned.

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
Successful entry registers a checkout and updates recency once, including partial
launch success. Fuzzy/ambiguous selection requires an exact selector until the
interactive picker is implemented. Info reports `approved`, `approval_required`,
or `not_evaluated` with a reason when launch preflight cannot complete.

On Windows, inspect `desk open . --dry-run` first. Run `desk .` in an interactive
console to approve the displayed recipe, or pass the dry run's digest with
`desk open . --trust <digest>`. JSON mode and redirected input never prompt.
Approval is local to that checkout and effective recipe; changing a target or
launcher profile requires new approval. Neither dry run nor info writes state.
The default editor is native VS Code; the default terminal is Windows Terminal.
Configure absolute native `.exe` paths in device TOML when defaults are unavailable.
Shared HTTP(S) URLs use the default browser. See SPEC for profile schemas.

Entry reports dispatch, not GUI readiness or process ownership. It attempts
remaining resources after a runtime failure and reports all results. A state-write
failure after launch is reported separately; inspect the results before retrying
because applications may already be open. Windows network checkouts are unsupported.
Existing personal workspace stores cause entry to fail explicitly until personal
resources are implemented; they are never silently skipped.

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
Approval corruption similarly preserves `trust.json`; inspect its `trust.json.bak`
before restoring a known-good copy. Restoration never transfers another checkout's
approval. An unfinished init must be recovered before entry can mutate device state.

Windows/amd64 verification includes concurrency/recovery, trust mutation, partial
launches, state-write failures, real argument/CWD preservation and detached-child
survival. The desktop smoke evidence and exact app versions are in
[PILOT.md](PILOT.md). macOS/Linux CI verifies portable logic and explicit unsupported
launch errors; it does not establish native desktop launch support. The next work
is increment 3 in IMPLEMENTATION.md.
