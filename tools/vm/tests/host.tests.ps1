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
    $s = Get-VMSettings $testRoot @{ VBoxPath = $fake }
    $s.StateRoot = Join-Path $testRoot 'private'
    Assert ($s.GuestUser -eq 'dev' -and $s.GuestRoot -eq '~/src/desky') 'Defaults differ'
    $config = Join-Path $testRoot 'local.json'
    '{"VmName":"Ubuntu Desktop 雪","GuestRoot":"~/projects/desky"}' | Set-Content $config
    $override = Get-VMSettings $testRoot @{ Config=$config; GuestRoot='/srv/desky' }
    Assert ($override.VmName -eq 'Ubuntu Desktop 雪' -and $override.GuestRoot -eq '/srv/desky') 'Precedence failure'
    '{"Password":"should not be accepted"}' | Set-Content $config
    Must-Fail { Get-VMSettings $testRoot @{Config=$config} } 'Unknown configuration'
    foreach ($setting in 'IsoPath', 'BaseFolder', 'MemoryMB', 'CPUs', 'DiskGB') {
        @{ $setting = 'old-setting' } | ConvertTo-Json | Set-Content $config
        Must-Fail { Get-VMSettings $testRoot @{Config=$config} } 'obsolete creation setting'
    }
    Must-Fail { Get-VMSettings $testRoot @{GuestRoot='/tmp/../root'} } 'GuestRoot'
    Must-Fail { Get-VMSettings $testRoot @{GuestUser='root'} } 'normal Linux account'
    $absent = Invoke-VMAction 'status' $s '' '' | ConvertFrom-Json
    Assert ($absent.State -eq 'absent') 'Missing VM status differs'
    Assert (-not (Test-Path $s.StateRoot)) 'Status wrote private state'
    foreach ($action in 'start', 'provision', 'collect') {
        Must-Fail { Invoke-VMAction $action $s '' '' } 'already-installed Ubuntu VM'
    }
    Assert (-not (Test-Path $s.StateRoot)) 'Missing VM prompted for credentials or wrote state'
    # Removed actions must fail at the public entry point before calling VBox.
    $before = (Get-Content (Join-Path $testRoot 'calls.jsonl')).Count
    $entryResult = & $module {
        param($helper, $executable)
        Invoke-VMProcess 'pwsh' @('-NoProfile', '-File', $helper, 'create', '-VBoxPath', $executable) -AllowFailure
    } (Join-Path $toolDir 'manage.ps1') $fake
    Assert ($entryResult.Code -ne 0) 'Removed create action was accepted'
    Assert ((Get-Content (Join-Path $testRoot 'calls.jsonl')).Count -eq $before) 'Removed action contacted VBox'
    Must-Fail { Invoke-VMAction 'create' $s '' '' } 'ValidateSet|validation|does not belong'
    & pwsh -NoProfile -File (Join-Path $toolDir 'manage.ps1') status -VBoxPath $fake -VmName 'missing-fixture' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Public helper entry point failed' }

    # Supply an existing VM; the fake executable has no creation operations.
    $id = 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee'
    $statePath = Join-Path $testRoot 'vm.json'
    $vm = @{ UUID=$id; name='Ubuntu Desktop 雪'; VMState='poweroff'; firmware='efi'; memory='8192'; 'SATA-0-0'='existing.vdi'; 'SATA-1-0'='existing.iso' }
    $vm | ConvertTo-Json | Set-Content $statePath
    $s.VmName = $vm.name
    $existing = Invoke-VMAction 'status' $s '' '' | ConvertFrom-Json
    Assert ($existing.State -eq 'poweroff') 'Existing VM lookup failed'
    $byId = $s.Clone()
    $byId.VmName = $id
    Assert ((Invoke-VMAction 'status' $byId '' '' | ConvertFrom-Json).State -eq 'poweroff') 'UUID lookup failed'
    Must-Fail { Invoke-VMAction 'provision' $s '' '' } 'Start the VM'
    Assert (-not (Test-Path $s.StateRoot)) 'Powered-off VM prompted for credentials'
    Invoke-VMAction 'start' $s '' '' | Out-Null
    Invoke-VMAction 'start' $byId '' '' | Out-Null
    $calls = @(Get-Content (Join-Path $testRoot 'calls.jsonl') | ForEach-Object { ,@($_ | ConvertFrom-Json) })
    Assert (@($calls | Where-Object { $_[0] -eq 'startvm' }).Count -eq 1) 'Repeated start was not harmless'
    $running = Get-Content $statePath -Raw | ConvertFrom-Json -AsHashtable
    foreach ($key in 'firmware', 'memory', 'SATA-0-0', 'SATA-1-0') { Assert ($running[$key] -eq $vm[$key]) 'Existing VM settings changed' }

    # Retired installer artifacts must neither gate reuse nor be modified/deleted.
    [IO.Directory]::CreateDirectory($s.StateRoot) | Out-Null
    $legacy = Join-Path $s.StateRoot "$id.json"
    [IO.File]::WriteAllText($legacy, 'obsolete incomplete record, deliberately not valid JSON')
    $legacyDir = Join-Path $s.StateRoot 'old-installation'
    [IO.Directory]::CreateDirectory($legacyDir) | Out-Null
    $legacyFile = Join-Path $legacyDir 'answer-file.fixture'
    [IO.File]::WriteAllText($legacyFile, 'synthetic private fixture')
    $legacyHash = (Get-FileHash $legacy).Hash
    $artifactHash = (Get-FileHash $legacyFile).Hash
    Invoke-VMAction 'status' $s '' '' | Out-Null
    # Exercise the real password-file lifetime using a synthetic masked response.
    # Stub only source/report delivery; their integrity tests remain separate.
    & $module {
        function script:Read-Host {
            param([string]$Prompt, [switch]$AsSecureString)
            ConvertTo-SecureString 'test-only-value' -AsPlainText -Force
        }
        function script:Send-VMSource($Settings, $UUID, $PasswordFile, $GuestRoot) {
            if (-not (Test-Path -LiteralPath $PasswordFile)) { throw 'Password file missing during operation' }
            if (Test-Path (Join-Path $Settings.Repo 'transfer-failure')) { throw 'Synthetic transfer failure' }
            @{ UUID=$UUID; Root=$GuestRoot } | ConvertTo-Json -Compress | Add-Content (Join-Path $Settings.Repo 'deliveries.jsonl')
        }
        function script:Receive-VMReport($Settings, $UUID, $PasswordFile, $GuestRoot, $Revision, $RunId) {
            if (-not (Test-Path -LiteralPath $PasswordFile)) { throw 'Password file missing during collection' }
            @{ UUID=$UUID; Root=$GuestRoot; Revision=$Revision; RunId=$RunId } | ConvertTo-Json -Compress | Add-Content (Join-Path $Settings.Repo 'collections.jsonl')
        }
    }
    Invoke-VMAction 'provision' $s '' '' | Out-Null
    Invoke-VMAction 'provision' $s '' '' | Out-Null
    $deliveries = @(Get-Content (Join-Path $testRoot 'deliveries.jsonl') | ForEach-Object { $_ | ConvertFrom-Json })
    Assert ($deliveries.Count -eq 2 -and $deliveries[0].Root -eq '/home/dev/src/desky' -and $deliveries[0].UUID -eq $id) 'Existing VM provisioning failed'
    Invoke-VMAction 'collect' $s ('a' * 40) ('b' * 32) | Out-Null
    $collection = Get-Content (Join-Path $testRoot 'collections.jsonl') | ConvertFrom-Json
    Assert ($collection.Revision -eq ('a' * 40) -and $collection.RunId -eq ('b' * 32)) 'Selected report identity changed'
    foreach ($failure in 'guestcontrol', 'guest-additions', 'live-session') {
        Set-Content (Join-Path $testRoot 'fail') $failure
        Must-Fail { Invoke-VMAction 'provision' $s '' '' } 'Guest not ready or authentication failed'
        Remove-Item -LiteralPath (Join-Path $testRoot 'fail')
    }
    Set-Content (Join-Path $testRoot 'guest-home') 'relative-home'
    Must-Fail { Invoke-VMAction 'provision' $s '' '' } 'Guest not ready'
    Remove-Item -LiteralPath (Join-Path $testRoot 'guest-home')
    Assert ((Get-Content (Join-Path $testRoot 'deliveries.jsonl')).Count -eq 2) 'Failed readiness transferred source'
    Set-Content (Join-Path $testRoot 'transfer-failure') 'fixture'
    Must-Fail { Invoke-VMAction 'provision' $s '' '' } 'Synthetic transfer failure'
    Remove-Item -LiteralPath (Join-Path $testRoot 'transfer-failure')
    Assert (@(Get-ChildItem $s.StateRoot -Directory).Count -eq 1) 'Temporary credential directories survived'
    Assert ((Get-FileHash $legacy).Hash -eq $legacyHash -and (Get-FileHash $legacyFile).Hash -eq $artifactHash) 'Retired installation artifacts changed'
    Assert (-not ((Get-Content (Join-Path $testRoot 'calls.jsonl') -Raw).Contains('test-only-value'))) 'Credential leaked to command arguments'
    $guestResult = & $module {
        param($settings, $uuid)
        Invoke-Guest $settings $uuid 'fixture-password-file' '/usr/bin/python3' @('-c', 'print("a b")', '雪 & ;')
    } $s $id
    $guestArgs = @($guestResult.Out | ConvertFrom-Json)
    Assert ($guestArgs.Count -eq 3 -and $guestArgs[0] -ceq '-c' -and $guestArgs[1] -ceq 'print("a b")' -and $guestArgs[2] -ceq '雪 & ;') 'Guest argument boundaries changed'
    Set-Content (Join-Path $testRoot 'fail') 'showvminfo'
    Must-Fail { Invoke-VMAction 'start' $s '' '' } 'failed'
    Remove-Item -LiteralPath (Join-Path $testRoot 'fail')
    $calls = @(Get-Content (Join-Path $testRoot 'calls.jsonl') | ForEach-Object { ,@($_ | ConvertFrom-Json) })
    Assert (@($calls | Where-Object { $_[0] -notin @('--version', 'list', 'showvminfo', 'startvm', 'guestcontrol') }).Count -eq 0) 'Unexpected VM management command'
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

    # Exercise real export/staging/copy/publication orchestration instead of the
    # lifecycle stub above. The fake models VBox's trailing-slash copy semantics.
    [IO.File]::WriteAllText((Join-Path $repo '.git/info/attributes'), '')
    [IO.Directory]::CreateDirectory((Join-Path $repo 'tools/vm')) | Out-Null
    Copy-Item -LiteralPath (Join-Path $toolDir 'guest.py') -Destination (Join-Path $repo 'tools/vm/guest.py')
    & git -C $repo add -- tools/vm/guest.py
    & git -C $repo -c user.name='VM fixture' -c user.email='fixture@example.invalid' commit -qm 'add guest fixture'
    if ($LASTEXITCODE) { throw 'Transfer fixture commit failed' }
    $exportSettings.VBoxPath = $fake
    $guestRoot = '/home/dev/source 雪 & spaces; punctuation'
    $revision = (& git -C $repo rev-parse HEAD).Trim()
    $before = @(Get-Content (Join-Path $testRoot 'calls.jsonl')).Count
    $result = & $module {
        param($sourceFile, $settings, $uuid, $root)
        . $sourceFile
        Send-VMSource $settings $uuid 'fixture-password-file' $root
    } (Join-Path $toolDir 'Source.ps1') $exportSettings $id $guestRoot
    Assert (($result -join "`n").Contains("Committed revision: $revision")) 'Transfer did not acknowledge publication'
    $transferCalls = @(Get-Content (Join-Path $testRoot 'calls.jsonl') | Select-Object -Skip $before | ForEach-Object { ,@($_ | ConvertFrom-Json) })
    $copies = @($transferCalls | Where-Object { $_[2] -eq 'copyto' })
    Assert ($copies.Count -eq 3) 'Transfer did not copy all three workflow files'
    $destination = $copies[0][[array]::IndexOf($copies[0], '--target-directory') + 1]
    Assert ($destination.StartsWith($guestRoot + '/.incoming-') -and $destination.EndsWith('/')) 'Guest directory argument lost its path or trailing slash'
    $staging = Join-Path $testRoot ($destination.TrimEnd('/').Split('/')[-1])
    $originalHash = (Get-FileHash -LiteralPath (Join-Path $staging 'source.tar')).Hash
    # A direct call without the separator must fail just as the real vendor does.
    Must-Fail { & $module {
        param($settings, $uuid, $dest, $source)
        Invoke-VBox $settings @('guestcontrol', $uuid, 'copyto', '--no-replace', '--target-directory', $dest.TrimEnd('/'), $source)
    } $exportSettings $id $destination $export.Archive } 'Tool operation.*failed'
    Must-Fail { & $module {
        param($settings, $uuid, $dest, $source)
        Copy-ToGuest $settings $uuid 'fixture-password-file' $source $dest
    } $exportSettings $id $destination $export.Archive } 'Tool operation.*failed'
    Assert ((Get-FileHash -LiteralPath (Join-Path $staging 'source.tar')).Hash -eq $originalHash) 'Existing guest copy was replaced'
    # Lost or noisy process output is harmless only with a matching receipt.
    foreach ($failure in 'publication-empty', 'publication-noisy', 'publication-wrong') {
        Set-Content (Join-Path $testRoot 'fail') $failure
        $result = & $module {
            param($sourceFile, $settings, $uuid, $root)
            . $sourceFile
            Send-VMSource $settings $uuid 'fixture-password-file' $root
        } (Join-Path $toolDir 'Source.ps1') $exportSettings $id $guestRoot
        Assert (($result -join "`n").Contains("Committed revision: $revision")) 'Valid receipt was rejected after lost/noisy process output'
        Assert (-not ($result -join "`n").Contains('fixture-private-detail')) 'Raw publication output was exposed'
        Remove-Item -LiteralPath (Join-Path $testRoot 'fail')
    }
    foreach ($case in @(
        @{ Failure='stage'; Message='Creating guest staging directory failed'; Copies=0; Publishes=0 },
        @{ Failure='copyto-source.tar'; Message='Copying source.tar to guest staging failed'; Copies=1; Publishes=0 },
        @{ Failure='copyto-manifest.json'; Message='Copying manifest.json to guest staging failed'; Copies=2; Publishes=0 },
        @{ Failure='copyto-transport.py'; Message='Copying transport.py to guest staging failed'; Copies=3; Publishes=0 },
        @{ Failure='publish'; Message='Verifying and publishing committed source failed'; Copies=3; Publishes=1 },
        @{ Failure='receipt-missing'; Message='Retrieving publication confirmation failed'; Copies=3; Publishes=1 },
        @{ Failure='receipt-copy'; Message='Retrieving publication confirmation failed'; Copies=3; Publishes=1 },
        @{ Failure='receipt-invalid'; Message='Invalid publication confirmation'; Copies=3; Publishes=1 },
        @{ Failure='receipt-wrong-schema'; Message='Publication confirmation does not match'; Copies=3; Publishes=1 },
        @{ Failure='receipt-wrong-revision'; Message='Publication confirmation does not match'; Copies=3; Publishes=1 },
        @{ Failure='receipt-wrong-directory'; Message='Publication confirmation does not match'; Copies=3; Publishes=1 },
        @{ Failure='receipt-wrong-archive_sha256'; Message='Publication confirmation does not match'; Copies=3; Publishes=1 },
        @{ Failure='receipt-wrong-incoming'; Message='Publication confirmation does not match'; Copies=3; Publishes=1 }
    )) {
        Set-Content (Join-Path $testRoot 'fail') $case.Failure
        $before = @(Get-Content (Join-Path $testRoot 'calls.jsonl')).Count
        $message = ''
        $result = @()
        try {
            $result = & $module {
                param($sourceFile, $settings, $uuid, $root)
                . $sourceFile
                Send-VMSource $settings $uuid 'fixture-password-file' $root
            } (Join-Path $toolDir 'Source.ps1') $exportSettings $id $guestRoot
        } catch { $message = $_.Exception.Message }
        Assert ($message.Contains($case.Message)) 'Transfer failure did not identify its stage'
        Assert (-not $message.Contains('fixture-private-detail')) 'Transfer failure exposed raw diagnostics'
        Assert (-not (($result -join "`n").Contains('Committed revision:'))) 'Failed transfer was acknowledged'
        $failedCalls = @(Get-Content (Join-Path $testRoot 'calls.jsonl') | Select-Object -Skip $before | ForEach-Object { ,@($_ | ConvertFrom-Json) })
        Assert (@($failedCalls | Where-Object { $_[2] -eq 'copyto' }).Count -eq $case.Copies) 'Transfer continued copying after a failure'
        Assert (@($failedCalls | Where-Object { $_ -contains 'publish' }).Count -eq $case.Publishes) 'Partial transfer reached publication'
        Remove-Item -LiteralPath (Join-Path $testRoot 'fail')
    }
    Write-Output 'VM host tests passed.'
} finally {
    Remove-Item Env:DESKY_FAKE_VBOX_ROOT -ErrorAction SilentlyContinue
    # Deliberately leave this bounded, non-secret fixture for diagnosis; no recursive cleanup.
    Write-Output "Test artifacts: $testRoot"
}
