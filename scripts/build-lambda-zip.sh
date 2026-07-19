#!/usr/bin/env bash
# Build pure-Go Lambda zip (no Docker).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "${ROOT}"

ARCH="${LAMBDA_ARCH:-arm64}"
OUT_DIR="${ROOT}/dist"
mkdir -p "${OUT_DIR}"

echo "Building bootstrap (linux/${ARCH})..."
CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" \
	go build -ldflags="-s -w" -trimpath -o "${OUT_DIR}/bootstrap" ./cmd/lambda

rm -f "${OUT_DIR}/function.zip"
( cd "${OUT_DIR}" && zip -q -j function.zip bootstrap )

# Bundle the generated alias list alongside the binary so the gateway can load
# it at cold start (no S3 dependency). The indexer regenerates this and the
# deploy rebuilds the zip.
cp "${ROOT}/aliases.json" "${OUT_DIR}/aliases.json"
( cd "${OUT_DIR}" && zip -q -g function.zip aliases.json )

echo "bootstrap: $(du -h "${OUT_DIR}/bootstrap" | cut -f1)"
echo "zip:       $(du -h "${OUT_DIR}/function.zip" | cut -f1)"
echo "path:      ${OUT_DIR}/function.zip"
