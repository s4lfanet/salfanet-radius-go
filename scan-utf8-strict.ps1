$root = 'e:\salfanet-go\salfanet-radius-go\src'
$files = Get-ChildItem -Path $root -Recurse -Include '*.tsx','*.ts'
$utf8 = New-Object System.Text.UTF8Encoding($false, $true)  # strict validation
$badFiles = @()
foreach ($file in $files) {
    try {
        $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
        $utf8.GetString($bytes)  # throws on invalid UTF-8
    } catch {
        $badFiles += $file.FullName
    }
}
if ($badFiles.Count -eq 0) {
    Write-Output "All TSX/TS files are valid UTF-8"
} else {
    Write-Output "Files with invalid UTF-8 ($($badFiles.Count)):"
    $badFiles | ForEach-Object { Write-Output $_ }
}
