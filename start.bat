@echo off
setlocal
cd /d "%~dp0"
start "Discord Bot" /b /wait go run .
set "exit_code=%errorlevel%"
endlocal & exit /b %exit_code%