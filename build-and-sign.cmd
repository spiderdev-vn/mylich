@echo off
setlocal
cd /d "%~dp0lich-cli"
call "%~dp0lich-cli\build-and-sign.cmd"
endlocal
