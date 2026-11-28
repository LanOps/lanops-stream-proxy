# Releasing

The Windows bundle is the artifact people download. It is built by CI from a
tag, attached to a draft release, and published by a person after the notes are
written.

## Versioning

Tags are `vX.Y.Z`. The tag is stamped into the binary and into the bundle name,
so any log or zip can be tied back to a commit:

```
LanOps stream proxy v0.2.0
INF Initializing Stream Proxy version=v0.2.0
```

`VERSION.txt` inside the bundle carries the version, the short commit and the
MediaMTX version that was packaged with it.

## What CI does

Two systems, split by what each can actually do.

**GitHub Actions** (`.github/workflows/windows.yml`) owns everything Windows and
the release, because its `windows-latest` runner can execute the binary:

1. `go vet` and `go test ./...` on Windows
2. builds `stream-proxy.exe` for windows/amd64
3. **runs it**: with nothing on the MediaMTX address it must print its version,
   refuse to start and exit 1. That is the check a Linux container cannot make
4. packages the bundle on Linux with `resources/windows/build-bundle.sh`, so the
   packaging path is exercised on every push to main
5. on tags, attaches the zip and its `.sha256` to a **draft** release

**Drone** keeps what it already did: lint, the Linux test run, and the Docker
image push on main and tags.

Nothing is published automatically. The draft is yours to check, describe and
publish.

## What a person does

1. Push a tag: `git tag v0.2.0 && git push origin v0.2.0`
2. Wait for the Drone build to go green, and for "Windows bundle built, draft
   release ready" in Discord.
3. Open the draft release, confirm the asset is attached and its size is around
   25 MB.
4. Write the notes: what changed, and anything that has to be done on upgrade.
   Keep the "Not verified" caveat if CI still cannot execute the binary.
5. Publish.

## Required settings

GitHub Actions needs no secrets: the draft release uses the workflow's own
`GITHUB_TOKEN` with `contents: write`, declared in the workflow.

Drone still needs its existing `dockerhub_username`, `dockerhub_token` and
`discord_webhook_url` secrets for the image push and notifications. The Windows
workflow is deliberately independent of them, so a missing Drone secret cannot
block a Windows build.

## Building and releasing by hand

If CI is unavailable, the same artifact comes out of:

```bash
VERSION=v0.2.0 ./resources/windows/build-bundle.sh
# dist/lanops-stream-proxy-windows-v0.2.0.zip
```

Then:

```bash
gh release create v0.2.0 dist/lanops-stream-proxy-windows-v0.2.0.zip \
  --repo LanOps/lanops-stream-proxy --draft --title v0.2.0 --notes "..."
```

Publishing a release creates the tag, and a tag fires the Docker build, so the
image and the bundle end up describing the same commit.

## Checking a build someone sends you

Unzip it and read `VERSION.txt`. If that is missing, the bundle predates this
process and the startup line in the proxy window is the only version marker.

## Upgrading people on an older bundle

The database gains columns on first start (`AutoMigrate`), so an old `bridge.db`
keeps working. OBS sources are recreated if their URLs no longer match
`MEDIAMTX_RTMP_ADDRESS`. Nothing needs deleting, and a stream an operator had
disabled stays disabled across a reconnect, because a stream is no longer
deleted when its publisher stops.
