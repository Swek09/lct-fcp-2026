param(
    [string]$BaseUrl = 'http://localhost:8080',
    [ValidateRange(1,50)][int]$Concurrency = 20,
    [ValidateRange(1,2000)][int]$Requests = 200
)
$ErrorActionPreference = 'Stop'
# Bounded local read-only smoke load. This does not estimate production capacity.
$session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
$headers = @{ Origin=$BaseUrl.TrimEnd('/'); 'X-Requested-With'='fsp' }
$body = @{ email='employer@demo.fsp'; password='DemoPass2026!' } | ConvertTo-Json
$user = Invoke-RestMethod "$BaseUrl/api/auth/login" -Method Post -WebSession $session -Headers $headers -ContentType 'application/json' -Body $body
$headers['X-CSRF-Token'] = $user.csrf
try {
    if (-not ('FspSmokeLoad' -as [type])) {
        Add-Type -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
public static class FspSmokeLoad {
    public static async Task<double[][]> Run(string url, string cookie, int count, int concurrency) {
        using (var client = new HttpClient()) {
            client.Timeout = TimeSpan.FromSeconds(15);
            client.DefaultRequestHeaders.Add("Cookie", cookie);
            using (var gate = new SemaphoreSlim(concurrency)) {
                var tasks = new List<Task<double[]>>();
                for (int i = 0; i < count; i++) tasks.Add(One(client, gate, url));
                return await Task.WhenAll(tasks);
            }
        }
    }
    private static async Task<double[]> One(HttpClient client, SemaphoreSlim gate, string url) {
        await gate.WaitAsync();
        var timer = Stopwatch.StartNew();
        try {
            using (var response = await client.GetAsync(url)) {
                await response.Content.ReadAsByteArrayAsync();
                return new double[] { timer.Elapsed.TotalMilliseconds, (int)response.StatusCode };
            }
        } catch { return new double[] { timer.Elapsed.TotalMilliseconds, 0 }; }
        finally { gate.Release(); }
    }
}
'@
    }
    $cookie = $session.Cookies.GetCookieHeader([uri]$BaseUrl)
    $query = [Uri]::EscapeDataString('middle backend Go PostgreSQL Redis')
    $watch = [Diagnostics.Stopwatch]::StartNew()
    $rows = [FspSmokeLoad]::Run("$BaseUrl/api/candidates?need=$query", $cookie, $Requests, $Concurrency).GetAwaiter().GetResult()
    $watch.Stop()
    $times = @($rows | ForEach-Object { $_[0] } | Sort-Object)
    $success = @($rows | Where-Object { $_[1] -eq 200 }).Count
    [ordered]@{
        measured_at_utc=[DateTime]::UtcNow.ToString('o')
        scope='Local Docker Compose, one authenticated demo employer, search, small synthetic database'
        requests=$Requests; concurrency=$Concurrency; successful=$success; failed=$Requests-$success
        elapsed_seconds=[Math]::Round($watch.Elapsed.TotalSeconds,3)
        requests_per_second=[Math]::Round($Requests/$watch.Elapsed.TotalSeconds,2)
        latency_ms=@{
            p50=[Math]::Round($times[[Math]::Ceiling($Requests*0.50)-1],2)
            p95=[Math]::Round($times[[Math]::Ceiling($Requests*0.95)-1],2)
            p99=[Math]::Round($times[[Math]::Ceiling($Requests*0.99)-1],2)
            max=[Math]::Round($times[-1],2)
        }
        limitations='Short read-only load with a shared session and small dataset. Does not measure independent user sessions, write contention, mail delivery, large databases or production capacity.'
    } | ConvertTo-Json -Depth 5
    if ($success -ne $Requests) { throw "Only $success/$Requests requests returned HTTP 200" }
} finally {
    Invoke-RestMethod "$BaseUrl/api/auth/logout" -Method Post -WebSession $session -Headers $headers -ContentType 'application/json' -Body '{}' | Out-Null
}
