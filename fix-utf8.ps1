$filePath = 'e:\salfanet-go\salfanet-radius-go\src\app\admin\whatsapp\templates\page.tsx'
$bytes = [System.IO.File]::ReadAllBytes($filePath)

# Replace 0x95 (Windows-1252 bullet •) with UTF-8 bullet (0xE2 0x80 0xA2)
# Also replace other common Windows-1252 chars if found
$replacements = @{
    0x95 = [byte[]](0xE2, 0x80, 0xA2)  # bullet •
    0x91 = [byte[]](0xE2, 0x80, 0x98)  # left single quote '
    0x92 = [byte[]](0xE2, 0x80, 0x99)  # right single quote '
    0x93 = [byte[]](0xE2, 0x80, 0x9C)  # left double quote "
    0x94 = [byte[]](0xE2, 0x80, 0x9D)  # right double quote "
    0x96 = [byte[]](0xE2, 0x80, 0x93)  # en dash –
    0x97 = [byte[]](0xE2, 0x80, 0x94)  # em dash —
    0x85 = [byte[]](0xE2, 0x80, 0xA6)  # ellipsis …
}

$result = New-Object System.Collections.Generic.List[byte]
$count = 0
for ($i = 0; $i -lt $bytes.Length; $i++) {
    $b = $bytes[$i]
    if ($replacements.ContainsKey([int]$b)) {
        $replacement = $replacements[[int]$b]
        foreach ($rb in $replacement) {
            $result.Add($rb)
        }
        $count++
        Write-Output "Replaced 0x$($b.ToString('X2')) at offset $i with UTF-8 equivalent"
    }
    else {
        $result.Add($b)
    }
}

[System.IO.File]::WriteAllBytes($filePath, $result.ToArray())
Write-Output "Total replacements: $count"
Write-Output "File size: $($result.Count) bytes (was $($bytes.Length) bytes)"
