# Desky

Reopen a project's editor, terminal, desktop applications and saved web pages
from one command. Desky's `desk` CLI remembers recent projects and lets you save
and pin URLs for your next visit. Applications keep running after the CLI exits;
Desky does not manage environments, own windows or shut applications down.

**Current status:** workspace registration, recent-project selection, personal
URLs and native launch adapters are implemented. The tested Windows/amd64 desktop
flow passed, including repeat opening. Ubuntu 26.04.1/amd64 passed with configured
profiles and the default editor. The Windows noninteractive-session fixture also
passed; macOS desktop qualification remains open. Passing automated tests alone
does not establish desktop support.
See [platform requirements](#platform-requirements) and the
[recorded evidence](PILOT.md#2026-10-21-windows-qualification-after-ubuntu-integration).

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

**macOS/Linux / shell:** the equivalent sequence is below. See the platform
matrix below for tested environments and remaining desktop checks.

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
| Windows | Native VS Code, Windows Terminal and the default browser; configure absolute `.exe` paths if defaults are unavailable | Windows build 26200.9457/amd64 desktop smoke, repeat opening, noninteractive-session fixture and native race tests passed; race setup used MSYS2 UCRT64 GCC |
| macOS | `code --reuse-window`, Terminal through `/usr/bin/open`, and the default browser | Adapter and native automated tests implemented; desktop qualification open |
| Linux | `code --reuse-window` and `xdg-open`; terminal resources require an explicit `launchers.terminal` profile | Ubuntu 26.04.1/amd64 GNOME Wayland smoke with configured profiles and the default editor passed; other environments remain unverified |

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
outstanding desktop checks are recorded in [PILOT.md](PILOT.md#2026-10-21-windows-qualification-after-ubuntu-integration).

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
[CI](.github/workflows/verify.yml). For a fresh clone, install the Go version in
[go.mod](go.mod) or a compatible newer release, configure the compiler as described
below, then check readiness and run verification from the repository root:

```sh
go run ./tools/verify -race-check  # Check setup; build and execute a race probe
go run ./tools/verify -race        # Normal verification plus uncached race tests
go run ./tools/verify             # Formatting, vet, tests and native build only
go run ./tools/verify -race -cross # Also six uninstrumented cross-platform builds
```

`-race-check` is standalone: it cannot be combined with `-race` or `-cross`. It
checks the native target, detector support and compiler, then links and executes
the runner's instrumented test binary with no tests selected. A successful build
with `[no test files]` is not accepted. It may populate Go's caches or download
module dependencies, but runs no project tests and creates no Desky build outputs.
`-race` performs the same prerequisite check before normal verification and the
full uncached race suite. Failures stop verification with diagnostics.

The runner enables `CGO_ENABLED=1` only for race-related child commands. Ordinary
commands retain your environment, and cross-builds use `CGO_ENABLED=0`. Verification
never installs packages or changes persistent Go settings or system PATH.

Run native verification without cross-compilation overrides for `GOOS`/`GOARCH`.
Those settings can prevent the outer `go run ./tools/verify` from starting before
the runner can explain the mismatch. In PowerShell, remove session overrides with
`Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue`; in a Unix shell,
use `unset GOOS GOARCH`. If you intentionally saved overrides with `go env -w`,
inspect them with `go env GOOS GOARCH` and remove those saved overrides explicitly
with `go env -u GOOS GOARCH` before native verification.

**Windows/amd64:** install [MSYS2](https://www.msys2.org/) using its default
`C:\msys64` location. Follow its update instructions, including reopening its
terminal if a core update requests it. In the **MSYS2 UCRT64** terminal, install
the native GCC package:

```sh
pacman -S --needed mingw-w64-ucrt-x86_64-gcc
```

Then select that compiler in the PowerShell session used for verification:

```powershell
$compilerDir = 'C:\msys64\ucrt64\bin' # Adjust for your installation
$env:Path = "$compilerDir;$env:Path"
$env:CC = '"' + (Join-Path $compilerDir 'gcc.exe') + '"'
go run ./tools/verify -race-check
go run ./tools/verify -race
```

The selected compiler must include mingw-w64 runtime 8 or later. The runner checks
that its `--print-file-name libsynchronization.a` resolves an existing absolute
library path, then tests actual linking and execution. Compiler paths containing
spaces must be quoted in `CC`; compiler arguments and wrappers use Go's quoted-field
syntax, without shell expansion. See [Go's requirements](https://go.dev/doc/articles/race_detector#Requirements).

**Linux:** install a native C compiler. On Ubuntu/Debian, use
`sudo apt-get update` followed by `sudo apt-get install build-essential`.
The [Ubuntu bootstrap](tools/vm/README.md) already installs these prerequisites.
Then run `-race-check` and `-race` from your normal shell.

**macOS:** run `-race-check` first. If the Go probe reports a missing compiler or
SDK, install Apple's Command Line Tools with `xcode-select --install`, complete
the installer, and retry. Use `xcode-select -p` to inspect the selected developer
directory. The runner relies on Go's build/execution probe on macOS rather than
requiring a separate GCC installation.

Go detector support is separate from Desky's desktop qualification. In particular,
native Windows/arm64 is not currently supported by Go's race detector; a compiler
installation does not enable it. The runner asks the installed Go toolchain about
support rather than maintaining another platform list. Race detection only covers
executed paths and can substantially increase runtime and memory use; see the
[Go race guide](https://go.dev/doc/articles/race_detector).

CI requires race verification on Windows, macOS and Linux. Windows provisions
UCRT64 GCC and runs the same commands from native PowerShell. Linux additionally
runs cross-builds and the Ubuntu workflow tests. Cross-builds cover Windows,
macOS and Linux on amd64/arm64; compilation does not verify desktop launching.
See [PILOT.md](PILOT.md) for actual results and remaining gates.

For Ubuntu development, create your own VM and clone this repository inside it,
then run `bash tools/vm/bootstrap.sh` to install prerequisites and build Desky.
The [Ubuntu setup workflow](tools/vm/README.md) also provides optional desktop
qualification with isolated reports. Repeated setup with installed prerequisites
and the Ubuntu desktop smoke with configured profiles have passed, as has the
[default editor desktop check](PILOT.md#2026-10-21-default-editor-desktop-qualification).
Installing missing packages on a fresh machine remains unverified. See the
[Ubuntu evidence](PILOT.md#2026-10-20-native-ubuntu-setup-and-launcher-qualification).

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
