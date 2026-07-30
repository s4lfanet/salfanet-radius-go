$root = 'e:\salfanet-go\salfanet-radius-go\src'
$files = Get-ChildItem -Path $root -Recurse -Include '*.tsx','*.ts'

# Windows-1252 bytes (0x80-0x9F range) that are invalid UTF-8 → their UTF-8 equivalents
$replacements = @{
    [byte]0x80 = [byte[]](0xE2, 0x82, 0xAC)  # €
    [byte]0x82 = [byte[]](0xE2, 0x80, 0x9A)  # ‚
    [byte]0x83 = [byte[]](0xC6, 0x92)        # ƒ
    [byte]0x84 = [byte[]](0xE2, 0x80, 0x9E)  # „
    [byte]0x85 = [byte[]](0xE2, 0x80, 0xA6)  # …
    [byte]0x86 = [byte[]](0xE2, 0x80, 0xA0)  # †
    [byte]0x87 = [byte[]](0xE2, 0x80, 0xA1)  # ‡
    [byte]0x88 = [byte[]](0xCB, 0x86)        # ˆ
    [byte]0x89 = [byte[]](0xE2, 0x80, 0xB0)  # ‰
    [byte]0x8A = [byte[]](0xC5, 0xA0)        # Š
    [byte]0x8B = [byte[]](0xE2, 0x80, 0xB9)  # ‹
    [byte]0x8C = [byte[]](0xC5, 0x92)        # Œ
    [byte]0x8E = [byte[]](0xC5, 0xBD)        # Ž
    [byte]0x91 = [byte[]](0xE2, 0x80, 0x98)  # '
    [byte]0x92 = [byte[]](0xE2, 0x80, 0x99)  # '
    [byte]0x93 = [byte[]](0xE2, 0x80, 0x9C)  # "
    [byte]0x94 = [byte[]](0xE2, 0x80, 0x9D)  # "
    [byte]0x95 = [byte[]](0xE2, 0x80, 0xA2)  # •
    [byte]0x96 = [byte[]](0xE2, 0x80, 0x93)  # –
    [byte]0x97 = [byte[]](0xE2, 0x80, 0x94)  # —
    [byte]0x98 = [byte[]](0xCB, 0x9C)        # ˜
    [byte]0x99 = [byte[]](0xE2, 0x84, 0xA2)  # ™
    [byte]0x9A = [byte[]](0xC5, 0xA1)        # š
    [byte]0x9C = [byte[]](0xC5, 0x93)        # œ
    [byte]0x9E = [byte[]](0xC5, 0xBE)        # ž
    [byte]0x9F = [byte[]](0xC5, 0xB8)        # Ÿ
    [byte]0xA1 = [byte[]](0xC2, 0xA1)        # ¡
    [byte]0xA2 = [byte[]](0xC2, 0xA2)        # ¢
    [byte]0xA3 = [byte[]](0xC2, 0xA3)        # £
    [byte]0xA5 = [byte[]](0xC2, 0xA5)        # ¥
    [byte]0xA6 = [byte[]](0xC2, 0xA6)        # ¦
    [byte]0xA7 = [byte[]](0xC2, 0xA7)        # §
    [byte]0xA8 = [byte[]](0xC2, 0xA8)        # ¨
    [byte]0xA9 = [byte[]](0xC2, 0xA9)        # ©
    [byte]0xAA = [byte[]](0xC2, 0xAA)        # ª
    [byte]0xAB = [byte[]](0xC2, 0xAB)        # «
    [byte]0xAC = [byte[]](0xC2, 0xAC)        # ¬
    [byte]0xAD = [byte[]](0xC2, 0xAD)        # ­
    [byte]0xAE = [byte[]](0xC2, 0xAE)        # ®
    [byte]0xAF = [byte[]](0xC2, 0xAF)        # ¯
    [byte]0xB0 = [byte[]](0xC2, 0xB0)        # °
    [byte]0xB1 = [byte[]](0xC2, 0xB1)        # ±
    [byte]0xB2 = [byte[]](0xC2, 0xB2)        # ²
    [byte]0xB3 = [byte[]](0xC2, 0xB3)        # ³
    [byte]0xB4 = [byte[]](0xC2, 0xB4)        # ´
    [byte]0xB5 = [byte[]](0xC2, 0xB5)        # µ
    [byte]0xB6 = [byte[]](0xC2, 0xB6)        # ¶
    [byte]0xB7 = [byte[]](0xC2, 0xB7)        # ·
    [byte]0xB8 = [byte[]](0xC2, 0xB8)        # ¸
    [byte]0xB9 = [byte[]](0xC2, 0xB9)        # ¹
    [byte]0xBA = [byte[]](0xC2, 0xBA)        # º
    [byte]0xBB = [byte[]](0xC2, 0xBB)        # »
    [byte]0xBC = [byte[]](0xC2, 0xBC)        # ¼
    [byte]0xBD = [byte[]](0xC2, 0xBD)        # ½
    [byte]0xBE = [byte[]](0xC2, 0xBE)        # ¾
    [byte]0xBF = [byte[]](0xC2, 0xBF)        # ¿
}

$totalFiles = 0
$totalReplacements = 0

foreach ($file in $files) {
    try {
        $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
        $hasBad = $false
        for ($i = 0; $i -lt $bytes.Length; $i++) {
            $b = $bytes[$i]
            # Check for bytes > 127 that aren't valid UTF-8 lead/continuation bytes
            if ($b -gt 127 -and $b -lt 0xC2) {
                $hasBad = $true
                break
            }
        }
        if (-not $hasBad) { continue }

        $result = New-Object System.Collections.Generic.List[byte]
        $fileCount = 0
        for ($i = 0; $i -lt $bytes.Length; $i++) {
            $b = $bytes[$i]
            if ($replacements.ContainsKey($b)) {
                foreach ($rb in $replacements[$b]) { $result.Add($rb) }
                $fileCount++
            } else {
                $result.Add($b)
            }
        }
        if ($fileCount -gt 0) {
            [System.IO.File]::WriteAllBytes($file.FullName, $result.ToArray())
            $totalFiles++
            $totalReplacements += $fileCount
            Write-Output "$($file.Name): $fileCount replacements"
        }
    } catch {
        Write-Output "SKIP: $($file.FullName) - $($_.Exception.Message)"
    }
}

Write-Output ""
Write-Output "Total files fixed: $totalFiles"
Write-Output "Total replacements: $totalReplacements"
