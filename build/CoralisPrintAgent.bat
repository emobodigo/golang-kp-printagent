@echo off
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    CoralisPrintAgent-x64.exe
) else if "%PROCESSOR_ARCHITEW6432%"=="AMD64" (
    CoralisPrintAgent-x64.exe
) else (
    CoralisPrintAgent-x86.exe
)
