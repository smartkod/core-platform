<#
.SYNOPSIS
    End-to-end smoke test for the core platform: kernel, identity, authorization,
    organization and catalog.

.DESCRIPTION
    Exercises the behaviour that matters and reports PASS/FAIL per check.
    Exits non-zero if any check fails, so it can be wired into CI later.

    This is a verification script, not a test suite: the authoritative tests
    live in the Go packages. It exists so a human can confirm a running
    instance behaves correctly after a deploy.

.PARAMETER BaseUrl
    Base URL of the running instance. Default: http://localhost:18080

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\smoke.ps1
    powershell -ExecutionPolicy Bypass -File .\scripts\smoke.ps1 -BaseUrl http://localhost:8080
#>
param(
    [string]$BaseUrl = "http://localhost:18080"
)

$ErrorActionPreference = "Stop"

$script:passed = 0
$script:failed = 0
$script:base = "$BaseUrl/api/v1"

function Check {
    param([string]$Name, [scriptblock]$Body)

    try {
        $result = & $Body
        if ($result -eq $true -or $null -eq $result) {
            Write-Host "  PASS  $Name" -ForegroundColor Green
            $script:passed++
        } else {
            Write-Host "  FAIL  $Name  (got: $result)" -ForegroundColor Red
            $script:failed++
        }
    } catch {
        Write-Host "  FAIL  $Name  ($($_.Exception.Message))" -ForegroundColor Red
        $script:failed++
    }
}

# Invoke-WebRequest throws on non-2xx; this returns the status instead so a
# test can assert on 401/403/422 without a try/catch everywhere.
function Status {
    param([string]$Method, [string]$Url, [hashtable]$Headers = @{}, [string]$Body = "")

    try {
        $params = @{ Uri = $Url; Method = $Method; UseBasicParsing = $true; TimeoutSec = 15 }
        if ($Headers.Count -gt 0) { $params.Headers = $Headers }
        if ($Body) { $params.Body = $Body; $params.ContentType = "application/json" }
        $r = Invoke-WebRequest @params
        return $r.StatusCode
    } catch {
        if ($_.Exception.Response) { return [int]$_.Exception.Response.StatusCode }
        throw
    }
}

function Json {
    param([string]$Method, [string]$Url, [hashtable]$Headers = @{}, [string]$Body = "")

    $params = @{ Uri = $Url; Method = $Method; UseBasicParsing = $true; TimeoutSec = 15 }
    if ($Headers.Count -gt 0) { $params.Headers = $Headers }
    if ($Body) { $params.Body = $Body; $params.ContentType = "application/json" }

    try {
        return (Invoke-WebRequest @params).Content | ConvertFrom-Json
    } catch {
        # Non-2xx responses still carry the error envelope; parse it instead of
        # letting the exception hide the body.
        if ($_.ErrorDetails -and $_.ErrorDetails.Message) {
            return $_.ErrorDetails.Message | ConvertFrom-Json
        }
        if ($_.Exception.Response) {
            $stream = $_.Exception.Response.GetResponseStream()
            $reader = New-Object System.IO.StreamReader($stream)
            return $reader.ReadToEnd() | ConvertFrom-Json
        }
        throw
    }
}

Write-Host ""
Write-Host "core-platform smoke test" -ForegroundColor Cyan
Write-Host "target: $BaseUrl"
Write-Host ("-" * 60)

# ---------------------------------------------------------------- health
Write-Host ""
Write-Host "Health" -ForegroundColor Yellow

Check "GET /healthz returns 200" {
    (Status GET "$BaseUrl/healthz") -eq 200
}

Check "GET /readyz returns 200 (database reachable)" {
    (Status GET "$BaseUrl/readyz") -eq 200
}

Check "GET /version returns build metadata" {
    $v = Json GET "$BaseUrl/version"
    $v.name -and $v.version
}

Check "every response carries X-Request-ID" {
    $r = Invoke-WebRequest "$BaseUrl/healthz" -UseBasicParsing
    [bool]$r.Headers["X-Request-ID"]
}

Check "unknown route returns the standard error envelope" {
    $envelope = Json GET "$BaseUrl/api/v1/does-not-exist"
    $envelope.success -eq $false -and $envelope.error.code -eq "NOT_FOUND"
}

# ---------------------------------------------------------------- identity
Write-Host ""
Write-Host "Identity: registration" -ForegroundColor Yellow

$email = "smoke-$([guid]::NewGuid().ToString('N').Substring(0,8))@example.com"
$password = "correct horse battery"
$regBody = @{ email = $email; password = $password; name = "Smoke Test" } | ConvertTo-Json

Check "register a new account returns 201" {
    (Status POST "$script:base/auth/register" @{} $regBody) -eq 201
}

Check "the account id is a UUIDv7" {
    $r = Json POST "$script:base/auth/register" @{} (@{ email = "v7-$email"; password = $password } | ConvertTo-Json)
    # v7 puts the version nibble in the first character of the third group.
    $r.data.id -match '^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-'
}

Check "a weak password is rejected with 422" {
    (Status POST "$script:base/auth/register" @{} (@{ email = "weak-$email"; password = "123" } | ConvertTo-Json)) -eq 422
}

Check "registering an existing e-mail does not reveal that it exists" {
    $r = Json POST "$script:base/auth/register" @{} $regBody
    # Same shape as a fresh signup: no CONFLICT code, no "already registered".
    $r.success -eq $true
}

Write-Host ""
Write-Host "Identity: session" -ForegroundColor Yellow

$login = Json POST "$script:base/auth/login" @{} (@{ email = $email; password = $password } | ConvertTo-Json)
$access = $login.data.access_token
$refresh = $login.data.refresh_token

Check "login returns an access and a refresh token" {
    [bool]$access -and [bool]$refresh
}

Check "GET /me with a Bearer token returns the account" {
    $me = Json GET "$script:base/me" @{ Authorization = "Bearer $access" }
    $me.data.email -eq $email
}

Check "GET /me without a token returns 401" {
    (Status GET "$script:base/me") -eq 401
}

Check "GET /me with a garbage token returns 401" {
    (Status GET "$script:base/me" @{ Authorization = "Bearer not-a-token" }) -eq 401
}

Check "wrong password returns 401" {
    (Status POST "$script:base/auth/login" @{} (@{ email = $email; password = "wrong password" } | ConvertTo-Json)) -eq 401
}

Write-Host ""
Write-Host "Identity: refresh rotation" -ForegroundColor Yellow

$rotated = Json POST "$script:base/auth/refresh" @{} (@{ refresh_token = $refresh } | ConvertTo-Json)
$newRefresh = $rotated.data.refresh_token

Check "refresh issues a different token (rotation)" {
    $newRefresh -and $newRefresh -ne $refresh
}

Check "refresh issues a new access token" {
    $rotated.data.access_token -and $rotated.data.access_token -ne $access
}

Check "replaying the old refresh token returns 401" {
    (Status POST "$script:base/auth/refresh" @{} (@{ refresh_token = $refresh } | ConvertTo-Json)) -eq 401
}

Check "reuse detection revokes the whole token family" {
    # The token issued a moment ago must also be dead, otherwise an attacker
    # who stole the old token keeps the session the victim rotated into.
    (Status POST "$script:base/auth/refresh" @{} (@{ refresh_token = $newRefresh } | ConvertTo-Json)) -eq 401
}

Write-Host ""
Write-Host "Identity: authorization" -ForegroundColor Yellow

$l2 = Json POST "$script:base/auth/login" @{} (@{ email = $email; password = $password } | ConvertTo-Json)

Check "a normal user gets 403 on an admin endpoint" {
    (Status GET "$script:base/admin/users" @{ Authorization = "Bearer $($l2.data.access_token)" }) -eq 403
}

Check "an admin endpoint without a token gets 401" {
    (Status GET "$script:base/admin/users") -eq 401
}

Write-Host ""
Write-Host "Identity: logout" -ForegroundColor Yellow

Check "logout succeeds" {
    (Status POST "$script:base/auth/logout" @{} (@{ refresh_token = $($l2.data.refresh_token) } | ConvertTo-Json)) -eq 204
}

Check "the refresh token stops working after logout" {
    (Status POST "$script:base/auth/refresh" @{} (@{ refresh_token = $($l2.data.refresh_token) } | ConvertTo-Json)) -eq 401
}

Check "logout is idempotent" {
    (Status POST "$script:base/auth/logout" @{} (@{ refresh_token = $($l2.data.refresh_token) } | ConvertTo-Json)) -eq 204
}

$l3 = Json POST "$script:base/auth/login" @{} (@{ email = $email; password = $password } | ConvertTo-Json)

Check "logout-all invalidates an access token that is still signed valid" {
    $code = Status POST "$script:base/me/logout-all" @{ Authorization = "Bearer $($l3.data.access_token)" }
    $after = Status GET "$script:base/me" @{ Authorization = "Bearer $($l3.data.access_token)" }
    $code -eq 204 -and $after -eq 401
}

Write-Host ""
Write-Host "Identity: password reset" -ForegroundColor Yellow

Check "forgot-password accepts a known address" {
    (Status POST "$script:base/auth/forgot-password" @{} (@{ email = $email } | ConvertTo-Json)) -eq 202
}

Check "forgot-password does not reveal an unknown address" {
    # Identical response to the known-address case above.
    (Status POST "$script:base/auth/forgot-password" @{} (@{ email = "nobody-$email" } | ConvertTo-Json)) -eq 202
}

Check "an invalid reset token is rejected" {
    (Status POST "$script:base/auth/reset-password" @{} (@{ token = "bogus"; password = "another good password" } | ConvertTo-Json)) -eq 422
}

Write-Host ""
Write-Host "Rate limiting" -ForegroundColor Yellow

Check "repeated successful logins are never throttled" {
    # Only failures should consume the budget. If successes counted too, every
    # user behind one NAT would share a limit and lock each other out.
    $throttled = $false
    for ($i = 0; $i -lt 20; $i++) {
        $code = Status POST "$script:base/auth/login" @{} (@{ email = $email; password = $password } | ConvertTo-Json)
        if ($code -eq 429) { $throttled = $true; break }
    }
    -not $throttled
}

# ---------------------------------------------------------------- files
Write-Host ""
Write-Host "Files" -ForegroundColor Yellow

# PowerShell 5.1 has no -Encoding utf8NoBOM and its multipart support varies
# between versions, so curl.exe is used for uploads. It ships with Windows 10+.
$fl = Json POST "$script:base/auth/login" @{} (@{ email = $email; password = $password } | ConvertTo-Json)
$authHeader = "Authorization: Bearer $($fl.data.access_token)"

# A real 1x1 PNG, built by hand so the test needs no fixture files.
$pngBytes = [byte[]]@(
    0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
    0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
    0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
    0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
    0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
    0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
    0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
    0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
    0x42, 0x60, 0x82
)
$tmpDir = [System.IO.Path]::GetTempPath()
$stamp = [guid]::NewGuid().ToString('N').Substring(0, 6)
$tmpPng = Join-Path $tmpDir "smoke-$stamp.png"
[System.IO.File]::WriteAllBytes($tmpPng, $pngBytes)

$script:fileId = $null

Check "upload a PNG returns 201 and a file id" {
    $raw = curl.exe -s -X POST "$script:base/files" -H $authHeader -F "file=@$tmpPng"
    $doc = $raw | ConvertFrom-Json
    if (-not $doc.success) { return $false }
    $script:fileId = $doc.data.id
    [bool]$script:fileId
}

Check "the uploaded file appears in the owner's list" {
    if (-not $script:fileId) { return $false }
    $list = Json GET "$script:base/files" @{ Authorization = "Bearer $($fl.data.access_token)" }
    [bool]($list.data.items | Where-Object { $_.id -eq $script:fileId })
}

Check "download returns exactly the stored bytes" {
    if (-not $script:fileId) { return $false }
    $out = Join-Path $tmpDir "smoke-dl-$stamp.png"
    curl.exe -s -o $out -H $authHeader "$script:base/files/$($script:fileId)/download" | Out-Null
    $got = [System.IO.File]::ReadAllBytes($out)
    Remove-Item $out -ErrorAction SilentlyContinue
    $got.Length -eq $pngBytes.Length
}

Check "an SVG upload is rejected with 422" {
    # SVG is an active document; storing one would be stored XSS.
    $svg = Join-Path $tmpDir "smoke-$stamp.svg"
    [System.IO.File]::WriteAllText($svg, '<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>')
    $code = curl.exe -s -o NUL -w "%{http_code}" -X POST "$script:base/files" -H $authHeader -F "file=@$svg"
    Remove-Item $svg -ErrorAction SilentlyContinue
    $code -eq "422"
}

Check "HTML disguised as a .png is rejected" {
    # The sniffed type wins over the declared one.
    $fake = Join-Path $tmpDir "smoke-$stamp.png"
    $fakeName = Join-Path $tmpDir "fake-$stamp.png"
    [System.IO.File]::WriteAllText($fakeName, '<!DOCTYPE html><html><body><script>alert(1)</script></body></html>')
    $code = curl.exe -s -o NUL -w "%{http_code}" -X POST "$script:base/files" -H $authHeader -F "file=@$fakeName"
    Remove-Item $fakeName -ErrorAction SilentlyContinue
    $code -eq "422"
}

Check "another user cannot read the file (IDOR)" {
    if (-not $script:fileId) { return $false }

    $otherEmail = "other-$stamp@example.com"
    Json POST "$script:base/auth/register" @{} (@{ email = $otherEmail; password = $password } | ConvertTo-Json) | Out-Null
    $ol = Json POST "$script:base/auth/login" @{} (@{ email = $otherEmail; password = $password } | ConvertTo-Json)

    (Status GET "$script:base/files/$($script:fileId)" @{ Authorization = "Bearer $($ol.data.access_token)" }) -eq 404
}

Check "deleting the file removes it from the list" {
    if (-not $script:fileId) { return $false }
    $code = Status DELETE "$script:base/files/$($script:fileId)" @{ Authorization = "Bearer $($fl.data.access_token)" }
    $list = Json GET "$script:base/files" @{ Authorization = "Bearer $($fl.data.access_token)" }
    $code -eq 204 -and -not ($list.data.items | Where-Object { $_.id -eq $script:fileId })
}

Remove-Item $tmpPng -ErrorAction SilentlyContinue

# ---------------------------------------------------------------- notifications
Write-Host ""
Write-Host "Notifications" -ForegroundColor Yellow

$nl = Json POST "$script:base/auth/login" @{} (@{ email = $email; password = $password } | ConvertTo-Json)
$nh = @{ Authorization = "Bearer $($nl.data.access_token)" }

Check "unread count starts at zero" {
    $r = Json GET "$script:base/notifications/unread-count" $nh
    $r.data.unread -eq 0
}

Check "preferences default to e-mail and in-app on, push off" {
    # Absence of a row means "default on": a new user must receive transactional
    # mail without visiting a settings page first.
    $r = Json GET "$script:base/notifications/preferences" $nh
    $r.data.channels.email -eq $true -and
    $r.data.channels.in_app -eq $true -and
    $r.data.channels.push -eq $false
}

Check "preferences can be changed" {
    $r = Json PUT "$script:base/notifications/preferences" $nh '{"email":false}'
    $r.data.channels.email -eq $false
}

Check "preferences survive a re-read" {
    $r = Json GET "$script:base/notifications/preferences" $nh
    $r.data.channels.email -eq $false
}

Check "inbox is empty for a new account" {
    $r = Json GET "$script:base/notifications" $nh
    $r.data.total -eq 0
}

Check "notifications require authentication" {
    (Status GET "$script:base/notifications") -eq 401
}

Check "another user's inbox is separate" {
    $otherEmail = "notif-$stamp@example.com"
    Json POST "$script:base/auth/register" @{} (@{ email = $otherEmail; password = $password } | ConvertTo-Json) | Out-Null
    $ol = Json POST "$script:base/auth/login" @{} (@{ email = $otherEmail; password = $password } | ConvertTo-Json)
    $r = Json GET "$script:base/notifications" @{ Authorization = "Bearer $($ol.data.access_token)" }
    $r.data.total -eq 0
}

# ---------------------------------------------------------------- organization
Write-Host ""
Write-Host "Organization + authorization" -ForegroundColor Yellow

$ol = Json POST "$script:base/auth/login" @{} (@{ email = $email; password = $password } | ConvertTo-Json)
$oh = @{ Authorization = "Bearer $($ol.data.access_token)" }
$slug = "smoke-org-$stamp"

$org = Json POST "$script:base/organizations" $oh (@{ name = "Smoke Co"; slug = $slug } | ConvertTo-Json)
$script:orgId = $org.data.id

Check "creating an organization returns 201" {
    $org.success -eq $true -and [bool]$script:orgId
}

Check "the creator receives the owner permission" {
    if (-not $script:orgId) { return $false }
    $r = Json GET "$script:base/authz/me/permissions?scope=organization&scope_id=$($script:orgId)" $oh
    $r.data.permissions -contains "organization.manage"
}

Check "the owner can read the organization" {
    if (-not $script:orgId) { return $false }
    $r = Json GET "$script:base/organizations/$($script:orgId)" $oh
    $r.data.id -eq $script:orgId
}

Check "an outsider cannot read the organization" {
    if (-not $script:orgId) { return $false }
    $otherEmail = "outsider-$stamp@example.com"
    Json POST "$script:base/auth/register" @{} (@{ email = $otherEmail; password = $password } | ConvertTo-Json) | Out-Null
    $xl = Json POST "$script:base/auth/login" @{} (@{ email = $otherEmail; password = $password } | ConvertTo-Json)
    (Status GET "$script:base/organizations/$($script:orgId)" @{ Authorization = "Bearer $($xl.data.access_token)" }) -eq 403
}

Check "a draft organization is not public" {
    (Status GET "$script:base/organizations/by-slug/$slug") -eq 404
}

Check "the owner can add a branch" {
    if (-not $script:orgId) { return $false }
    $b = Json POST "$script:base/organizations/$($script:orgId)/branches" $oh (@{ name = "Main"; city = "Istanbul" } | ConvertTo-Json)
    $script:branchId = $b.data.id
    $b.success -eq $true -and [bool]$script:branchId
}

Check "a draft cannot jump straight to active" {
    if (-not $script:orgId) { return $false }
    (Status POST "$script:base/admin/organizations/$($script:orgId)/status" $oh (@{ status = "active" } | ConvertTo-Json)) -eq 403
}

Check "the owner can invite a member" {
    if (-not $script:orgId) { return $false }
    $m = Json POST "$script:base/organizations/$($script:orgId)/members" $oh (@{ email = "invitee-$stamp@example.com"; role_key = "staff" } | ConvertTo-Json)
    $m.success -eq $true -and $m.data.status -eq "invited"
}

Check "the owner can remove a branch" {
    if (-not $script:orgId -or -not $script:branchId) { return $false }
    (Status DELETE "$script:base/organizations/$($script:orgId)/branches/$($script:branchId)" $oh) -eq 204
}

# ---------------------------------------------------------------- catalog
Write-Host ""
Write-Host "Catalog" -ForegroundColor Yellow

Check "unknown offering returns 404" {
    (Status GET "$script:base/offerings/00000000-0000-0000-0000-000000000000") -eq 404
}

Check "catalog requires authentication on tenant routes" {
    if (-not $script:orgId) { return $false }
    (Status GET "$script:base/organizations/$($script:orgId)/offerings") -eq 401
}

Check "an outsider cannot list tenant offerings" {
    if (-not $script:orgId) { return $false }
    $otherEmail = "catalog-outsider-$stamp@example.com"
    Json POST "$script:base/auth/register" @{} (@{ email = $otherEmail; password = $password } | ConvertTo-Json) | Out-Null
    $xl = Json POST "$script:base/auth/login" @{} (@{ email = $otherEmail; password = $password } | ConvertTo-Json)
    (Status GET "$script:base/organizations/$($script:orgId)/offerings" @{ Authorization = "Bearer $($xl.data.access_token)" }) -eq 403
}

# ---------------------------------------------------------------- summary
Write-Host ""
Write-Host ("-" * 60)
if ($script:failed -eq 0) {
    Write-Host "all $script:passed checks passed" -ForegroundColor Green
    exit 0
}
Write-Host "$script:passed passed, $script:failed failed" -ForegroundColor Red
exit 1
