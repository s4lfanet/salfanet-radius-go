$json = Get-Content "C:\Users\yanz\Downloads\salfanet-radius-go\src\locales\id.json" -Raw | ConvertFrom-Json

function Get-FlatKeys {
    param($obj, $prefix = "")
    foreach ($prop in $obj.PSObject.Properties) {
        $key = if ($prefix) { "$prefix.$($prop.Name)" } else { $prop.Name }
        if ($prop.Value -is [System.Management.Automation.PSCustomObject]) {
            Get-FlatKeys -obj $prop.Value -prefix $key
        } else {
            $key
        }
    }
}

$jsonKeys = Get-FlatKeys -obj $json | Sort-Object
$jsonKeys | Out-File -FilePath "C:\Users\yanz\Downloads\salfanet-radius-go\json_keys.txt" -Encoding utf8
Write-Host "Total keys in id.json: $($jsonKeys.Count)"
