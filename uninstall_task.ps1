#requires -RunAsAdministrator
$ErrorActionPreference = "Stop"

$exe = Join-Path $env:APPDATA "YZU-WLAN\yzu-wlan.exe"
$envFile = Join-Path $env:APPDATA "YZU-WLAN\.env"

if (!(Test-Path $exe)) {
    throw "$exe not found. Remove tasks manually in taskschd.msc or run build.ps1 and install_task.ps1 first."
}

& $exe -uninstall -e $envFile

Write-Host "Uninstalled YZU-WLAN scheduled tasks."
