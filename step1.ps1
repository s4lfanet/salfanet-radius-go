$files = Get-ChildItem -Path "C:\Users\yanz\Downloads\salfanet-radius-go\src" -Recurse -Include "*.tsx","*.ts" | Where-Object { $_.FullName -notlike "*node_modules*" }
$pattern = "t\('([a-zA-Z][a-zA-Z0-9]*(?:\.[a-zA-Z][a-zA-Z0-9]*)+)'\)"
$keys = [System.Collections.Generic.List[string]]::new()
foreach ($file in $files) {
    $content = Get-Content $file.FullName -Raw -ErrorAction SilentlyContinue
    if ($content) {
        $m2 = [regex]::Matches($content, $pattern)
        foreach ($m in $m2) { $keys.Add($m.Groups[1].Value) }
    }
}
$uniqueKeys = $keys | Sort-Object -Unique
$uniqueKeys | Out-File -FilePath "C:\Users\yanz\Downloads\salfanet-radius-go\used_keys.txt" -Encoding utf8
Write-Host "Total unique keys used: $($uniqueKeys.Count)"
