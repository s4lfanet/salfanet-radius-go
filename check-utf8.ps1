$bytes = [System.IO.File]::ReadAllBytes('e:\salfanet-go\salfanet-radius-go\src\app\admin\whatsapp\templates\page.tsx')
$found = $false
for ($i=0; $i -lt $bytes.Length; $i++) {
    $b = $bytes[$i]
    if ($b -gt 127 -and ($b -lt 0xC2 -or $b -gt 0xF4)) {
        Write-Output "Bad byte at offset ${i}: 0x$($b.ToString('X2'))"
        $start = [Math]::Max(0, $i - 30)
        $end = [Math]::Min($bytes.Length - 1, $i + 30)
        $context = $bytes[$start..$end]
        Write-Output "Context: $([System.Text.Encoding]::ASCII.GetString($context))"
        $found = $true
        break
    }
}
if (-not $found) { Write-Output 'All bytes valid UTF-8' }
