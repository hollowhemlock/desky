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
            # Stop waiting; do not kill a possibly active installation or guest operation.
            throw 'Tool deadline exceeded; outcome unknown. Inspect status before retrying.'
        }
        $result = @{ Code = $process.ExitCode; Out = $stdout.GetAwaiter().GetResult(); Err = $stderr.GetAwaiter().GetResult() }
        if ($result.Code -ne 0 -and -not $AllowFailure) {
            # VBox errors can include command lines or installation answer-file contents.
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
        VmName = 'desky-dev'; IsoPath = ''; BaseFolder = ''; GuestUser = 'dev'; GuestRoot = '~/src/desky'
        MemoryMB = 8192; CPUs = 4; DiskGB = 100
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
            if (-not $s.ContainsKey($key)) { throw "Unknown configuration key: $key" }
            $s[$key] = $local[$key]
        }
    }
    foreach ($key in @($s.Keys)) { if ($Options.ContainsKey($key)) { $s[$key] = $Options[$key] } }
    if ($s.VmName -notmatch '^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$') { throw 'Use a VM name or UUID with letters, numbers, dots, underscores and hyphens.' }
    if ($s.GuestUser -notmatch '^[a-z_][a-z0-9_-]{0,30}$') { throw 'GuestUser must be a conventional Linux account name.' }
    if ($s.GuestRoot -notmatch '^(~/|/)[^\x00-\x1f\x7f]*$' -or $s.GuestRoot.Split('/') -contains '..') { throw 'GuestRoot must be an absolute or ~/ path without parent traversal.' }
    foreach ($limit in @(@('MemoryMB', 2048, 262144), @('CPUs', 1, 64), @('DiskGB', 20, 2048))) {
        $value = 0
        if (-not [int]::TryParse([string]$s[$limit[0]], [ref]$value) -or $value -lt $limit[1] -or $value -gt $limit[2]) { throw "Invalid $($limit[0])." }
        $s[$limit[0]] = $value
    }
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
            if ($Matches[1] -ceq $Settings.VmName -or $Matches[2] -ieq $Settings.VmName) { $id = $Matches[2] }
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

function Save-VMRecord {
    param([string]$Path, $Value)
    $temp = "$Path.$([guid]::NewGuid().ToString('N')).tmp"
    [IO.File]::WriteAllText($temp, ($Value | ConvertTo-Json -Depth 12), [Text.UTF8Encoding]::new($false))
    [IO.File]::Move($temp, $Path, $true)
}

function Get-VMRecord {
    param($Settings, $Info)
    if (-not (Test-Path -LiteralPath $Settings.StateRoot)) { return $null }
    $records = @(Get-ChildItem -LiteralPath $Settings.StateRoot -Filter '*.json' -File | ForEach-Object {
        $item = Get-Content -LiteralPath $_.FullName -Raw | ConvertFrom-Json -AsHashtable
        if ($item.UUID -notmatch '^[0-9a-f-]{36}$' -or $item.Attempt -notmatch '^[0-9a-f-]{36}$' -or
            $item.Stage -notin @('configuring', 'installing', 'installed', 'ready to provision', 'provisioned')) {
            throw 'Invalid VM ownership record; preserving it for inspection.'
        }
        if (($Info -and $item.UUID -eq $Info.UUID) -or (-not $Info -and $item.Name -ceq $Settings.VmName)) { $item }
    })
    if ($records.Count -gt 1) { throw 'Multiple ownership records match; inspect local state without recreating the VM.' }
    if ($records.Count -eq 1) { return $records[0] }
    return $null
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
    Invoke-VBox $Settings @('guestcontrol', $UUID, 'copyto', '--username', $Settings.GuestUser, '--passwordfile', $PasswordFile,
        '--no-replace', '--target-directory', $Destination, $Source) 300 | Out-Null
}

function New-FinalizationTemplate {
    param([string]$Vendor, [string]$UserName, [string]$Attempt)
    if ($UserName -notmatch '^[a-z_][a-z0-9_-]{0,30}$' -or $Attempt -notmatch '^[0-9a-f-]{36}$') { throw 'Invalid finalization identity.' }
    $vendorText = [IO.File]::ReadAllText($Vendor).Replace("`r`n", "`n")
    if (($vendorText -split '(?m)^exit \$\{MY_EXITCODE\}$').Count -ne 2 -or
        $vendorText -notmatch '(?m)^MY_EXITCODE=0$' -or $vendorText -notmatch 'log_command_in_target\(\)' -or
        $vendorText -notmatch 'MY_TARGET="/"' -or $vendorText -notmatch 'MY_TARGET="/target"') {
        throw 'Unsupported VirtualBox Ubuntu template structure; VM creation was not started.'
    }
    $fragment = [IO.File]::ReadAllText((Join-Path $PSScriptRoot 'finalize.sh')).Replace('__GUEST_USER__', $UserName).Replace('__ATTEMPT__', $Attempt)
    # Insert after all vendor work, respecting both direct and chroot execution modes.
    $insert = @'
if [ "${MY_EXITCODE}" = 0 ]; then
    chroot "${MY_TARGET}" /bin/bash -s <<'DESKY_FINALIZE'
__FRAGMENT__
DESKY_FINALIZE
    if [ "$?" != 0 ]; then MY_EXITCODE=1; fi
fi
exit ${MY_EXITCODE}
'@.Replace('__FRAGMENT__', $fragment)
    return $vendorText.Replace('exit ${MY_EXITCODE}', $insert)
}

function Assert-InstallationRecord {
    param($Settings, [string]$UUID, [string]$PasswordFile, $Record)
    $code = @'
import json,os,stat,sys
p='/var/lib/desky-vm'; f=p+'/installed.json'
for x in ('/var','/var/lib',p,f):
 s=os.lstat(x)
 if stat.S_ISLNK(s.st_mode) or s.st_uid!=0 or s.st_mode & 0o022: raise SystemExit('Untrusted completion record')
d=json.load(open(f))
if d != {'attempt':sys.argv[1],'user':sys.argv[2],'root_locked':True,'sudo':True}: raise SystemExit('Installation record mismatch')
print('verified')
'@
    Invoke-Guest $Settings $UUID $PasswordFile '/usr/bin/python3' @('-c', $code, $Record.Attempt, $Settings.GuestUser) | Out-Null
}

function Assert-GuestReady {
    param($Settings, $Info, [string]$PasswordFile)
    if ($Info.VMState -ne 'running') { throw 'Start the VM and wait for Ubuntu to boot before provisioning.' }
    try {
        $probe = Invoke-Guest $Settings $Info.UUID $PasswordFile '/usr/bin/python3' @('-c', 'import os; print(os.path.expanduser("~"))') 30
        $guestHome = $probe.Out.Trim()
        if ($guestHome -notmatch '^/[^\r\n]+$') { throw 'Invalid guest home.' }
        Invoke-Guest $Settings $Info.UUID $PasswordFile '/usr/bin/pgrep' @('-x', 'VBoxService') 30 | Out-Null
        return $guestHome
    } catch { throw 'Guest not ready or authentication failed. Check the password and Guest Additions service; see tools/vm/README.md recovery instructions.' }
}

function Complete-InstallationCleanup {
    param($Settings, $Info, $Record)
    if ($Record.Cleaned) { return }
    if (-not $Record.ContainsKey('Artifacts')) { throw 'Installation artifact inventory was interrupted. Preserve the private installation directory for manual inspection; no files were removed.' }
    $installDir = Join-Path $Settings.StateRoot $Record.Attempt
    $prefix = [IO.Path]::GetFullPath($installDir) + [IO.Path]::DirectorySeparatorChar
    if (Test-Path -LiteralPath $installDir) {
        if ((Get-Item -LiteralPath $installDir).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Linked installation directory refused.' }
        foreach ($file in Get-ChildItem -LiteralPath $installDir -Force) {
            if ($file.PSIsContainer -or ($file.Attributes -band [IO.FileAttributes]::ReparsePoint) -or
                -not $Record.Artifacts.ContainsKey($file.Name) -or $Record.Artifacts[$file.Name] -ne (Get-FileHash -LiteralPath $file.FullName).Hash) {
                throw 'Unexpected or changed installation artifact; cleanup stopped.'
            }
        }
    }
    foreach ($slot in @(@('SATA', 1, 0), @('SATA', 2, 0))) {
        $key = "$($slot[0])-$($slot[1])-$($slot[2])"
        if ($Info.ContainsKey($key) -and $Info[$key] -ne 'none' -and $Info[$key] -ne 'emptydrive') {
            $medium = [IO.Path]::GetFullPath($Info[$key])
            $owned = $medium.StartsWith($prefix, [StringComparison]::OrdinalIgnoreCase) -or $medium -eq $Record.ISO -or $medium -eq $Record.AdditionsISO
            if (-not $owned) { throw 'Installation drive was changed by another operation; leaving media and files untouched.' }
            Invoke-VBox $Settings @('storageattach', $Info.UUID, '--storagectl', $slot[0], '--port', [string]$slot[1], '--device', '0', '--type', 'dvddrive', '--medium', 'emptydrive') | Out-Null
        }
    }
    # Delete only inventoried direct children; a partly completed cleanup is retryable.
    if (Test-Path -LiteralPath $installDir) {
        foreach ($file in Get-ChildItem -LiteralPath $installDir -Force) {
            Remove-Item -LiteralPath $file.FullName
        }
        [IO.Directory]::Delete($installDir, $false)
    }
    $Record.Cleaned = $true
    Save-VMRecord (Join-Path $Settings.StateRoot "$($Record.UUID).json") $Record
}

function New-ManagedVM {
    param($Settings, $Info, $Record)
    if ($Info -and (-not $Record -or $Record.Stage -ne 'configuring')) {
        Write-Output 'VM already exists. Use status, start or provision; its OS will not be reinstalled.'
        return
    }
    if ($Record -and $Record.Stage -ne 'configuring') { throw 'A prior installation attempt exists but its VM is missing; inspect the ownership record. No recreation was attempted.' }
    if (-not (Test-Path -LiteralPath $Settings.IsoPath -PathType Leaf)) { throw 'Supply -IsoPath with a local Ubuntu 24.04 Desktop amd64 ISO.' }
    $iso = [IO.Path]::GetFullPath($Settings.IsoPath)
    $detected = ConvertFrom-VBoxLines (Invoke-VBox $Settings @('unattended', 'detect', "--iso=$iso", '--machine-readable')).Out
    if ($detected.IsInstallSupported -ne 'on' -or $detected.OSVersion -notmatch '^24\.04' -or $detected.OSTypeId -notmatch '^Ubuntu.*_64$' -or [IO.Path]::GetFileName($iso) -notmatch 'desktop-amd64\.iso$') {
        throw 'Only detected Ubuntu 24.04 Desktop amd64 unattended installation is supported.'
    }
    $vendor = Join-Path (Split-Path $Settings.VBoxPath) 'UnattendedTemplates/ubuntu_postinstall.sh'
    $additions = Join-Path (Split-Path $Settings.VBoxPath) 'VBoxGuestAdditions.iso'
    if (-not (Test-Path -LiteralPath $additions)) { throw 'Matching bundled Guest Additions ISO is missing.' }
    $attempt = if ($Record) { $Record.Attempt } else { [guid]::NewGuid().ToString() }
    $template = New-FinalizationTemplate $vendor $Settings.GuestUser $attempt
    $isoHash = (Get-FileHash -LiteralPath $iso -Algorithm SHA256).Hash.ToLowerInvariant()
    if (-not $Record) {
        $base = $Settings.BaseFolder
        if (-not $base) {
            $properties = (Invoke-VBox $Settings @('list', 'systemproperties')).Out
            if ($properties -notmatch '(?m)^Default machine folder:\s*(.+)\r?$') { throw 'Cannot determine VirtualBox machine folder; supply -BaseFolder.' }
            $base = $Matches[1].Trim()
        }
        $base = [IO.Path]::GetFullPath($base)
        if (Test-Path -LiteralPath (Join-Path $base $Settings.VmName)) {
            throw 'The new machine folder already exists; choose another name or inspect it. Existing disks are never adopted.'
        }
        $Record = @{ UUID = [guid]::NewGuid().ToString(); Name = $Settings.VmName; Attempt = $attempt; Stage = 'configuring'; Cleaned = $false
            ISO = $iso; ISOHash = $isoHash; AdditionsISO = $additions; Base = $base; GuestUser = $Settings.GuestUser
            MemoryMB = $Settings.MemoryMB; CPUs = $Settings.CPUs; DiskGB = $Settings.DiskGB
            VendorHash = (Get-FileHash -LiteralPath $vendor).Hash; Version = (Invoke-VBox $Settings @('--version')).Out.Trim() }
        New-PrivateDirectory $Settings.StateRoot
        Save-VMRecord (Join-Path $Settings.StateRoot "$($Record.UUID).json") $Record
    } elseif ($Record.ISOHash -ne $isoHash -or $Record.GuestUser -ne $Settings.GuestUser -or $Record.VendorHash -ne (Get-FileHash -LiteralPath $vendor).Hash) {
        throw 'Installation inputs changed; inspect the incomplete VM instead of resuming with different inputs.'
    }
    $recordPath = Join-Path $Settings.StateRoot "$($Record.UUID).json"
    if (-not $Info) {
        if (Test-Path -LiteralPath (Join-Path $Record.Base $Record.Name)) {
            throw 'An unregistered machine folder exists from an ambiguous creation; inspect it before proceeding.'
        }
        Invoke-VBox $Settings @('createvm', '--name', $Record.Name, '--uuid', $Record.UUID, '--ostype', $detected.OSTypeId, '--basefolder', $Record.Base, '--register') | Out-Null
        $Info = Get-VMInfo $Settings
    }
    if ($Info.UUID -ne $Record.UUID -or $Info.VMState -ne 'poweroff') { throw 'VM identity/state does not permit resuming configuration.' }
    $folder = Split-Path $Info.CfgFile
    $disk = Join-Path $folder 'desky-system.vdi'
    if ($Record.ContainsKey('DiskUUID')) {
        if (-not (Test-Path -LiteralPath $disk -PathType Leaf) -or $Record.DiskPath -cne $disk) { throw 'Recorded system disk is missing or moved; refusing recreation.' }
    } elseif (Test-Path -LiteralPath $disk) {
        throw 'Existing system disk has no recorded ownership; refusing to attach or overwrite it.'
    }
    if (Test-Path -LiteralPath $disk) {
        if ((Get-Item -LiteralPath $disk).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Linked system disk refused.' }
        $medium = (Invoke-VBox $Settings @('showmediuminfo', 'disk', $disk)).Out
        if ($medium -notmatch '(?m)^UUID:\s*([0-9a-fA-F-]{36})\s*$' -or $Matches[1] -ne $Record.DiskUUID) {
            throw 'System disk identity changed; refusing installation.'
        }
    }
    foreach ($port in 0, 1, 2) {
        $slot = "SATA-$port-0"
        if ($Info.ContainsKey($slot) -and $Info[$slot] -notin @('none', 'emptydrive', $(if ($port -eq 0) { $disk } else { '' }))) {
            throw 'A creation-stage storage attachment changed; refusing to replace it.'
        }
    }
    Invoke-VBox $Settings @('modifyvm', $Record.UUID, '--memory', [string]$Record.MemoryMB, '--cpus', [string]$Record.CPUs,
        '--ioapic', 'on', '--rtc-use-utc', 'on', '--firmware', 'efi', '--graphicscontroller', 'vmsvga', '--vram', '128', '--nic1', 'nat', '--clipboard-mode', 'disabled', '--drag-and-drop', 'disabled') | Out-Null
    if (-not (Test-Path -LiteralPath $disk)) {
        Invoke-VBox $Settings @('createmedium', 'disk', '--filename', $disk, '--size', [string]($Record.DiskGB * 1024), '--format', 'VDI', '--variant', 'Standard') | Out-Null
        $medium = (Invoke-VBox $Settings @('showmediuminfo', 'disk', $disk)).Out
        if ($medium -notmatch '(?m)^UUID:\s*([0-9a-fA-F-]{36})\s*$') { throw 'Cannot establish created disk identity; no installation was attempted.' }
        $Record.DiskUUID = $Matches[1]
        $Record.DiskPath = $disk
        Save-VMRecord $recordPath $Record
    }
    if (-not ($Info.Values -contains 'SATA')) { Invoke-VBox $Settings @('storagectl', $Record.UUID, '--name', 'SATA', '--add', 'sata', '--controller', 'IntelAhci', '--portcount', '3', '--bootable', 'on') | Out-Null }
    Invoke-VBox $Settings @('storageattach', $Record.UUID, '--storagectl', 'SATA', '--port', '0', '--device', '0', '--type', 'hdd', '--medium', $disk) | Out-Null
    foreach ($port in 1, 2) { Invoke-VBox $Settings @('storageattach', $Record.UUID, '--storagectl', 'SATA', '--port', [string]$port, '--device', '0', '--type', 'dvddrive', '--medium', 'emptydrive') | Out-Null }
    $installDir = Join-Path $Settings.StateRoot $attempt
    New-PrivateDirectory $installDir
    $templatePath = Join-Path $installDir 'postinstall.sh'
    [IO.File]::WriteAllText($templatePath, $template, [Text.UTF8Encoding]::new($false))
    Use-GuestPassword $Settings {
        param($passwordFile)
        $rootFile = Join-Path $installDir 'root-password'
        [IO.File]::WriteAllText($rootFile, [Convert]::ToBase64String([Security.Cryptography.RandomNumberGenerator]::GetBytes(48)))
        try {
            # Intent precedes VBox's mutation; unknown outcome never authorizes another install.
            $Record.Stage = 'installing'
            Save-VMRecord $recordPath $Record
            Invoke-VBox $Settings @('unattended', 'install', $Record.UUID, "--iso=$iso", "--user=$($Settings.GuestUser)",
                "--user-password-file=$passwordFile", "--admin-password-file=$rootFile", "--hostname=$($Record.Name).home.arpa",
                '--install-additions', "--additions-iso=$additions", "--post-install-template=$templatePath",
                "--auxiliary-base-path=$(Join-Path $installDir 'unattended-')", '--start-vm=gui') 300 | Out-Null
            $Record.Artifacts = @{}
            foreach ($file in Get-ChildItem -LiteralPath $installDir -File) {
                if ($file.Name -ne 'root-password') { $Record.Artifacts[$file.Name] = (Get-FileHash -LiteralPath $file.FullName).Hash }
            }
            Save-VMRecord $recordPath $Record
        } finally { if (Test-Path -LiteralPath $rootFile) { Remove-Item -LiteralPath $rootFile } }
    }
    Write-Output 'Installation started. Log in to Ubuntu when ready, then run provision. Installer media remains private until successful finalization.'
}

. (Join-Path $PSScriptRoot 'Source.ps1')

function Invoke-VMAction {
    param([string]$Action, $Settings, [string]$Revision, [string]$RunId)
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
        $record = Get-VMRecord $Settings $info
        if ($Action -eq 'status') {
            @{ VirtualBox = $version; VM = $Settings.VmName; State = $(if ($info) { $info.VMState } else { 'absent' });
                Installation = $(if ($record) { $record.Stage } else { 'unmanaged' });
                GuestReadiness = 'not authenticated; provision performs readiness checks' } | ConvertTo-Json
            return
        }
        if ($Action -eq 'create') { New-ManagedVM $Settings $info $record; return }
        if (-not $info) { throw 'Target VM does not exist; run create or select an existing VM.' }
        if ($Action -eq 'start') {
            if ($info.VMState -eq 'running') { Write-Output 'VM is already running.'; return }
            if ($info.VMState -notin @('poweroff', 'saved')) { throw "VM state '$($info.VMState)' is not safe to start." }
            Invoke-VBox $Settings @('startvm', $info.UUID, '--type', 'gui') | Out-Null
            return
        }
        Use-GuestPassword $Settings {
            param($passwordFile)
            $guestHome = Assert-GuestReady $Settings $info $passwordFile
            $guestRoot = if ($Settings.GuestRoot.StartsWith('~/')) { $guestHome + $Settings.GuestRoot.Substring(1) } else { $Settings.GuestRoot }
            if ($record) {
                if ($record.GuestUser -ne $Settings.GuestUser) { throw 'Guest user differs from the installation record.' }
                Assert-InstallationRecord $Settings $info.UUID $passwordFile $record
                if ($Action -eq 'provision') {
                    $record.Stage = 'ready to provision'
                    Save-VMRecord (Join-Path $Settings.StateRoot "$($record.UUID).json") $record
                }
            }
            if ($Action -eq 'provision') {
                if ($record) { Complete-InstallationCleanup $Settings $info $record }
                Send-VMSource $Settings $info.UUID $passwordFile $guestRoot
                if ($record) { $record.Stage = 'provisioned'; Save-VMRecord (Join-Path $Settings.StateRoot "$($record.UUID).json") $record }
            } else { Receive-VMReport $Settings $info.UUID $passwordFile $guestRoot $Revision $RunId }
        }
    } finally { if ($locked) { $mutex.ReleaseMutex() }; $mutex.Dispose() }
}

Export-ModuleMember -Function Get-VMSettings, Invoke-VMAction
