# Runs the security audit suite against a running instance.
#
# The suite itself is ordinary Go code (test/security). This script exists only
# so a human does not have to remember the environment variable and the flags,
# and so a CI job can call one thing.
#
# It is a verification tool, not a test suite replacement: the tests live in
# test/security and carry their own assertions.

param(
    [string]$BaseUrl = "http://127.0.0.1:18080",
    [switch]$Verbose
)

$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $PSScriptRoot)

Write-Host "security audit against $BaseUrl"
Write-Host ("-" * 60)

try {
    $health = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 "$BaseUrl/healthz"
    if ($health.StatusCode -ne 200) {
        throw "healthz returned $($health.StatusCode)"
    }
} catch {
    Write-Host "the instance is not reachable: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "start it with: docker compose up -d"
    exit 1
}

$env:TEST_BASE_URL = $BaseUrl

$goArgs = @('test', './test/security/...', '-count=1')
if ($Verbose) { $goArgs += '-v' }

go @goArgs
exit $LASTEXITCODE
