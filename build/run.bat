@echo off
title CoralisPrintAgent - Press Ctrl+C to stop
echo.
echo ================================================
echo     CoralisPrintAgent - Local Printer Service
echo ================================================
echo.
echo Starting CoralisPrintAgent...
echo Server: http://localhost:8081
echo.
echo 💡 This window shows real-time logs
echo 💡 Minimize this window (don't close it)
echo 💡 Press Ctrl+C to stop the service
echo.
echo ================================================
echo.

if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    CoralisPrintAgent-x64.exe
) else if "%PROCESSOR_ARCHITEW6432%"=="AMD64" (
    CoralisPrintAgent-x64.exe
) else (
    CoralisPrintAgent-x86.exe
)

echo.
echo CoralisPrintAgent has stopped.
pause
