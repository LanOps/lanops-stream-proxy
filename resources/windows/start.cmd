@echo off
rem Starts MediaMTX minimised, then the proxy in this window.
rem Both read their config from this folder.
cd /d "%~dp0"

if not exist .env (
    echo Missing .env - copy .env.example to .env and fill it in.
    pause
    exit /b 1
)

if not exist mediamtx.yml (
    echo Missing mediamtx.yml - copy it from resources/windows in the repo.
    pause
    exit /b 1
)

start "MediaMTX" /min mediamtx.exe mediamtx.yml
stream-proxy.exe
pause
