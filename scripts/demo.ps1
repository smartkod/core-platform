<#
.SYNOPSIS
    Creates a demo account with realistic data so the UI can be shown.

.DESCRIPTION
    Each run makes a fresh account with a random e-mail suffix, then fills one
    organization with branches, staff and offerings, so every screen has
    something to look at.

    Transport note: every JSON body is written to a UTF-8 file and sent with
    curl --data-binary. Invoke-WebRequest's handling of a byte[] body on Windows
    PowerShell mangles non-ASCII characters — "Stüdyo" arrived at the API as
    CP437 box-drawing characters — and a demo whose data looks broken makes the
    UI look broken. curl sends the file's bytes unchanged, which is what the
    browser does too.

    Use this only against a development instance.

.PARAMETER BaseUrl
    API base URL. Default: http://127.0.0.1:18080

.PARAMETER Origin
    Origin the UI is served from. Sent as Origin/X-Client-Type so the API uses
    cookie transport and CORS matches the real browser. Default: http://localhost:3100
#>
param(
    [string]$BaseUrl = "http://127.0.0.1:18080",
    [string]$Origin = "http://localhost:3100"
)

$ErrorActionPreference = "Stop"

$api = "$BaseUrl/api/v1"
$stamp = Get-Random -Minimum 100000 -Maximum 999999
$email = "demo$stamp@example.com"
$password = "demo-parola-1234"

$work = Join-Path ([System.IO.Path]::GetTempPath()) ("core-demo-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $work | Out-Null
$jar = Join-Path $work "cookies.txt"

$utf8 = New-Object System.Text.UTF8Encoding($false)

Write-Host "demo verisi hazırlanıyor..." -ForegroundColor Cyan

function Write-Json {
    param([string]$Path, $Value)
    [System.IO.File]::WriteAllText($Path, ($Value | ConvertTo-Json -Depth 8 -Compress), $script:utf8)
}

# Invoke-Api runs curl and returns the parsed envelope. A non-2xx response is
# still parsed, so the caller can see the API's error instead of a stack trace.
function Invoke-Api {
    param(
        [string]$Method,
        [string]$Path,
        $Body = $null,
        [switch]$Anonymous
    )

    $args = @("-s", "-X", $Method, "$script:api$Path",
        "-H", "Origin: $Origin", "-H", "X-Client-Type: web",
        "-H", "Accept: application/json",
        "-b", $script:jar, "-c", $script:jar)

    if (-not $Anonymous) {
        if ($script:token) { $args += @("-H", "Authorization: Bearer $script:token") }
        if ($script:csrf) { $args += @("-H", "X-CSRF-Token: $script:csrf") }
    }

    if ($null -ne $Body) {
        $file = Join-Path $script:work ("body-" + [guid]::NewGuid().ToString("N") + ".json")
        [System.IO.File]::WriteAllText($file, ($Body | ConvertTo-Json -Depth 8 -Compress), $script:utf8)
        $args += @("-H", "Content-Type: application/json; charset=utf-8", "--data-binary", "@$file")
    }

    $raw = & curl.exe @args
    if ($LASTEXITCODE -ne 0) { throw "curl failed for $Method $Path" }
    if ([string]::IsNullOrWhiteSpace($raw)) { return $null }
    return $raw | ConvertFrom-Json
}

function Read-Csrf {
    # curl's cookie jar: domain, flag, path, secure, expiry, name, value
    if (-not (Test-Path $script:jar)) { return "" }
    foreach ($line in Get-Content $script:jar) {
        if ($line.StartsWith("#") -or [string]::IsNullOrWhiteSpace($line)) { continue }
        $parts = $line -split "\t"
        if ($parts.Count -ge 7 -and $parts[5] -eq "csrf") { return $parts[6] }
    }
    return ""
}

try {
    Invoke-Api -Method POST -Path "/auth/register" -Anonymous -Body @{
        email = $email; password = $password; name = "Demo Kullanıcı"
    } | Out-Null

    $login = Invoke-Api -Method POST -Path "/auth/login" -Anonymous -Body @{
        email = $email; password = $password
    }
    $script:token = $login.data.access_token
    $script:csrf = Read-Csrf

    $org = Invoke-Api -Method POST -Path "/organizations" -Body @{
        name = "Stüdyo Nova"; slug = "studyo-nova-$stamp"
    }
    $orgId = $org.data.id

    foreach ($b in @(
        @{ name = "Merkez Stüdyo"; type = "physical"; city = "İstanbul"; district = "Kadıköy"; address_line = "Moda Cad. 14"; public = $true },
        @{ name = "Beşiktaş Şube"; type = "physical"; city = "İstanbul"; district = "Beşiktaş"; public = $true },
        @{ name = "Online Seans"; type = "online"; city = "İstanbul"; public = $false })) {
        Invoke-Api -Method POST -Path "/organizations/$orgId/branches" -Body $b | Out-Null
    }

    $taxonomy = Invoke-Api -Method GET -Path "/taxonomy"
    $taxId = $taxonomy.data[0].id

    $first = $null
    foreach ($o in @(
        @{ name = "60 dakikalık seans"; slug = "60-dk-seans"; summary = "Birebir çalışma, tüm seviyeler"; service_mode = "on_site" },
        @{ name = "4 seanslık paket"; slug = "4-seans-paketi"; summary = "Aylık düzenli çalışma programı"; service_mode = "on_site" },
        @{ name = "Online değerlendirme"; slug = "online-degerlendirme"; summary = "30 dakika, görüntülü"; service_mode = "online" })) {
        $o["taxonomy_id"] = $taxId
        $created = Invoke-Api -Method POST -Path "/organizations/$orgId/offerings" -Body $o
        if (-not $first) { $first = $created.data.id }
    }

    Invoke-Api -Method POST -Path "/organizations/$orgId/offerings/$first/status" -Body @{
        status = "published"
    } | Out-Null
    Invoke-Api -Method POST -Path "/organizations/$orgId/offerings/$first/variants" -Body @{ name = "Yüz yüze" } | Out-Null
    Invoke-Api -Method POST -Path "/organizations/$orgId/offerings/$first/variants" -Body @{ name = "Görüntülü" } | Out-Null

    $i = 0
    foreach ($role in @("manager", "staff", "member")) {
        $i++
        Invoke-Api -Method POST -Path "/organizations/$orgId/members" -Body @{
            email = "uye$i$stamp@example.com"; role_key = $role
        } | Out-Null
    }

    Invoke-Api -Method PATCH -Path "/organizations/$orgId" -Body @{
        description = "Kadıköy merkezli çalışma stüdyosu. Birebir ve grup seansları."
        email       = "merhaba@studyonova.example"
        phone       = "0216 000 00 00"
        website     = "https://studyonova.example"
        locale      = "tr"
    } | Out-Null

    Write-Host ""
    Write-Host "hazır" -ForegroundColor Green
    Write-Host "  e-posta : $email"
    Write-Host "  parola  : $password"
    Write-Host "  işletme : Stüdyo Nova ($orgId)"
    Write-Host "  arayüz  : $Origin"
    Write-Host ""
}
finally {
    if (Test-Path $work) { Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue }
}
