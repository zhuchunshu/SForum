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

if ! command -v air >/dev/null 2>&1; then
  echo "air is required. Install it with: go install github.com/air-verse/air@latest"
  exit 1
fi

"$ROOT_DIR/scripts/build-builtin-plugins.sh"
# 与 api-dev 一致：内置插件从 gitignored staging 加载，不污染 extensions/builtin。
export BUILTIN_EXTENSION_ROOT="${SFORUM_BUILTIN_DEV_ROOT:-$ROOT_DIR/storage/builtin-dev}"

# 主题激活 worker 需要能从 apps/api 找到宿主 Nuxt 应用和 Bun。
export THEME_WEB_ROOT="${THEME_WEB_ROOT:-../web}"
export THEME_BUN_PATH="${THEME_BUN_PATH:-bun}"

cd "$ROOT_DIR/apps/api"
echo "Starting SForum worker with .air.worker.toml. Air builds cmd/worker into sforum-worker."
exec air -c .air.worker.toml
