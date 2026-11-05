# Serve the web UI from a separate origin.
#
# The UI is plain static files, so any static server works. This one needs only
# Python (already a build dependency of the project's tooling) and starts
# instantly, which keeps the loop short: the API runs in Docker on :18080, the
# UI runs here on :3000, and the browser is the only missing piece.
#
# The API's `security.allowed_origins` must contain this origin or the browser
# will block every request. That is deliberate: CORS is an allow-list, not a
# default-open policy.

param(
    [int]$Port = 3000,
    [string]$Root = (Join-Path $PSScriptRoot '..\web')
)

$ErrorActionPreference = 'Stop'
$Root = (Resolve-Path $Root).Path

Write-Host "Serving $Root"
Write-Host "Open http://localhost:$Port"
Write-Host "API expected at http://localhost:18080 (change it in the UI under 'Baglanti ayari')."

python -m http.server $Port --bind 127.0.0.1 --directory $Root
