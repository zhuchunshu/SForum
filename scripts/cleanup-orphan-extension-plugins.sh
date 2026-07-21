#!/usr/bin/env bash
# 清理 air/热重载后被 init 收养的 SForum 扩展 backend plugin 孤儿进程。
#
# 选择与终止逻辑唯一实现：
#   apps/api/app/Support/DevHygiene (CleanupOrphanExtensionPlugins)
# 本脚本只是薄封装，避免 shell 与 Go 规则漂移。
#
# 用法：
#   ./scripts/cleanup-orphan-extension-plugins.sh
#   DRY_RUN=1 ./scripts/cleanup-orphan-extension-plugins.sh
#
# 注意：air pre_cmd 在「旧 API 被杀之前」执行，只能清掉上一轮遗留；
# 开发 API 进程内 reaper 会在 kill_delay 之后再扫（见 bootstrap/orphan_plugin_reaper.go）。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_DIR="$ROOT_DIR/apps/api"

args=()
if [ "${DRY_RUN:-0}" = "1" ]; then
  args+=(--dry-run)
fi

cd "$API_DIR"

# 优先已编译的 sforum CLI（若有），否则 go run 同一命令入口。
if [ -x "$API_DIR/tmp/sforum" ]; then
  exec "$API_DIR/tmp/sforum" dev:cleanup-orphan-plugins "${args[@]+"${args[@]}"}"
fi

exec go run ./cmd/sforum dev:cleanup-orphan-plugins "${args[@]+"${args[@]}"}"
