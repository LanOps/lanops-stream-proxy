LanOps stream proxy - Windows bundle
====================================

Everything the proxy needs is in this folder:

  stream-proxy.exe       the proxy
  mediamtx.exe           RTMP ingest
  mediamtx.yml           MediaMTX config, API and RTMP only
  start.cmd              starts both, proxy log in this window
  .env                   proxy config, edit this
  MEDIAMTX-LICENSE.txt   MediaMTX licence


Before you start: OBS
---------------------
The proxy drives OBS over obs-websocket and does not touch the mouse.

1. Tools > WebSocket Server Settings
     [x] Enable WebSocket server
         Server Port: 4455
     [ ] Enable Authentication          (see note below)

   If you tick Authentication, copy the password into
   OBS_WEBSOCKET_PASSWORD in .env. If you leave it unticked, leave that
   value empty. The two must agree.

2. Settings > Video
     Base (Canvas) Resolution: 1920x1080

   The proxy scales each media source to fit a 1920x1080 box and puts the
   stream name at 102,47. On a 1280x720 canvas the sources are larger than
   the canvas and get cropped. On 2560x1440 they sit in the top left corner.
   Output resolution does not matter, only the canvas.

3. Put the proxy scene on air. On first run the proxy creates a scene called
   "Proxy Scenes" and adds its sources there. It never switches to it.
   Either make "Proxy Scenes" the program scene, or in the scene you already
   use: Add Source > Scene > "Proxy Scenes".

   Skipping this is the usual reason for "the proxy is working but nothing
   shows on stream".

OBS 28 and newer has obs-websocket built in. On 27 or older, install the
obs-websocket 5.x plugin first.


First run
---------
1. Finish the three OBS settings above, with OBS open.
2. Double click start.cmd.
3. Windows Firewall asks twice. Allow both: mediamtx.exe on 1935 is how other
   machines push streams in, stream-proxy.exe on 8080 is the API.
4. Watch the window. You should see it connect, find or create "Proxy Scenes",
   and add a media source and name label per live stream.

Streams are added hidden, then the proxy shows one at a time, rotating every
30 seconds. The first one appears within about 30 seconds of a stream going
live: sources are added on a 7 second cycle, rotation runs on a 30 second one.

The API password is in .env (API_ADMIN_PASSWORD). Open it with

  notepad .env

from a command prompt in this folder. Notepad's Open dialog filters for .txt
and will not show the file otherwise. Change the password before the event.

The .exe is not code signed. If Windows says "Windows protected your PC",
click More info then Run anyway, or run Unblock-File stream-proxy.exe in
PowerShell to clear the download flag.


Turning streams on and off
--------------------------
Every stream the proxy finds is enabled automatically. The API is how you
turn them off, or check what it has seen:

  curl -u admin:PASSWORD http://127.0.0.1:8080/streams

  curl -u admin:PASSWORD -X POST -H "Content-Type: application/json" ^
       -d "{\"enabled\":false}" http://127.0.0.1:8080/streams/cam1/enable

Streamers publish to  rtmp://<this-pc-lan-ip>:1935/<name>
The <name> is what shows up in OBS and in the API.


Leave these alone
-----------------
- Do not rename or delete the sources called media_source-<name> and
  text-<name>. The proxy recreates missing ones, but the fade filters are
  looked up by source name.
- Do not change scene collections during the event. The proxy resolves the
  scene UUID when it connects and keeps using it.
- Each enabled stream is a live decode inside OBS, running whether it is
  visible or not. Ten enabled streams is ten concurrent H.264 decodes. Turn
  off the ones you are not showing.


If OBS is on a different machine
--------------------------------
In .env:
  OBS_WEBSOCKET_ADDRESS=192.168.1.50:4455     (the OBS machine)
  MEDIAMTX_RTMP_ADDRESS=192.168.1.60:1935     (this machine's LAN address)

Do not leave MEDIAMTX_RTMP_ADDRESS at 127.0.0.1 if OBS is elsewhere. The
sources will be created, OBS will fail to open them, and they stay black.
Open port 4455 on the OBS machine.


Starting it automatically
-------------------------
Point Task Scheduler at start.cmd, "Run whether user is logged on or not",
with the start-in folder set to this one. OBS still has to be running for
anything to happen; start OBS first or let the proxy's retry loop wait for it.


Troubleshooting
---------------
"Waiting for OBS" repeating every 5 seconds
  OBS is not reachable. Check OBS is running, the WebSocket server is enabled,
  the port matches OBS_WEBSOCKET_ADDRESS, and the password matches. If OBS is
  on another machine, check its firewall.

Sources appear in OBS but the stream is black
  OBS cannot open the RTMP URL. Check MEDIAMTX_RTMP_ADDRESS is an address the
  OBS machine can reach, and that something is actually publishing to that
  stream name.

Nothing appears on stream but OBS looks right
  The "Proxy Scenes" scene is not on air. See step 3 above.

Only some streams rotate
  Only enabled streams rotate. The rest are in OBS but hidden.

"no supported text source found in OBS"
  OBS has neither the FreeType2 nor the GDI+ text source. This is unusual on
  Windows; a stock OBS install has both.

The proxy exits immediately
  A required value is missing from .env. The log names the variable.
