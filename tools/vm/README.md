# Installed Ubuntu VM workflow

These optional development tools use an **already-installed Ubuntu Desktop VM**
for Desky development and qualification. They are separate from `desk`: the
application never manages its users' environments.

Requirements: Windows, PowerShell **7.4+**, Git, VirtualBox **7.2**, and an existing
Ubuntu **26.04 or 24.04 LTS Desktop amd64** VM with Python 3.12 or newer and working
Guest Additions. Use an existing normal guest account with sudo access for package
installation. Package installation requires guest internet access.

The helper does not create VMs, install Ubuntu, change hardware/accounts, attach
or detach media, configure SSH/shared folders, or stop VMs. Configure and install
the VM separately. Power state alone does not establish guest readiness.

**Verification status:** deterministic Windows host and Linux guest tests pass.
Real Guest Control transfer, repeated package bootstrap and desktop qualification
remain unverified. An earlier unattended installation stalled during essential
driver loading; that installation workflow has been retired. See [PILOT](../../PILOT.md).

## Select an installed VM

Copy `config.example.json` to `.cache/vm/config.local.json` if that local file
does not already exist. Set `VmName` to the registered name or UUID and
`GuestUser` to your existing Ubuntu account. Local settings stay outside Git.

Alternatively, provide those values explicitly on each invocation:

```powershell
./tools/vm/manage.ps1 status -VmName 'ubuntu-dev'
./tools/vm/manage.ps1 start -VmName 'ubuntu-dev'
./tools/vm/manage.ps1 provision -VmName 'ubuntu-dev' -GuestUser dev
```

Use the installed system, then log in to its desktop. A VM still booting its
installer is not ready. Provisioning rejects detected live-media boot modes
(`boot=casper` or `boot=live`) and live/transient root filesystems such as
overlay or squashfs. A persistent root filesystem is required; installed systems
deliberately using an overlay root are outside this workflow. Root execution,
failed authentication, missing Python and an unavailable Guest Additions service
also prevent source transfer.

| Action | Behavior |
|---|---|
| `status` | Inspect the selected VM's power state without prompting or changing it |
| `start` | Open a powered-off/saved VM; succeed harmlessly if already running |
| `provision` | Check authenticated Guest Control and transfer the current committed source |
| `collect` | Retrieve one explicitly selected qualification report |

`provision` requests the normal user's password at a local masked prompt. Keep
it in your password manager, never in JSON, Git, command arguments or chat.
It reports staging, each file copy and source publication separately. If a
transfer fails, its error identifies the stage while keeping raw guest diagnostics
private. Retry `provision` after addressing the failure; each attempt uses a new
staging directory, and an incomplete transfer is never published.
Each file first lands in a new private subdirectory, then its size and SHA-256 are
checked through authenticated guest execution before an exclusive hard link places
it in staging. Existing files are preserved. This avoids the empty-file behavior
of VirtualBox's `copyto --no-replace` without allowing replacements. A damaged copy
stops the transfer before the publisher runs; partial copies remain in their
attempt directory for inspection.
After successful verification and publication, the guest writes a small
`publication.json` confirmation inside that attempt's staging directory. The host
retrieves only this workflow file and checks its revision, archive checksum,
destination and staging identity. Success does not depend on VirtualBox delivering
the process's final stdout. Missing, incomplete or mismatched confirmations still
fail without exposing raw guest content; retry `provision` to verify and reuse an
intact revision in a new attempt. The local confirmation stays beside the ignored
source archive beneath `.cache/vm/`.
Run the printed bootstrap command inside Ubuntu's logged-in desktop terminal.
Bootstrap uses standard sudo prompts to install missing Git/compiler tools, Go,
VS Code, Firefox, GNOME Terminal and GNOME Text Editor. It checks actual
prerequisites on reruns and performs no general OS upgrade.

## Qualify and collect

From the exported revision directory inside the desktop terminal:

```sh
python3 tools/vm/guest.py qualify .
```

Qualification runs `go run ./tools/verify -race`, saves a verification log and
guides two real Desky entries. Inspect the displayed fixture path, terminal
`pwd`, browser content/URL, editor and note. Answer each observation with `p`,
`f` or Enter for unverified. Leave applications open until their survival is
observed. The script does not close them or bypass headless/SSH checks.

Collect the printed revision/run ID on Windows using the same configured target
(or repeat `-VmName` and `-GuestUser`):

```powershell
./tools/vm/manage.ps1 collect -Revision <full-commit-id> -RunId <run-id>
```

Only the selected report JSON is collected beneath `.cache/vm/reports/`; logs
stay in Ubuntu. Collection never changes observations or support claims. Review
the evidence before updating PILOT manually.

## Configuration and defaults

Explicit arguments override local JSON settings, which override defaults.
Unknown keys are rejected. `-Config` selects another local JSON file; the
default `.cache/vm/config.local.json` is relative to the repository.

| Setting / argument | Default and reason |
|---|---|
| `VmName` | `desky-dev`; select an existing registered name or UUID |
| `GuestUser` | `dev`; override with your existing normal Ubuntu account |
| `GuestRoot` | `~/src/desky`; native guest filesystem, one directory per commit |
| `VBoxPath` | Standard installation location; override for custom installations |

VM names may include spaces and Unicode; quote them in PowerShell. A missing
target is an error for start, provision and collect. The helper never creates
a replacement.

### Migrating from the retired installer

Remove `IsoPath`, `BaseFolder`, `MemoryMB`, `CPUs` and `DiskGB` from old
local configuration. These creation settings and the `create` action are no
longer supported. Select an already-installed VM and account instead.

Old VM disks, mounted media and private installation records remain untouched.
The helper no longer reads ownership/finalization records or cleans up installer
artifacts. Reusing a VM after a manual installation does not require a Desky
completion marker or account changes. Boot its installed system before provisioning.

Earlier unattended artifacts may remain under `%LOCALAPPDATA%\DeskyVM` and
may contain credential-bearing answer files. Keep them private. If you later
clean up the failed installation, inspect that directory and the VM's attached
media first; the helper does not remove or detach them automatically.

## Privacy and reruns

Local configuration, exports and reports belong in ignored `.cache/vm/`.
Temporary password files live in access-restricted directories under
`%LOCALAPPDATA%\DeskyVM` and are removed on success or failure. The helper
authenticates as the normal user; it never changes or authenticates as root.

Source is exported with `git archive HEAD` and checked against the Git tree.
Host line-ending preferences are disabled for that export. Working changes,
ignored files, history and authentication are excluded. Already-committed secrets
are not detected or removed. Submodules and export attributes that omit/transform
committed content are rejected. Transfers use checksums, staging and atomic
publication. Partial `.incoming-*` directories remain for inspection.

Reruns validate exact source contents and reject modifications or unexpected
files, including extra `.go`/`_test.go` files. Those files are preserved. Only
designated metadata, `bin/desk` and workflow run outputs are allowed; arbitrary
files under `bin/` are not exempt. Exports are not automatically pruned.

Host operations and guest publication/bootstrap/checks are serialized. Deadlines
mean an unknown outcome; inspect before retrying because active operations are
not killed. Guest bootstrap records actual versions without freezing upstream
repositories. Go uses a versioned user-owned location when required; shell
profiles and unrelated installations are preserved. Qualification isolates
Desky's XDG storage while retaining the actual desktop session.

### Guest Additions unavailable

First check the selected guest username/password and that the installed system
has finished booting. If necessary, repair Additions inside Ubuntu:

1. Mount the matching bundled `VBoxGuestAdditions.iso` through VirtualBox's Devices menu.
2. Run `sudo apt-get install build-essential dkms linux-headers-$(uname -r)`.
3. Run `sudo sh VBoxLinuxAdditions.run` from the mounted ISO directory; reboot if requested.
4. Check `sudo rcvboxadd status-user` and `sudo rcvboxadd status-kernel`, then retry provisioning.

See [Oracle's repair instructions](https://docs.oracle.com/en/virtualization/virtualbox/7.2/user/guestadditions.html).

## Verification

On Windows (Go builds the fake executable):

```powershell
./tools/vm/tests/host.tests.ps1
```

On Linux:

```sh
bash -n tools/vm/bootstrap.sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/vm/tests -p 'test_*.py' -v
```

CI adds these checks to the existing three-platform workflow. Host tests use a
fake VirtualBox executable, synthetic credentials and stubbed source/report
delivery to check lifecycle orchestration and preservation. Separate host tests
exercise real source export and transfer orchestration against the fake, including
VirtualBox's trailing-slash directory semantics, no-replace behavior and failures
at every transfer stage. Linux tests cover actual publication/integrity.
Windows retains bounded test artifacts for diagnosis; none contain real
credentials. These checks are not desktop acceptance.

| Contract | Evidence |
|---|---|
| Persistence, locking, recovery and privacy | Existing native suite through `tools/verify` |
| Missing launchers and safe lookup | `TestUnixLookupAndProfiles` |
| Absent desktop | `TestUnixUnsupportedSession`, `TestRequireLocalDesktop`, headless CLI tests |
| URL helper failure, no shell evaluation | `TestUnixURLDispatch` |
| Dispatch failure and timeout survival | `TestDispatchHelperFailureAndTimeout` |
| Arguments, CWD and disconnected handles | `TestNativeArgumentsCWDAndDetachment` |
| Editor, terminal directory, URL content and named app | Individual qualification observations |
| Consent, CLI return, survival and repeat entry | Two-entry checklist and exit results |

Failure tests use controlled helpers without changing browser associations.
Passing them does not establish desktop readiness. Reports retain exact versions
and unverified observations; all remaining increment-4 platform gates stay open.
