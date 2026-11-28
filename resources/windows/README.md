# Running the stream proxy on Windows

The proxy takes every stream published to MediaMTX and puts it in OBS as a media source, with a name label, on one scene. Only one stream is visible at a time and it rotates. You control which streams are in the rotation through a small HTTP API.

Everything runs on the OBS machine as two ordinary programs. No Docker, no WSL, no install.

## What you need

- The bundle zip, `lanops-stream-proxy-windows.zip`. It contains `stream-proxy.exe`, `mediamtx.exe`, the configs and `start.cmd`.
- OBS Studio 28 or newer. Older versions need the obs-websocket 5.x plugin installed first.
- A folder you can write to, such as `C:\lanops\stream-proxy`.

## 1. Unzip somewhere writable

Do not use `C:\Program Files`. The proxy writes `bridge.db` into its own folder and Windows will not let it.

If Windows showed "Windows protected your PC" while unzipping, clear the download flag once from PowerShell in that folder:

```powershell
Unblock-File .\stream-proxy.exe
```

## 2. Configure OBS

OBS has to be running and listening before the proxy can do anything.

### 2a. Enable the WebSocket server

**Tools > WebSocket Server Settings**

- tick **Enable WebSocket server**
- **Server Port**: 4455
- **Enable Authentication**: optional

If you tick authentication, the password you set there must also go in `OBS_WEBSOCKET_PASSWORD` in `.env`. If you leave it off, leave that value empty. The two must agree, in either direction.

### 2b. Set the canvas to 1920x1080

**Settings > Video > Base (Canvas) Resolution**: `1920x1080`

The proxy scales every media source to fit a 1920x1080 box and puts the name label at 102,47. On a 1280x720 canvas the sources are larger than the canvas and get cropped. On 2560x1440 they sit in the top left corner. Output (Scaled) Resolution does not matter, only the canvas.

## 3. Set the proxy config

Open `.env` in the same folder. From a command prompt in that folder:

```
notepad .env
```

Notepad's Open dialog filters for `.txt` and will not show the file, hence the command.

| Value | What it should be |
|---|---|
| `DB_PATH` | `bridge.db`. Leave it |
| `OBS_WEBSOCKET_ADDRESS` | `127.0.0.1:4455` when OBS is on this machine |
| `OBS_WEBSOCKET_PASSWORD` | Empty if OBS auth is off, otherwise the OBS password |
| `OBS_PROXY_SCENE_NAME` | `Proxy Scenes`. The scene the proxy creates |
| `OBS_CLOSE_WHEN_INACTIVE` | `true`. Stops OBS decoding streams that are not on screen. Only set it to `false` if the picture fails to come back after a changeover |
| `MEDIAMTX_API_ADDRESS` | `127.0.0.1:9997`. Leave it |
| `MEDIAMTX_RTMP_ADDRESS` | `127.0.0.1:1935` when OBS is on this machine |
| `API_ADMIN_USERNAME` | The username for the API |
| `API_ADMIN_PASSWORD` | Already set to a generated value. Change it if you like |
| `API_PORT` | `8080` |

The two addresses are the ones that go wrong. They are what OBS is given, so they have to be reachable from the machine OBS runs on. If OBS is on a different PC, see "OBS on another machine" below.

## 4. Start it

Double-click **start.cmd**. It opens a window, starts MediaMTX minimised, then runs the proxy in that window.

Windows Firewall asks about both programs. Allow both. The `mediamtx.exe` rule on port 1935 is how other machines push streams in; `stream-proxy.exe` on 8080 is the API.

A first run against an empty OBS looks like this (trimmed, lines can interleave):

```
INF Loading API
INF Initializing Stream Proxy
INF Loading Config
INF Waiting for OBS
INF OBS Scene doesn't exist. Creating...
INF OBS Scene created!
INF OBS Scene Name: Proxy Scenes
INF OBS Scene UUID: 6a1f...
INF Starting API
INF Starting Stream Syncs
INF Starting Active Stream Rotation
INF Stream Found! Adding cam1 to the DB
INF Adding stream cam1 to OBS
INF Stream Found! Adding cam2 to the DB
INF Adding stream cam2 to OBS
```

`Waiting for OBS` repeating every 5 seconds means it cannot reach OBS. Check OBS is running, the WebSocket server is enabled, the port and password match. It keeps retrying rather than exiting, so you can fix OBS and watch it connect.

After the first minute the window is quiet. It has nothing to say unless something changes or breaks.

## 5. Put the proxy scene on air

The proxy creates a scene called `Proxy Scenes` and puts its sources there. **It never switches to it.** That is your job, and it is the step people miss.

Either:

- make `Proxy Scenes` the program scene, or
- in the scene you already use, **Add Source > Scene > Proxy Scenes**

Until you do this the proxy works perfectly and nothing appears on stream.

## 6. Check it is working

Streams are picked up automatically. To see what the proxy knows about:

```
curl -u admin:PASSWORD http://127.0.0.1:8080/streams
```

```json
[{"id":1,"name":"cam1","enabled":true,"obsStreamUuid":"...","obsStreamId":5,"obsTextUuid":"...","obsTextId":6}]
```

`enabled` is whether the stream is in the rotation. New streams start enabled.

To take a stream out of the rotation, which also removes its sources from OBS:

```
curl -u admin:PASSWORD -X POST -H "Content-Type: application/json" -d "{\"enabled\":false}" http://127.0.0.1:8080/streams/cam1/enable
```

Those are Command Prompt commands. In PowerShell `curl` can be an alias for `Invoke-WebRequest`, which takes different arguments, so use `curl.exe` or this instead:

```powershell
$cred = Get-Credential admin
Invoke-RestMethod -Uri http://127.0.0.1:8080/streams -Credential $cred
Invoke-RestMethod -Uri http://127.0.0.1:8080/streams/cam1/enable -Method Post -Credential $cred `
  -ContentType 'application/json' -Body '{"enabled":false}'
```

Setting it back to `true` re-adds the sources within about 7 seconds.

## 7. Tell the streamers where to push

Streamers point their own OBS at MediaMTX:

**Settings > Stream > Service: Custom**

- Server: `rtmp://<this-pc-lan-ip>:1935`
- Stream Key: the name they want to appear, for example `cam1`

Or from ffmpeg:

```
ffmpeg -re -i input.mp4 -c:v libx264 -preset veryfast -tune zerolatency -f flv rtmp://<this-pc-ip>:1935/cam1
```

The stream key becomes the stream name in OBS and in the API. Stream names with spaces or slashes make awkward source names, so keep them simple.

## While it is running

- **One visible at a time.** The rotation shows one stream, then fades to the next, every 30 seconds, in the order the streams were first seen. A changeover fades over 5 seconds.
- **First appearance takes up to 30 seconds.** Sources are added on a 7 second cycle; the rotation that makes one visible runs on a 30 second cycle.
- **Stopping a stream cleans up.** When a publisher stops, MediaMTX drops the stream, and the proxy removes its sources from OBS within about 7 seconds. If it was the visible one, the rotation moves on.
- **Do not rename or delete `media_source-<name>` or `text-<name>`.** The proxy recreates missing ones, but the fade filters are looked up by source name.
- **Stay on one scene collection.** The proxy resolves the scene UUID when it connects and keeps using it.
- **Only the visible stream decodes.** Sources are created with `close_when_inactive`, so OBS drops an RTMP stream as soon as the rotation hides it and reopens it on the way back in. That keeps CPU flat however many streams are enabled. If a build misbehaves with it, set `OBS_CLOSE_WHEN_INACTIVE=false` in `.env`.

## OBS on another machine

Set in `.env`:

```
OBS_WEBSOCKET_ADDRESS=192.168.1.50:4455
MEDIAMTX_RTMP_ADDRESS=192.168.1.60:1935
```

The first is the OBS machine, the second is this machine's LAN address. Open port 4455 on the OBS machine, and leave `MEDIAMTX_RTMP_ADDRESS` at `127.0.0.1` only when OBS is local. If OBS is elsewhere and this is left at `127.0.0.1`, the sources are created, OBS cannot open them, and they stay black.

## Starting it automatically

Point Task Scheduler at `start.cmd`, with **Run whether user is logged on or not** and the start-in folder set to this folder.

OBS still has to be running for anything to happen. Start OBS first, or let the proxy wait: it retries every 5 seconds.

## Troubleshooting

| What you see | What it means |
|---|---|
| `Waiting for OBS` repeating every 5s | OBS unreachable: not running, WebSocket server off, wrong port, wrong password, or its firewall |
| Sources in OBS but the picture is black | OBS cannot open the RTMP URL. Check `MEDIAMTX_RTMP_ADDRESS` is reachable from the OBS machine, and that something is publishing that stream name |
| Everything looks right, nothing on stream | The `Proxy Scenes` scene is not on air. See step 5 |
| Only some streams rotate | Only enabled streams rotate. The rest are in OBS and hidden |
| `no supported text source found in OBS` | OBS has neither text source plugin. A stock OBS install has both |
| Exits immediately | A required value is missing from `.env`. The log names it |
| Exits at startup with a database error | The folder is not writable. Move it out of `Program Files` |

## Building the bundle yourself

Only needed if you are changing the code. The script needs `go`, `curl`, `unzip` and `zip`, so run it in WSL, Git Bash or Linux:

```bash
./resources/windows/build-bundle.sh
# dist/lanops-stream-proxy-windows.zip
```

It builds the proxy for windows/amd64, downloads the MediaMTX version pinned in the script, and assembles the same layout that ships in the zip. `MEDIAMTX_VERSION` overrides the version.
