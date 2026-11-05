<#
.SYNOPSIS
    Starts the API console (cmd/uiconsole) against a running core platform.

.DESCRIPTION
    The platform is API-only; this console is a development tool. It serves one
    HTML page and proxies /api/* to the API, so no CORS configuration is needed
    and Bearer tokens work out of the box.

    Start the API first (.\scripts\run.ps1 or docker compose up -d --build),
    then run this. Use -Stop to shut the console down again.

.PARAMETER ApiUrl
    Base URL of the running API. Defaults to http://127.0.0.1:18080.

.PARAMETER Port
    Port the console listens on. Defaults to 19090.

.PARAMETER Stop
    Stops a console previously started by this script.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\ui.ps1
    powershell -ExecutionPolicy Bypass -File .\scripts\ui.ps1 -ApiUrl http://127.0.0.1:8082 -Port 19091
    powershell -ExecutionPolicy Bypass -File .\scripts\ui.ps1 -Stop
#>
param(
    [string]$ApiUrl = "http://127.0.0.1:18080",
    [int]$Port = 19090,
    [switch]$Stop,
    [switch]$NoBrowser
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$exe = Join-Path $root "bin\uiconsole.exe"
$pidFile = Join-Path $root "bin\uiconsole.pid"
$logFile = Join-Path $root "bin\uiconsole.log"

function Test-PortInUse([int]$Port) {
    if (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue) {
        $conn = Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue
        return [bool]$conn
    }
    try {
        $client = New-Object System.Net.Sockets.TcpClient
        $client.Connect("127.0.0.1", $Port)
        $client.Close()
        return $true
    } catch {
        return $false
    }
}

function Stop-Console {
    if (Test-Path $pidFile) {
        # Get-Content is avoided on purpose: this host's profile sets
        # $PSDefaultParameterValues['*:Encoding'], which PS 5.1 rejects.
        $existing = if (Test-Path $pidFile) { [System.IO.File]::ReadAllText($pidFile).Trim() } else { "" }
        if ($existing) {
            Stop-Process -Id ([int]$existing) -Force -ErrorAction SilentlyContinue
            Write-Host "  console stopped (pid $existing)" -ForegroundColor Yellow
        }
        Remove-Item $pidFile -ErrorAction SilentlyContinue
    }
    # Fallback when the pid file is gone but the process is still around.
    Get-Process uiconsole -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}

Write-Host "core-platform api console" -ForegroundColor Cyan

if ($Stop) {
    Stop-Console
    exit 0
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "  go not found in PATH; the console is built from source" -ForegroundColor Red
    exit 1
}

Write-Host "  building..."
$previous = $ErrorActionPreference
$ErrorActionPreference = "Continue"
go build -o $exe ./cmd/uiconsole 2>&1 | Out-Null
$exit = $LASTEXITCODE
$ErrorActionPreference = $previous
if ($exit -ne 0) {
    Write-Host "  build failed (exit $exit)" -ForegroundColor Red
    exit 1
}

# A console left over from a previous run would hold the port.
Stop-Console
if (Test-PortInUse $Port) {
    Write-Host "  port $Port is busy; pass -Port <n>" -ForegroundColor Red
    exit 1
}

Write-Host "  starting console on port $Port (API: $ApiUrl)..."
$proc = Start-Process -FilePath $exe `
    -ArgumentList @("-addr", "127.0.0.1:$Port", "-api", $ApiUrl) `
    -WindowStyle Hidden -PassThru `
    -RedirectStandardOutput $logFile -RedirectStandardError "$logFile.err"
Set-Content -Path $pidFile -Value $proc.Id -Encoding ascii

$ready = $false
for ($i = 0; $i -lt 40; $i++) {
    if (Test-PortInUse $Port) { $ready = $true; break }
    Start-Sleep -Milliseconds 250
}

if (-not $ready) {
    Write-Host "  console did not start; see bin\uiconsole.log.err" -ForegroundColor Red
    Stop-Console
    exit 1
}

$url = "http://127.0.0.1:$Port/"
Write-Host ""
Write-Host "  running" -ForegroundColor Green
Write-Host "    console      $url"
Write-Host "    proxying to  $ApiUrl"
Write-Host ""
if (-not (Test-PortInUse ([int]([uri]$ApiUrl).Port))) {
    Write-Host "  warning: nothing listens on $ApiUrl - start the API first" -ForegroundColor Yellow
    Write-Host ""
}
Write-Host "  stop:  powershell -ExecutionPolicy Bypass -File .\scripts\ui.ps1 -Stop"
Write-Host ""

if (-not $NoBrowser) {
    Start-Process $url
}
