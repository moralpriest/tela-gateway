#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TAG="${1:-tela-gateway:local}"
docker build -t "${TAG}" "${ROOT}"
echo "Built ${TAG}"
docker images "${TAG}"
