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

Every variable is required and the process exits at startup if one is missing.

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

## RTMP Stream Generator

In the `resources/tools` dir there is a go script to generate RTMP streams. It needs `ffmpeg` on the PATH.

```bash
go run resources/tools/rtmpStreamGenerator.go
```
