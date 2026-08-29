@echo off
chcp 65001 >nul
cd /d "%~dp0"

set ENV_FILE=%APPDATA%\YZU-WLAN\.env
set APP_EXE=%APPDATA%\YZU-WLAN\yzu-wlan.exe

if not exist "%ENV_FILE%" (
    echo %ENV_FILE% not found. Copy .env.example there first.
    pause
    exit /b 1
)

if not exist "%APP_EXE%" (
    powershell -ExecutionPolicy Bypass -File "%~dp0build.ps1"
    if not exist "%APPDATA%\YZU-WLAN" mkdir "%APPDATA%\YZU-WLAN"
    copy /Y "%~dp0dist\yzu-wlan.exe" "%APP_EXE%" >nul
)

"%APP_EXE%" -e "%ENV_FILE%"
