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
| `MEDIAMTX_API_ADDRESS` | `127.0.0.1:1936`. Must match `apiAddress` in `mediamtx.yml`. Not MediaMTX's default 9997, because Windows refuses that on machines with Hyper-V or WSL installed |
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

## 6. Connecting to the proxy

The proxy has no web interface. It listens on a port and answers three endpoints over HTTP, with basic auth.

### Address

| Connecting from | Address |
|---|---|
| the OBS machine | `http://127.0.0.1:8080` |
| another machine on the LAN, a phone or a laptop | `http://<proxy-machine-lan-ip>:8080` |

The port is `API_PORT` in `.env`, 8080 in the bundle.

### Credentials

`API_ADMIN_USERNAME` and `API_ADMIN_PASSWORD` from `.env`. The bundle generates the password, so read it from the file rather than guessing:

```
notepad .env
```

Every request needs them. There are no anonymous endpoints.

Windows Firewall asks about `stream-proxy.exe` on 8080 during the first run. Allow it if anyone needs the API from another machine; if only the OBS machine ever calls it, that rule can stay blocked.

### Endpoints

| Endpoint | Method | Body | What it does |
|---|---|---|---|
| `/streams` | GET | | every stream the proxy has seen, with `enabled` |
| `/streams/<name>` | GET | | one stream |
| `/streams/<name>/enable` | POST | `{"enabled": true}` or `{"enabled": false}` | puts a stream into or out of the rotation |

### From a browser

Open `http://127.0.0.1:8080/streams`. The browser asks for the username and password, then shows the raw JSON. That is the whole interface: there is nothing to click.

### From a command prompt

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

Setting it back to `true` re-adds the sources within about 7 seconds.

Those are Command Prompt commands. In PowerShell `curl` can be an alias for `Invoke-WebRequest`, which takes different arguments, so use `curl.exe` or this instead:

```powershell
$cred = Get-Credential admin
Invoke-RestMethod -Uri http://127.0.0.1:8080/streams -Credential $cred
Invoke-RestMethod -Uri http://127.0.0.1:8080/streams/cam1/enable -Method Post -Credential $cred `
  -ContentType 'application/json' -Body '{"enabled":false}'
```

### If you cannot reach it from another machine

Work from the proxy machine outwards, so you know which part is broken.

**1. On the proxy machine:**

```
curl -u admin:PASSWORD http://127.0.0.1:8080/streams
```

If that answers, the proxy is listening and the credentials are right. If it
does not, the problem is here, not the network: check the proxy window for a
startup error, and that `API_PORT` matches the port you used.

**2. Still on the proxy machine, confirm it is listening on every interface and
not just loopback:**

```
netstat -ano | findstr :8080
```

The line should read `0.0.0.0:8080`, not `127.0.0.1:8080`.

**3. From the other machine:**

```powershell
Test-NetConnection <proxy-machine-lan-ip> -Port 8080
```

`TcpTestSucceeded : False` means something between the two is dropping it. In
order of likelihood:

- **Windows Firewall** on the proxy machine. Allow it from an admin prompt:

  ```
  netsh advfirewall firewall add rule name="lanops stream proxy" dir=in action=allow protocol=TCP localport=8080
  ```

- **Network profile.** If the firewall prompt was cancelled once, or the LAN is
  detected as a Public network where the app's rules do not apply, the port
  stays closed. Settings > Network & Internet > Ethernet/Wi-Fi > Network
  profile, set it to Private, or add the rule above.
- **AP or client isolation.** Many LAN-party switches and venue access points
  block wireless clients from talking to each other. Test from a wired port
  before believing anything else.

`TcpTestSucceeded : True` but the request still fails means the network is fine
and it is auth: check you are using `API_ADMIN_USERNAME` and `API_ADMIN_PASSWORD`
from `.env`, not the OBS or MediaMTX password.

### This is not how viewers watch

The proxy does not serve video. It only drives OBS. Viewers watch whatever OBS outputs, to Twitch, YouTube or a file. MediaMTX on this machine is configured with RTMP and the API only, so there is no viewer page or stream here either.

## 7. How the streamers connect

Any RTMP encoder works. Most people will use OBS on their own machine, and it
only needs three things.

### In their OBS

**Settings > Stream**

```
Service     Custom...
Server      rtmp://<proxy-machine-lan-ip>:1935
Stream Key  cam1
```

Replace the stream key with their own name. Then **Start Streaming**. Nothing else needs changing, and they do not need to
install anything.

Anything that can publish RTMP works the same way: Streamlabs, vMix, Larix on a
phone, or ffmpeg:

```
ffmpeg -re -i input.mp4 -c:v libx264 -preset veryfast -tune zerolatency \
       -g 30 -f flv rtmp://<proxy-machine-lan-ip>:1935/cam1
```

### The stream key is the stream name

That string becomes the source name in OBS, the label on screen, and the name
in the API. Four rules, and the first two have already cost people an evening:

- **Every key must be unique.** Two publishers on the same key fight. MediaMTX
  drops the one that connected first, so the older streamer's encoder dies with
  a broken pipe, reconnects, drops the other one, and the source stutters. If
  two streams keep flickering, look for a duplicate key.
- **Keys are case sensitive.** `cam1` and `Cam1` are two different streams, and
  both will appear in OBS.
- Keep them short and lowercase, with no spaces or slashes. `cam1`, `cam2`,
  `stage`, `desk-left`.
- Agree the names before the event. The label is what the audience sees, so the
  key is also the caption.

### What happens when they go live

- The stream appears in the `Proxy Scenes` scene within about 30 seconds, as a
  media source and a name label. It is enabled automatically, so it joins the
  rotation.
- Widescreen 16:9 sources fill the frame. Anything else, a phone in portrait for
  example, is centred with black bars down the sides.
- Their audio comes through with the picture. If you do not want a streamer's
  mic in the programme mix, mute the `media_source-<name>` source in OBS.
- When they stop, MediaMTX drops the stream and the proxy removes their sources
  within about 7 seconds. When they start again they come back, enabled.

### Check it without a streamer

Before anyone arrives, publish a test pattern to the proxy from the OBS machine:

```
ffmpeg -re -f lavfi -i testsrc=size=1280x720:rate=30 -c:v libx264 -preset veryfast \
       -tune zerolatency -g 30 -f flv rtmp://127.0.0.1:1935/test
```

Within 30 seconds `test` appears in the API and in OBS. Stop it with Ctrl+C.

### If their stream does not show up

| What you see | What it means |
|---|---|
| `GET /streams` does not list their key | Nothing reached MediaMTX. Wrong IP, wrong port, key misspelled, or Windows Firewall blocking 1935 |
| It is in the API but not in OBS | Wait a full 30 seconds. If it is still missing, check the proxy window for a sync error |
| Two entries that look like the same person | Case or spelling difference. Disable the stray one with the API, or ask them to align their keys |
| Their stream keeps dropping | Duplicate key. See the first rule above |

## While it is running

- **One visible at a time.** The rotation shows one stream, then fades to the next, every 30 seconds, in the order the streams were first seen. A changeover fades over 5 seconds.
- **First appearance takes up to 30 seconds.** Sources are added on a 7 second cycle; the rotation that makes one visible runs on a 30 second cycle.
- **Only live streams count.** A path exists in MediaMTX the moment someone publishes to it, and disappears when they stop. Paths that are listed but have no publisher, for example ones defined in `mediamtx.yml`, report as not ready and are ignored. Nothing creates a media source that can never connect.
- **Stopping a stream cleans up.** When a publisher stops, the proxy removes their sources from OBS within about 7 seconds. If they were the visible one, the rotation moves straight to the next stream instead of waiting out the rest of the 30 seconds. Measured at 7 seconds end to end: publisher killed, sources removed, next stream on screen.
- **A stream that comes back is enabled again.** Its sources are recreated and it rejoins the rotation. If you had deliberately dropped someone from the rotation, disable them again after they reconnect.
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
| MediaMTX exits with `bind: An attempt was made to access a socket in a way forbidden by its access permissions` | Windows has reserved the port. Not a permissions problem with the file, and not something unblocking the exe fixes. See below |

### MediaMTX cannot open its port

Windows reserves blocks of TCP ports for Hyper-V, WSL and Docker. If a port
falls inside one of those blocks, nothing can listen on it and MediaMTX exits
at startup. The message Windows gives is misleading: it reads like a file
permission problem, but changing the file's permissions does nothing.

MediaMTX opens two ports, and either can hit this:

| Port | What it is | Where it is set |
|---|---|---|
| 1936 | the API the proxy polls | `apiAddress` in `mediamtx.yml`, `MEDIAMTX_API_ADDRESS` in `.env` |
| 1935 | RTMP, where streamers publish | `rtmpAddress` in `mediamtx.yml`, `MEDIAMTX_RTMP_ADDRESS` in `.env` |

The bundle ships the API on 1936 rather than MediaMTX's own default of 9997,
because 9997 is the one that gets refused on a Windows machine with WSL or
Hyper-V installed.

Read which port MediaMTX named in its error, then check it in an admin Command
Prompt, replacing 1936 with whichever port was refused:

```
netstat -ano | findstr :1936
netsh int ipv4 show excludedportrange protocol=tcp
```

If `netstat` shows a PID, something else is using the port. If the second
command shows a range covering it, Windows reserved it. Either reserve the port
for MediaMTX yourself, which needs an admin prompt:

```
net stop winnat
netsh int ipv4 add excludedportrange protocol=tcp startport=1936 numberofports=1
net start winnat
```

or move that one port. Moving the **API** port is the smaller change, because
nothing outside this folder refers to it. Change both lines to the new port and
leave everything else alone:

```
mediamtx.yml        apiAddress: 127.0.0.1:<port>
.env                MEDIAMTX_API_ADDRESS=127.0.0.1:<port>
```

Moving the **RTMP** port also changes what the streamers type, so three places
have to agree:

```
mediamtx.yml        rtmpAddress: :1936
.env                MEDIAMTX_RTMP_ADDRESS=127.0.0.1:1936
the streamers        rtmp://<this-machine>:1936/<key>
```

`start.cmd` reads the API port out of `MEDIAMTX_API_ADDRESS` when it checks that
MediaMTX came up, so it follows whichever port you choose. The Windows Firewall
rule follows the RTMP port, so moving that one asks again.

## Building the bundle yourself

Only needed if you are changing the code. The script needs `go`, `curl`, `unzip` and `zip`, so run it in WSL, Git Bash or Linux:

```bash
./resources/windows/build-bundle.sh
# dist/lanops-stream-proxy-windows.zip
```

It builds the proxy for windows/amd64, downloads the MediaMTX version pinned in the script, and assembles the same layout that ships in the zip. `MEDIAMTX_VERSION` overrides the version.
