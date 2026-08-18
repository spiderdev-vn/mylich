@echo off
setlocal
cd /d "%~dp0"

where pwsh.exe >nul 2>&1
if %ERRORLEVEL% equ 0 (
    pwsh.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\build-and-sign.ps1"
) else (
    powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0scripts\build-and-sign.ps1"
)

if %ERRORLEVEL% neq 0 (
    echo [ERROR] Build and Sign failed with code %ERRORLEVEL%
    exit /b %ERRORLEVEL%
)
endlocal
