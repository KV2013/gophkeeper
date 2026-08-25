#!/usr/bin/env bash
# Запуск сервера GophKeeper.
#
# При необходимости поднимает зависимости (Postgres, MinIO) через docker compose
# и собирает бинарник, если он отсутствует.
#
# Настройки задаются переменными окружения (в скрипте — dev-значения по умолчанию).
set -euo pipefail

cd "$(dirname "$0")"

# --- Настройки сервера (переопределяются переменными окружения) ---
export SERVER_ADDRESS="${SERVER_ADDRESS:-:8080}"
export LOG_LEVEL="${LOG_LEVEL:-info}"
export DATABASE_DSN="${DATABASE_DSN:-postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper?sslmode=disable}"
export JWT_SECRET="${JWT_SECRET:-dev-secret}"
export TOKEN_TTL="${TOKEN_TTL:-1h}"
export ENABLE_HTTPS="${ENABLE_HTTPS:-false}"
export S3_ENDPOINT="${S3_ENDPOINT:-localhost:9000}"
export S3_ACCESS_KEY="${S3_ACCESS_KEY:-minioadmin}"
export S3_SECRET_KEY="${S3_SECRET_KEY:-minioadmin}"
export S3_BUCKET="${S3_BUCKET:-gophkeeper}"
export S3_USE_SSL="${S3_USE_SSL:-false}"

BIN="${BIN:-bin/server}"

# --- Зависимости: Postgres и MinIO ---
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    echo "==> Поднимаем зависимости (Postgres, MinIO)..."
    docker compose up -d db filestorage
else
    echo "==> docker compose недоступен — считаем, что зависимости уже запущены."
fi

# --- Сборка, если бинарника нет ---
if [[ ! -x "$BIN" ]]; then
    echo "==> Бинарник '$BIN' не найден, собираем..."
    ./build_server_and_client.sh
fi

echo "==> Запуск сервера ($BIN) на $SERVER_ADDRESS"
exec "$BIN"
