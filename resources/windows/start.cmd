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
    echo Missing mediamtx.yml - it ships in the bundle next to this script.
    pause
    exit /b 1
)

if not exist mediamtx.exe (
    echo Missing mediamtx.exe.
    echo Download mediamtx_v1.14.0_windows_amd64.zip from
    echo https://github.com/bluenviron/mediamtx/releases and put
    echo mediamtx.exe in this folder.
    pause
    exit /b 1
)

start "MediaMTX" /min mediamtx.exe mediamtx.yml
stream-proxy.exe
pause
