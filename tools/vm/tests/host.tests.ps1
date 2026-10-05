#Requires -Version 7.4
$ErrorActionPreference = 'Stop'
$toolDir = Split-Path $PSScriptRoot
Import-Module (Join-Path $toolDir 'Vm.psm1') -Force
$module = Get-Module Vm
$testRoot = Join-Path ([IO.Path]::GetTempPath()) ('desky-vm-tests-' + [guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($testRoot) | Out-Null
$env:DESKY_FAKE_VBOX_ROOT = $testRoot
function Assert($Condition, $Message) { if (-not $Condition) { throw $Message } }
function Must-Fail([scriptblock]$Body, [string]$Pattern) {
    try { & $Body; throw 'Expected operation to fail' } catch {
        if ($_.Exception.Message -notmatch $Pattern) { throw }
    }
}
try {
    $fake = Join-Path $testRoot 'VBoxManage.exe'
    & go build -o $fake (Join-Path $toolDir 'testdata/fakevbox/main.go')
    if ($LASTEXITCODE) { throw 'Fake VirtualBox build failed' }
    $vendorDir = Join-Path $testRoot 'UnattendedTemplates'
    [IO.Directory]::CreateDirectory($vendorDir) | Out-Null
    @'
#!/bin/bash
MY_EXITCODE=0
if [ "$1" = "--direct" ]; then MY_TARGET="/"; else MY_TARGET="/target"; fi
log_command_in_target() { :; }
exit ${MY_EXITCODE}
'@ | Set-Content (Join-Path $vendorDir 'ubuntu_postinstall.sh')
    Set-Content (Join-Path $testRoot 'VBoxGuestAdditions.iso') 'fixture'
    $iso = Join-Path $testRoot 'ubuntu-26.04.1-desktop-amd64.iso'
    $isoFiles = Join-Path $testRoot 'iso contents'
    [IO.Directory]::CreateDirectory((Join-Path $isoFiles '.disk')) | Out-Null
    [IO.Directory]::CreateDirectory((Join-Path $isoFiles 'boot/grub')) | Out-Null
    function New-MediaFixture([string]$Release = '26.04.1', [string]$BootTitle = 'Try or Install Ubuntu', [string]$BuildDate = '20260826') {
        [IO.File]::WriteAllText((Join-Path $isoFiles '.disk/info'), "Ubuntu $Release LTS `"Fixture`" - Release amd64 ($BuildDate)")
        [IO.File]::WriteAllText((Join-Path $isoFiles 'boot/grub/grub.cfg'), "menuentry `"$BootTitle`" {`n linux /casper/vmlinuz --- quiet splash`n initrd /casper/initrd`n}`n")
        # A small archive exercises the real Windows reader; VBox itself is fake.
        & (Join-Path ([Environment]::GetFolderPath('System')) 'tar.exe') -cf $iso -C $isoFiles .disk/info boot/grub/grub.cfg
        if ($LASTEXITCODE) { throw 'Media fixture archive failed' }
    }
    New-MediaFixture
    $s = Get-VMSettings $testRoot @{ VBoxPath = $fake; IsoPath = $iso }
    $s.StateRoot = Join-Path $testRoot 'private'
    Assert ($s.MemoryMB -eq 8192 -and $s.GuestUser -eq 'dev') 'Defaults differ'
    $config = Join-Path $testRoot 'local.json'
    '{"VmName":"custom","CPUs":2}' | Set-Content $config
    $override = Get-VMSettings $testRoot @{ Config=$config; CPUs=6 }
    Assert ($override.VmName -eq 'custom' -and $override.CPUs -eq 6) 'Precedence failure'
    '{"Password":"should not be accepted"}' | Set-Content $config
    Must-Fail { Get-VMSettings $testRoot @{Config=$config} } 'Unknown configuration'
    Must-Fail { Get-VMSettings $testRoot @{GuestRoot='/tmp/../root'} } 'GuestRoot'
    Invoke-VMAction 'status' $s '' '' | Out-Null
    Assert (-not (Test-Path $s.StateRoot)) 'Status wrote state'
    & pwsh -NoProfile -File (Join-Path $toolDir 'manage.ps1') status -VBoxPath $fake -VmName 'missing-fixture' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Public helper entry point failed' }
    # Supply a synthetic test password without prompting or touching user credentials.
    & $module {
        function script:Use-GuestPassword($Settings, [scriptblock]$Body) {
            $file = Join-Path $Settings.StateRoot 'test-password'
            [IO.File]::WriteAllText($file, 'test-only-value')
            try { & $Body $file } finally { Remove-Item -LiteralPath $file }
        }
    }
    # A failed detector may still print plausible metadata. Never accept it or
    # expose its raw diagnostics, and keep unsupported/missing metadata actionable.
    $detectionFixture = Join-Path $testRoot 'detect-result.json'
    $supported = "OSTypeId=`"Ubuntu25_64`"`nOSVersion=`"26.04.1 LTS Ubuntu`"`nIsInstallSupported=`"on`""
    $unsupported = "OSTypeId=`"Ubuntu25_64`"`nOSVersion=`"25.10 Ubuntu`"`nIsInstallSupported=`"on`""
    $partial = "VBoxManage.exe: error: Code E_NOTIMPL (0x80004001) (extended info not available)`r`nVBoxManage.exe: error: Context: `"DetectIsoOS()`" at line 2235 of file VBoxManageMisc.cpp`r`n"
    foreach ($case in @(
        @{ Code=1; Out=$unsupported; Err='Code E_NOTIMPL: fixture-private-detail'; Pattern='ISO detection failed.*E_NOTIMPL.*Detected Ubuntu 25\.10.*requires.*26\.04' },
        @{ Code=1; Out=$supported; Err='fixture-private-detail'; Pattern='ISO detection failed.*Detected Ubuntu 26\.04\.1' },
        @{ Code=1; Out=$supported; Err=($partial + 'Failed to open ISO: fixture-private-detail'); Pattern='ISO detection failed' },
        @{ Code=1; Out=$supported.Replace('"on"', '"off"'); Err=$partial; Pattern='requires.*26\.04' },
        @{ Code=1; Out=$supported.Replace('Ubuntu25_64', 'Ubuntu26_arm64'); Err=$partial; Pattern='requires.*26\.04' },
        @{ Code=1; Out=''; Err=$partial; Pattern='requires.*26\.04' },
        @{ Code=2; Out=$supported; Err=$partial; Pattern='ISO detection failed' },
        @{ Code=1; Out=''; Err='fixture-private-detail'; Pattern='ISO detection failed.*requires.*26\.04' },
        @{ Code=0; Out=$unsupported; Err=''; Pattern='Detected Ubuntu 25\.10.*requires.*26\.04' },
        @{ Code=0; Out=''; Err=''; Pattern='requires.*26\.04' }
    )) {
        $case | ConvertTo-Json | Set-Content -LiteralPath $detectionFixture
        $failure = $null
        try { Invoke-VMAction 'create' $s '' '' | Out-Null } catch { $failure = $_.Exception.Message }
        Assert ($failure -and $failure -match $case.Pattern) 'ISO rejection did not explain the detection failure or supported release'
        Assert (-not $failure.Contains('fixture-private-detail')) 'Raw detection diagnostics were exposed'
        Assert (-not (Test-Path -LiteralPath $s.StateRoot)) 'Rejected ISO wrote installation state'
        $detectionCalls = Get-Content (Join-Path $testRoot 'calls.jsonl') -Raw
        Assert ($detectionCalls -notmatch 'createvm|createmedium|modifyvm|storageattach|"install"') 'Rejected ISO mutated a VM'
    }
    Remove-Item -LiteralPath $detectionFixture
    # The known partial result still requires independent matching desktop media.
    New-MediaFixture '24.04.5.1'
    Must-Fail { Invoke-VMAction 'create' $s '' '' } 'ISO contents do not confirm'
    New-MediaFixture '26.04.1' 'Try or Install Ubuntu Server'
    Must-Fail { Invoke-VMAction 'create' $s '' '' } 'ISO contents do not confirm'
    [IO.File]::WriteAllText($iso, 'not an archive')
    Must-Fail { Invoke-VMAction 'create' $s '' '' } 'ISO contents do not confirm'
    Assert (-not (Test-Path -LiteralPath $s.StateRoot)) 'Uncorroborated ISO wrote installation state'
    # Keep the previous LTS path working, including four-component point releases.
    New-MediaFixture '24.04.5.1'
    @{ Code=0; Out=$supported.Replace('26.04.1', '24.04.5.1'); Err='' } | ConvertTo-Json | Set-Content $detectionFixture
    $priorMedia = & $module { param($settings) Get-InstallationMedia $settings } $s
    Assert ($priorMedia.Release -eq '24.04.5.1' -and $priorMedia.Detection -eq 'complete') 'Prior LTS media was rejected'
    New-MediaFixture '24.04.1' 'Try or Install Ubuntu' '20240827.1'
    @{ Code=0; Out=$supported.Replace('26.04.1', '24.04.1'); Err='' } | ConvertTo-Json | Set-Content $detectionFixture
    $respunMedia = & $module { param($settings) Get-InstallationMedia $settings } $s
    Assert ($respunMedia.Release -eq '24.04.1') 'A valid ISO build-date respin suffix was rejected'
    New-MediaFixture
    @{ Code=0; Out=$supported; Err='' } | ConvertTo-Json | Set-Content $detectionFixture
    $completeMedia = & $module { param($settings) Get-InstallationMedia $settings } $s
    Assert ($completeMedia.Release -eq '26.04.1' -and $completeMedia.Detection -eq 'complete') 'Complete detection was rejected'
    Remove-Item -LiteralPath $detectionFixture
    Invoke-VMAction 'create' $s '' '' | Out-Null
    $records = @(Get-ChildItem $s.StateRoot -Filter '*.json')
    Assert ($records.Count -eq 1) 'Ownership record missing'
    $record = Get-Content $records[0].FullName -Raw | ConvertFrom-Json -AsHashtable
    Assert ($record.Stage -eq 'installing') 'Installation intent missing'
    Assert ($record.Release -eq '26.04.1' -and $record.Detection -eq 'known-linux-partial') 'Validated media identity was not recorded'
    Invoke-VMAction 'create' $s '' '' | Out-Null
    Invoke-VMAction 'start' $s '' '' | Out-Null
    $calls = @(Get-Content (Join-Path $testRoot 'calls.jsonl') | ForEach-Object { ,($_ | ConvertFrom-Json) })
    Assert (@($calls | Where-Object { $_[0] -eq 'unattended' -and $_[1] -eq 'install' }).Count -eq 1) 'Repeated OS installation'
    Assert (-not ((Get-Content (Join-Path $testRoot 'calls.jsonl') -Raw).Contains('test-only-value'))) 'Credential leaked to command arguments'
    Assert (-not (Test-Path (Join-Path $s.StateRoot 'test-password'))) 'Password file survived'
    Assert (-not (Test-Path (Join-Path $s.StateRoot "$($record.Attempt)/root-password"))) 'Root password survived'
    $guestResult = & $module {
        param($settings, $id)
        Invoke-Guest $settings $id 'fixture-password-file' '/usr/bin/python3' @('-c', 'print("a b")', '雪 & ;')
    } $s $record.UUID
    $guestArgs = @($guestResult.Out | ConvertFrom-Json)
    Assert ($guestArgs.Count -eq 3 -and $guestArgs[0] -ceq '-c' -and $guestArgs[1] -ceq 'print("a b")' -and $guestArgs[2] -ceq '雪 & ;') 'Guest argument boundaries changed'
    # Unmanaged reuse never modifies hardware or starts an installation.
    $unmanaged = $s.Clone()
    $unmanaged.StateRoot = Join-Path $testRoot 'unmanaged'
    $before = (Get-Content (Join-Path $testRoot 'calls.jsonl')).Count
    Invoke-VMAction 'create' $unmanaged '' '' | Out-Null
    $after = @(Get-Content (Join-Path $testRoot 'calls.jsonl') | Select-Object -Skip $before)
    Assert (-not ($after -match 'modifyvm|unattended|storageattach')) 'Unmanaged VM changed'
    # A failed tool is a failure, even when it produces plausible output.
    Set-Content (Join-Path $testRoot 'fail') 'showvminfo'
    Must-Fail { Invoke-VMAction 'start' $s '' '' } 'failed'
    Remove-Item -LiteralPath (Join-Path $testRoot 'fail')
    # Resume interrupted configuration, but never repeat an ambiguous installation.
    $retry = $s.Clone()
    $retry.VmName = 'retry-vm'
    Set-Content (Join-Path $testRoot 'fail') 'storagectl'
    Must-Fail { Invoke-VMAction 'create' $retry '' '' } 'failed'
    $retryRecord = @(Get-ChildItem $s.StateRoot -Filter '*.json' | ForEach-Object { Get-Content $_.FullName -Raw | ConvertFrom-Json -AsHashtable } | Where-Object Name -eq 'retry-vm')[0]
    Assert ($retryRecord.Stage -eq 'configuring') 'Recoverable creation stage was lost'
    Assert ($retryRecord.ContainsKey('DiskUUID')) 'Disk ownership was not recorded'
    Set-Content (Join-Path $testRoot 'disk-identity') '11111111-1111-1111-1111-111111111111'
    Must-Fail { Invoke-VMAction 'create' $retry '' '' } 'disk identity changed'
    Remove-Item -LiteralPath (Join-Path $testRoot 'disk-identity')
    Set-Content (Join-Path $testRoot 'fail') 'unattended install'
    Must-Fail { Invoke-VMAction 'create' $retry '' '' } 'failed'
    $retryRecord = Get-Content (Join-Path $s.StateRoot "$($retryRecord.UUID).json") -Raw | ConvertFrom-Json -AsHashtable
    Assert ($retryRecord.Stage -eq 'installing') 'Ambiguous install was not retained'
    Assert (-not (Test-Path (Join-Path $s.StateRoot "$($retryRecord.Attempt)/root-password"))) 'Root credential retained after failure'
    Remove-Item -LiteralPath (Join-Path $testRoot 'fail')
    Invoke-VMAction 'create' $retry '' '' | Out-Null
    $calls = @(Get-Content (Join-Path $testRoot 'calls.jsonl') | ForEach-Object { ,($_ | ConvertFrom-Json) })
    Assert (@($calls | Where-Object { $_[0] -eq 'createvm' -and $_ -contains 'retry-vm' }).Count -eq 1) 'Configuration retry recreated the VM'
    Assert (@($calls | Where-Object { $_[0] -eq 'unattended' -and $_[1] -eq 'install' -and $_ -contains $retryRecord.UUID }).Count -eq 1) 'Failed installation was repeated'
    $collision = $s.Clone()
    $collision.VmName = 'existing-disk'
    $unownedDir = Join-Path $testRoot 'machines/existing-disk'
    [IO.Directory]::CreateDirectory($unownedDir) | Out-Null
    $unownedDisk = Join-Path $unownedDir 'desky-system.vdi'
    [IO.File]::WriteAllText($unownedDisk, 'unrelated data')
    Must-Fail { Invoke-VMAction 'create' $collision '' '' } 'machine folder already exists'
    Assert ([IO.File]::ReadAllText($unownedDisk) -ceq 'unrelated data') 'Unowned disk was modified'
    # Export a real temporary Git commit; private, ignored and dirty files must not arrive.
    $repo = Join-Path $testRoot 'repo with spaces'
    [IO.Directory]::CreateDirectory($repo) | Out-Null
    & git -C $repo init -q
    & git -C $repo config core.autocrlf true
    & git -C $repo config core.eol crlf
    [IO.File]::WriteAllText((Join-Path $repo 'public.txt'), "committed`n")
    [IO.File]::WriteAllText((Join-Path $repo '.gitignore'), ".private`n")
    & git -C $repo add -- public.txt .gitignore
    & git -C $repo -c user.name='VM fixture' -c user.email='fixture@example.invalid' commit -qm fixture
    'dirty' | Set-Content (Join-Path $repo 'public.txt')
    'private' | Set-Content (Join-Path $repo '.private')
    'untracked' | Set-Content (Join-Path $repo 'untracked.txt')
    $exportSettings = Get-VMSettings $repo @{}
    $export = & $module { param($settings) New-SourceExport $settings } $exportSettings
    $manifest = Get-Content $export.Manifest -Raw | ConvertFrom-Json
    Assert ($manifest.files.Count -eq 2) 'Private or untracked files were exported'
    Assert ($manifest.files.path -notcontains '.private') 'Private file in manifest'
    $public = @($manifest.files | Where-Object path -eq 'public.txt')[0]
    $committedHash = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData([Text.Encoding]::UTF8.GetBytes("committed`n"))).ToLowerInvariant()
    Assert ($public.sha256 -ceq $committedHash) 'Host line-ending settings changed exported contents'
    # Repository attributes take precedence over core settings and remain guarded.
    [IO.File]::WriteAllText((Join-Path $repo '.git/info/attributes'), "public.txt text eol=crlf`n")
    Must-Fail { & $module { param($settings) New-SourceExport $settings } $exportSettings } 'Archive transformations'
    Write-Output 'VM host tests passed.'
} finally {
    Remove-Item Env:DESKY_FAKE_VBOX_ROOT -ErrorAction SilentlyContinue
    # Deliberately leave this bounded, non-secret fixture for diagnosis; no recursive cleanup.
    Write-Output "Test artifacts: $testRoot"
}
