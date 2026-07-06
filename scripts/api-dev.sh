#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"

if [ -f "$ENV_FILE" ]; then
  set -a
  . "$ENV_FILE"
  set +a
else
  echo "Missing .env. Run ./scripts/dev.sh once to create it from .env.example."
  exit 1
fi

HTTP_PORT="${HTTP_PORT:-8080}"

if command -v lsof >/dev/null 2>&1; then
  if lsof -nP -iTCP:"$HTTP_PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "API port $HTTP_PORT is already in use. Not stopping the existing process."
    lsof -nP -iTCP:"$HTTP_PORT" -sTCP:LISTEN
    exit 1
  fi
fi

if ! command -v air >/dev/null 2>&1; then
  echo "air is required. Install it with: go install github.com/air-verse/air@latest"
  exit 1
fi

cd "$ROOT_DIR/apps/api"
exec air
