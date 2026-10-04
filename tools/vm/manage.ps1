#Requires -Version 7.4
[CmdletBinding()]
param(
    [Parameter(Mandatory, Position = 0)]
    [ValidateSet('status', 'create', 'start', 'provision', 'collect')][string]$Action,
    [string]$Config, [string]$VmName, [string]$IsoPath, [string]$BaseFolder,
    [string]$GuestUser, [string]$GuestRoot, [string]$VBoxPath,
    [int]$MemoryMB, [int]$CPUs, [int]$DiskGB,
    [string]$Revision, [string]$RunId
)
$ErrorActionPreference = 'Stop'
Import-Module (Join-Path $PSScriptRoot 'Vm.psm1') -Force
try {
    if (-not $IsWindows) { throw 'The VM host helper requires Windows and PowerShell 7.4 or newer.' }
    $repo = Split-Path (Split-Path $PSScriptRoot)
    $settings = Get-VMSettings $repo $PSBoundParameters
    Invoke-VMAction $Action $settings $Revision $RunId
} catch {
    Write-Error $_.Exception.Message -ErrorAction Continue
    exit 1
}
