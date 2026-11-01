<#
.SYNOPSIS
    Stops the core platform.

.DESCRIPTION
    Without -Drop it keeps the PostgreSQL volume, so the next `run.ps1` comes up
    with the same data (accounts, tokens). With -Drop the database is destroyed.

.PARAMETER Drop
    Also remove the database volume. All data is lost.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\stop.ps1
    powershell -ExecutionPolicy Bypass -File .\scripts\stop.ps1 -Drop
#>
param(
    [switch]$Drop
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

$previous = $ErrorActionPreference
$ErrorActionPreference = "Continue"
try {
    if ($Drop) {
        Write-Host "stopping and dropping the database volume" -ForegroundColor Yellow
        docker compose down -v
    } else {
        Write-Host "stopping (data is kept)" -ForegroundColor Cyan
        docker compose down
    }
    $exit = $LASTEXITCODE
} finally {
    $ErrorActionPreference = $previous
}

if ($exit -ne 0) {
    Write-Host "docker compose down failed (exit $exit)" -ForegroundColor Red
    exit $exit
}

Write-Host "stopped" -ForegroundColor Green
