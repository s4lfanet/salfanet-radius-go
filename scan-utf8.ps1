$root = 'e:\salfanet-go\salfanet-radius-go\src'
$files = Get-ChildItem -Path $root -Recurse -Include '*.tsx','*.ts'
$badFiles = @()
foreach ($file in $files) {
    try {
        $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
        for ($i = 0; $i -lt $bytes.Length; $i++) {
            $b = $bytes[$i]
            if ($b -gt 127 -and ($b -lt 0xC2 -or $b -gt 0xF4)) {
                $badFiles += "$($file.FullName) offset=$i byte=0x$($b.ToString('X2'))"
                break
            }
        }
    } catch {
        # Skip files that can't be read
    }
}
if ($badFiles.Count -eq 0) {
    Write-Output "All TSX/TS files are valid UTF-8"
} else {
    Write-Output "Files with invalid UTF-8:"
    $badFiles | ForEach-Object { Write-Output $_ }
}
