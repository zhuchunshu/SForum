#!/usr/bin/env bash
# 受保护内置插件：为本地 dev 构建 backend 二进制并刷新 V3 digest。
#
# 故意不写回 extensions/builtin/**/sforum.extension.json，避免每次
# api-dev / worker-dev 弄脏 git。构建与 digest 全部落在
# storage/builtin-dev（已在 .gitignore 的 storage/ 下）。
#
# 调用方应 export BUILTIN_EXTENSION_ROOT 指向该 staging 树（api-dev /
# worker-dev 已处理）。CI/Docker 仍在镜像内对源路径 digest --write。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAGING_ROOT="${SFORUM_BUILTIN_DEV_ROOT:-$ROOT_DIR/storage/builtin-dev}"

prepare_staging_tree() {
  mkdir -p "$STAGING_ROOT"
  # 从源树同步主题 + 插件源码；删除 staging 中被排除的旧二进制，防止已移除插件残留空目录。
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete --delete-excluded \
      --exclude 'plugins/*/backend/plugin' \
      --exclude '.DS_Store' \
      --exclude 'air.env' \
      "$ROOT_DIR/extensions/builtin/" "$STAGING_ROOT/"
  else
    # macOS/BSD 兜底：全量复制后删掉旧二进制。
    rm -rf "$STAGING_ROOT"
    mkdir -p "$STAGING_ROOT"
    cp -R "$ROOT_DIR/extensions/builtin/." "$STAGING_ROOT/"
    find "$STAGING_ROOT/plugins" -path '*/backend/plugin' -type f -delete 2>/dev/null || true
  fi
}

build_builtin_plugin() {
  local id="$1"
  local dir="$2"
  echo "Building protected built-in plugin: $id"
  (cd "$dir" && go build -trimpath -buildvcs=false -ldflags="-s -w" -o plugin .)
}

refresh_v3_plugin_digest() {
  local package_dir="$1"
  # 仅在 staging 写 digest；源树 extensions/builtin 保持可提交的稳定摘要。
  (cd "$ROOT_DIR/apps/api" && \
    go run ./cmd/sforum extension digest --write "$package_dir" && \
    go run ./cmd/sforum extension test "$package_dir")
}

write_air_env() {
  # air 从 apps/api 加载 env_files；写入绝对路径，避免 cwd 相对路径漂移。
  printf 'BUILTIN_EXTENSION_ROOT=%s\n' "$STAGING_ROOT" >"$STAGING_ROOT/air.env"
}

main() {
  prepare_staging_tree

  build_builtin_plugin "sforum.smtp" \
    "$STAGING_ROOT/plugins/sforum-smtp/backend"
  refresh_v3_plugin_digest \
    "$STAGING_ROOT/plugins/sforum-smtp"

  build_builtin_plugin "sforum.content-policy" \
    "$STAGING_ROOT/plugins/sforum-content-policy/backend"
  refresh_v3_plugin_digest \
    "$STAGING_ROOT/plugins/sforum-content-policy"

  build_builtin_plugin "sforum.storage-fs" \
    "$STAGING_ROOT/plugins/sforum-storage-fs/backend"
  refresh_v3_plugin_digest \
    "$STAGING_ROOT/plugins/sforum-storage-fs"

  build_builtin_plugin "sforum.storage-s3" \
    "$STAGING_ROOT/plugins/sforum-storage-s3/backend"
  refresh_v3_plugin_digest \
    "$STAGING_ROOT/plugins/sforum-storage-s3"

  build_builtin_plugin "sforum.search-site" \
    "$STAGING_ROOT/plugins/sforum-search-site/backend"
  refresh_v3_plugin_digest \
    "$STAGING_ROOT/plugins/sforum-search-site"

  build_builtin_plugin "sforum.auth-github" \
    "$STAGING_ROOT/plugins/sforum-auth-github/backend"
  refresh_v3_plugin_digest \
    "$STAGING_ROOT/plugins/sforum-auth-github"

  build_builtin_plugin "sforum.web-push" \
    "$STAGING_ROOT/plugins/sforum-web-push/backend"
  refresh_v3_plugin_digest \
    "$STAGING_ROOT/plugins/sforum-web-push"

  write_air_env

  echo "Built-in plugins staged at: $STAGING_ROOT"
  echo "Source tree manifests under extensions/builtin were not modified."
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
