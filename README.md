# Desky

Reopen a project's editor, terminal, desktop applications and saved web pages
from one command. Desky's `desk` CLI remembers recent projects and lets you save
and pin URLs for your next visit. Applications keep running after the CLI exits;
Desky does not manage environments, own windows or shut applications down.

**Current status:** workspace registration, recent-project selection, personal
URLs and native launch adapters are implemented. Windows has desktop smoke
evidence with follow-up checks outstanding. macOS/Linux desktop qualification
remains open; passing automated tests does not establish desktop support.
See [platform requirements](#platform-requirements) and the
[recorded evidence](PILOT.md#windows-desktop-follow-up-2026-10-03).

## Quick start

Install the Go version specified by [go.mod](go.mod), or a compatible newer
release, and VS Code (the default editor). Run these steps from a local desktop
terminal. The first build may download the dependencies pinned in go.mod/go.sum.

From the Desky repository root, verify and build:

```sh
go run ./tools/verify
```

This checks formatting, vets and tests the code, and builds `bin/desk.exe` on
Windows or `bin/desk` on macOS/Linux. It does not install the binary or change PATH.

**Windows / PowerShell:** save the executable's absolute path, then switch to
your project. Replace the example project path with an existing local directory.

```powershell
$desk = (Resolve-Path ./bin/desk.exe).Path
Set-Location 'C:\path\to\your-project'
& $desk init
& $desk open . --dry-run
& $desk .
```

**macOS/Linux / shell:** the equivalent sequence is below. Native desktop
behavior on these platforms still needs qualification.

```sh
desk_bin="$(pwd)/bin/desk"
cd '/path/to/your-project'
"$desk_bin" init
"$desk_bin" open . --dry-run
"$desk_bin" .
```

`init` creates `workspace.toml` with a stable project ID and one editor resource,
and registers the checkout outside the repository. It does not stage the file
in Git; repeating a completed init refuses to overwrite it. For an already
initialized project, skip `init`.

The dry run previews the launch recipe without opening applications or writing
state. The final command asks you to approve that recipe, then requests the
editor launch. Approval belongs to this checkout and effective recipe; changing
a target or launcher profile requires approval again. The invoking shell's
directory stays unchanged by Desky.

## Everyday usage

Below, `desk` means a binary on your PATH. Otherwise use `& $desk` in PowerShell
or `"$desk_bin"` in the shell above. Run `desk --help` for the full command catalog.

```sh
desk                         # Choose a recent project
desk .                       # Open the current project
desk list                    # List registered checkouts
desk info --json              # Inspect identity, resources and approval state
desk url add 'https://example.com/docs' --title 'Project docs' --pin
desk url list --all           # Include archived URLs and show conflicts
```

A pinned URL opens on the next entry. Omit `--pin` to save it without opening it
automatically. Use `desk url unpin <id>` to stop opening it, or
`desk url archive <id>` to archive it; `restore` returns it to saved status.
These commands use the personal resource ID from the listing and preserve history.
Adding an existing normalized personal URL keeps its current ID, title and status.
Shared URLs are read-only to these commands and are deduplicated with personal pins.

In the recent picker, type a query to filter, press Enter for the first displayed
item, use `/` to clear, or `:q` to cancel. Ambiguous names show checkout paths.
Scripts require exact selectors; JSON mode and redirected input never prompt.
For scripted entry, inspect `desk open . --dry-run`, then explicitly approve its
digest with `desk open . --trust <digest>`.

## Configuration and privacy

Desky keeps three kinds of data separate:

| Scope | Contents | Sharing |
|---|---|---|
| Project: `workspace.toml` | Stable workspace ID and editor, terminal, URL or named-app resources | May be committed with the project; contains no launcher commands |
| Personal: `personal_data_dir` | Intentionally saved URLs and their revision history | Outside the checkout; may be synchronized by your own filesystem provider |
| Device: configuration and state | Launcher paths, app profiles, checkout registry, recency and approvals | Local to this device; approvals are not transferred between checkouts |

Use `desk config path` to locate device TOML, even before the file exists, and
`desk config get personal_data_dir` to inspect the effective personal directory.
Edit TOML directly using the [configuration examples](SPEC.md#configuration-contracts).
Add terminal, URL and named-app resources to `workspace.toml` as needed; `init`
does not add them. Executable paths and arguments belong in device profiles.

An explicit workspace ID reconnects copied personal URLs across clones or devices.
Directory-only identity depends on its original local registry association.
To relocate personal data, close Desky, copy the entire personal directory,
verify the copy, then update device TOML. A custom directory must already exist;
unavailable storage never silently falls back to the default. See
[storage locations and durability](SPEC.md#storage-and-durability).

Desky does not fetch saved URLs, scrape titles or read the clipboard. URLs can
contain sensitive query strings; store only what you intend to retain. Ordinary
output omits URLs used as default titles; `--json` includes full URL snapshots.
Metadata inspection and dry runs do not register a checkout or write state.

## Platform requirements

| Platform | Default launchers and required setup | Desktop verification |
|---|---|---|
| Windows | Native VS Code, Windows Terminal and the default browser; configure absolute `.exe` paths if defaults are unavailable | Windows 11/amd64 smoke evidence; terminal directory, browser contents and complete repeat-open follow-up remain open |
| macOS | `code --reuse-window`, Terminal through `/usr/bin/open`, and the default browser | Adapter and native automated tests implemented; desktop qualification open |
| Linux | `code --reuse-window` and `xdg-open`; terminal resources require an explicit `launchers.terminal` profile | Adapter and native automated tests implemented; desktop qualification open |

On macOS/Linux, the default `code` executable must resolve from an absolute PATH
entry outside the checkout. Linux terminal profiles must use that terminal's
directory flag and exactly one `{path}` argument; see the
[profile contract](SPEC.md#configuration-contracts). Unix profiles may use
executable shebang scripts. Windows network checkouts are unsupported.

Launching requires a local desktop session. Preflight rejects SSH, WSL and
detected headless sessions; metadata inspection and URL saving still work there.
Windows requires a visible process window station. Session detection does not
prove that an application is healthy.

Successful dispatch means a launch request was issued, not that a window is ready.
Dispatch helpers have a five-second deadline; a timeout means the outcome is
unknown and does not kill the application. Desky attempts remaining resources
after a runtime failure and reports each result. Exact OS/app versions and
outstanding desktop checks are recorded in [PILOT.md](PILOT.md#windows-desktop-follow-up-2026-10-03).

## Troubleshooting and recovery

- **Launcher unavailable or approval needed:** inspect `desk open . --dry-run`
  and `desk config get launchers`. Check the local desktop session and configured
  executable paths, then approve the displayed recipe from an interactive terminal.
- **Some resources failed:** inspect all results before retrying; applications
  may already be open. A state-write failure after launch is reported separately.
  Conflicting or damaged personal URLs are skipped while healthy resources open.
  Exit 9 indicates skipped resources, or exit 8 when dispatch also failed.
- **URL conflicts:** inspect `desk url list --all --json`. If every revision head
  agrees on URL/title, explicitly choose a status, for example
  `desk url unpin <id> --resolve` to keep the URL saved without opening it.
  Malformed or incomplete records, differing URL/title and unknown schemas need
  recovery from backup/provider history. Preserve all files; Desky never chooses
  a timestamp winner. Failure to enumerate the overall store blocks entry.
- **Interrupted init:** rerun `desk init` in the same checkout. It resumes only
  the exact prepared configuration. If `workspace.toml` differs from the device
  `pending-init.json` journal, preserve both and restore the prepared configuration
  from the journal's base64 `config` field before retrying. Restore an unavailable
  pending checkout before initializing another one.
- **Damaged device state:** preserve the damaged file and, while no Desky commands
  are running, restore a known-good `registry.json.bak` or `trust.json.bak` from
  the [device state directory](SPEC.md#storage-and-durability). Clearing device
  state does not delete personal URLs, but directory-only identities need their
  original association restored. Approval restoration does not transfer trust to
  another checkout.
- **Interrupted URL save:** retrying keeps any registered identity and reuses an
  already-published matching URL. If an incomplete resource directory remains,
  preserve it and inspect backup/provider history; it may contain incoming data.
  Filesystems must support hard links for new-file publication; unsupported
  filesystems fail without overwriting data.

The [storage contract](SPEC.md#storage-and-durability) describes journals,
revision history, publication and recovery limits in detail.

## Development and verification

The authoritative [verification runner](tools/verify/main.go) is shared with
[CI](.github/workflows/verify.yml). From the repository root:

```sh
go run ./tools/verify             # Formatting, vet, tests and native build
go run ./tools/verify -race -cross # Also race tests and six cross-platform builds
```

Race checks require a supported C toolchain. Cross-builds cover Windows, macOS
and Linux on amd64/arm64; compilation does not verify desktop launching. Native
CI exercises persistence, concurrency, recovery, privacy and process boundaries
on all three OSes. See [PILOT.md](PILOT.md) for actual results and remaining gates.

For Linux desktop qualification from Windows, the optional
[installed Ubuntu VM workflow](tools/vm/README.md) transfers committed source to
an existing VM and records isolated reports. VM creation and OS installation are
outside that tooling. Real transfer/bootstrap and desktop acceptance remain open.

| Document | Purpose |
|---|---|
| [AGENTS.md](AGENTS.md) | Repository maintenance rules, reading order and autonomy pilot policy |
| [internal/README.md](internal/README.md) | Implemented features, code ownership, entry points and tests |
| [SPEC.md](SPEC.md) | Selected behavior, configuration schemas, architecture and platform contracts |
| [IMPLEMENTATION.md](IMPLEMENTATION.md) | Incremental acceptance and remaining verification work |
| [PILOT.md](PILOT.md) | Verification evidence, desktop matrix, outstanding gates and pilot history |
| [plan.md](plan.md) | Original product intent and boundaries |
| [namespace.md](namespace.md) | CLI design intent, provisional examples and future scope |

The source briefs include proposals and future features. Use the implemented CLI
and SPEC.md for current behavior; preserve that distinction when contributing.
