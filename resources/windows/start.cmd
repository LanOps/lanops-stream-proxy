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

rem MediaMTX fails fast if Windows will not let it have a port, and its window
rem is minimised, so the reason is easy to miss. The port to check comes from
rem .env, since the API port is as likely to be the contested one as RTMP.
rem Warn, then carry on: the proxy retries, so a false alarm costs nothing.
timeout /t 3 /nobreak >nul
powershell -NoProfile -ExecutionPolicy Bypass -Command "$port='9997'; foreach ($line in Get-Content -LiteralPath '.env' -ErrorAction SilentlyContinue) { if ($line -match '^MEDIAMTX_API_ADDRESS\s*=\s*(.+)$') { $port = ($matches[1] -split ':')[-1].Trim() } }; $open = Test-NetConnection -ComputerName 127.0.0.1 -Port ([int]$port) -InformationLevel Quiet -WarningAction SilentlyContinue; if (-not $open) { Write-Host ''; Write-Host ('WARNING: MediaMTX is not listening on port ' + $port + ', so it probably did not start.'); Write-Host 'Restore its window from the taskbar to read the error.'; Write-Host 'If it mentions bind and access permissions, Windows has reserved that port.'; Write-Host 'See the README section: MediaMTX cannot open its port.'; Write-Host '' }"

stream-proxy.exe
pause
