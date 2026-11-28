# LanOps OBS Stream Proxy

RTMP server & bridge to automatically (if enabled) inject RTMP streams into OBS under a scene as a media source. When rotation is active, it will rotate visibility of each stream.

Intended for use at [LanOps Events](https://www.lanops.co.uk)

Thanks to everyone involved in the following projects:

https://github.com/bluenviron/mediamtx
https://github.com/andreykaipov/goobs

## Prerequisites

- OBS running with the WebSocket server enabled (Tools > WebSocket Server Settings)
- MediaMTX running with the API enabled (docker compose available in `resources/docker/compose`)
- Either Docker, or Go 1.23 to build it yourself

## Running with Docker Compose

The compose file in `resources/docker/compose` starts MediaMTX and the stream proxy together:

```bash
cd resources/docker/compose
cp .env.example .env   # then fill it in
docker compose up -d
```

Both containers use host networking, so the proxy reaches OBS on `127.0.0.1:4455` and MediaMTX on `127.0.0.1:9997` without any port mapping. If OBS runs on a different machine, set `MEDIAMTX_RTMP_ADDRESS` to an address that machine can reach, otherwise OBS will fail to open the media sources.

`./data` next to the compose file holds the SQLite database, so the stream list survives container restarts.

The proxy waits for OBS to appear instead of exiting, and reconnects if OBS restarts. It is safe to start the stack before OBS is open.

## Running locally

```bash
cp src/.env.example src/.env   # fill in the same values as above
cd src
go mod tidy
go run ./cmd/stream-proxy
```

The `.env` is read from the current working directory, so run it from `src`.

## API Endpoints

| Endpoint                | Method | URL Params | JSON Input            | Description                                |
|-------------------------|--------|------------|-----------------------|--------------------------------------------|
| `/streams`              | GET    | None       | None                  | Retrieves a list of all available streams. |
| `/streams/:name`        | GET    | `name`     | None                  | Retrieves details for a specific stream.   |
| `/streams/:name/enable` | POST   | `name`     | `{ "enabled": bool }` | Enables/Disables the specified stream.     |

All endpoints use HTTP basic auth with `API_ADMIN_USERNAME` / `API_ADMIN_PASSWORD`.

## Env

| Variable                 | Description                               |
|--------------------------|-------------------------------------------|
| `DB_PATH`                | Path to the SQLite database file.         |
| `OBS_WEBSOCKET_ADDRESS`  | Address of the OBS WebSocket server.      |
| `OBS_WEBSOCKET_PASSWORD` | Password for authenticating with OBS.     |
| `OBS_PROXY_SCENE_NAME`   | Name of the scene used as a proxy in OBS. |
| `MEDIAMTX_API_ADDRESS`   | Address of the MediaMTX API endpoint.     |
| `MEDIAMTX_RTMP_ADDRESS`  | RTMP address for MediaMTX streaming.      |
| `API_ADMIN_USERNAME`     | Username to access the API.               |
| `API_ADMIN_PASSWORD`     | Password to access the API.               |
| `API_PORT`               | Port to expose the API.                   |

Every variable is required and the process exits at startup if one is missing, except `OBS_WEBSOCKET_PASSWORD`, which can be empty when auth is turned off in OBS.

## Docker

```bash
docker build -f resources/docker/Dockerfile -t lanops-stream-proxy .
```

```bash
docker run -d \
  --name stream-proxy \
  --restart unless-stopped \
  --network host \
  -e DB_PATH=/data/bridge.db \
  -e OBS_WEBSOCKET_ADDRESS=127.0.0.1:4455 \
  -e OBS_WEBSOCKET_PASSWORD= \
  -e OBS_PROXY_SCENE_NAME="Proxy Scenes" \
  -e MEDIAMTX_API_ADDRESS=127.0.0.1:9997 \
  -e MEDIAMTX_RTMP_ADDRESS=127.0.0.1:1935 \
  -e API_ADMIN_USERNAME=admin \
  -e API_ADMIN_PASSWORD=changeme \
  -e API_PORT=8080 \
  -v "$(pwd)/data:/data" \
  lanopsdev/lanops-stream-proxy:latest
```

`--network host` is required for the default `127.0.0.1` addresses to resolve to the host rather than the container.

## Windows

Everything runs on the OBS machine as two native binaries. No Docker, no WSL, no port forwarding. The proxy has no cgo dependencies, so it builds anywhere Go runs and needs no C compiler.

### The bundle

`resources/windows/build-bundle.sh` assembles one zip: the proxy built for windows/amd64, the pinned MediaMTX release, a MediaMTX config with only the API and RTMP enabled, a `.env` with a generated API password, `start.cmd` and `README.txt`.

```bash
./resources/windows/build-bundle.sh
# dist/lanops-stream-proxy-windows.zip
```

It needs `go`, `curl`, `unzip` and `zip`, so run it in WSL, Git Bash or any Linux. Unzip the result on the OBS machine and run `start.cmd`. Nothing else to install.

To build only the proxy:

```powershell
cd src
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-s -w" -o stream-proxy.exe ./cmd/stream-proxy
```

### OBS

The proxy drives OBS over obs-websocket. OBS 28 and newer has it built in; on 27 or older install the obs-websocket 5.x plugin.

1. **Enable the WebSocket server.** Tools > WebSocket Server Settings, tick "Enable WebSocket server", leave the port at 4455. Authentication is optional. If you enable it, put the same password in `OBS_WEBSOCKET_PASSWORD`; if you leave it off, leave that value empty.
2. **Set the canvas to 1920x1080.** Settings > Video > Base (Canvas) Resolution. The proxy scales every media source to fit a 1920x1080 box and places the stream name at 102,47. On a 1280x720 canvas the sources are larger than the canvas and get cropped; on 2560x1440 they sit in the top left corner. Output resolution is irrelevant.
3. **Put the proxy scene on air.** On first run the proxy creates a scene called `Proxy Scenes` and adds its sources there, but it never switches to it. Either make `Proxy Scenes` the program scene, or add it to your own layout with Add Source > Scene > `Proxy Scenes`.

Step 3 is the one that gets missed. Without it the proxy runs correctly and nothing appears on stream.

### Operating notes

- The proxy creates two sources per stream, `media_source-<name>` and `text-<name>`, hidden. Do not rename or delete them. It recreates missing ones, but the fade filters are looked up by name.
- Stay on one scene collection for the event. The scene UUID is resolved when the proxy connects.
- Every enabled stream is a live decode inside OBS, running whether or not it is visible. Ten enabled streams is ten concurrent decodes. Turn off the ones you are not showing with the API.
- Streams are enabled as they appear, so anything pushed to MediaMTX joins the rotation within 30 seconds. The first one becomes visible when the rotation runs; sources are added on a 7 second cycle, rotation on a 30 second one.
- Windows Firewall will ask about `mediamtx.exe` on 1935 and `stream-proxy.exe` on 8080. Allow both. The 1935 rule is what lets other machines push streams.

`resources/windows/README.txt` is the operator version of this, with troubleshooting, and ships inside the bundle.

## RTMP Stream Generator

In the `resources/tools` dir there is a go script to generate RTMP streams. It needs `ffmpeg` on the PATH.

```bash
go run resources/tools/rtmpStreamGenerator.go
```
