$root = 'e:\salfanet-go\salfanet-radius-go\src'
$files = Get-ChildItem -Path $root -Recurse -Include '*.tsx','*.ts'

# Windows-1252 byte → UTF-8 byte sequence (only for bytes 0x80-0x9F that aren't valid UTF-8)
$cp1252 = @{
    0x80 = [byte[]](0xE2,0x82,0xAC)  # €
    0x81 = $null                       # undefined
    0x82 = [byte[]](0xE2,0x80,0x9A)  # ‚
    0x83 = [byte[]](0xC6,0x92)       # ƒ
    0x84 = [byte[]](0xE2,0x80,0x9E)  # „
    0x85 = [byte[]](0xE2,0x80,0xA6)  # …
    0x86 = [byte[]](0xE2,0x80,0xA0)  # †
    0x87 = [byte[]](0xE2,0x80,0xA1)  # ‡
    0x88 = [byte[]](0xCB,0x86)       # ˆ
    0x89 = [byte[]](0xE2,0x80,0xB0)  # ‰
    0x8A = [byte[]](0xC5,0xA0)       # Š
    0x8B = [byte[]](0xE2,0x80,0xB9)  # ‹
    0x8C = [byte[]](0xC5,0x92)       # Œ
    0x8D = $null                       # undefined
    0x8E = [byte[]](0xC5,0xBD)       # Ž
    0x8F = $null                       # undefined
    0x90 = $null                       # undefined
    0x91 = [byte[]](0xE2,0x80,0x98)  # '
    0x92 = [byte[]](0xE2,0x80,0x99)  # '
    0x93 = [byte[]](0xE2,0x80,0x9C)  # "
    0x94 = [byte[]](0xE2,0x80,0x9D)  # "
    0x95 = [byte[]](0xE2,0x80,0xA2)  # •
    0x96 = [byte[]](0xE2,0x80,0x93)  # –
    0x97 = [byte[]](0xE2,0x80,0x94)  # —
    0x98 = [byte[]](0xCB,0x9C)       # ˜
    0x99 = [byte[]](0xE2,0x84,0xA2)  # ™
    0x9A = [byte[]](0xC5,0xA1)       # š
    0x9B = [byte[]](0xE2,0x80,0xBA)  # ›
    0x9C = [byte[]](0xC5,0x93)       # œ
    0x9D = $null                       # undefined
    0x9E = [byte[]](0xC5,0xBE)       # ž
    0x9F = [byte[]](0xC5,0xB8)       # Ÿ
    0xA0 = [byte[]](0xC2,0xA0)       # nbsp
    0xA1 = [byte[]](0xC2,0xA1)       # ¡
    0xA2 = [byte[]](0xC2,0xA2)       # ¢
    0xA3 = [byte[]](0xC2,0xA3)       # £
    0xA4 = [byte[]](0xC2,0xA4)       # ¤
    0xA5 = [byte[]](0xC2,0xA5)       # ¥
    0xA6 = [byte[]](0xC2,0xA6)       # ¦
    0xA7 = [byte[]](0xC2,0xA7)       # §
    0xA8 = [byte[]](0xC2,0xA8)       # ¨
    0xA9 = [byte[]](0xC2,0xA9)       # ©
    0xAA = [byte[]](0xC2,0xAA)       # ª
    0xAB = [byte[]](0xC2,0xAB)       # «
    0xAC = [byte[]](0xC2,0xAC)       # ¬
    0xAD = [byte[]](0xC2,0xAD)       # ­
    0xAE = [byte[]](0xC2,0xAE)       # ®
    0xAF = [byte[]](0xC2,0xAF)       # ¯
    0xB0 = [byte[]](0xC2,0xB0)       # °
    0xB1 = [byte[]](0xC2,0xB1)       # ±
    0xB2 = [byte[]](0xC2,0xB2)       # ²
    0xB3 = [byte[]](0xC2,0xB3)       # ³
    0xB4 = [byte[]](0xC2,0xB4)       # ´
    0xB5 = [byte[]](0xC2,0xB5)       # µ
    0xB6 = [byte[]](0xC2,0xB6)       # ¶
    0xB7 = [byte[]](0xC2,0xB7)       # ·
    0xB8 = [byte[]](0xC2,0xB8)       # ¸
    0xB9 = [byte[]](0xC2,0xB9)       # ¹
    0xBA = [byte[]](0xC2,0xBA)       # º
    0xBB = [byte[]](0xC2,0xBB)       # »
    0xBC = [byte[]](0xC2,0xBC)       # ¼
    0xBD = [byte[]](0xC2,0xBD)       # ½
    0xBE = [byte[]](0xC2,0xBE)       # ¾
    0xBF = [byte[]](0xC2,0xBF)       # ¿
}

$totalFiles = 0
$totalReplacements = 0

foreach ($file in $files) {
    try {
        $bytes = [System.IO.File]::ReadAllBytes($file.FullName)
    } catch { continue }

    $result = New-Object System.Collections.Generic.List[byte]
    $i = 0
    $fileReplacements = 0
    $len = $bytes.Length

    while ($i -lt $len) {
        $b = $bytes[$i]

        if ($b -lt 0x80) {
            # ASCII — pass through
            $result.Add($b)
            $i++
            continue
        }

        # Try to parse as valid UTF-8 sequence
        $seqLen = 0
        $valid = $false

        if ($b -ge 0xC2 -and $b -le 0xDF) {
            $seqLen = 2
        } elseif ($b -ge 0xE0 -and $b -le 0xEF) {
            $seqLen = 3
        } elseif ($b -ge 0xF0 -and $b -le 0xF4) {
            $seqLen = 4
        }

        if ($seqLen -gt 0 -and ($i + $seqLen) -le $len) {
            $valid = $true
            for ($j = 1; $j -lt $seqLen; $j++) {
                if ($bytes[$i + $j] -lt 0x80 -or $bytes[$i + $j] -gt 0xBF) {
                    $valid = $false
                    break
                }
            }
            # Extra checks for overlong sequences
            if ($valid -and $b -eq 0xE0 -and $bytes[$i+1] -lt 0xA0) { $valid = $false }
            if ($valid -and $b -eq 0xED -and $bytes[$i+1] -gt 0x9F) { $valid = $false }
            if ($valid -and $b -eq 0xF0 -and $bytes[$i+1] -lt 0x90) { $valid = $false }
            if ($valid -and $b -eq 0xF4 -and $bytes[$i+1] -gt 0x8F) { $valid = $false }
        }

        if ($valid) {
            # Valid UTF-8 sequence — copy as-is
            for ($j = 0; $j -lt $seqLen; $j++) {
                $result.Add($bytes[$i + $j])
            }
            $i += $seqLen
        } else {
            # Invalid byte — check if it's a Windows-1252 char
            if ($cp1252.ContainsKey([int]$b) -and $cp1252[[int]$b] -ne $null) {
                $replacement = $cp1252[[int]$b]
                foreach ($rb in $replacement) { $result.Add($rb) }
                $fileReplacements++
            } else {
                # Unknown byte — replace with '?' (0x3F)
                $result.Add(0x3F)
                $fileReplacements++
            }
            $i++
        }
    }

    if ($fileReplacements -gt 0) {
        [System.IO.File]::WriteAllBytes($file.FullName, $result.ToArray())
        $totalFiles++
        $totalReplacements += $fileReplacements
        Write-Output "$($file.Name): $fileReplacements replacements"
    }
}

Write-Output ""
Write-Output "Total files fixed: $totalFiles"
Write-Output "Total replacements: $totalReplacements"
