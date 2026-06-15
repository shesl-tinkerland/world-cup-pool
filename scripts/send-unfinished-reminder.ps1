[CmdletBinding(SupportsShouldProcess)]
param(
	[string]$BaseUrl = 'http://localhost:8090',
	[string]$EnvFile = (Join-Path ([System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))) '.env')
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

function Get-EnvValue {
	param(
		[Parameter(Mandatory = $true)][string]$Path,
		[Parameter(Mandatory = $true)][string]$Name
	)

	$line = Get-Content -LiteralPath $Path | Where-Object { $_ -match ('^{0}=' -f [regex]::Escape($Name)) } | Select-Object -First 1
	if (-not $line) {
		return ''
	}
	return ($line -replace '^[^=]+=','').Trim()
}

if (-not (Test-Path -LiteralPath $EnvFile -PathType Leaf)) {
	throw '.env file not found: {0}' -f $EnvFile
}

$adminEmail = Get-EnvValue -Path $EnvFile -Name 'PB_ADMIN_EMAIL'
$adminPassword = Get-EnvValue -Path $EnvFile -Name 'PB_ADMIN_PASSWORD'
if ([string]::IsNullOrWhiteSpace($adminEmail) -or [string]::IsNullOrWhiteSpace($adminPassword)) {
	throw 'PB_ADMIN_EMAIL and PB_ADMIN_PASSWORD must both be set in {0}.' -f $EnvFile
}

$base = $BaseUrl.TrimEnd('/')
if (-not $PSCmdlet.ShouldProcess($base, 'Send pre-kickoff reminder email to all unfinished users')) {
	return
}

$authBody = @{
	identity = $adminEmail
	password = $adminPassword
} | ConvertTo-Json

$auth = Invoke-RestMethod -Method Post -Uri ('{0}/api/collections/_superusers/auth-with-password' -f $base) -ContentType 'application/json' -Body $authBody
if (-not $auth.token) {
	throw 'Superuser login succeeded without returning a token.'
}

$headers = @{ Authorization = 'Bearer {0}' -f $auth.token }
$response = Invoke-RestMethod -Method Post -Uri ('{0}/api/notifications/send-incomplete' -f $base) -Headers $headers -ContentType 'application/json' -Body '{}'

$summary = $response.summary
Write-Host ('Unfinished reminder run completed. Incomplete: {0}, sent: {1}, already sent: {2}, failed: {3}.' -f $summary.incomplete, $summary.sent, $summary.alreadySent, $summary.failed)