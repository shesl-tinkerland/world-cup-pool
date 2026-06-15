$root = 'c:\FotballVM\frontend\src'
$files = Get-ChildItem -Path $root -Recurse -Include *.svelte,*.ts | Where-Object { $_.FullName -notmatch '\\node_modules\\|\\\.svelte-kit\\' }
$utf8 = New-Object System.Text.UTF8Encoding($false)
$pattern = '\\u([0-9a-fA-F]{4})'
$total = 0
foreach ($f in $files) {
  $orig = [System.IO.File]::ReadAllText($f.FullName, $utf8)
  $m = [regex]::Matches($orig, $pattern)
  if ($m.Count -eq 0) { continue }
  $new = [regex]::Replace($orig, $pattern, { param($x) [string][char][Convert]::ToInt32($x.Groups[1].Value, 16) })
  [System.IO.File]::WriteAllText($f.FullName, $new, $utf8)
  Write-Host "fixed $($m.Count): $($f.FullName)"
  $total += $m.Count
}
Write-Host "Total replacements: $total"
