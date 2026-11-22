# RailShack · 50 Microservices Project by DevOps Shack
# Copyright (c) 2026 DevOps Shack (Aditya Jaiswal) · https://devopsshack.com · https://www.youtube.com/@devopsshack
# SPDX-License-Identifier: Apache-2.0 · Keep this notice and the NOTICE file in any copy or derivative work.
#
# End-to-end smoke test (PowerShell): health of all 50 services, sign up / sign in, then
# search -> book -> PNR -> My Trips -> cancel, checking that booking, My Trips and cancelling
# refuse requests without a login. Works in Windows PowerShell 5.1 and PowerShell 7.
#
# Usage:  .\scripts\smoke-test.ps1 https://railshack.prod-bat.site
param([Parameter(Mandatory = $true)][string]$BaseUrl)

$ErrorActionPreference = "Stop"
$BaseUrl = $BaseUrl.TrimEnd("/")
Add-Type -AssemblyName System.Net.Http
$http = New-Object System.Net.Http.HttpClient
$http.Timeout = [TimeSpan]::FromSeconds(30)

function Pass([string]$msg) { Write-Host "  PASS  $msg" -ForegroundColor Green }
function Fail([string]$msg) { Write-Host "  FAIL  $msg" -ForegroundColor Red; exit 1 }

# One HTTP call. Returns the status code, the body (parsed when it is JSON) and the response headers.
function Call([string]$Method, [string]$Path, $Body = $null, [string]$Token = "") {
    $req = New-Object System.Net.Http.HttpRequestMessage ([System.Net.Http.HttpMethod]::new($Method)), ($BaseUrl + $Path)
    if ($Token) { $req.Headers.Add("Authorization", "Bearer $Token") }
    if ($null -ne $Body) {
        $json = $Body | ConvertTo-Json -Depth 10 -Compress
        $req.Content = New-Object System.Net.Http.StringContent ($json, [Text.Encoding]::UTF8, "application/json")
    }
    try { $resp = $http.SendAsync($req).GetAwaiter().GetResult() }
    catch { return [pscustomobject]@{ Code = 0; Data = $null; Text = ""; Headers = $null } }
    $text = $resp.Content.ReadAsStringAsync().GetAwaiter().GetResult()
    $data = $null
    try { $data = $text | ConvertFrom-Json } catch { }
    [pscustomobject]@{ Code = [int]$resp.StatusCode; Data = $data; Text = $text; Headers = $resp.Headers }
}

$date = (Get-Date).AddDays(1).ToString("yyyy-MM-dd")
Write-Host "RailShack smoke test -> $BaseUrl"

$r = Call GET "/api/platform/status"
if ($r.Code -ne 200) { Fail "platform status unreachable" }
if ($r.Data.up -eq 50) { Pass "all 50 microservices healthy" } else { Fail "only $($r.Data.up)/50 services healthy" }

$user = "smoke" + [DateTimeOffset]::UtcNow.ToUnixTimeSeconds() + (Get-Random -Maximum 99999)
$r = Call POST "/api/auth/signup" @{ user = $user; password = "smoke-pass-123"; name = "Smoke Test" }
if ($r.Code -lt 200 -or $r.Code -ge 300 -or -not $r.Data.token) { Fail "sign up returned $($r.Code)" }
$token = $r.Data.token
Pass "sign up created $user and issued a token"
$r = Call POST "/api/auth/login" @{ user = $user; password = "smoke-pass-123" }
if ($r.Code -ne 200) { Fail "sign in returned $($r.Code)" }
$r = Call POST "/api/auth/login" @{ user = $user; password = "wrong-password" }
if ($r.Code -eq 401) { Pass "sign in works, wrong password is refused" } else { Fail "wrong password returned $($r.Code)" }

$r = Call GET "/api/search/trains?from=NDLS&to=BSB&date=$date"
if ($r.Code -eq 200 -and $r.Data.count -ge 1) { Pass "search NDLS -> BSB returned $($r.Data.count) trains" } else { Fail "search returned no trains" }

$booking = @{
    train = "22436"; from = "NDLS"; to = "BSB"; date = $date; class = "CC"; quota = "GN"
    passengers = @(@{ name = "Asha"; age = 31; gender = "F" }, @{ name = "Ravi"; age = 64; gender = "M" })
    payment = "wallet"; coupon = "BLACKFRIDAY"; insurance = $true
}
$r = Call POST "/api/booking/book" $booking
if ($r.Code -eq 401) { Pass "booking without signing in is refused" } else { Fail "booking without login returned $($r.Code)" }
$r = Call POST "/api/booking/book" $booking $token
if ($r.Code -lt 200 -or $r.Code -ge 300) { Fail "booking returned $($r.Code): $($r.Text)" }
$pnr = $r.Data.pnr
$hops = 0
$trace = $null
if ($r.Headers.TryGetValues("X-Trace", [ref]$trace)) { $hops = (($trace -join ",") -split ",").Count }
Pass "booked PNR $pnr (status $($r.Data.status), total Rs $($r.Data.total), touched $hops services)"

$r = Call GET "/api/pnr/$pnr"
if ($r.Data.state -eq "ACTIVE") { Pass "PNR lookup works" } else { Fail "unexpected PNR state '$($r.Data.state)'" }
$r = Call GET "/api/pnr/$pnr"
switch ($r.Data.servedFrom) {
    "redis"  { Pass "PNR saved in PostgreSQL, second read served from the Redis cache" }
    "memory" { Pass "PNR stored in memory (no DATABASE_URL: local mode)" }
    default  { Fail "second PNR read came from '$($r.Data.servedFrom)', expected the Redis cache" }
}

$r = Call GET "/api/booking-history/$user"
if ($r.Code -eq 401) { Pass "My Trips without signing in is refused" } else { Fail "My Trips without login returned $($r.Code)" }
$r = Call GET "/api/booking-history/devopsshack" $null $token
if ($r.Code -eq 403) { Pass "someone else's trips are refused" } else { Fail "other user's trips returned $($r.Code)" }
$trips = 0
foreach ($i in 1..5) {   # booking-history is written in the background
    $r = Call GET "/api/booking-history/$user" $null $token
    $trips = @($r.Data.trips).Count
    if ($trips -ge 1) { break }
    Start-Sleep -Seconds 1
}
if ($trips -ge 1) { Pass "trip saved in My Trips" } else { Fail "trip not saved" }

$r = Call POST "/api/cancellation/cancel" @{ pnr = $pnr }
if ($r.Code -eq 401) { Pass "cancelling without signing in is refused" } else { Fail "cancel without login returned $($r.Code)" }
$r = Call POST "/api/cancellation/cancel" @{ pnr = $pnr } $token
if ($r.Code -ne 200) { Fail "cancellation returned $($r.Code)" }
Pass "cancelled with refund Rs $($r.Data.refund.amount)"

if ((Call GET "/api/live-status/12301").Code -eq 200) { Pass "live train status" } else { Fail "live status" }
if ((Call GET "/api/admin/dashboard").Code -eq 200) { Pass "ops dashboard" } else { Fail "admin dashboard" }
$r = Call GET "/"
if ($r.Code -eq 200 -and $r.Text -match "RailShack") { Pass "web UI served" } else { Fail "web UI" }
Write-Host "Smoke test passed" -ForegroundColor Green
