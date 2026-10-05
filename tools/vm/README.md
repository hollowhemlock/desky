# Ubuntu desktop VM workflow

These optional development tools create or reuse a VM and prepare Desky desktop
qualification. They are separate from `desk`: the application never installs or
manages its users' environments.

Requirements: Windows with its built-in `tar.exe`, PowerShell **7.4+**, Git,
VirtualBox **7.2**, and a local **Ubuntu 26.04 LTS Desktop amd64** ISO. The current
target is **26.04.1**; Ubuntu 24.04 LTS remains accepted for existing workflows.
Both releases include Python 3.12 or newer, as required by the guest scripts.
Package installation requires guest internet access. The helper does not install
VirtualBox, download operating systems, enable SSH or configure shared folders.

**Verification status:** deterministic host and Linux guest tests exercise the
helper. Real unattended installation, Guest Control transfer, repeated package
bootstrap and desktop qualification remain unverified. The local Ubuntu 26.04.1
ISO passes the host's read-only media validation with VirtualBox 7.2.20.
Helper tests and WSL checks do not qualify desktops; see [PILOT](../../PILOT.md).

If `create` reports an ISO detection failure, that attempt made no VM changes.
VirtualBox's Linux detector can return a bare `E_NOTIMPL` despite identifying
installable media; [its installer explicitly accepts this partial result](https://github.com/VirtualBox/virtualbox/blob/master/src/VBox/Main/src-server/UnattendedImpl.cpp).
The helper accepts that exact diagnostic only with a supported Ubuntu release,
amd64 type, `IsInstallSupported="on"`, and matching `.disk/info` and Desktop GRUB
boot entries read independently from the ISO. It records the release and whether
detection was complete or this known partial result. Other failures, incomplete
metadata, mismatches and server images are rejected. A successful detection exit
also requires those content checks. The `Ubuntu25_64` type returned for 26.04.1 is
VirtualBox's older hardware profile; the detected release still controls acceptance.
For rejected media, verify the ISO against Ubuntu's published checksum and check
the installed VirtualBox 7.2 build. Renaming an ISO does not change its contents.

## First run

From the repository root in PowerShell, using a committed revision containing
these tools:

```powershell
./tools/vm/manage.ps1 status
./tools/vm/manage.ps1 create -IsoPath 'D:\Downloads\ubuntu-26.04.1-desktop-amd64.iso'
```

Enter the guest password at the local masked prompt and keep it in your password
manager, never in JSON, Git, command arguments or chat. Creation returns after
starting installation. When Ubuntu is ready, log in as `dev`, then run on Windows:

```powershell
./tools/vm/manage.ps1 provision
```

Provisioning prompts for that user's password, checks guest readiness and transfers
committed source. Run its printed command inside Ubuntu's desktop terminal.
Bootstrap uses normal sudo prompts and installs missing Git/compiler tools, Go,
VS Code, Firefox, GNOME Terminal and GNOME Text Editor without a general OS upgrade.

Then run inside the exported revision directory:

```sh
python3 tools/vm/guest.py qualify .
```

Qualification runs `go run ./tools/verify -race`, saves a verification log and
guides two real Desky entries. Inspect the displayed fixture path, terminal `pwd`,
browser content/URL, editor and note. Answer each observation with `p`, `f` or
Enter for unverified. Leave applications open until their survival is observed.
The script does not close them.

Collect the printed revision/run ID on Windows (replace both placeholders):

```powershell
./tools/vm/manage.ps1 collect -Revision <full-commit-id> -RunId <run-id>
```

Only the selected report JSON is collected beneath `.cache/vm/reports/`; logs stay
in Ubuntu. Collection never changes observations or support claims. Review the
evidence before updating PILOT manually.

## Configuration and defaults

Copy `config.example.json` to `.cache/vm/config.local.json`, or supply `-Config`
with another local file. Explicit arguments override local settings, which
override defaults. Unknown keys are errors. The default config path is relative
to the repository, not the invoking shell.

| Setting / argument | Default and reason |
|---|---|
| `VmName` | `desky-dev`; select an existing name or UUID to reuse a VM |
| `IsoPath` | Required only for fresh creation or resuming initial configuration |
| `BaseFolder` | VirtualBox's configured machine folder; keep disks outside Git |
| `GuestUser` | `dev`; neutral account name, also used for Guest Control |
| `GuestRoot` | `~/src/desky`; native guest filesystem, one directory per commit |
| `MemoryMB` | 8192; room for desktop applications and ordinary builds |
| `CPUs` | 4; useful build parallelism while leaving host capacity |
| `DiskGB` | 100; ceiling for a single dynamically allocated VDI |
| `VBoxPath` | Standard installation location; override for custom installations |

Names allow letters, digits, dots, underscores and hyphens. New VMs use EFI,
VMSVGA, NAT, disabled clipboard/drag-and-drop and matching bundled Guest Additions.
Defaults do not modify existing VM hardware, disks or accounts. For your shared
VM, set `VmName` to `ubuntu-dev`. Starting a running VM is harmless:

```powershell
./tools/vm/manage.ps1 start -VmName ubuntu-dev
```

## Privacy, reruns and recovery

Local configuration, exports and reports belong in ignored `.cache/vm/`. Do not
force-add them. Ownership records and unattended media live under
`%LOCALAPPDATA%\DeskyVM`, restricted to the current Windows user. Password files
are temporary and never logged. The independent random installation-only root
credential is discarded; successful finalization locks root and verifies sudo
access before writing a root-owned completion record. Reused VMs keep their
account policy. The helper never authenticates as root.

`status` is read-only and reports persisted progress without prompting or claiming
authenticated readiness. `provision` verifies the completion record and Guest
Additions, then detaches installation media and removes inventoried private
artifacts. Unexpected/changed files stop cleanup. Interrupted cleanup with a
valid inventory is retryable. If artifact inventory itself was interrupted,
inspect the restricted directory and detach its media manually before removing
credential-bearing files. Never share answer files or raw installer logs.

Creation resumes helper-owned configuration only before installation intent is
recorded. An ambiguous installation is never automatically retried. Inspect the
VM window and ownership record. Unmanaged VMs are not adopted for installation.
Existing machine directories are refused during new creation. The created disk's
medium UUID is recorded and checked on retries; pre-existing, replaced or
ambiguously created disks are never adopted for installation.
The helper never deletes VMs, resets snapshots or forces shutdown. Deadlines
mean an unknown outcome; inspect before retrying because active operations are
not killed. Host mutations and guest publication/bootstrap/checks are serialized.

Source is exported with `git archive HEAD` and checked against the Git tree.
Host line-ending preferences are disabled for that export to preserve committed bytes.
Working changes, ignored files, history and authentication are excluded. Already
committed secrets are not detected or removed. Submodules and export attributes
that omit/transform committed content are rejected. Transfers use checksums,
staging and atomic publication. Partial `.incoming-*` directories remain for
inspection, never replacing a published revision.

Reruns validate exact source contents and reject modifications or unexpected
files, including extra `.go`/`_test.go` files. They preserve those files instead
of resetting them. Only designated metadata, `bin/desk` and workflow run outputs
are allowed; arbitrary files under `bin/` are not exempt. The workflow does not
automatically prune exports or disks.

Bootstrap checks actual prerequisites on each run. It records real versions;
upstream package repositories are not frozen. If necessary, Go is downloaded
using official checksum metadata into a versioned user-owned location.
Qualification uses that executable without editing shell profiles. Desky storage
is isolated using XDG locations while retaining the actual desktop session;
application preferences can still affect observations.

### Guest Additions unavailable

First check the guest username/password. Source transfer requires Guest Control.
If necessary, repair Additions inside Ubuntu:

1. Mount the matching bundled `VBoxGuestAdditions.iso` using VirtualBox's Devices
   menu, without replacing unrelated disks.
2. In Ubuntu run `sudo apt-get install build-essential dkms linux-headers-$(uname -r)`.
3. Run `sudo sh VBoxLinuxAdditions.run` from the mounted ISO directory; reboot if requested.
4. Check `sudo rcvboxadd status-user` and `sudo rcvboxadd status-kernel`, then retry provisioning.

See [Oracle's repair instructions](https://docs.oracle.com/en/virtualization/virtualbox/7.2/user/guestadditions.html).
A failed vendor installation cannot produce a success record merely by reaching
the finalizer. Repair or recreate an unfinalized test VM manually; do not
manufacture a completion record.

## Verification

On Windows (Go builds the fake executable):

```powershell
./tools/vm/tests/host.tests.ps1
```

On Linux:

```sh
bash -n tools/vm/bootstrap.sh tools/vm/finalize.sh
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/vm/tests -p 'test_*.py' -v
```

CI adds these checks to the existing three-platform workflow. Host fixtures use
a fake VirtualBox executable and synthetic credentials. Linux tests use temporary
directories and fake account commands. Windows retains bounded test artifacts
for diagnosis; none contain real credentials. These are not install acceptance.

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
