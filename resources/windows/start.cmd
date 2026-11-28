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

rem MediaMTX fails fast if Windows will not let it have its port, and its
rem window is minimised, so the reason is easy to miss. Warn, then carry on:
rem the proxy retries, so a false alarm here costs nothing.
timeout /t 3 /nobreak >nul
powershell -NoProfile -ExecutionPolicy Bypass -Command "if (-not (Test-NetConnection -ComputerName 127.0.0.1 -Port 9997 -InformationLevel Quiet -WarningAction SilentlyContinue)) { Write-Host ''; Write-Host 'WARNING: MediaMTX is not listening on port 9997, so it probably did not start.'; Write-Host 'Restore its window from the taskbar to read the error.'; Write-Host 'If it mentions bind and access permissions, Windows has reserved the port.'; Write-Host 'See the README section: MediaMTX cannot open its port.'; Write-Host '' }"

stream-proxy.exe
pause
