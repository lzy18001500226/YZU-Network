$ErrorActionPreference = "Stop"

Set-Location $PSScriptRoot
New-Item -ItemType Directory -Force -Path ".\dist" | Out-Null
New-Item -ItemType Directory -Force -Path ".\.gocache" | Out-Null

$goExe = (Get-Command go -CommandType Application -ErrorAction Stop).Source

$env:GOCACHE = Join-Path $PSScriptRoot ".gocache"
& $goExe build -buildvcs=false -trimpath -ldflags="-s -w" -o ".\dist\yzu-wlan.exe" ".\cmd\yzu-wlan"
if ($LASTEXITCODE -ne 0) {
    throw "go build failed with exit code $LASTEXITCODE"
}

Write-Host "Built .\dist\yzu-wlan.exe"
