#!/usr/bin/env bash
# 释放本仓库 air 产出的 sforum-api 对开发端口的占用。
#
# 默认（ORPHANS_ONLY=1，供 air pre_cmd 使用）：
#   只杀「父进程不是 air」的孤儿 sforum-api，避免热重载时提前杀掉当前实例。
# 强制（ORPHANS_ONLY=0，供 api-dev.sh 启动前使用）：
#   杀掉端口上所有 sforum-api，便于重新拉起。
#
# 不会杀 docker / 其他服务；未知进程占用时返回非 0 并打印 lsof。
set -euo pipefail

PORT="${1:-}"
if [ -z "$PORT" ]; then
  PORT="${HTTP_PORT:-8080}"
fi

ORPHANS_ONLY="${ORPHANS_ONLY:-1}"

if ! [[ "$PORT" =~ ^[0-9]+$ ]] || [ "$PORT" -lt 1 ] || [ "$PORT" -gt 65535 ]; then
  echo "free-api-dev-port: invalid port: ${PORT}" >&2
  exit 2
fi

if ! command -v lsof >/dev/null 2>&1; then
  echo "free-api-dev-port: lsof is required" >&2
  exit 1
fi

is_sforum_api_cmd() {
  case "$1" in
    *sforum-api*) return 0 ;;
    *) return 1 ;;
  esac
}

# 父进程命令行是否像 air 热重载宿主（避免误伤其它同名工具）。
parent_is_air() {
  local pid="$1"
  local ppid parent_cmd
  ppid="$(ps -o ppid= -p "$pid" 2>/dev/null | tr -d ' ')"
  if [ -z "${ppid:-}" ] || [ "$ppid" = "0" ] || [ "$ppid" = "1" ]; then
    return 1
  fi
  parent_cmd="$(ps -o command= -p "$ppid" 2>/dev/null || true)"
  case "$parent_cmd" in
    *[[:space:]]air|*/air|air|air\ *|*air\ -c*|*air\ -*) return 0 ;;
  esac
  # macOS 上 air 可能显示为完整路径末尾 /air
  case "$parent_cmd" in
    */air|*/air\ *) return 0 ;;
  esac
  return 1
}

collect_listeners() {
  lsof -nP -iTCP:"$PORT" -sTCP:LISTEN -F pc 2>/dev/null | awk '
    /^p/ { pid = substr($0, 2); next }
    /^c/ {
      cmd = substr($0, 2)
      if (pid != "") print pid "\t" cmd
    }
  '
}

stop_pid() {
  local pid="$1"
  local cmd="$2"
  if ! kill -0 "$pid" 2>/dev/null; then
    return 0
  fi
  echo "free-api-dev-port: releasing ${cmd} (pid ${pid}) on :${PORT}"
  kill -INT "$pid" 2>/dev/null || true
  for _ in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
    if ! kill -0 "$pid" 2>/dev/null; then
      return 0
    fi
    sleep 0.1
  done
  if kill -0 "$pid" 2>/dev/null; then
    kill -KILL "$pid" 2>/dev/null || true
    sleep 0.05
  fi
}

listeners="$(collect_listeners || true)"
if [ -z "$listeners" ]; then
  exit 0
fi

killed_any=0
foreign=0
kept_managed=0

while IFS=$'\t' read -r pid cmd; do
  [ -z "${pid:-}" ] && continue

  if ! is_sforum_api_cmd "$cmd"; then
    foreign=1
    echo "free-api-dev-port: port ${PORT} held by non-sforum process: pid=${pid} cmd=${cmd}" >&2
    continue
  fi

  if [ "$ORPHANS_ONLY" = "1" ] && parent_is_air "$pid"; then
    # 仍由当前/其他 air 托管：热重载交给 kill_delay + Listen 重试，不要提前杀。
    kept_managed=1
    continue
  fi

  stop_pid "$pid" "$cmd"
  killed_any=1
done <<< "$listeners"

still="$(collect_listeners || true)"
if [ -n "$still" ]; then
  # 强制模式：再清一轮残留 sforum-api
  if [ "$ORPHANS_ONLY" != "1" ]; then
    while IFS=$'\t' read -r pid cmd; do
      [ -z "${pid:-}" ] && continue
      if is_sforum_api_cmd "$cmd"; then
        stop_pid "$pid" "$cmd"
        killed_any=1
      else
        foreign=1
      fi
    done <<< "$still"
    still="$(collect_listeners || true)"
  fi
fi

if [ -n "$still" ] && [ "$foreign" -eq 1 ]; then
  echo "free-api-dev-port: refusing to free port ${PORT} (foreign listener remains)" >&2
  lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >&2 || true
  exit 1
fi

# 孤儿模式且仍有 air 托管实例时，端口仍占用是预期（交给 Listen 重试）。
if [ -n "$still" ] && [ "$ORPHANS_ONLY" = "1" ] && [ "$kept_managed" -eq 1 ]; then
  exit 0
fi

if [ -n "$still" ] && [ "$ORPHANS_ONLY" != "1" ]; then
  echo "free-api-dev-port: port ${PORT} still busy after cleanup" >&2
  lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >&2 || true
  exit 1
fi

if [ "$killed_any" -eq 1 ]; then
  sleep 0.15
fi

exit 0
