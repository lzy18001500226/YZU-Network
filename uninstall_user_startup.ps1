$ErrorActionPreference = "Stop"

$runKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
Remove-ItemProperty -Path $runKey -Name "YZU-WLAN-AutoLogin" -ErrorAction SilentlyContinue

Write-Host "Uninstalled current-user startup entry: YZU-WLAN-AutoLogin"
