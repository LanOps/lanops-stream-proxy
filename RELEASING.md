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

On every push to `main` and every tag, Drone:

1. lints and tests (`go test -race ./...`)
2. compiles the Windows binary, a fast check that stays cgo-free
3. builds the bundle with `resources/windows/build-bundle.sh`, so a broken
   packaging step fails before a release depends on it

On tags only, it additionally:

4. uploads `dist/lanops-stream-proxy-windows-<tag>.zip` to a **draft** release,
   with a `.sha256` alongside it
5. pushes the Docker image tagged with the version (`auto_tag`)

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

## Required secret

The release step needs a Drone repository secret named `github_token`, a token
with `repo` scope on `LanOps/lanops-stream-proxy`. Without it, tag builds fail at
`release-windows` after the bundle has been built. The existing `dockerhub_*`
and `discord_webhook_url` secrets are unaffected.

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
