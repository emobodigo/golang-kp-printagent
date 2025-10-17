CoralisPrintAgent - Local Printer Service
==========================================

INSTALLATION OPTIONS:

Option 1: Windows Service (Recommended for Production)
  - Run 'install-service.bat' as Administrator
  - Runs in background, starts automatically on boot
  - No console window
  - More stable and reliable

Option 2: Startup Folder (Simple, User-level)
  - Run 'install-startup.bat' (No admin needed)
  - Starts when user logs in
  - Shows console window (can be minimized)
  - Easy to disable

Option 3: Manual Run
  - Double-click 'CoralisPrintAgent.exe' or 'run.bat'
  - Run only when needed

USAGE:
1. A console window will open showing logs
2. Service runs on http://localhost:8081
3. DO NOT close the console window (minimize it instead)

ENDPOINTS:
- GET  /health    - Check service status
- GET  /printers  - List available printers
- POST /print     - Send print job

UNINSTALL:
- Service: Run 'uninstall-service.bat' as Administrator
- Startup: Run 'uninstall-startup.bat'

FILES:
- printagent.log - Application logs
- README.txt     - This file

Support: https://coralishealthcare.com
