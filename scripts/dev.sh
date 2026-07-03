#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is required. Please install Docker first."
  exit 1
fi

if ! docker compose version >/dev/null 2>&1; then
  echo "Docker Compose plugin is required. Please install or update Docker."
  exit 1
fi

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env from .env.example"
fi

if ! grep -q "^APP_LOCALE=" .env; then
  printf "\nAPP_LOCALE=zh-CN\n" >> .env
fi

if ! grep -q "^SUPPORTED_LOCALES=" .env; then
  printf "SUPPORTED_LOCALES=zh-CN,en-US\n" >> .env
fi

set -a
. ./.env
set +a

WATCH_FLAG=()
if docker compose up --help | grep -q -- "--watch"; then
  WATCH_FLAG=(--watch)
fi

echo "Starting SForum development stack..."
echo "Web: http://localhost:${WEB_PORT:-3000}"
echo "API: http://localhost:${API_PORT:-8080}/api/v1/health"
echo "Meilisearch: http://localhost:${MEILI_PORT:-7700}"
echo "PostgreSQL: localhost:${POSTGRES_PORT:-5432}"
echo "Redis: localhost:${REDIS_PORT:-6379}"

docker compose -f compose.yaml -f compose.dev.yaml up --build "${WATCH_FLAG[@]}"
