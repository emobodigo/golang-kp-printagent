@echo off
setlocal
title Build PrintAgent

echo 🚀 Building PrintAgent...

rem === Cleanup old builds ===
rmdir /s /q build 2>nul
rmdir /s /q dist 2>nul
mkdir build
mkdir dist

rem === Compile Go binary ===
go build -ldflags="-s -w" -o build\PrintAgent.exe main.go
if errorlevel 1 (
    echo ❌ Build failed.
    pause
    exit /b
)

rem === Copy dependencies (if any) ===
copy main.go build\ >nul 2>&1

rem === Create ZIP package ===
powershell -Command "Compress-Archive -Path build\* -DestinationPath dist\PrintAgent.zip -Force"

echo ✅ Build completed successfully!
echo 📦 Output: dist\PrintAgent.zip
pause
