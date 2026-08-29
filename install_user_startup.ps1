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

$command = "powershell -WindowStyle Hidden -ExecutionPolicy Bypass -Command `"Start-Sleep -Seconds 30; & '$installedExe' -e '$envFile'`""
$runKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
New-ItemProperty -Path $runKey -Name "YZU-WLAN-AutoLogin" -Value $command -PropertyType String -Force | Out-Null

Write-Host "Installed current-user startup entry: YZU-WLAN-AutoLogin"
Write-Host "Executable: $installedExe"
Write-Host "Config: $envFile"
