#!/usr/bin/env bash
# 受保护内置扩展：为本地 dev 构建 backend 二进制并刷新 V3 digest。
#
# 故意不写回 extensions/builtin/**/sforum.extension.json，避免每次
# api-dev / worker-dev 弄脏 git。构建与 digest 全部落在
# storage/builtin-dev（已在 .gitignore 的 storage/ 下）。
#
# 调用方应 export BUILTIN_EXTENSION_ROOT 指向该 staging 树（api-dev /
# worker-dev 已处理）。CI/Docker 仍在镜像内对源路径 digest --write。
#
# 包清单由 `sforum extension builtins` 从清单树发现（M0 ADR 6.2.5）：本脚本
# 不再硬编码任何 id，新增/改名内置包只改包本身，脚本、README 与镜像构建不会漂移。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# 注意：内置包的 go.mod 用相对路径 replace 指向 apps/api，因此覆盖 staging 根时
# 必须保持同样的目录深度（例如 $ROOT_DIR/storage/<name>），否则 go build 解析失败。
STAGING_ROOT="${SFORUM_BUILTIN_DEV_ROOT:-$ROOT_DIR/storage/builtin-dev}"
# 源根在调用时从 ROOT_DIR 推导：测试与调用方通过覆盖 ROOT_DIR 指向临时树。
builtin_source_root() { printf '%s/extensions/builtin' "$ROOT_DIR"; }

prepare_staging_tree() {
  mkdir -p "$STAGING_ROOT"
  # 从源树同步主题 + 插件源码；删除 staging 中被排除的旧二进制，防止已移除插件残留空目录。
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete --delete-excluded \
      --exclude 'plugins/*/backend/plugin' \
      --exclude '.DS_Store' \
      --exclude 'air.env' \
      "$(builtin_source_root)/" "$STAGING_ROOT/"
  else
    # macOS/BSD 兜底：全量复制后删掉旧二进制。
    rm -rf "$STAGING_ROOT"
    mkdir -p "$STAGING_ROOT"
    cp -R "$(builtin_source_root)/." "$STAGING_ROOT/"
    find "$STAGING_ROOT/plugins" -path '*/backend/plugin' -type f -delete 2>/dev/null || true
  fi
}

discover_builtin_packages() {
  (cd "$ROOT_DIR/apps/api" && go run ./cmd/sforum extension builtins --root "$(builtin_source_root)")
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

  local built=0
  local skipped=0
  while IFS=$'\t' read -r id relative_path backend_entry has_go_backend; do
    [ -n "${id:-}" ] || continue
    if [ "${has_go_backend:-false}" != "true" ]; then
      # 主题与纯前端包没有可编译后端：rsync 已同步，无需构建。
      skipped=$((skipped + 1))
      continue
    fi
    build_builtin_plugin "$id" "$STAGING_ROOT/$relative_path/$(dirname "$backend_entry")"
    refresh_v3_plugin_digest "$STAGING_ROOT/$relative_path"
    built=$((built + 1))
  done < <(discover_builtin_packages)

  if [ "$built" -eq 0 ]; then
    echo "No built-in package with a Go backend was discovered under $(builtin_source_root)" >&2
    exit 1
  fi

  write_air_env

  echo "Built-in plugins staged at: $STAGING_ROOT (built=$built skipped=$skipped)"
  echo "Source tree manifests under extensions/builtin were not modified."
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
