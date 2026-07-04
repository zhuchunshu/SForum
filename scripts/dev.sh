#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

usage() {
  cat <<'USAGE'
Usage: ./scripts/dev.sh [options]

Options:
  --build, --rebuild  Rebuild development images before starting.
  --no-migrate        Start dependency services without running migrations.
  --print-command     Print the resolved Docker Compose command and exit.
  -h, --help          Show this help message.

Default mode starts only development dependency services: PostgreSQL, Redis,
Meilisearch, and Mailpit. Run the frontend and backend locally with:

  cd apps/web && bun run dev
  cd apps/api && air
USAGE
}

BUILD_ENABLED=0
MIGRATE_ENABLED=1
PRINT_COMMAND=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --build | --rebuild)
      BUILD_ENABLED=1
      ;;
    --no-migrate)
      MIGRATE_ENABLED=0
      ;;
    --watch | --worker | --with-worker)
      echo "Option $1 is no longer supported by ./scripts/dev.sh."
      echo "Start frontend/backend locally with 'bun run dev' and 'air'."
      exit 1
      ;;
    --print-command)
      PRINT_COMMAND=1
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      usage
      exit 1
      ;;
  esac
  shift
done

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

set -a
. ./.env
set +a

COMPOSE_FILES=(-f compose.yaml -f compose.dev.yaml)
DEPENDENCY_SERVICES=(postgres redis meilisearch mailpit)
UP_ARGS=("${COMPOSE_FILES[@]}" up --remove-orphans --wait)
if [ "$BUILD_ENABLED" -eq 1 ]; then
  UP_ARGS+=(--build)
fi
UP_ARGS+=("${DEPENDENCY_SERVICES[@]}")

MIGRATE_DATABASE_URL="postgres://${POSTGRES_USER:-sforum}:${POSTGRES_PASSWORD:-sforum}@postgres:5432/${POSTGRES_DB:-sforum}?sslmode=disable"
MIGRATE_PRINT_DATABASE_URL="postgres://${POSTGRES_USER:-sforum}:***@postgres:5432/${POSTGRES_DB:-sforum}?sslmode=disable"
MIGRATE_ARGS=("${COMPOSE_FILES[@]}" run --rm --no-deps -e "DATABASE_URL=${MIGRATE_DATABASE_URL}")
MIGRATE_PRINT_ARGS=("${COMPOSE_FILES[@]}" run --rm --no-deps -e "DATABASE_URL=${MIGRATE_PRINT_DATABASE_URL}")
if [ "$BUILD_ENABLED" -eq 1 ]; then
  MIGRATE_ARGS+=(--build)
  MIGRATE_PRINT_ARGS+=(--build)
fi
MIGRATE_ARGS+=(migrate)
MIGRATE_PRINT_ARGS+=(migrate)

echo "Starting SForum development dependencies..."
if [ "$BUILD_ENABLED" -eq 1 ]; then
  echo "Mode: rebuild migration image when needed"
else
  echo "Mode: fast dependency start, no forced rebuild"
fi
echo "Postgres: 127.0.0.1:${POSTGRES_PORT:-15432}"
echo "Redis: 127.0.0.1:${REDIS_PORT:-16379}"
echo "Meilisearch: http://127.0.0.1:${MEILI_PORT:-17700}"
echo "Mailpit: http://127.0.0.1:${MAILPIT_UI_PORT:-18025}"
EXPECTED_NUXT_API_INTERNAL_BASE_URL="http://127.0.0.1:${HTTP_PORT:-8080}/api/v1"
case "${NUXT_API_INTERNAL_BASE_URL:-}" in
  http://127.0.0.1:* | http://localhost:*)
    if [ "${NUXT_API_INTERNAL_BASE_URL}" != "$EXPECTED_NUXT_API_INTERNAL_BASE_URL" ]; then
      echo "Warning: NUXT_API_INTERNAL_BASE_URL=${NUXT_API_INTERNAL_BASE_URL}"
      echo "         but HTTP_PORT=${HTTP_PORT:-8080}; expected ${EXPECTED_NUXT_API_INTERNAL_BASE_URL} for host-run API."
    fi
    ;;
esac
if [ "$MIGRATE_ENABLED" -eq 1 ]; then
  echo "Database migrations run after PostgreSQL is healthy."
else
  echo "Database migrations skipped by --no-migrate."
fi
echo "Then run: cd apps/web && bun run dev"
echo "Then run: cd apps/api && air"
echo "Use './scripts/dev.sh --build' after Dockerfile or dependency changes."

if [ "$PRINT_COMMAND" -eq 1 ]; then
  printf "docker compose"
  printf " %q" "${UP_ARGS[@]}"
  printf "\n"
  if [ "$MIGRATE_ENABLED" -eq 1 ]; then
    printf "docker compose"
    printf " %q" "${MIGRATE_PRINT_ARGS[@]}"
    printf "\n"
  fi
  exit 0
fi

# 旧开发流会留下 Compose 管理的前后端容器；先停掉，避免占用本机端口。
docker compose "${COMPOSE_FILES[@]}" --profile worker stop web api worker >/dev/null 2>&1 || true

docker compose "${UP_ARGS[@]}"

if [ "$MIGRATE_ENABLED" -eq 1 ]; then
  docker compose "${MIGRATE_ARGS[@]}"
fi
