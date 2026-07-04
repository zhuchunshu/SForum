#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

usage() {
  cat <<'USAGE'
Usage: ./scripts/dev.sh [options]

Options:
  --build, --rebuild  Rebuild development images before starting.
  --watch             Enable Docker Compose Watch explicitly.
  --print-command     Print the resolved Docker Compose command and exit.
  -h, --help          Show this help message.

Default mode reuses existing containers and images. Source changes reload
through bind mounts, Nuxt/Vite HMR, and Air.
USAGE
}

BUILD_ENABLED=0
WATCH_ENABLED=0
PRINT_COMMAND=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --build | --rebuild)
      BUILD_ENABLED=1
      ;;
    --watch)
      WATCH_ENABLED=1
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

ensure_env_key() {
  local key="$1"
  local value="$2"

  if ! grep -q "^${key}=" .env; then
    printf "\n%s=%s\n" "$key" "$value" >> .env
  fi
}

set_env_key() {
  local key="$1"
  local value="$2"
  local tmp_file

  if grep -Fqx "${key}=${value}" .env; then
    return
  fi

  tmp_file="$(mktemp)"
  awk -v key="$key" -v value="$value" '
    BEGIN { replaced = 0 }
    $0 ~ "^" key "=" {
      if (!replaced) {
        print key "=" value
        replaced = 1
      }
      next
    }
    { print }
    END {
      if (!replaced) {
        print key "=" value
      }
    }
  ' .env > "$tmp_file"
  mv "$tmp_file" .env
}

replace_env_value() {
  local key="$1"
  local old_value="$2"
  local new_value="$3"
  local tmp_file

  if grep -q "^${key}=${old_value}$" .env; then
    tmp_file="$(mktemp)"
    awk -v key="$key" -v old_value="$old_value" -v new_value="$new_value" '
      $0 == key "=" old_value {
        print key "=" new_value
        next
      }
      { print }
    ' .env > "$tmp_file"
    mv "$tmp_file" .env
  fi
}

ensure_env_key APP_LOCALE zh-CN
ensure_env_key SUPPORTED_LOCALES zh-CN,en-US
ensure_env_key APP_URL http://127.0.0.1:3000
replace_env_value APP_URL http://localhost:3000 http://127.0.0.1:3000
set_env_key NUXT_PUBLIC_API_BASE_URL /api/v1
ensure_env_key NUXT_API_INTERNAL_BASE_URL http://api:8080/api/v1

set -a
. ./.env
set +a

if [ "$WATCH_ENABLED" -eq 1 ] && ! docker compose up --help | grep -q -- "--watch"; then
  echo "Docker Compose Watch is not available in this Docker Compose version."
  exit 1
fi

COMPOSE_ARGS=(-f compose.yaml -f compose.dev.yaml up)
if [ "$BUILD_ENABLED" -eq 1 ]; then
  COMPOSE_ARGS+=(--build)
fi
if [ "$WATCH_ENABLED" -eq 1 ]; then
  COMPOSE_ARGS+=(--watch)
fi

echo "Starting SForum development stack..."
if [ "$BUILD_ENABLED" -eq 1 ]; then
  echo "Mode: rebuild development images before starting"
elif [ "$WATCH_ENABLED" -eq 1 ]; then
  echo "Mode: fast start with explicit Docker Compose Watch"
else
  echo "Mode: fast start, no forced rebuild, no Compose Watch"
fi
echo "Web: http://127.0.0.1:${WEB_PORT:-3000}"
echo "Web health: http://127.0.0.1:${WEB_PORT:-3000}/health"
echo "API health via web: http://127.0.0.1:${WEB_PORT:-3000}/api/v1/health"
echo "Internal services stay on the Compose network: api, postgres, redis, meilisearch, mailpit"
echo "Database migrations run before API and worker start."
echo "Use './scripts/dev.sh --build' after Dockerfile or dependency changes."

if [ "$PRINT_COMMAND" -eq 1 ]; then
  printf "docker compose"
  printf " %q" "${COMPOSE_ARGS[@]}"
  printf "\n"
  exit 0
fi

docker compose "${COMPOSE_ARGS[@]}"
