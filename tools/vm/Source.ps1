function New-SourceExport {
    param($Settings)
    $repo = $Settings.Repo
    $revision = (Invoke-VMProcess 'git' @('-C', $repo, 'rev-parse', '--verify', 'HEAD')).Out.Trim()
    if ($revision -notmatch '^[0-9a-f]{40,64}$') { throw 'Cannot resolve committed revision.' }
    if ((Invoke-VMProcess 'git' @('-C', $repo, 'status', '--porcelain')).Out) {
        Write-Host 'Working-tree changes are excluded; only the committed revision will be transferred.'
    }
    $dir = Join-Path $Settings.Cache ([guid]::NewGuid().ToString('N'))
    [IO.Directory]::CreateDirectory($dir) | Out-Null
    $archive = Join-Path $dir 'source.tar'
    Invoke-VMProcess 'git' @('-C', $repo, 'archive', '--format=tar', "--output=$archive", $revision) | Out-Null
    $tree = (Invoke-VMProcess 'git' @('-C', $repo, 'ls-tree', '-rz', '--full-tree', $revision)).Out
    $expected = [Collections.Generic.Dictionary[string,object]]::new([StringComparer]::Ordinal)
    foreach ($item in $tree.Split([char]0, [StringSplitOptions]::RemoveEmptyEntries)) {
        if ($item -notmatch '(?s)^(100644|100755|120000) blob ([0-9a-f]+)\t(.+)$') { throw 'Unsupported Git tree entry (submodules are not exported).' }
        $expected.Add($Matches[3], @{ Mode = $Matches[1]; Hash = $Matches[2] })
    }
    $files = [Collections.Generic.List[object]]::new()
    $stream = [IO.File]::OpenRead($archive)
    $tar = [System.Formats.Tar.TarReader]::new($stream)
    try {
        while ($null -ne ($entry = $tar.GetNextEntry())) {
            if ($entry.EntryType -in @([System.Formats.Tar.TarEntryType]::Directory, [System.Formats.Tar.TarEntryType]::GlobalExtendedAttributes)) { continue }
            $name = $entry.Name
            if (-not $expected.ContainsKey($name) -or $name.StartsWith('/') -or ($name.Split('/') -contains '..')) { throw 'Archive entry does not match the committed tree.' }
            $known = $expected[$name]
            $memory = [IO.MemoryStream]::new()
            try {
                if ($entry.EntryType -eq [System.Formats.Tar.TarEntryType]::SymbolicLink) {
                    $bytes = [Text.Encoding]::UTF8.GetBytes($entry.LinkName)
                    $memory.Write($bytes)
                    $type = 'symlink'
                    if ($known.Mode -ne '120000') { throw 'Archive symlink type mismatch.' }
                } elseif ($entry.EntryType -in @([System.Formats.Tar.TarEntryType]::RegularFile, [System.Formats.Tar.TarEntryType]::V7RegularFile)) {
                    if ($entry.DataStream) { $entry.DataStream.CopyTo($memory) }
                    $type = 'file'
                    if ($known.Mode -eq '120000') { throw 'Archive file type mismatch.' }
                } else { throw 'Unsupported archive entry type.' }
                $data = $memory.ToArray()
                $hash = [Security.Cryptography.IncrementalHash]::CreateHash($(if ($known.Hash.Length -eq 40) { [Security.Cryptography.HashAlgorithmName]::SHA1 } else { [Security.Cryptography.HashAlgorithmName]::SHA256 }))
                try {
                    $hash.AppendData([Text.Encoding]::UTF8.GetBytes("blob $($data.Length)`0"))
                    $hash.AppendData($data)
                    if ([Convert]::ToHexString($hash.GetHashAndReset()).ToLowerInvariant() -ne $known.Hash) { throw 'Archive transformations changed committed contents; remove local export attributes before transferring.' }
                } finally { $hash.Dispose() }
                $file = @{ path = $name; type = $type; sha256 = [Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($data)).ToLowerInvariant(); executable = ($known.Mode -eq '100755') }
                if ($type -eq 'symlink') { $file.target = $entry.LinkName }
                $files.Add($file)
            } finally { $memory.Dispose() }
            $expected.Remove($name) | Out-Null
        }
    } finally { $tar.Dispose(); $stream.Dispose() }
    if ($expected.Count -ne 0) { throw 'Archive omitted committed files; export-ignore attributes are not supported for qualification.' }
    $archiveHash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    $manifest = Join-Path $dir 'manifest.json'
    @{ schema = 1; revision = $revision; archive_sha256 = $archiveHash; files = @($files.ToArray()) } |
        ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $manifest -Encoding utf8NoBOM
    return @{ Directory = $dir; Archive = $archive; Manifest = $manifest; Revision = $revision; Hash = $archiveHash }
}

function Send-VMSource {
    param($Settings, [string]$UUID, [string]$PasswordFile, [string]$GuestRoot)
    $export = New-SourceExport $Settings
    # No helper from the uncommitted host tree is executed in the guest. It is extracted from HEAD.
    $transport = Join-Path $export.Directory 'transport.py'
    $source = Invoke-VMProcess 'git' @('-C', $Settings.Repo, 'show', "$($export.Revision):tools/vm/guest.py") -AllowFailure
    if ($source.Code -ne 0) { throw 'The selected commit does not contain the VM tools. Commit the tooling before provisioning.' }
    [IO.File]::WriteAllText($transport, $source.Out, [Text.UTF8Encoding]::new($false))
    $code = @'
import os,sys,tempfile
p=os.path.abspath(sys.argv[1])
for part in [p]+list(__import__('pathlib').Path(p).parents):
 if os.path.islink(part): raise SystemExit('Linked source root refused')
os.makedirs(p,exist_ok=True)
print(tempfile.mkdtemp(prefix='.incoming-',dir=p))
'@
    $incoming = (Invoke-Guest $Settings $UUID $PasswordFile '/usr/bin/python3' @('-c', $code, $GuestRoot)).Out.Trim()
    if (-not $incoming.StartsWith($GuestRoot.TrimEnd('/') + '/.incoming-') -or $incoming -match '[\r\n]') { throw 'Unexpected guest staging path.' }
    foreach ($file in $export.Archive, $export.Manifest, $transport) { Copy-ToGuest $Settings $UUID $PasswordFile $file $incoming }
    $published = (Invoke-Guest $Settings $UUID $PasswordFile '/usr/bin/python3' @("$incoming/transport.py", 'publish', $incoming, $GuestRoot, $export.Hash)).Out.Trim()
    $expected = $GuestRoot.TrimEnd('/') + '/' + $export.Revision
    if ($published -cne $expected) { throw 'Guest source publication did not return the expected revision directory.' }
    $quoted = "'" + $published.Replace("'", "'\''") + "'"
    Write-Output "Committed revision: $($export.Revision)"
    Write-Output "In Ubuntu's desktop terminal, run: cd -- $quoted && bash tools/vm/bootstrap.sh"
    Write-Output 'Then run: python3 tools/vm/guest.py qualify .'
}

function Receive-VMReport {
    param($Settings, [string]$UUID, [string]$PasswordFile, [string]$GuestRoot, [string]$Revision, [string]$RunId)
    if ($Revision -notmatch '^[0-9a-f]{40,64}$' -or $RunId -notmatch '^[0-9a-f]{32}$') { throw 'collect requires -Revision (full commit ID) and -RunId (printed by qualification).' }
    $guestPath = "$($GuestRoot.TrimEnd('/'))/$Revision/.cache/vm-runs/$RunId/report.json"
    $code = @'
import json,os,stat,sys
from pathlib import Path
p=Path(sys.argv[1]); revision=sys.argv[2]; run=sys.argv[3]
for x in [p]+list(p.parents):
 if x.is_symlink(): raise SystemExit('Linked report path refused')
if p.stat().st_size>1048576: raise SystemExit('Report too large')
d=json.loads(p.read_text())
if d.get('schema')!=1 or d.get('revision')!=revision or d.get('run_id')!=run: raise SystemExit('Report identity mismatch')
print(json.dumps(d))
'@
    $result = Invoke-Guest $Settings $UUID $PasswordFile '/usr/bin/python3' @('-c', $code, $guestPath, $Revision, $RunId)
    $report = $result.Out | ConvertFrom-Json -AsHashtable
    if ($report.revision -ne $Revision -or $report.run_id -ne $RunId) { throw 'Unexpected report identity.' }
    $dir = Join-Path $Settings.Cache 'reports'
    [IO.Directory]::CreateDirectory($dir) | Out-Null
    $path = Join-Path $dir "$Revision-$RunId.json"
    if (Test-Path -LiteralPath $path) {
        if ((Get-Content -LiteralPath $path -Raw).Trim() -cne $result.Out.Trim()) { throw 'A differing report already exists locally; preserving it.' }
    } else { [IO.File]::WriteAllText($path, $result.Out, [Text.UTF8Encoding]::new($false)) }
    Write-Output "Report collected: $path"
}
