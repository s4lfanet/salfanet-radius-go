cd "C:\Users\yanz\Downloads\salfanet-radius-go"

# Extract keys from code
$files = Get-ChildItem -Path "src" -Recurse -Include "*.tsx","*.ts" | Where-Object { $_.FullName -notlike "*node_modules*" }
$pattern = "t\('([a-zA-Z][a-zA-Z0-9]*(?:\.[a-zA-Z][a-zA-Z0-9]*)+)'\)"
$keys = [System.Collections.Generic.List[string]]::new()
foreach ($file in $files) {
    $content = [System.IO.File]::ReadAllText($file.FullName)
    $m2 = [regex]::Matches($content, $pattern)
    foreach ($m in $m2) { $keys.Add($m.Groups[1].Value) }
}
$usedKeys = $keys | Sort-Object -Unique
$usedKeys | Set-Content "used_keys.txt" -Encoding UTF8
Write-Host "CODE KEYS: $($usedKeys.Count)"

# Extract keys from id.json
$json = [System.IO.File]::ReadAllText("src\locales\id.json") | ConvertFrom-Json
function Get-FlatKeys { 
    param($obj, $prefix = "")
    $result = [System.Collections.Generic.List[string]]::new()
    foreach ($prop in $obj.PSObject.Properties) {
        $key = if ($prefix) { "$prefix.$($prop.Name)" } else { $prop.Name }
        if ($prop.Value -is [System.Management.Automation.PSCustomObject]) { 
            foreach ($k in (Get-FlatKeys -obj $prop.Value -prefix $key)) { $result.Add($k) }
        } else { $result.Add($key) }
    }
    return $result
}
$jsonKeys = Get-FlatKeys -obj $json | Sort-Object
$jsonKeys | Set-Content "json_keys.txt" -Encoding UTF8
Write-Host "JSON KEYS: $($jsonKeys.Count)"

# Find missing
$missing = $usedKeys | Where-Object { $_ -notin $jsonKeys }
Write-Host "MISSING: $($missing.Count)"
$missing | ForEach-Object { Write-Host "  MISSING: $_" }

# Find orphaned
$orphaned = $jsonKeys | Where-Object { $_ -notin $usedKeys }
Write-Host "ORPHANED: $($orphaned.Count)"
