#requires -RunAsAdministrator
$ErrorActionPreference = "Stop"

Set-Location $PSScriptRoot

$envDir = Join-Path $env:APPDATA "YZU-WLAN"
$envFile = Join-Path $envDir ".env"
$builtExe = Join-Path $PSScriptRoot "dist\yzu-wlan.exe"
$installedExe = Join-Path $envDir "yzu-wlan.exe"

if (!(Test-Path $envFile)) {
    throw "$envFile not found. Copy .env.example to $envFile and fill SSO_USERNAME / SSO_PASSWORD first."
}

if (!(Test-Path $builtExe)) {
    Write-Host "dist\yzu-wlan.exe not found, building first..."
    & "$PSScriptRoot\build.ps1"
}

New-Item -ItemType Directory -Force -Path $envDir | Out-Null
Copy-Item -Force $builtExe $installedExe

& $installedExe -install -e $envFile
if ($LASTEXITCODE -ne 0) {
    throw "install scheduled tasks failed with exit code $LASTEXITCODE"
}

Write-Host "Installed Windows scheduled tasks:"
Write-Host "  YZU-WLAN-AutoLogin"
Write-Host "  YZU-WLAN-AutoLogin-Repeat"
Write-Host "Executable: $installedExe"
Write-Host "Config: $envFile"
