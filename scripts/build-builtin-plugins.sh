#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# 受保护内置插件后端：开发启动前构建到 backend/plugin（gitignored）。
build_builtin_plugin() {
  local id="$1"
  local dir="$2"
  echo "Building protected built-in plugin: $id"
  (cd "$dir" && go build -trimpath -buildvcs=false -o plugin .)
}

refresh_v3_plugin_digest() {
  local package_dir="$1"
  # V3 将 manifest 绑定到平台相关二进制；同步 built-in 前必须刷新摘要并跑宿主契约门禁。
  (cd "$ROOT_DIR/apps/api" && \
    go run ./cmd/sforum extension digest --write "$package_dir" && \
    go run ./cmd/sforum extension test "$package_dir")
}

# 三个受保护内置插件均构建后刷新 V3 packageFiles digest，避免本地产物与 manifest 漂移。
build_builtin_plugin "sforum.smtp" \
  "$ROOT_DIR/extensions/builtin/plugins/sforum-smtp/backend"
refresh_v3_plugin_digest \
  "$ROOT_DIR/extensions/builtin/plugins/sforum-smtp"

build_builtin_plugin "sforum.content-policy" \
  "$ROOT_DIR/extensions/builtin/plugins/sforum-content-policy/backend"
refresh_v3_plugin_digest \
  "$ROOT_DIR/extensions/builtin/plugins/sforum-content-policy"

build_builtin_plugin "sforum.storage-fs" \
  "$ROOT_DIR/extensions/builtin/plugins/sforum-storage-fs/backend"
refresh_v3_plugin_digest \
  "$ROOT_DIR/extensions/builtin/plugins/sforum-storage-fs"
