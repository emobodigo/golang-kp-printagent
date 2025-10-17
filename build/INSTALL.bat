@echo off
setlocal
title CoralisPrintAgent - Quick Installer

echo ================================================
echo     CoralisPrintAgent - Installation Wizard
echo ================================================
echo.
echo Please choose installation method:
echo.
echo 1. Windows Service (Recommended)
echo    ✓ Runs automatically on system boot
echo    ✓ Runs in background (no console)
echo    ✓ More stable
echo    ⚠ Requires Administrator rights
echo.
echo 2. Startup Folder
echo    ✓ Runs when you log in
echo    ✓ Shows console window (can minimize)
echo    ✓ No admin rights needed
echo    ✓ Easy to remove
echo.
echo 3. Manual - Don't install to startup
echo    ✓ Run only when needed
echo.
set /p CHOICE="Enter your choice (1, 2, or 3^): "

if "%CHOICE%"=="1" (
    echo.
    echo Installing as Windows Service...
    echo This requires Administrator privileges.
    echo.
    pause
    call install-service.bat
) else if "%CHOICE%"=="2" (
    echo.
    echo Installing to Startup folder...
    echo.
    pause
    call install-startup.bat
) else if "%CHOICE%"=="3" (
    echo.
    echo ✅ Installation skipped.
    echo.
    echo To run CoralisPrintAgent manually:
    echo - Double-click: CoralisPrintAgent.exe
    echo - Or use: run.bat
    echo.
    pause
) else (
    echo.
    echo ❌ Invalid choice. Please run this installer again.
    echo.
    pause
)
