param([string]$BaseUrl = 'http://localhost:8080')
$ErrorActionPreference = 'Stop'

# Synthetic labels are declared before querying; this is a regression benchmark,
# not an estimate of hiring quality on an independent population.
$datasetPath = Join-Path $PSScriptRoot '../docs/matching-dataset.json'
$dataset = Get-Content -LiteralPath $datasetPath -Raw -Encoding utf8 | ConvertFrom-Json
$cases = $dataset.cases
$session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
$headers = @{ 'X-Requested-With'='fsp'; Origin=$BaseUrl.TrimEnd('/') }
$body = @{ email='employer@demo.fsp'; password='DemoPass2026!' } | ConvertTo-Json
$user = Invoke-RestMethod "$BaseUrl/api/auth/login" -Method Post -WebSession $session -Headers $headers -ContentType 'application/json' -Body $body
$headers['X-CSRF-Token'] = $user.csrf
try {
    $results = foreach ($case in $cases) {
        $query = [Uri]::EscapeDataString($case.need)
        $response = Invoke-RestMethod "$BaseUrl/api/candidates?need=$query" -WebSession $session
        $top = @($response.candidates | Select-Object -First $case.k)
        $hits = @($top | Where-Object { $_.name -in $case.relevant }).Count
        [ordered]@{
            need=$case.need; k=$case.k; expected_relevant=$case.relevant
            top=@($top | ForEach-Object { @{name=$_.name; score=$_.match_score} })
            total=$response.total; precision_at_k=$hits / [double]$case.k
        }
    }
    $report = [ordered]@{
        dataset="$($dataset.candidates.Count) synthetic candidates, $($cases.Count) manually labelled descriptions"
        limitations='Same demo fixtures used for development. No independent expert or user validation. Labels measure regression on this small synthetic set only.'
        mean_precision_at_k=($results.precision_at_k | Measure-Object -Average).Average
        results=@($results)
    }
    $report | ConvertTo-Json -Depth 8
} finally {
    Invoke-RestMethod "$BaseUrl/api/auth/logout" -Method Post -WebSession $session -Headers $headers -ContentType 'application/json' -Body '{}' | Out-Null
}
