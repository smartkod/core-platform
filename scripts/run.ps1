<#
.SYNOPSIS
    Starts the core platform, picking free host ports automatically.

.DESCRIPTION
    Removes three recurring annoyances:
      * remembering APP_PORT / DB_PORT_HOST every session
      * a port already taken by another project
      * the containers being gone after a Docker Desktop restart

    Idempotent: running it twice is harmless. It never destroys data; use
    -Fresh for that.

.PARAMETER AppPort
    Host port for the API. Defaults to 18080, or the first free port after it.

.PARAMETER DbPort
    Host port for PostgreSQL. Defaults to 15532, or the first free port after it.

.PARAMETER Fresh
    Drops the database volume before starting. Destroys all data.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\run.ps1
    powershell -ExecutionPolicy Bypass -File .\scripts\run.ps1 -Fresh
#>
param(
    [int]$AppPort = 18080,
    [int]$DbPort = 15532,
    [switch]$Fresh
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

# Native tools (docker, go) write progress and warnings to stderr. With
# ErrorActionPreference=Stop, PowerShell promotes that stderr to a terminating
# error and kills the script mid-build. Native exit codes are checked explicitly
# instead, which is what actually signals failure.
function Invoke-Native {
    param([scriptblock]$Command)

    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        & $Command 2>&1 | Out-Null
        return $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previous
    }
}

# Port detection on Windows cannot rely on binding a TcpListener: Docker
# Desktop publishes on 0.0.0.0 through a proxy, so a loopback bind still
# succeeds and the port looks free when it is not. Get-NetTCPConnection sees the
# real listeners, including that proxy.
function Test-PortInUse([int]$Port) {
    if (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue) {
        $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
        return [bool]$conn
    }
    # Fallback for hosts without the NetTCPIP module.
    try {
        $client = New-Object System.Net.Sockets.TcpClient
        $client.Connect("127.0.0.1", $Port)
        $client.Close()
        return $true
    } catch {
        return $false
    }
}

# Ports published by THIS compose project are not a conflict: restarting on the
# same port is the normal case and must stay idempotent.
function Test-OwnPort([int]$Port) {
    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $names = docker compose ps --format "{{.Ports}}" 2>$null
    } finally {
        $ErrorActionPreference = $previous
    }
    foreach ($line in $names) {
        if ($line -match ":$Port->") { return $true }
    }
    return $false
}

function Find-Free([int]$Start) {
    for ($p = $Start; $p -lt $Start + 50; $p++) {
        if (-not (Test-PortInUse $p)) { return $p }
    }
    throw "no free port found starting at $Start"
}

Write-Host "core-platform" -ForegroundColor Cyan

if ($Fresh) {
    Write-Host "  dropping the database volume (-Fresh)" -ForegroundColor Yellow
    Invoke-Native { docker compose down -v } | Out-Null
}

# Ports that are already published by a running compose project are fine to
# reuse; only unrelated listeners matter.
$appPort = if (-not (Test-PortInUse $AppPort) -or (Test-OwnPort $AppPort)) { $AppPort } else { Find-Free ($AppPort + 1) }
$dbPort = if (-not (Test-PortInUse $DbPort) -or (Test-OwnPort $DbPort)) { $DbPort } else { Find-Free ($DbPort + 1) }

if ($appPort -ne $AppPort) {
    Write-Host "  port $AppPort is busy, using $appPort for the API" -ForegroundColor Yellow
}
if ($dbPort -ne $DbPort) {
    Write-Host "  port $DbPort is busy, using $dbPort for PostgreSQL" -ForegroundColor Yellow
}

$env:APP_PORT = "$appPort"
$env:DB_PORT_HOST = "$dbPort"

Write-Host "  starting containers..."
$exit = Invoke-Native { docker compose up -d --build }
if ($exit -ne 0) {
    Write-Host "  docker compose failed (exit $exit)" -ForegroundColor Red
    Write-Host "  run 'docker compose up -d --build' to see the full output" -ForegroundColor Yellow
    exit 1
}

# Wait for readiness rather than sleeping a fixed time.
Write-Host "  waiting for readiness..."
$url = "http://localhost:$appPort/readyz"
$ready = $false
for ($i = 0; $i -lt 120; $i++) {
    try {
        $r = Invoke-WebRequest $url -UseBasicParsing -TimeoutSec 2
        if ($r.StatusCode -eq 200) { $ready = $true; break }
    } catch {
        # still starting, or retrying after the Docker DNS hiccup
    }
    Start-Sleep -Milliseconds 500
}

if (-not $ready) {
    Write-Host "  not ready after 60s" -ForegroundColor Red
    Write-Host ""
    Write-Host "  app logs:" -ForegroundColor Yellow
    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    docker compose logs app --tail 20
    $ErrorActionPreference = $previous
    exit 1
}

Write-Host ""
Write-Host "  running" -ForegroundColor Green
Write-Host "    API          http://localhost:$appPort"
Write-Host "    health       http://localhost:$appPort/healthz"
Write-Host "    PostgreSQL   localhost:$dbPort"
Write-Host ""
Write-Host "  smoke test:"
Write-Host "    powershell -ExecutionPolicy Bypass -File .\scripts\smoke.ps1 -BaseUrl http://localhost:$appPort"
Write-Host ""
Write-Host "  stop:  docker compose down        (keeps data)"
Write-Host "         docker compose down -v     (drops data)"
