[CmdletBinding()]
param(
    [string]$PublicRepoUrl = 'https://github.com/oyvhov/world-cup-pool.git',
    [string]$CommitMessage = '',
    [switch]$SkipLeakScan,
    [switch]$KeepExport
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

function Invoke-Checked {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $true)][string[]]$Arguments,
        [Parameter(Mandatory = $true)][string]$Description
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw ('{0} failed with exit code {1}.' -f $Description, $LASTEXITCODE)
    }
}

$repoRoot = (& git rev-parse --show-toplevel).Trim()
if (-not $repoRoot) {
    throw 'Not inside a git repository.'
}

Set-Location $repoRoot

$dirty = (& git status --porcelain)
if ($dirty) {
    throw 'Working tree is not clean. Commit your changes first, then run sync-public.ps1.'
}

$head = (& git rev-parse --short HEAD).Trim()
$stamp = Get-Date -Format 'yyyyMMddHHmmss'
if (-not $CommitMessage) {
    $CommitMessage = ((& git log -1 --format=%B HEAD) | Out-String).TrimEnd([char[]]"`r`n")
    if (-not $CommitMessage) {
        $CommitMessage = 'Public sync from private HEAD ' + $head
    }
}

$exportRoot = 'C:\public-sync-temp'
if (-not (Test-Path -LiteralPath $exportRoot -PathType Container)) {
    New-Item -ItemType Directory -Path $exportRoot -Force | Out-Null
}
$exportDir = Join-Path $exportRoot ('world-cup-pool-public-' + $stamp)
$archivePath = Join-Path $exportDir 'source.zip'
$commitMessagePath = Join-Path $exportDir '.public-commit-message.txt'

New-Item -ItemType Directory -Path $exportDir -Force | Out-Null

Invoke-Checked -FilePath 'git' -Arguments @('archive', '--format=zip', '-o', $archivePath, 'HEAD') -Description 'git archive'
Expand-Archive -Path $archivePath -DestinationPath $exportDir -Force
if (Test-Path -LiteralPath $archivePath) {
    Remove-Item -Force -LiteralPath $archivePath
}

Push-Location $exportDir
try {
    Invoke-Checked -FilePath 'git' -Arguments @('init', '-b', 'main') -Description 'git init'
    Invoke-Checked -FilePath 'git' -Arguments @('add', '-A') -Description 'git add'
    [System.IO.File]::WriteAllText(
        $commitMessagePath,
        $CommitMessage,
        (New-Object System.Text.UTF8Encoding($false))
    )
    Invoke-Checked -FilePath 'git' -Arguments @('commit', '-F', $commitMessagePath) -Description 'git commit'
    if (Test-Path -LiteralPath $commitMessagePath) {
        Remove-Item -Force -LiteralPath $commitMessagePath
    }

    if (-not $SkipLeakScan) {
        $dockerCmd = Get-Command docker -ErrorAction SilentlyContinue
        if (-not $dockerCmd) {
            throw 'Docker is required for leak scan. Install Docker or run with -SkipLeakScan.'
        }

        Invoke-Checked -FilePath 'docker' -Arguments @(
            'run', '--rm', '-v', ('{0}:/repo' -f $exportDir),
            'zricethezav/gitleaks:latest',
            'detect', '--source=/repo', '--redact', '--no-banner'
        ) -Description 'gitleaks scan'
    }

    Invoke-Checked -FilePath 'git' -Arguments @('remote', 'add', 'origin', $PublicRepoUrl) -Description 'git remote add origin'
    Invoke-Checked -FilePath 'git' -Arguments @('push', '--force', '-u', 'origin', 'main') -Description 'git push'
}
finally {
    if (Test-Path -LiteralPath $commitMessagePath) {
        Remove-Item -Force -LiteralPath $commitMessagePath
    }
    Pop-Location
}

if (-not $KeepExport) {
    if (Test-Path -LiteralPath $exportDir) {
        Remove-Item -Recurse -Force -LiteralPath $exportDir
    }
}

Write-Host ('Public sync complete from private HEAD {0}' -f $head)
Write-Host ('Public repo: {0}' -f $PublicRepoUrl)
