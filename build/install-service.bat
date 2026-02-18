@echo off
setlocal
title Install CoralisPrintAgent Service

echo ================================================
echo   Installing CoralisPrintAgent as Windows Service
echo ================================================
echo.
echo This will install CoralisPrintAgent to run automatically
echo in the background when Windows starts.
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
set "EXE_PATH=%~dp0CoralisPrintAgent-x86.exe"

rem === Detect architecture ===
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set "EXE_PATH=%~dp0CoralisPrintAgent-x64.exe"
) else if "%PROCESSOR_ARCHITEW6432%"=="AMD64" (
    set "EXE_PATH=%~dp0CoralisPrintAgent-x64.exe"
) else (
    set "EXE_PATH=%~dp0CoralisPrintAgent-x86.exe"
)

echo Creating service...
sc create %SERVICE_NAME% binPath= "%EXE_PATH%" start= auto DisplayName= "Coralis Print Agent"
if errorlevel 1 (
    echo.
    echo ❌ Failed to create service
    echo    Service may already exist. Try uninstall-service.bat first.
    pause
    exit /b 1
)

sc description %SERVICE_NAME% "Local printer service for Coralis Healthcare web applications"

echo Starting service...
sc start %SERVICE_NAME%

echo.
echo ✅ CoralisPrintAgent installed successfully!
echo.
echo 📋 Service Information:
echo    Name: %SERVICE_NAME%
echo    URL: http://localhost:8081
echo    Logs: %~dp0printagent.log
echo.
echo 💡 The service will start automatically when Windows boots.
echo    No console window will be shown.
echo.
pause
