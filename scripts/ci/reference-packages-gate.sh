#!/usr/bin/env bash
# M9 出口门禁：每个参考包都要在**干净临时目录**里过 build/digest/validate/test。
#
# 为什么是「暂存副本」而不是直接跑源码树：仓库里的 fixtures 是 CI 夹具，不是可直接分发的
# 作者包——带后端的包在 go.mod 里声明 `replace … => ../../../../../apps/api`，并且不入库
# `backend/plugin` 二进制（由测试/本脚本现场构建）。因此本门禁的做法是：
#   复制到临时目录 → 把 replace 改写成仓库内 apps/api 的绝对路径 → 构建后端（若有）
#   → digest --write → validate → test。
#
# 诚实的边界：这**不是**真正的 clean-room。真正的 clean-room 需要已发布的 SDK 制品
# （`go.mod` 依赖发布版本而非 replace），属于发布流程（M10 行 9 的 release image）。
# 本脚本把这一区别写在输出里，不冒充 clean-room 通过。
#
# 用法：
#   bash scripts/ci/reference-packages-gate.sh            # 全部门禁
#   SFORUM_REFERENCE_VERBOSE=1 bash scripts/ci/reference-packages-gate.sh
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
API_DIR="$REPO_ROOT/apps/api"
FIXTURE_DIR="$REPO_ROOT/extensions/fixtures/plugins"
WORK_ROOT="$(mktemp -d)"
FAILURES=0
CHECKED=0

cleanup() { rm -rf "$WORK_ROOT"; }
trap cleanup EXIT

note() { printf '  %s\n' "$*"; }
fail() {
  printf '  FAIL %s\n' "$*"
  FAILURES=$((FAILURES + 1))
}

run_author_command() {
  local label="$1" package_root="$2"
  shift 2
  local output
  if ! output="$(cd "$API_DIR" && go run ./cmd/sforum "$@" "$package_root" 2>&1)"; then
    fail "$label"
    printf '%s\n' "$output" | tail -6 | sed 's/^/      /'
    return 1
  fi
  [ "${SFORUM_REFERENCE_VERBOSE:-0}" = "1" ] && printf '%s\n' "$output" | tail -3 | sed 's/^/      /'
  return 0
}

printf 'reference package gate (repo=%s)\n' "$REPO_ROOT"
printf 'note: staged copies rewrite the SDK replace to %s; a true clean room needs published SDK artifacts\n' "$API_DIR"

for source in "$FIXTURE_DIR"/*/; do
  id="$(basename "$source")"
  [ -f "$source/sforum.extension.json" ] || continue
  CHECKED=$((CHECKED + 1))
  target="$WORK_ROOT/$id"
  mkdir -p "$target"
  cp -R "$source/." "$target/"

  # 1) 暂存副本不带构建产物：后端必须现场构建，否则 `digest` 会因缺文件失败。
  rm -f "$target/backend/plugin"
  has_backend=0
  if [ -f "$target/sforum.extension.json" ] && grep -q '"backend"' "$target/sforum.extension.json"; then
    has_backend=1
  fi

  if [ "$has_backend" = "1" ]; then
    if [ -f "$target/backend/go.mod" ]; then
      sed -i "s#\.\./\.\./\.\./\.\./\.\./apps/api#$API_DIR#" "$target/backend/go.mod"
    fi
    if ! build_output="$(cd "$target/backend" && CGO_ENABLED=0 GOWORK=off go build -mod=mod -trimpath -buildvcs=false -o plugin . 2>&1)"; then
      fail "$id: build backend"
      printf '%s\n' "$build_output" | tail -5 | sed 's/^/      /'
      continue
    fi
    note "$id: backend built"
  fi

  ok=1
  run_author_command "$id: extension digest --write" "$target" extension digest --write || ok=0
  [ "$ok" = "1" ] && { run_author_command "$id: extension validate" "$target" extension validate || ok=0; }
  [ "$ok" = "1" ] && { run_author_command "$id: extension test" "$target" extension test || ok=0; }
  if [ "$ok" = "1" ]; then
    note "$id: digest + validate + test passed"
  fi
done

printf 'reference packages checked: %d\n' "$CHECKED"
if [ "$FAILURES" -gt 0 ]; then
  printf 'reference package gate: %d failure(s)\n' "$FAILURES"
  exit 1
fi
printf 'reference package gate: OK (staged in-tree copies, not clean-room)\n'
