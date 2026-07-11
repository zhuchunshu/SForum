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

if ! command -v air >/dev/null 2>&1; then
  echo "air is required. Install it with: go install github.com/air-verse/air@latest"
  exit 1
fi

# 启动前强制回收本仓库遗留的 sforum-api；docker/其他服务占用则失败，不误杀。
if ! ORPHANS_ONLY=0 "$ROOT_DIR/scripts/free-api-dev-port.sh" "$HTTP_PORT"; then
  echo "API port $HTTP_PORT is already in use by a non-sforum process. Not stopping it."
  exit 1
fi

"$ROOT_DIR/scripts/build-builtin-plugins.sh"

cd "$ROOT_DIR/apps/api"
exec air
