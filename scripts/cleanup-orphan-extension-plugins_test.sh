#!/usr/bin/env bash
# 验证 shell 封装调用 Go DevHygiene 路径，并用 dry-run 做 allow/deny 实机断言。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT_DIR/scripts/cleanup-orphan-extension-plugins.sh"
SCRATCH="${SCRATCH:-}"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

# 结构：脚本必须委托 Go CLI，不得内嵌独立选择实现。
grep -q 'dev:cleanup-orphan-plugins' "$SCRIPT" || fail "shell must invoke dev:cleanup-orphan-plugins"
grep -q 'go run ./cmd/sforum' "$SCRIPT" || fail "shell must fall back to go run ./cmd/sforum"
if grep -q 'is_extension_backend_plugin_cmd' "$SCRIPT"; then
  fail "shell must not reimplement plugin selection predicates"
fi

# CLI 帮助可运行（驱动真实 cobra 注册）
cd "$ROOT_DIR/apps/api"
go run ./cmd/sforum dev:cleanup-orphan-plugins --help >/dev/null

# dry-run 必须成功
out="$(DRY_RUN=1 bash "$SCRIPT" 2>&1)" || fail "dry-run cleanup failed: $out"

# live API 的子进程不得出现在 dry-run 选中列表
api_pid="$(pgrep -f 'tmp/sforum-api' 2>/dev/null | head -1 || true)"
if [ -n "${api_pid:-}" ]; then
  children="$(pgrep -P "$api_pid" 2>/dev/null || true)"
  for child in $children; do
    if echo "$out" | grep -E "pid=${child}([[:space:]]|$)" >/dev/null 2>&1; then
      cmd="$(ps -o command= -p "$child" 2>/dev/null || true)"
      case "$cmd" in
        *backend/plugin*) fail "must not select live API child pid=${child} cmd=${cmd}" ;;
      esac
    fi
  done
fi

# 若存在 PPID=1 的扩展 plugin，dry-run 应选中它们
while IFS= read -r line; do
  [ -z "$line" ] && continue
  pid="$(echo "$line" | awk '{print $1}')"
  cmd="$(echo "$line" | awk '{for(i=4;i<=NF;i++) printf $i" "; print ""}')"
  case "$cmd" in
    *storage/extensions/*backend/plugin*|*\/extensions\/*backend/plugin*)
      echo "$out" | grep -E "pid=${pid}([[:space:]]|$)" >/dev/null 2>&1 \
        || fail "expected orphan pid=${pid} in dry-run output"
      ;;
  esac
done < <(ps -axo pid=,ppid=,rss=,command= | awk '$2==1 {print}')

echo "cleanup-orphan-extension-plugins_test: ok"
if [ -n "$SCRATCH" ]; then
  echo "$out" > "$SCRATCH/orphan-cleanup-dry-run-output.txt"
fi
