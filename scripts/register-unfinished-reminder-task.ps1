[CmdletBinding(SupportsShouldProcess)]
param(
	[Parameter(Mandatory = $true)]
	[datetime]$RunAtNorway,
	[string]$TaskName = 'FotballVM Unfinished Reminder',
	[string]$BaseUrl = 'http://localhost:8090',
	[string]$UserId = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

$repoRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$scriptPath = Join-Path $repoRoot 'scripts\send-unfinished-reminder.ps1'
$launcherPath = Join-Path $repoRoot 'scripts\send-unfinished-reminder.cmd'

function Quote-PowerShellLiteral {
	param([Parameter(Mandatory = $true)][string]$Value)
	return "'{0}'" -f $Value.Replace("'", "''")
}

if (-not (Test-Path -LiteralPath $scriptPath -PathType Leaf)) {
	throw 'Reminder script not found: {0}' -f $scriptPath
}
if (-not (Test-Path -LiteralPath $launcherPath -PathType Leaf)) {
	throw 'Reminder launcher not found: {0}' -f $launcherPath
}

$norwayZone = [TimeZoneInfo]::FindSystemTimeZoneById('W. Europe Standard Time')
$runAtNorwayUnspecified = [DateTime]::SpecifyKind($RunAtNorway, [DateTimeKind]::Unspecified)
$runAtLocal = [TimeZoneInfo]::ConvertTime($runAtNorwayUnspecified, $norwayZone, [TimeZoneInfo]::Local)
$taskNameSafe = $TaskName.Replace(':', '-')
if ($taskNameSafe -ne $TaskName) {
	Write-Host ('Adjusted task name to avoid unsupported characters: "{0}".' -f $taskNameSafe)
}
if ($runAtLocal -le (Get-Date)) {
	throw 'The requested run time is not in the future on this machine: {0}' -f $runAtLocal
}

$action = New-ScheduledTaskAction -Execute $launcherPath -WorkingDirectory $repoRoot
$trigger = New-ScheduledTaskTrigger -Once -At $runAtLocal
$settings = New-ScheduledTaskSettingsSet -StartWhenAvailable -MultipleInstances IgnoreNew -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
$principal = New-ScheduledTaskPrincipal -UserId $UserId -LogonType S4U -RunLevel Highest

if ($PSCmdlet.ShouldProcess($taskNameSafe, 'Register one-off unfinished reminder task')) {
	try {
		Register-ScheduledTask -TaskName $taskNameSafe -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Description 'One-off unfinished-user reminder email send for FotballVM.' -Force -ErrorAction Stop | Out-Null
		Write-Host ('Registered scheduled task "{0}".' -f $taskNameSafe)
	} catch {
		if ($_.Exception.Message -notmatch 'Access is denied') {
			throw
		}

		Write-Host 'Register-ScheduledTask was denied. Falling back to schtasks with current user context.'
		& schtasks /Create /SC ONCE /ST $runAtLocal.ToString('HH:mm') /TN $taskNameSafe /TR $launcherPath /F | Out-Null
		if ($LASTEXITCODE -ne 0) {
			throw 'schtasks fallback failed with exit code {0}.' -f $LASTEXITCODE
		}
		Write-Host ('Registered scheduled task "{0}" via schtasks fallback.' -f $taskNameSafe)
	}
	Write-Host ('Norway time: {0:yyyy-MM-dd HH:mm} | Local task time: {1:yyyy-MM-dd HH:mm zzz}' -f $runAtNorwayUnspecified, $runAtLocal)
} else {
	Write-Host ('WhatIf: would register scheduled task "{0}" for {1:yyyy-MM-dd HH:mm zzz} local time.' -f $taskNameSafe, $runAtLocal)
}