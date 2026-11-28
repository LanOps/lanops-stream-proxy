#!/usr/bin/env bash
# Assembles the Windows bundle: the proxy built for windows/amd64, the pinned
# MediaMTX release, the config and the start script, all in one zip that can be
# unzipped on the OBS machine and run without installing anything else.
#
# Needs go, curl, unzip and zip. Runs in WSL, Git Bash or Linux.
set -euo pipefail

MEDIAMTX_VERSION="${MEDIAMTX_VERSION:-1.14.0}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${REPO_ROOT}/dist/lanops-stream-proxy-windows"
ZIP="${REPO_ROOT}/dist/lanops-stream-proxy-windows.zip"

for bin in go curl unzip zip; do
	if ! command -v "${bin}" >/dev/null 2>&1; then
		echo "missing required command: ${bin}" >&2
		exit 1
	fi
done

rm -rf "${OUT_DIR}" "${ZIP}"
mkdir -p "${OUT_DIR}"

echo ">> building stream-proxy.exe"
(
	cd "${REPO_ROOT}/src"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		go build -ldflags="-s -w" -o "${OUT_DIR}/stream-proxy.exe" ./cmd/stream-proxy
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

# A generated API password beats shipping a known default.
if command -v openssl >/dev/null 2>&1; then
	password="$(openssl rand -hex 8)"
	sed -i "s/^API_ADMIN_PASSWORD=.*/API_ADMIN_PASSWORD=${password}/" "${OUT_DIR}/.env"
fi

echo ">> writing ${ZIP}"
(cd "${OUT_DIR}" && zip -r -q -9 "${ZIP}" .)
echo ">> done: ${ZIP}"
