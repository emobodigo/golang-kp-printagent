@echo off
setlocal
title Install CoralisPrintAgent to Startup

echo ================================================
echo   Adding CoralisPrintAgent to Startup Folder
echo ================================================
echo.
echo This will make CoralisPrintAgent start automatically
echo when you log in to Windows.
echo.
echo ℹ️  No administrator rights required
echo ℹ️  A console window will appear on startup
echo ℹ️  You can minimize the console window
echo.
pause

set "APP_PATH=%~dp0CoralisPrintAgent-x86.exe"
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" set "APP_PATH=%~dp0CoralisPrintAgent-x64.exe"
if "%PROCESSOR_ARCHITEW6432%"=="AMD64" set "APP_PATH=%~dp0CoralisPrintAgent-x64.exe"
set "STARTUP_FOLDER=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"

echo Creating shortcut in Startup folder...

rem === Create VBS script to make shortcut ===
set "VBS_FILE=%TEMP%\create_shortcut.vbs"
(
Set oWS = WScript.CreateObject("WScript.Shell")
sLinkFile = "%SHORTCUT_PATH%"
Set oLink = oWS.CreateShortcut(sLinkFile)
oLink.TargetPath = "%APP_PATH%"
oLink.WorkingDirectory = "%~dp0"
oLink.Description = "Coralis Print Agent - Local Printer Service"
oLink.WindowStyle = 7
oLink.Save
) > "%VBS_FILE%"

cscript //nologo "%VBS_FILE%"
del "%VBS_FILE%"

if exist "%SHORTCUT_PATH%" (
    echo.
    echo ✅ CoralisPrintAgent added to startup successfully!
    echo.
    echo 📋 Startup Information:
    echo    Location: %SHORTCUT_PATH%
    echo    URL: http://localhost:8081
    echo.
    echo 💡 The application will start automatically when you log in.
    echo    A console window will appear - you can minimize it.
    echo.
    echo 🧪 To test now, double-click: %~dp0run.bat
    echo.
) else (
    echo.
    echo ❌ Failed to create startup shortcut
    echo.
)

pause
