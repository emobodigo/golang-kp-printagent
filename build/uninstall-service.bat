@echo off
setlocal
title Uninstall CoralisPrintAgent Service

echo ================================================
echo   Uninstalling CoralisPrintAgent Service
echo ================================================
echo.

rem === Check admin privileges ===
net session >nul 2>&1
if errorlevel 1 (
    echo ❌ This script must be run as Administrator
    echo.
    echo Right-click this file and select "Run as administrator"
    pause
    exit /b 1
)

set "SERVICE_NAME=CoralisPrintAgent"

echo Stopping service...
sc stop %SERVICE_NAME%
timeout /t 2 >nul

echo Deleting service...
sc delete %SERVICE_NAME%

echo.
echo ✅ CoralisPrintAgent service uninstalled successfully!
echo.
pause
