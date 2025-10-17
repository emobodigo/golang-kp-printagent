@echo off
setlocal
title Remove CoralisPrintAgent from Startup

echo ================================================
echo   Removing CoralisPrintAgent from Startup
echo ================================================
echo.

set "STARTUP_FOLDER=%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup"
set "SHORTCUT_PATH=%STARTUP_FOLDER%\CoralisPrintAgent.lnk"

if exist "%SHORTCUT_PATH%" (
    del "%SHORTCUT_PATH%"
    echo ✅ CoralisPrintAgent removed from startup successfully!
    echo.
    echo The application will no longer start automatically.
    echo.
) else (
    echo ⚠️  CoralisPrintAgent shortcut not found in startup folder
    echo    It may have already been removed.
    echo.
)

pause
