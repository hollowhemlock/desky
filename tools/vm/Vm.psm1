Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Invoke-VMProcess {
    param([string]$File, [string[]]$Arguments, [int]$TimeoutSeconds = 120, [switch]$AllowFailure)
    $info = [Diagnostics.ProcessStartInfo]::new($File)
    $info.UseShellExecute = $false
    $info.CreateNoWindow = $true
    $info.RedirectStandardOutput = $true
    $info.RedirectStandardError = $true
    $info.StandardOutputEncoding = [Text.UTF8Encoding]::new($false)
    $info.StandardErrorEncoding = [Text.UTF8Encoding]::new($false)
    foreach ($argument in $Arguments) { $info.ArgumentList.Add($argument) }
    $process = [Diagnostics.Process]::new()
    $process.StartInfo = $info
    try {
        if (-not $process.Start()) { throw 'Could not start the requested tool.' }
        $stdout = $process.StandardOutput.ReadToEndAsync()
        $stderr = $process.StandardError.ReadToEndAsync()
        if (-not $process.WaitForExit($TimeoutSeconds * 1000)) {
            # Stop waiting; do not kill a possibly active guest operation.
            throw 'Tool deadline exceeded; outcome unknown. Inspect status before retrying.'
        }
        $result = @{ Code = $process.ExitCode; Out = $stdout.GetAwaiter().GetResult(); Err = $stderr.GetAwaiter().GetResult() }
        if ($result.Code -ne 0 -and -not $AllowFailure) {
            # Guest errors can include command lines or private file contents.
            throw "Tool operation '$($Arguments[0])' failed (exit $($result.Code)); no raw diagnostic output was logged."
        }
        return $result
    } finally { $process.Dispose() }
}

function Invoke-VBox {
    param($Settings, [string[]]$Arguments, [int]$TimeoutSeconds = 120, [switch]$AllowFailure)
    Invoke-VMProcess $Settings.VBoxPath $Arguments $TimeoutSeconds -AllowFailure:$AllowFailure
}

function Get-VMSettings {
    param([string]$Repo, [System.Collections.IDictionary]$Options)
    $s = @{
        VmName = 'desky-dev'; GuestUser = 'dev'; GuestRoot = '~/src/desky'
        VBoxPath = [IO.Path]::Combine([Environment]::GetFolderPath('ProgramFiles'), 'Oracle', 'VirtualBox', 'VBoxManage.exe')
    }
    $config = Join-Path $Repo '.cache/vm/config.local.json'
    if ($Options.ContainsKey('Config') -and $Options.Config) {
        $config = [IO.Path]::GetFullPath($Options.Config)
        if (-not (Test-Path -LiteralPath $config -PathType Leaf)) { throw 'Configuration file does not exist.' }
    }
    if (Test-Path -LiteralPath $config) {
        $local = Get-Content -LiteralPath $config -Raw | ConvertFrom-Json -AsHashtable
        foreach ($key in $local.Keys) {
            if ($key -in @('IsoPath', 'BaseFolder', 'MemoryMB', 'CPUs', 'DiskGB')) { throw "Remove obsolete creation setting '$key' from local configuration. Select an already-installed VM with VmName and GuestUser." }
            if (-not $s.ContainsKey($key)) { throw "Unknown configuration key: $key" }
            $s[$key] = $local[$key]
        }
    }
    foreach ($key in @($s.Keys)) { if ($Options.ContainsKey($key)) { $s[$key] = $Options[$key] } }
    if ($s.VmName -notmatch '^[^\x00-\x1f\x7f]{1,128}$' -or -not $s.VmName.Trim()) { throw 'Use an existing VM name or UUID without control characters.' }
    if ($s.GuestUser -eq 'root' -or $s.GuestUser -notmatch '^[a-z_][a-z0-9_-]{0,30}$') { throw 'GuestUser must be a normal Linux account name, not root.' }
    if ($s.GuestRoot -notmatch '^(~/|/)[^\x00-\x1f\x7f]*$' -or $s.GuestRoot.Split('/') -contains '..') { throw 'GuestRoot must be an absolute or ~/ path without parent traversal.' }
    $s.Repo = [IO.Path]::GetFullPath($Repo)
    $s.Cache = Join-Path $s.Repo '.cache/vm'
    $s.StateRoot = Join-Path ([Environment]::GetFolderPath('LocalApplicationData')) 'DeskyVM'
    return $s
}

function ConvertFrom-VBoxLines {
    param([string]$Value)
    $result = @{}
    foreach ($line in ($Value -split "`r?`n")) {
        if ($line -match '^"?([^"=]+)"?="(.*)"$') {
            $result[$Matches[1]] = $Matches[2].Replace('\"', '"').Replace('\\', '\')
        }
    }
    return $result
}

function Get-VMInfo {
    param($Settings)
    $listing = (Invoke-VBox $Settings @('list', 'vms')).Out
    $id = $null
    foreach ($line in ($listing -split "`r?`n")) {
        if ($line -match '^"(.*)" \{([0-9a-fA-F-]{36})\}$') {
            $name = $Matches[1].Replace('\"', '"').Replace('\\', '\')
            if ($name -ceq $Settings.VmName -or $Matches[2] -ieq $Settings.VmName) { $id = $Matches[2] }
        }
    }
    if (-not $id) { return $null }
    ConvertFrom-VBoxLines (Invoke-VBox $Settings @('showvminfo', $id, '--machinereadable')).Out
}

function New-PrivateDirectory {
    param([string]$Path)
    if (Test-Path -LiteralPath $Path) {
        if ((Get-Item -LiteralPath $Path).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing a linked private directory.' }
    } else { [IO.Directory]::CreateDirectory($Path) | Out-Null }
    if ($IsWindows) {
        $acl = [Security.AccessControl.DirectorySecurity]::new()
        $acl.SetAccessRuleProtection($true, $false)
        $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User
        $rule = [Security.AccessControl.FileSystemAccessRule]::new($sid, 'FullControl', 'ContainerInherit,ObjectInherit', 'None', 'Allow')
        $acl.AddAccessRule($rule)
        [IO.FileSystemAclExtensions]::SetAccessControl([IO.DirectoryInfo]::new($Path), $acl)
    }
}

function Use-GuestPassword {
    param($Settings, [scriptblock]$Body)
    New-PrivateDirectory $Settings.StateRoot
    $dir = Join-Path $Settings.StateRoot ([guid]::NewGuid().ToString('N'))
    New-PrivateDirectory $dir
    $path = Join-Path $dir 'password'
    $secret = Read-Host "Password for guest user $($Settings.GuestUser) (not saved)" -AsSecureString
    $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secret)
    try {
        $plain = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer)
        if (-not $plain -or $plain -match '[\r\n\x00]') { throw 'A nonempty single-line password is required.' }
        [IO.File]::WriteAllText($path, $plain, [Text.UTF8Encoding]::new($false))
        $plain = $null
        & $Body $path
    } finally {
        [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer)
        $secret.Dispose()
        if (Test-Path -LiteralPath $path) { Remove-Item -LiteralPath $path }
        [IO.Directory]::Delete($dir, $false)
    }
}

function Invoke-Guest {
    param($Settings, [string]$UUID, [string]$PasswordFile, [string]$Executable, [string[]]$Arguments, [int]$TimeoutSeconds = 300)
    $argsList = @('guestcontrol', $UUID, 'run', '--username', $Settings.GuestUser, '--passwordfile', $PasswordFile,
        '--exe', $Executable, '--timeout', [string]($TimeoutSeconds * 1000), '--wait-stdout', '--wait-stderr', '--') + $Arguments
    Invoke-VBox $Settings $argsList ($TimeoutSeconds + 10)
}

function Copy-ToGuest {
    param($Settings, [string]$UUID, [string]$PasswordFile, [string]$Source, [string]$Destination)
    # VBox's GuestPath::BuildDestinationPath appends the source basename only
    # for a trailing separator, even when --target-directory is specified.
    $directory = $Destination.TrimEnd('/') + '/'
    Invoke-VBox $Settings @('guestcontrol', $UUID, 'copyto', '--username', $Settings.GuestUser, '--passwordfile', $PasswordFile,
        '--no-replace', '--target-directory', $directory, $Source) 300 | Out-Null
}

function Assert-GuestReady {
    param($Settings, $Info, [string]$PasswordFile)
    if ($Info.VMState -ne 'running') { throw 'Start the VM and wait for Ubuntu to boot before provisioning.' }
    try {
        $code = @'
import os,shlex
from pathlib import Path
if os.getuid()==0: raise SystemExit('Use a normal guest account')
if any(x.lower() in ('boot=casper','boot=live') for x in shlex.split(Path('/proc/cmdline').read_text())):
 raise SystemExit('Boot the installed system, not live installation media')
mounts=(line.split() for line in Path('/proc/mounts').read_text().splitlines())
roots=[m[2] for m in mounts if len(m)>=3 and m[1]=='/']
if not roots or any(t in ('overlay','aufs','squashfs','tmpfs','ramfs','rootfs') for t in roots):
 raise SystemExit('Boot an installed system with a persistent root filesystem')
print(os.path.expanduser('~'))
'@
        $probe = Invoke-Guest $Settings $Info.UUID $PasswordFile '/usr/bin/python3' @('-c', $code) 30
        $guestHome = $probe.Out.Trim()
        if ($guestHome -notmatch '^/[^\r\n]+$') { throw 'Invalid guest home.' }
        Invoke-Guest $Settings $Info.UUID $PasswordFile '/usr/bin/pgrep' @('-x', 'VBoxService') 30 | Out-Null
        return $guestHome
    } catch { throw 'Guest not ready or authentication failed. Boot the installed Ubuntu system and check the normal-user password, Python 3 and Guest Additions service; see tools/vm/README.md.' }
}

. (Join-Path $PSScriptRoot 'Source.ps1')

function Invoke-VMAction {
    param([ValidateSet('status', 'start', 'provision', 'collect')][string]$Action, $Settings, [string]$Revision, [string]$RunId)
    if (-not (Test-Path -LiteralPath $Settings.VBoxPath -PathType Leaf)) { throw 'VirtualBox was not found; install it or set VBoxPath.' }
    $version = (Invoke-VBox $Settings @('--version')).Out.Trim()
    if ($version -notmatch '^7\.2\.') { throw 'This helper supports VirtualBox 7.2 only.' }
    $mutex = [Threading.Mutex]::new($false, 'Local\DeskyVM-management')
    $locked = $false
    try {
        if ($Action -ne 'status') {
            try { $locked = $mutex.WaitOne(0) } catch [Threading.AbandonedMutexException] { $locked = $true }
            if (-not $locked) { throw 'Another VM helper operation is running. Retry when it finishes.' }
        }
        $info = Get-VMInfo $Settings
        if ($Action -eq 'status') {
            @{ VirtualBox = $version; VM = $Settings.VmName; State = $(if ($info) { $info.VMState } else { 'absent' });
                GuestReadiness = 'not authenticated; provision performs readiness checks' } | ConvertTo-Json
            return
        }
        if (-not $info) { throw 'Target VM does not exist. Select an already-installed Ubuntu VM with -VmName; this helper does not create VMs or install operating systems.' }
        if ($Action -eq 'start') {
            if ($info.VMState -eq 'running') { Write-Output 'VM is already running.'; return }
            if ($info.VMState -notin @('poweroff', 'saved')) { throw "VM state '$($info.VMState)' is not safe to start." }
            Invoke-VBox $Settings @('startvm', $info.UUID, '--type', 'gui') | Out-Null
            return
        }
        if ($info.VMState -ne 'running') { throw 'Start the VM and boot its installed Ubuntu system before provisioning or collecting reports.' }
        Use-GuestPassword $Settings {
            param($passwordFile)
            $guestHome = Assert-GuestReady $Settings $info $passwordFile
            $guestRoot = if ($Settings.GuestRoot.StartsWith('~/')) { $guestHome + $Settings.GuestRoot.Substring(1) } else { $Settings.GuestRoot }
            if ($Action -eq 'provision') {
                Send-VMSource $Settings $info.UUID $passwordFile $guestRoot
            } else { Receive-VMReport $Settings $info.UUID $passwordFile $guestRoot $Revision $RunId }
        }
    } finally { if ($locked) { $mutex.ReleaseMutex() }; $mutex.Dispose() }
}

Export-ModuleMember -Function Get-VMSettings, Invoke-VMAction
