# Ubuntu development setup

Create and install your own Ubuntu Desktop VM with your preferred VM software,
then clone Desky inside Ubuntu. Run the setup script in that checkout.

Requirements: an installed **Ubuntu 26.04 or 24.04 LTS Desktop amd64** system,
Python **3.12+**, Git, internet access and a normal user with sudo access.
Install Git before cloning if your Ubuntu installation does not include it.
Use the native Linux filesystem, such as `~/src/desky`.

## Clone and set up

In Ubuntu's desktop terminal:

```sh
mkdir -p ~/src
git clone https://github.com/hollowhemlock/desky.git ~/src/desky
cd ~/src/desky
bash tools/vm/bootstrap.sh
./bin/desk --help
```

If already cloned, enter that repository and run `bash tools/vm/bootstrap.sh`.
Run it as your normal user; it requests sudo through Ubuntu's standard prompt.
Choose the branch or commit you want before running setup.

The script checks the checkout against its committed revision, installs missing
compiler tools, certificates, download utilities, `xdg-utils`, GNOME Terminal,
GNOME Text Editor, Firefox and VS Code, then builds `bin/desk`. It uses Ubuntu's
package sources and Microsoft's signed VS Code source. It reads the required Go
version from `go.mod`, reuses a compatible Go installation or downloads the
checksum-verified official toolchain into `~/.local/share/desky-vm/toolchains`.
That toolchain is used by setup and qualification without editing shell profiles.

Rerun the same script after pulling changes or after an interrupted setup. It
checks installed prerequisites each time, reuses package sources and toolchains,
and rebuilds Desky. It performs no general OS upgrade. Completion is recorded
only after successful checks and build in `.cache/vm/bootstrap.json`.

## Optional desktop qualification

From the same checkout inside Ubuntu's logged-in desktop terminal:

```sh
python3 tools/vm/guest.py qualify .
```

This runs the authoritative `go run ./tools/verify -race` checks, then guides two
Desky invocations using isolated configuration, personal data, state and fixtures.
Inspect VS Code, GNOME Terminal's `pwd`, the browser URL/content and GNOME Text
Editor's note. Answer each observation with `p`, `f` or Enter for unverified.
Keep the applications open so their survival after Desky exits can be observed.
Actual consent prompts and desktop-session checks remain enabled.

Each run writes its report, verification log and fixtures under
`.cache/vm-runs/<run-id>/`. The script prints the report path. Copy a selected
`report.json` yourself when sharing results; these files remain ignored by Git.
Reports include the commit, source integrity, OS, architecture, session, installed
versions, automated results and individual observations. Unanswered observations
stay unverified. Reports never update platform support claims automatically.

## Source integrity and local files

Use a checkout whose source matches `HEAD`. Setup and qualification compare every
tracked file's contents, type and executable mode against Git, including symlink
targets. Changed, missing or unexpected files are preserved and reported. Extra
`.go` or `_test.go` files invalidate qualification even if Git ignores them.
Git metadata, the specific setup metadata, `bin/desk` and designated qualification
outputs are allowed; arbitrary files under `bin/` or `.cache/` are not exempt.
Submodules are unsupported. Source integrity and commit identity are checked again
after verification, so a changed checkout cannot produce a passing report for the
original revision. Commit or move your own changes before qualification; the
script never resets or cleans a checkout.

Setup and qualification serialize work within this checkout using
`.cache/vm/operation.lock`. Package versions follow the configured upstream
repositories; reports record actual versions rather than promising frozen ones.
The tools are development helpers separate from Desky's application behavior.

## Migrating from the retired host helper

The PowerShell/VirtualBox provisioning, transfer and collection commands have
been removed. The workflow starts from your own clone inside Ubuntu; it does not
require Guest Control credentials, Guest Additions or a host-side JSON config.

Existing VM disks, exports, local configuration and `%LOCALAPPDATA%/DeskyVM`
artifacts are left in place. Earlier unattended-install answer files can contain
credentials and must remain private. Use a fresh clone for this workflow rather
than an old transferred revision directory.

## Verification status

Linux tests exercise real temporary Git clones, exact source checks, locking,
repeat setup and recovery from a partial setup failure. Package installation and
GUI interaction are mocked in those tests. Actual package installation and native
desktop observations remain acceptance gates; increment 4 remains open.

```sh
bash -n tools/vm/bootstrap.sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/vm/tests -p 'test_*.py' -v
```

CI runs these alongside the existing native verification workflow. The desktop
checklist complements native tests for persistence, locking, recovery, privacy,
missing launchers, absent sessions, quoting, helper failures, timeouts and detached
handles. See [IMPLEMENTATION](../../IMPLEMENTATION.md) and [PILOT](../../PILOT.md).
