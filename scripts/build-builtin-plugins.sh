#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# 受保护内置插件后端：开发启动前构建到 backend/plugin（gitignored）。
build_builtin_plugin() {
  local id="$1"
  local dir="$2"
  echo "Building protected built-in plugin: $id"
  (cd "$dir" && go build -o plugin .)
}

build_builtin_plugin "sforum.smtp" \
  "$ROOT_DIR/extensions/builtin/plugins/sforum-smtp/backend"
build_builtin_plugin "sforum.content-policy" \
  "$ROOT_DIR/extensions/builtin/plugins/sforum-content-policy/backend"
build_builtin_plugin "sforum.storage-fs" \
  "$ROOT_DIR/extensions/builtin/plugins/sforum-storage-fs/backend"
