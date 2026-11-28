#!/usr/bin/env bash
# Assembles the Windows bundle: the proxy built for windows/amd64, the pinned
# MediaMTX release, the config and the start script, all in one zip that can be
# unzipped on the OBS machine and run without installing anything else.
#
# Needs go, curl, unzip and zip. Runs in WSL, Git Bash or Linux.
set -euo pipefail

MEDIAMTX_VERSION="${MEDIAMTX_VERSION:-1.14.0}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# The version is stamped into the binary and the bundle name, so a report of
# "it does not work" can be tied to an exact build. CI passes the tag; locally
# it falls back to git describe.
VERSION="${VERSION:-$(git -C "${REPO_ROOT}" describe --tags --always --dirty 2>/dev/null || echo dev)}"
COMMIT="$(git -C "${REPO_ROOT}" rev-parse --short HEAD 2>/dev/null || echo unknown)"

OUT_DIR="${REPO_ROOT}/dist/lanops-stream-proxy-windows"
ZIP="${REPO_ROOT}/dist/lanops-stream-proxy-windows-${VERSION}.zip"

for bin in go curl unzip zip; do
	if ! command -v "${bin}" >/dev/null 2>&1; then
		echo "missing required command: ${bin}" >&2
		exit 1
	fi
done

rm -rf "${OUT_DIR}" "${ZIP}"
mkdir -p "${OUT_DIR}"

echo ">> building stream-proxy.exe ${VERSION} (${COMMIT})"
(
	cd "${REPO_ROOT}/src"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build -ldflags="-s -w -X main.version=${VERSION}" \
		-o "${OUT_DIR}/stream-proxy.exe" ./cmd/stream-proxy
)

echo ">> fetching MediaMTX ${MEDIAMTX_VERSION}"
work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT
curl -fsSL -o "${work}/mediamtx.zip" \
	"https://github.com/bluenviron/mediamtx/releases/download/v${MEDIAMTX_VERSION}/mediamtx_v${MEDIAMTX_VERSION}_windows_amd64.zip"
unzip -q -o "${work}/mediamtx.zip" mediamtx.exe LICENSE -d "${work}"

mv "${work}/mediamtx.exe" "${OUT_DIR}/mediamtx.exe"
mv "${work}/LICENSE" "${OUT_DIR}/MEDIAMTX-LICENSE.txt"

cp "${REPO_ROOT}/resources/windows/mediamtx.yml" \
	"${REPO_ROOT}/resources/windows/start.cmd" \
	"${REPO_ROOT}/resources/windows/README.md" \
	"${OUT_DIR}/"
cp "${REPO_ROOT}/resources/windows/.env.example" "${OUT_DIR}/.env"

cat > "${OUT_DIR}/VERSION.txt" <<EOF
${VERSION}
commit ${COMMIT}
MediaMTX ${MEDIAMTX_VERSION}
EOF

# A generated API password beats shipping a known default.
if command -v openssl >/dev/null 2>&1; then
	password="$(openssl rand -hex 8)"
	sed -i "s/^API_ADMIN_PASSWORD=.*/API_ADMIN_PASSWORD=${password}/" "${OUT_DIR}/.env"
fi

echo ">> writing ${ZIP}"
(cd "${OUT_DIR}" && zip -r -q -9 "${ZIP}" .)

# A checksum next to the zip, so a download can be verified without trusting the
# page it came from.
if command -v sha256sum >/dev/null 2>&1; then
	(cd "$(dirname "${ZIP}")" && sha256sum "$(basename "${ZIP}")" > "$(basename "${ZIP}").sha256")
elif command -v shasum >/dev/null 2>&1; then
	(cd "$(dirname "${ZIP}")" && shasum -a 256 "$(basename "${ZIP}")" > "$(basename "${ZIP}").sha256")
fi

echo ">> done: ${ZIP}"
