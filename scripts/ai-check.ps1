<#
.SYNOPSIS
    Verifies the configured AI provider with one real request.

.DESCRIPTION
    Loads AI_* from .env, then runs cmd/aicheck, which uses the same adapter the
    application uses. This checks the endpoint, the key and the model together
    without starting the server.

    The key is never printed. Only its length is shown, so a wrong or truncated
    key is obvious without exposing it in a terminal or a log.

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .\scripts\ai-check.ps1
#>
param(
    [string]$EnvFile = ".env"
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

if (-not (Test-Path $EnvFile)) {
    Write-Host "$EnvFile not found. Copy .env.example and fill in the AI_* values." -ForegroundColor Red
    exit 1
}

function ReadEnv([string]$name) {
    $line = Get-Content $EnvFile | Where-Object { $_ -like "$name=*" } | Select-Object -First 1
    if (-not $line) { return "" }
    return $line.Split('=', 2)[1].Trim()
}

$base = ReadEnv "AI_BASE_URL"
$key = ReadEnv "AI_API_KEY"
$model = ReadEnv "AI_DEFAULT_MODEL"

if (-not $base -or -not $key) {
    Write-Host "AI_BASE_URL and AI_API_KEY must both be set in $EnvFile" -ForegroundColor Red
    exit 1
}

Write-Host "core-platform AI check" -ForegroundColor Cyan
Write-Host "  base  : $base"
Write-Host "  key   : $($key.Substring(0, [Math]::Min(12, $key.Length)))... ($($key.Length) chars)"
Write-Host "  model : $model"
Write-Host ""

$env:AI_BASE_URL = $base
$env:AI_API_KEY = $key
$env:AI_DEFAULT_MODEL = $model

$previous = $ErrorActionPreference
$ErrorActionPreference = "Continue"
try {
    go run ./cmd/aicheck
    $exit = $LASTEXITCODE
} finally {
    $ErrorActionPreference = $previous
    # Do not leave the secret in the process environment of this session.
    Remove-Item Env:\AI_API_KEY -ErrorAction SilentlyContinue
}

exit $exit
