#!/usr/bin/env bash
# Сборка сервера и клиента GophKeeper с информацией о сборке (версия/дата/коммит).
#
# Сервер собирается под нативную платформу, клиент — под Linux, Windows и macOS.
#
# Переменные окружения:
#   BUILD_VERSION — версия (по умолчанию из git describe, иначе "dev")
#   BUILD_DATE    — дата сборки (по умолчанию текущее время UTC)
#   BUILD_COMMIT  — коммит (по умолчанию короткий SHA из git, иначе "unknown")
#   OUT_DIR       — каталог для бинарников (по умолчанию "bin")
set -euo pipefail

cd "$(dirname "$0")"

BUILD_VERSION="${BUILD_VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}"
BUILD_DATE="${BUILD_DATE:-$(date -u '+%Y-%m-%dT%H:%M:%SZ')}"
BUILD_COMMIT="${BUILD_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"

OUT_DIR="${OUT_DIR:-bin}"
mkdir -p "$OUT_DIR"

LDFLAGS="-X main.buildVersion=${BUILD_VERSION} -X main.buildDate=${BUILD_DATE} -X main.buildCommit=${BUILD_COMMIT}"

echo "==> Сборка сервера -> ${OUT_DIR}/server"
go build -ldflags "$LDFLAGS" -o "${OUT_DIR}/server" ./cmd/server

# Клиент для целевых платформ (чистый Go, без CGO).
build_client() {
    local goos="$1" goarch="$2" output="$3"
    echo "==> Сборка клиента (${goos}/${goarch}) -> ${output}"
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
        go build -ldflags "$LDFLAGS" -o "$output" ./cmd/client
}

build_client linux   amd64 "${OUT_DIR}/client-linux-amd64"
build_client windows amd64 "${OUT_DIR}/client-windows-amd64.exe"
build_client darwin  amd64 "${OUT_DIR}/client-darwin-amd64"
build_client darwin  arm64 "${OUT_DIR}/client-darwin-arm64"

echo "Готово:"
ls -lh "${OUT_DIR}"/server "${OUT_DIR}"/client-*
