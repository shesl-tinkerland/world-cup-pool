[CmdletBinding(SupportsShouldProcess)]
param(
	[string]$BackupRoot = '',
	[string]$BackupName = '',
	[string]$ContainerName = 'fhun_tips_restore_test',
	[string]$VolumeName = 'world_cup_pool_restore_test_pb_data',
	[ValidateRange(1024, 65535)]
	[int]$Port = 8092,
	[string]$Image = 'wm-pickems:latest',
	[string]$EnvFile = '',
	[string]$TestUserIdentity = '',
	[System.Security.SecureString]$TestUserPassword
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
if (-not $EnvFile) {
	$EnvFile = Join-Path $repoRoot '.env'
}
if (-not $BackupRoot) {
	$BackupRoot = Join-Path $repoRoot 'backups\prod'
}
if ($Port -eq 8090) {
	throw 'Refusing to use port 8090. That port is reserved for production in this workspace.'
}

$BackupRoot = [System.IO.Path]::GetFullPath($BackupRoot)
$logRoot = [System.IO.Path]::GetFullPath((Join-Path $repoRoot 'backups'))
$logPath = Join-Path $logRoot 'backup.log'

function Ensure-Directory {
	param([Parameter(Mandatory = $true)][string]$Path)
	if (-not (Test-Path -LiteralPath $Path -PathType Container)) {
		New-Item -ItemType Directory -Path $Path -Force | Out-Null
	}
}

Ensure-Directory -Path $logRoot

function Write-Log {
	param(
		[Parameter(Mandatory = $true)][string]$Message,
		[ValidateSet('INFO', 'WARN', 'ERROR')]
		[string]$Level = 'INFO'
	)

	$line = '[{0}] [{1}] {2}' -f (Get-Date).ToString('s'), $Level, $Message
	Write-Host $line
	Add-Content -LiteralPath $logPath -Value $line
}

function Invoke-Checked {
	param(
		[Parameter(Mandatory = $true)][string]$FilePath,
		[Parameter(Mandatory = $true)][string[]]$Arguments,
		[Parameter(Mandatory = $true)][string]$Description
	)

	& $FilePath @Arguments
	if ($LASTEXITCODE -ne 0) {
		throw '{0} failed with exit code {1}.' -f $Description, $LASTEXITCODE
	}
}

function Invoke-LoggedStep {
	param(
		[Parameter(Mandatory = $true)][string]$Target,
		[Parameter(Mandatory = $true)][string]$Action,
		[Parameter(Mandatory = $true)][scriptblock]$Operation
	)

	if ($PSCmdlet.ShouldProcess($Target, $Action)) {
		$script:LASTEXITCODE = 0
		return (& $Operation)
	}

	Write-Log ('WhatIf: {0} -> {1}' -f $Action, $Target)
	$script:LASTEXITCODE = 0
	if ($LASTEXITCODE -ne 0) {
		throw '{0} failed with exit code {1}.' -f $Action, $LASTEXITCODE
	}
}

function Test-DockerResource {
	param(
		[ValidateSet('container', 'volume', 'image')]
		[string]$Kind,
		[Parameter(Mandatory = $true)][string]$Name
	)

	$previousPreference = $ErrorActionPreference
	try {
		$ErrorActionPreference = 'Continue'
		& docker $Kind inspect $Name *> $null
		return ($LASTEXITCODE -eq 0)
	} finally {
		$ErrorActionPreference = $previousPreference
	}
}

function Get-BackupDirectories {
	param([Parameter(Mandatory = $true)][string]$Root)

	if (-not (Test-Path -LiteralPath $Root -PathType Container)) {
		return @()
	}

	return @(Get-ChildItem -LiteralPath $Root -Directory |
		Where-Object { $_.Name -match '^prod-\d{8}-\d{6}$' } |
		Sort-Object Name -Descending)
}

function Resolve-BackupDirectory {
	param(
		[Parameter(Mandatory = $true)][string]$Root,
		[string]$Name
	)

	if ($Name) {
		$path = Join-Path $Root $Name
		if (-not (Test-Path -LiteralPath $path -PathType Container)) {
			if ($WhatIfPreference) {
				return [pscustomobject]@{
					Name = $Name
					FullName = $path
				}
			}
			throw 'Backup snapshot not found: {0}' -f $path
		}
		return Get-Item -LiteralPath $path
	}

	$latest = @(Get-BackupDirectories -Root $Root) | Select-Object -First 1
	if (-not $latest) {
		if ($WhatIfPreference) {
			return [pscustomobject]@{
				Name = 'prod-whatif'
				FullName = (Join-Path $Root 'prod-whatif')
			}
		}
		throw 'No backup snapshots found under {0}' -f $Root
	}
	return $latest
}

function Wait-ForHealth {
	param(
		[Parameter(Mandatory = $true)][string]$Url,
		[int]$Attempts = 30,
		[int]$DelaySeconds = 2
	)

	for ($attempt = 1; $attempt -le $Attempts; $attempt += 1) {
		try {
			$response = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 5
			if ($response.StatusCode -eq 200) {
				return $response
			}
		} catch {
			if ($attempt -eq $Attempts) {
				throw
			}
		}
		Start-Sleep -Seconds $DelaySeconds
	}

	throw 'Timed out waiting for {0}' -f $Url
}

function Get-RestoredDataCounts {
	param([Parameter(Mandatory = $true)][string]$RestoredVolumeName)

	$scriptText = @'
import json, sqlite3
conn = sqlite3.connect('/pb_data/data.db')
counts = {}
for table in ('leagues', 'tips', 'users'):
    try:
        counts[table] = conn.execute(f'SELECT COUNT(*) FROM {table}').fetchone()[0]
    except Exception as exc:
        counts[table] = str(exc)
print(json.dumps(counts))
'@
	$raw = $scriptText | docker run --rm -i -v ('{0}:/pb_data:ro' -f $RestoredVolumeName) python:3.12-alpine python -
	if ($LASTEXITCODE -ne 0) {
		throw 'Failed to inspect restored SQLite data.'
	}
	return ($raw | ConvertFrom-Json)
}

function ConvertTo-PlainText {
	param([Parameter(Mandatory = $true)][System.Security.SecureString]$SecureValue)

	$ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($SecureValue)
	try {
		return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr)
	} finally {
		[Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr)
	}
}

function Test-RestoredLogin {
	param(
		[Parameter(Mandatory = $true)][string]$BaseUrl,
		[Parameter(Mandatory = $true)][string]$Identity,
		[Parameter(Mandatory = $true)][System.Security.SecureString]$Password
	)

	$plainPassword = ConvertTo-PlainText -SecureValue $Password
	try {
		$body = @{
			identity = $Identity
			password = $plainPassword
		} | ConvertTo-Json
		$response = Invoke-RestMethod -Method Post -Uri ('{0}/api/collections/users/auth-with-password' -f $BaseUrl) -ContentType 'application/json' -Body $body
		if (-not $response.token) {
			throw 'Login succeeded without returning a PocketBase auth token.'
		}
	} finally {
		$plainPassword = $null
	}
}

function Test-AutoRegistrationLogin {
	param([Parameter(Mandatory = $true)][string]$BaseUrl)

	$suffix = [Guid]::NewGuid().ToString('N').Substring(0, 10)
	$email = 'restore-{0}@example.invalid' -f $suffix
	$password = 'Restore-{0}-Aa1!' -f $suffix
	$registerBody = @{
		name = 'Restore Test {0}' -f $suffix
		email = $email
		password = $password
		passwordConfirm = $password
	} | ConvertTo-Json

	Invoke-RestMethod -Method Post -Uri ('{0}/api/collections/users/records' -f $BaseUrl) -ContentType 'application/json' -Body $registerBody | Out-Null
	$loginBody = @{
		identity = $email
		password = $password
	} | ConvertTo-Json
	$response = Invoke-RestMethod -Method Post -Uri ('{0}/api/collections/users/auth-with-password' -f $BaseUrl) -ContentType 'application/json' -Body $loginBody
	if (-not $response.token) {
		throw 'Auto-created restore test user could not log in.'
	}
	return $email
}

try {
	if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
		throw 'Required tool not found on PATH: docker'
	}

	if (-not (Test-Path -LiteralPath $EnvFile -PathType Leaf)) {
		throw '.env file not found: {0}' -f $EnvFile
	}

	if ($TestUserIdentity -and (-not $PSBoundParameters.ContainsKey('TestUserPassword'))) {
		throw 'TestUserPassword is required when TestUserIdentity is provided.'
	}

	$backupDir = Resolve-BackupDirectory -Root $BackupRoot -Name $BackupName
	$archive = $null
	if (Test-Path -LiteralPath $backupDir.FullName -PathType Container) {
		$archive = Get-ChildItem -LiteralPath $backupDir.FullName -Filter '*-volume.tar.gz' -File |
			Sort-Object Name -Descending |
			Select-Object -First 1
	}
	if (-not $archive) {
		if ($WhatIfPreference) {
			$archive = [pscustomobject]@{
				Name = 'prod-whatif-volume.tar.gz'
				FullName = (Join-Path $backupDir.FullName 'prod-whatif-volume.tar.gz')
			}
		} else {
		throw 'No volume archive found in {0}' -f $backupDir.FullName
		}
	}

	Write-Log ('Restoring snapshot {0} into volume {1} on port {2}' -f $backupDir.Name, $VolumeName, $Port)

	if (Test-DockerResource -Kind 'container' -Name $ContainerName) {
		Invoke-LoggedStep -Target $ContainerName -Action 'remove existing restore container' -Operation {
			Invoke-Checked -FilePath 'docker' -Arguments @('rm', '-f', $ContainerName) -Description 'docker rm restore container'
		}
	}

	if (Test-DockerResource -Kind 'volume' -Name $VolumeName) {
		Invoke-LoggedStep -Target $VolumeName -Action 'remove existing restore volume' -Operation {
			Invoke-Checked -FilePath 'docker' -Arguments @('volume', 'rm', '-f', $VolumeName) -Description 'docker volume rm restore volume'
		}
	}

	Invoke-LoggedStep -Target $VolumeName -Action 'create restore volume' -Operation {
		Invoke-Checked -FilePath 'docker' -Arguments @('volume', 'create', $VolumeName) -Description 'docker volume create restore volume'
	}

	Invoke-LoggedStep -Target $archive.FullName -Action 'extract backup archive into restore volume' -Operation {
		Invoke-Checked -FilePath 'docker' -Arguments @(
			'run', '--rm',
			'-v', ('{0}:/restore' -f $VolumeName),
			'-v', ('{0}:/backup:ro' -f $backupDir.FullName),
			'alpine', 'sh', '-lc',
			('tar xzf /backup/{0} -C /restore' -f $archive.Name)
		) -Description 'restore volume archive'
	}

	if (-not (Test-DockerResource -Kind 'image' -Name $Image)) {
		Invoke-LoggedStep -Target $Image -Action 'build local application image' -Operation {
			Push-Location $repoRoot
			try {
				Invoke-Checked -FilePath 'docker' -Arguments @('compose', 'build', 'app') -Description 'docker compose build app'
			} finally {
				Pop-Location
			}
		}
	}

	Invoke-LoggedStep -Target $ContainerName -Action 'start restore validation container' -Operation {
		Invoke-Checked -FilePath 'docker' -Arguments @(
			'run', '-d',
			'--name', $ContainerName,
			'--restart', 'unless-stopped',
			'-p', ('{0}:8090' -f $Port),
			'--env-file', $EnvFile,
			'-e', 'WMP_DEV=1',
			'-v', ('{0}:/pb_data' -f $VolumeName),
			$Image
		) -Description 'docker run restore validation container'
	}

	if ($WhatIfPreference) {
		Write-Log 'WhatIf mode: skipping health, data and login verification.'
		return
	}

	$baseUrl = 'http://127.0.0.1:{0}' -f $Port
	Wait-ForHealth -Url ('{0}/api/health' -f $baseUrl) | Out-Null
	Write-Log ('Health check passed at {0}/api/health' -f $baseUrl)

	$dataCounts = Get-RestoredDataCounts -RestoredVolumeName $VolumeName
	if ([int]$dataCounts.leagues -le 0 -or [int]$dataCounts.tips -le 0) {
		throw 'Restored data did not contain leagues and tips. Counts: leagues={0}, tips={1}' -f $dataCounts.leagues, $dataCounts.tips
	}
	Write-Log ('Restored data counts: leagues={0}, tips={1}, users={2}' -f $dataCounts.leagues, $dataCounts.tips, $dataCounts.users)

	if ($TestUserIdentity) {
		Test-RestoredLogin -BaseUrl $baseUrl -Identity $TestUserIdentity -Password $TestUserPassword
		Write-Log ('Login check passed for {0}' -f $TestUserIdentity)
	} else {
		$autoUser = Test-AutoRegistrationLogin -BaseUrl $baseUrl
		Write-Log ('Login check passed with temporary restore-only user {0}' -f $autoUser)
	}

	Write-Log ('Restore validation is ready at {0}' -f $baseUrl)
} catch {
	Write-Log ('Restore validation failed: {0}' -f $_.Exception.Message) 'ERROR'
	throw
}