#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="$ROOT_DIR/.env"
API_DIR="$ROOT_DIR/apps/api"
BIN="$API_DIR/tmp/sforum"

if [ -f "$ENV_FILE" ]; then
  set -a
  . "$ENV_FILE"
  set +a
else
  echo "Missing .env. Run ./scripts/dev.sh once to create it from .env.example."
  exit 1
fi

mkdir -p "$API_DIR/tmp"
cd "$API_DIR"

go build -o "$BIN" ./cmd/sforum
exec "$BIN" "$@"
