param(
  [ValidateSet("add", "remove")][string]$Action,
  [string]$Directory,
  [switch]$SelfTest
)

$ErrorActionPreference = "Stop"

function Get-NormalizedPathEntry {
  param([string]$Entry)

  if ([string]::IsNullOrWhiteSpace($Entry)) {
    return ""
  }
  $expanded = [Environment]::ExpandEnvironmentVariables($Entry.Trim().Trim('"'))
  try {
    return [IO.Path]::GetFullPath($expanded).TrimEnd('\', '/')
  } catch {
    return $expanded.TrimEnd('\', '/')
  }
}

function Update-PathValue {
  param(
    [string]$Value,
    [ValidateSet("add", "remove")][string]$Action,
    [string]$Directory
  )

  $target = Get-NormalizedPathEntry $Directory
  if ([string]::IsNullOrWhiteSpace($target)) {
    throw "PATH directory must not be empty"
  }

  $entries = @()
  if (-not [string]::IsNullOrWhiteSpace($Value)) {
    $entries = @($Value -split ';' | ForEach-Object { $_.Trim() } | Where-Object { $_ -ne "" })
  }
  $matchesTarget = {
    param([string]$Entry)
    [StringComparer]::OrdinalIgnoreCase.Equals((Get-NormalizedPathEntry $Entry), $target)
  }

  if ($Action -eq "add") {
    if (-not ($entries | Where-Object { & $matchesTarget $_ })) {
      $entries += $Directory.Trim().TrimEnd('\', '/')
    }
  } else {
    $entries = @($entries | Where-Object { -not (& $matchesTarget $_) })
  }
  return $entries -join ';'
}

function Assert-Equal {
  param([string]$Actual, [string]$Expected, [string]$Message)
  if ($Actual -cne $Expected) {
    throw "$($Message): got '$Actual', want '$Expected'"
  }
}

if ($SelfTest) {
  Assert-Equal (Update-PathValue "C:\Windows;C:\Tools" add "c:\tools\") "C:\Windows;C:\Tools" "add must be idempotent"
  Assert-Equal (Update-PathValue "C:\Windows" add "C:\AT Starter") "C:\Windows;C:\AT Starter" "add must append the directory"
  Assert-Equal (Update-PathValue "C:\AT Starter;C:\AT Starter\bin;C:\Windows" remove "c:\at starter\") "C:\AT Starter\bin;C:\Windows" "remove must match only the exact directory"
  Write-Host "update-path self-test passed"
  exit 0
}

if ([string]::IsNullOrWhiteSpace($Action) -or [string]::IsNullOrWhiteSpace($Directory)) {
  throw "Action and Directory are required"
}

$current = [Environment]::GetEnvironmentVariable("Path", "Machine")
$updated = Update-PathValue $current $Action $Directory
if ($updated -cne $current) {
  [Environment]::SetEnvironmentVariable("Path", $updated, "Machine")
}
