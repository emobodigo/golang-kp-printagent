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

CoralisPrintAgent.exe

echo.
echo CoralisPrintAgent has stopped.
pause
