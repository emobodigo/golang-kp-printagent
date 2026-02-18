#!/bin/bash
# ================================================
#   Build CoralisPrintAgent - Windows 7 Compatible
#   Requires: Go 1.20.x toolchain
# ================================================

set -e

GO120="$HOME/go/bin/go1.20.14"
GO_CMD="$GO120"

echo ""
echo "================================================"
echo "   Building CoralisPrintAgent (Windows 7 Safe)"
echo "================================================"
echo ""

# === Check if Go 1.20 is available ===
if ! command -v go1.20.14 &>/dev/null && [ ! -f "$GO120" ]; then
    echo "⚠️  Go 1.20.14 not found. Installing..."
    echo "   (Go 1.21+ does NOT support Windows 7)"
    echo ""
    go install golang.org/dl/go1.20.14@latest
    go1.20.14 download
    echo ""
    echo "✅ Go 1.20.14 installed."
fi

echo "🔍 Build toolchain: $($GO_CMD version)"
echo ""

# === Cleanup ===
echo "🧹 Cleaning old builds..."
rm -rf build dist
mkdir -p build dist

# === Build 64-bit (Windows 7 x64) ===
echo "🔨 Compiling 64-bit (Windows 7 x64)..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 "$GO_CMD" build \
    -ldflags="-s -w" \
    -o build/CoralisPrintAgent-x64.exe .
echo "   ✅ x64 done"

# === Build 32-bit (Windows 7 x86) ===
echo "🔨 Compiling 32-bit (Windows 7 x86)..."
CGO_ENABLED=0 GOOS=windows GOARCH=386 "$GO_CMD" build \
    -ldflags="-s -w" \
    -o build/CoralisPrintAgent-x86.exe .
echo "   ✅ x86 done"

# === Verify PE MinOSVersion ===
echo ""
echo "🔍 Verifying minimum OS version in PE headers..."
python3 -c "
import struct, sys

def check_pe(fname):
    with open(fname, 'rb') as f:
        data = f.read()
    pe_off = struct.unpack_from('<I', data, 0x3C)[0]
    opt_off = pe_off + 4 + 20
    magic = struct.unpack_from('<H', data, opt_off)[0]
    if magic == 0x10b:
        major = struct.unpack_from('<H', data, opt_off + 40)[0]
        minor = struct.unpack_from('<H', data, opt_off + 42)[0]
    else:
        major = struct.unpack_from('<H', data, opt_off + 48)[0]
        minor = struct.unpack_from('<H', data, opt_off + 50)[0]
    ok = major == 6 and minor == 1
    status = '✅' if ok else '❌ WARNING: Not Win7 compatible!'
    print(f'   {status} {fname}: MinOSVersion = {major}.{minor} (need 6.1 for Win7)')

check_pe('build/CoralisPrintAgent-x64.exe')
check_pe('build/CoralisPrintAgent-x86.exe')
"

# === Create auto-detect launcher ===
cat > build/CoralisPrintAgent.bat << 'EOF'
@echo off
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    CoralisPrintAgent-x64.exe
) else if "%PROCESSOR_ARCHITEW6432%"=="AMD64" (
    CoralisPrintAgent-x64.exe
) else (
    CoralisPrintAgent-x86.exe
)
EOF

# === Create run.bat ===
cat > build/run.bat << 'EOF'
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
echo This window shows real-time logs
echo Minimize this window (don't close it)
echo Press Ctrl+C to stop the service
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
EOF

# === Copy existing build scripts from build/ if they exist ===
for f in install-service.bat uninstall-service.bat install-startup.bat uninstall-startup.bat INSTALL.bat README.txt; do
    if [ -f "build/$f" ]; then
        echo "   Keeping existing: $f"
    fi
done

# === Get git hash ===
GIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# === Create ZIP ===
echo ""
echo "📦 Creating ZIP package..."
cd build
zip -r "../dist/CoralisPrintAgent-v1.0-${GIT_HASH}-win7.zip" . -x "*.DS_Store"
cd ..

echo ""
echo "================================================"
echo "          ✅ Build Completed Successfully!"
echo "================================================"
echo ""
echo "📁 Build folder : build/"
echo "📦 Package      : dist/CoralisPrintAgent-v1.0-${GIT_HASH}-win7.zip"
echo ""
echo "📋 Files:"
echo "   - CoralisPrintAgent-x64.exe  (Windows 7/8/10/11 64-bit)"
echo "   - CoralisPrintAgent-x86.exe  (Windows 7/8/10/11 32-bit)"
echo "   - CoralisPrintAgent.bat      (Auto-detect launcher)"
echo "   - run.bat                    (Manual run with logs)"
echo ""
echo "💡 Compatible with: Windows 7 SP1+, Windows 8, Windows 10, Windows 11"
echo ""
