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

Everything runs on GitHub Actions. There is no Drone pipeline.

**`ci.yml`** — Linux: `golangci-lint` at the version the repo has always used,
`go test -race ./...`, and the Docker image. The image is built on pull requests
for validation and pushed only from `main` and tags, tagged `latest` on `main`
and semver on tags.

**`windows.yml`** — Windows and the release, because only a Windows runner can
execute the binary:

1. `go vet` and `go test ./...` on Windows
2. builds `stream-proxy.exe` for windows/amd64
3. **runs it**: with nothing on the MediaMTX address it must print its version,
   refuse to start and exit 1
4. packages the bundle on Linux with `resources/windows/build-bundle.sh`, so the
   packaging path is exercised on every push to main
5. on tags, attaches the zip and its `.sha256` to a **draft** release

Nothing is published automatically. The draft is yours to check, describe and
publish.

## What a person does

1. Push a tag: `git tag v0.2.0 && git push origin v0.2.0`
2. Wait for the Actions run to go green. The workflow called `windows` builds
   the bundle and creates the draft.
3. Open the draft release, confirm the asset is attached and its size is around
   25 MB.
4. Write the notes: what changed, and anything that has to be done on upgrade.
   Keep the "Not verified" caveat if CI still cannot execute the binary.
5. Publish.

## Required settings

No secret is needed for linting, testing, the Windows build or the draft
release: the release uses the workflow's own `GITHUB_TOKEN` with
`contents: write`.

The Docker image push needs repository secrets `DOCKERHUB_USERNAME` and
`DOCKERHUB_TOKEN`. Until they exist the image job builds the image, skips the
push and shows a warning, so the rest of the pipeline stays green.

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
