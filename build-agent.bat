@echo off
setlocal
title Build CoralisPrintAgent

echo.
echo ================================================
echo          🚀 Building CoralisPrintAgent
echo ================================================
echo.

rem === Check if Go is installed ===
where go >nul 2>nul
if errorlevel 1 (
    echo ❌ Go is not installed or not in PATH
    echo    Please install Go from https://go.dev/dl/
    pause
    exit /b 1
)

rem === Cleanup old builds ===
echo 🧹 Cleaning old builds...
if exist build rmdir /s /q build
if exist dist rmdir /s /q dist
mkdir build
mkdir dist

rem === Build Go binary WITH console window ===
echo 🔨 Compiling Go binary...
set CGO_ENABLED=0
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w" -o build\CoralisPrintAgent.exe main.go

if errorlevel 1 (
    echo.
    echo ❌ Build failed!
    pause
    exit /b 1
)

rem === Create README ===
echo 📝 Creating README...
(
echo CoralisPrintAgent - Local Printer Service
echo ==========================================
echo.
echo INSTALLATION OPTIONS:
echo.
echo Option 1: Windows Service (Recommended for Production^)
echo   - Run 'install-service.bat' as Administrator
echo   - Runs in background, starts automatically on boot
echo   - No console window
echo   - More stable and reliable
echo.
echo Option 2: Startup Folder (Simple, User-level^)
echo   - Run 'install-startup.bat' (No admin needed^)
echo   - Starts when user logs in
echo   - Shows console window (can be minimized^)
echo   - Easy to disable
echo.
echo Option 3: Manual Run
echo   - Double-click 'CoralisPrintAgent.exe' or 'run.bat'
echo   - Run only when needed
echo.
echo USAGE:
echo 1. A console window will open showing logs
echo 2. Service runs on http://localhost:8081
echo 3. DO NOT close the console window (minimize it instead^)
echo.
echo ENDPOINTS:
echo - GET  /health    - Check service status
echo - GET  /printers  - List available printers
echo - POST /print     - Send print job
echo.
echo UNINSTALL:
echo - Service: Run 'uninstall-service.bat' as Administrator
echo - Startup: Run 'uninstall-startup.bat'
echo.
echo FILES:
echo - printagent.log - Application logs
echo - README.txt     - This file
echo.
echo Support: https://coralishealthcare.com
) > build\README.txt

rem === Create Service Install Script ===
echo 📝 Creating service install script...
(
echo @echo off
echo setlocal
echo title Install CoralisPrintAgent Service
echo.
echo echo ================================================
echo echo   Installing CoralisPrintAgent as Windows Service
echo echo ================================================
echo echo.
echo echo This will install CoralisPrintAgent to run automatically
echo echo in the background when Windows starts.
echo echo.
echo.
echo rem === Check admin privileges ===
echo net session ^>nul 2^>^&1
echo if errorlevel 1 ^(
echo     echo ❌ This script must be run as Administrator
echo     echo.
echo     echo Right-click this file and select "Run as administrator"
echo     pause
echo     exit /b 1
echo ^)
echo.
echo set "SERVICE_NAME=CoralisPrintAgent"
echo set "EXE_PATH=%%~dp0CoralisPrintAgent.exe"
echo.
echo echo Creating service...
echo sc create %%SERVICE_NAME%% binPath= "%%EXE_PATH%%" start= auto DisplayName= "Coralis Print Agent"
echo if errorlevel 1 ^(
echo     echo.
echo     echo ❌ Failed to create service
echo     echo    Service may already exist. Try uninstall-service.bat first.
echo     pause
echo     exit /b 1
echo ^)
echo.
echo sc description %%SERVICE_NAME%% "Local printer service for Coralis Healthcare web applications"
echo.
echo echo Starting service...
echo sc start %%SERVICE_NAME%%
echo.
echo echo.
echo echo ✅ CoralisPrintAgent installed successfully!
echo echo.
echo echo 📋 Service Information:
echo echo    Name: %%SERVICE_NAME%%
echo echo    URL: http://localhost:8081
echo echo    Logs: %%~dp0printagent.log
echo echo.
echo echo 💡 The service will start automatically when Windows boots.
echo echo    No console window will be shown.
echo echo.
echo pause
) > build\install-service.bat

rem === Create Service Uninstall Script ===
echo 📝 Creating service uninstall script...
(
echo @echo off
echo setlocal
echo title Uninstall CoralisPrintAgent Service
echo.
echo echo ================================================
echo echo   Uninstalling CoralisPrintAgent Service
echo echo ================================================
echo echo.
echo.
echo rem === Check admin privileges ===
echo net session ^>nul 2^>^&1
echo if errorlevel 1 ^(
echo     echo ❌ This script must be run as Administrator
echo     echo.
echo     echo Right-click this file and select "Run as administrator"
echo     pause
echo     exit /b 1
echo ^)
echo.
echo set "SERVICE_NAME=CoralisPrintAgent"
echo.
echo echo Stopping service...
echo sc stop %%SERVICE_NAME%%
echo timeout /t 2 ^>nul
echo.
echo echo Deleting service...
echo sc delete %%SERVICE_NAME%%
echo.
echo echo.
echo echo ✅ CoralisPrintAgent service uninstalled successfully!
echo echo.
echo pause
) > build\uninstall-service.bat

rem === Create Startup Folder Install Script ===
echo 📝 Creating startup install script...
(
echo @echo off
echo setlocal
echo title Install CoralisPrintAgent to Startup
echo.
echo echo ================================================
echo echo   Adding CoralisPrintAgent to Startup Folder
echo echo ================================================
echo echo.
echo echo This will make CoralisPrintAgent start automatically
echo echo when you log in to Windows.
echo echo.
echo echo ℹ️  No administrator rights required
echo echo ℹ️  A console window will appear on startup
echo echo ℹ️  You can minimize the console window
echo echo.
echo pause
echo.
echo set "APP_PATH=%%~dp0CoralisPrintAgent.exe"
echo set "STARTUP_FOLDER=%%APPDATA%%\Microsoft\Windows\Start Menu\Programs\Startup"
echo set "SHORTCUT_PATH=%%STARTUP_FOLDER%%\CoralisPrintAgent.lnk"
echo.
echo echo Creating shortcut in Startup folder...
echo.
echo rem === Create VBS script to make shortcut ===
echo set "VBS_FILE=%%TEMP%%\create_shortcut.vbs"
echo ^(
echo Set oWS = WScript.CreateObject^("WScript.Shell"^)
echo sLinkFile = "%%SHORTCUT_PATH%%"
echo Set oLink = oWS.CreateShortcut^(sLinkFile^)
echo oLink.TargetPath = "%%APP_PATH%%"
echo oLink.WorkingDirectory = "%%~dp0"
echo oLink.Description = "Coralis Print Agent - Local Printer Service"
echo oLink.WindowStyle = 7
echo oLink.Save
echo ^) ^> "%%VBS_FILE%%"
echo.
echo cscript //nologo "%%VBS_FILE%%"
echo del "%%VBS_FILE%%"
echo.
echo if exist "%%SHORTCUT_PATH%%" ^(
echo     echo.
echo     echo ✅ CoralisPrintAgent added to startup successfully!
echo     echo.
echo     echo 📋 Startup Information:
echo     echo    Location: %%SHORTCUT_PATH%%
echo     echo    URL: http://localhost:8081
echo     echo.
echo     echo 💡 The application will start automatically when you log in.
echo     echo    A console window will appear - you can minimize it.
echo     echo.
echo     echo 🧪 To test now, double-click: %%~dp0run.bat
echo     echo.
echo ^) else ^(
echo     echo.
echo     echo ❌ Failed to create startup shortcut
echo     echo.
echo ^)
echo.
echo pause
) > build\install-startup.bat

rem === Create Startup Uninstall Script ===
echo 📝 Creating startup uninstall script...
(
echo @echo off
echo setlocal
echo title Remove CoralisPrintAgent from Startup
echo.
echo echo ================================================
echo echo   Removing CoralisPrintAgent from Startup
echo echo ================================================
echo echo.
echo.
echo set "STARTUP_FOLDER=%%APPDATA%%\Microsoft\Windows\Start Menu\Programs\Startup"
echo set "SHORTCUT_PATH=%%STARTUP_FOLDER%%\CoralisPrintAgent.lnk"
echo.
echo if exist "%%SHORTCUT_PATH%%" ^(
echo     del "%%SHORTCUT_PATH%%"
echo     echo ✅ CoralisPrintAgent removed from startup successfully!
echo     echo.
echo     echo The application will no longer start automatically.
echo     echo.
echo ^) else ^(
echo     echo ⚠️  CoralisPrintAgent shortcut not found in startup folder
echo     echo    It may have already been removed.
echo     echo.
echo ^)
echo.
echo pause
) > build\uninstall-startup.bat

rem === Create manual run script (Shows Console) ===
echo 📝 Creating run script...
(
echo @echo off
echo title CoralisPrintAgent - Press Ctrl+C to stop
echo echo.
echo echo ================================================
echo echo     CoralisPrintAgent - Local Printer Service
echo echo ================================================
echo echo.
echo echo Starting CoralisPrintAgent...
echo echo Server: http://localhost:8081
echo echo.
echo echo 💡 This window shows real-time logs
echo echo 💡 Minimize this window (don't close it^)
echo echo 💡 Press Ctrl+C to stop the service
echo echo.
echo echo ================================================
echo echo.
echo.
echo CoralisPrintAgent.exe
echo.
echo echo.
echo echo CoralisPrintAgent has stopped.
echo pause
) > build\run.bat

rem === Create Quick Installer (User Choice) ===
echo 📝 Creating quick installer...
(
echo @echo off
echo setlocal
echo title CoralisPrintAgent - Quick Installer
echo.
echo echo ================================================
echo echo     CoralisPrintAgent - Installation Wizard
echo echo ================================================
echo echo.
echo echo Please choose installation method:
echo echo.
echo echo 1. Windows Service (Recommended^)
echo echo    ✓ Runs automatically on system boot
echo echo    ✓ Runs in background (no console^)
echo echo    ✓ More stable
echo echo    ⚠ Requires Administrator rights
echo echo.
echo echo 2. Startup Folder
echo echo    ✓ Runs when you log in
echo echo    ✓ Shows console window (can minimize^)
echo echo    ✓ No admin rights needed
echo echo    ✓ Easy to remove
echo echo.
echo echo 3. Manual - Don't install to startup
echo echo    ✓ Run only when needed
echo echo.
echo set /p CHOICE="Enter your choice (1, 2, or 3^): "
echo.
echo if "%%CHOICE%%"=="1" ^(
echo     echo.
echo     echo Installing as Windows Service...
echo     echo This requires Administrator privileges.
echo     echo.
echo     pause
echo     call install-service.bat
echo ^) else if "%%CHOICE%%"=="2" ^(
echo     echo.
echo     echo Installing to Startup folder...
echo     echo.
echo     pause
echo     call install-startup.bat
echo ^) else if "%%CHOICE%%"=="3" ^(
echo     echo.
echo     echo ✅ Installation skipped.
echo     echo.
echo     echo To run CoralisPrintAgent manually:
echo     echo - Double-click: CoralisPrintAgent.exe
echo     echo - Or use: run.bat
echo     echo.
echo     pause
echo ^) else ^(
echo     echo.
echo     echo ❌ Invalid choice. Please run this installer again.
echo     echo.
echo     pause
echo ^)
) > build\INSTALL.bat

rem === Get version info ===
for /f "tokens=*" %%i in ('git rev-parse --short HEAD 2^>nul') do set GIT_HASH=%%i
if "%GIT_HASH%"=="" set GIT_HASH=unknown

rem === Create ZIP package ===
echo 📦 Creating ZIP package...
powershell -Command "Compress-Archive -Path build\* -DestinationPath dist\CoralisPrintAgent-v1.0-%GIT_HASH%.zip -Force"

if errorlevel 1 (
    echo ❌ Failed to create ZIP
    pause
    exit /b 1
)

rem === Show results ===
echo.
echo ================================================
echo          ✅ Build Completed Successfully!
echo ================================================
echo.
echo 📁 Build folder: build\
echo 📦 Package: dist\CoralisPrintAgent-v1.0-%GIT_HASH%.zip
echo.
echo 📋 Package contents:
echo    - CoralisPrintAgent.exe    (Main executable^)
echo    - INSTALL.bat              (Quick installer wizard^)
echo    - install-service.bat      (Install as Windows Service^)
echo    - uninstall-service.bat    (Remove service^)
echo    - install-startup.bat      (Add to startup folder^)
echo    - uninstall-startup.bat    (Remove from startup^)
echo    - run.bat                  (Manual run^)
echo    - README.txt               (Instructions^)
echo.
echo 💡 User Instructions:
echo    1. Extract the ZIP
echo    2. Run INSTALL.bat and choose installation method
echo    3. Done! Service will start automatically
echo.
echo 🎯 Installation Options:
echo    Option 1 (Service^)  = Best for production, runs on boot
echo    Option 2 (Startup^)  = Simple, runs on login
echo    Option 3 (Manual^)   = Run only when needed
echo.

rem === Optional: Test run ===
set /p TEST="🧪 Do you want to test run now? (Y/N): "
if /i "%TEST%"=="Y" (
    echo.
    echo Starting CoralisPrintAgent with console...
    cd build
    start cmd /k "title CoralisPrintAgent Console && CoralisPrintAgent.exe"
    timeout /t 2 >nul
    echo.
    echo Opening browser...
    start http://localhost:8081/health
)

pause