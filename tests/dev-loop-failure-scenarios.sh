#!/usr/bin/env bash
# M10 行 6 的可复现场景脚本：开发循环在失败与 API 重启下的行为。
#
# 为什么需要脚本而不是单元测试：这些断言依赖真实的构建子进程（bun）、真实的
# digest/validate/test 实现、真实的文件事件与真实的重启时序。单元测试用替身
# runner 覆盖了编排契约（见 cmd/sforum/dev_extension_test.go），这里覆盖的是
# 「真机行为」。
#
# 场景：
#   1. 启动对账：一次成功构建。
#   2. 包失败（清单声明的文件缺失）→ 报告 failed + 保留最后健康摘要，循环存活。
#   3. 恢复（文件回归 + 一次编辑）→ 重新 built。
#   4. 构建失败（作者前端构建脚本非零退出）→ 报告 failed + 保留最后健康摘要。
#   5. 恢复后，重启 API（文档化的注释探针手法）→ 循环存活并在下一次编辑后重建。
#   6. 重命名源文件（delete+create 合并为一次事务）→ 重新 built。
#   7. SIGINT → 干净退出。
#
# 用法：bash tests/dev-loop-failure-scenarios.sh
# 需要：能构建 apps/api（go）、APP_ENV=development、以及运行中的 API（第 5 步会重启它）。
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROBE_ID="acme.row6scenarios"
PROBE_DIR="$REPO/extensions/dev/$PROBE_ID"
BIN="/tmp/sforum-row6scenarios"
LOG="/tmp/row6scenarios.out"
LOOP_PID=""
FAILURES=0

note() { printf '  %s\n' "$*"; }
fail() { printf '  FAIL %s\n' "$*"; FAILURES=$((FAILURES + 1)); }

cleanup() {
  if [ -n "$LOOP_PID" ] && kill -0 "$LOOP_PID" 2>/dev/null; then
    kill -INT "$LOOP_PID" 2>/dev/null || true
    sleep 2
  fi
  rm -rf "$PROBE_DIR" "$BIN" "$LOG" /tmp/row6scenarios-backup
}
trap cleanup EXIT

phases() {
  python3 - "$LOG" <<'PY'
import json, sys
out = []
for line in open(sys.argv[1], encoding='utf-8', errors='replace'):
    if not line.startswith('{'):
        continue
    try:
        payload = json.loads(line)
    except Exception:
        continue
    out.append(payload)
print(json.dumps(out))
PY
}

assert_event() {
  local expression="$1" description="$2"
  python3 - "$LOG" "$expression" "$description" <<'PY' || FAILURES=$((FAILURES + 1))
import json, sys
log, expression, description = sys.argv[1], sys.argv[2], sys.argv[3]
events = []
for line in open(log, encoding='utf-8', errors='replace'):
    if line.startswith('{'):
        try:
            events.append(json.loads(line))
        except Exception:
            pass
namespace = {'events': events, 'count': lambda phase: sum(1 for e in events if e.get('phase') == phase),
             'failed': [e for e in events if e.get('event') == 'failed'],
             'built': [e for e in events if e.get('phase') == 'built']}
if not eval(expression, {}, namespace):
    print(f'  FAIL {description}')
    raise SystemExit(1)
print(f'  ok   {description}')
PY
}

touch_source() {
  python3 - "$PROBE_DIR/backend/plugin" <<'PY'
import pathlib, sys, time
path = pathlib.Path(sys.argv[1])
path.write_text(path.read_text() + f'\n# edit {time.time()}\n')
PY
}

wait_for_phase() {
  local phase="$1" timeout="${2:-20}"
  for _ in $(seq 1 "$((timeout * 2))"); do
    if python3 - "$LOG" "$phase" <<'PY'
import json, sys
log, phase = sys.argv[1], sys.argv[2]
seen = 0
for line in open(log, encoding='utf-8', errors='replace'):
    if line.startswith('{'):
        try:
            payload = json.loads(line)
        except Exception:
            continue
        if payload.get('phase') == phase:
            seen += 1
print('1' if seen > 0 else '0')
PY
    then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

main() {
  echo "dev loop failure scenarios (repo=$REPO)"
  rm -rf "$PROBE_DIR" "$LOG"
  (cd "$REPO/apps/api" && go build -o "$BIN" ./cmd/sforum) || { fail "cannot build the CLI"; return 1; }
  (cd "$REPO/apps/api" && go run ./cmd/sforum make:plugin --no-interaction \
    --id "$PROBE_ID" --name "Row6 Scenarios" --description "dev loop failure scenarios" \
    --url https://example.com/row6 --author-name Acme --backend --out "$PROBE_DIR" >/dev/null) \
    || { fail "cannot scaffold the probe package"; return 1; }

  APP_ENV=development setsid "$BIN" extension dev "extensions/dev/$PROBE_ID" --json --debounce 150ms >"$LOG" 2>&1 &
  LOOP_PID=$!
  sleep 6
  assert_event "count('built') >= 1" "startup reconcile built the package"

  # 1) 包失败：清单声明的文件缺失。
  mv "$PROBE_DIR/backend/plugin" /tmp/row6scenarios-backup
  python3 - "$PROBE_DIR/sforum.extension.json" <<'PY'
import pathlib, sys
path = pathlib.Path(sys.argv[1])
path.write_text(path.read_text().replace('"name": "Row6 Scenarios"', '"name": "Row6 Scenarios edited"'))
PY
  sleep 5
  assert_event "any('no such file' in (e.get('error') or '') for e in failed)" "missing declared package file is reported"
  assert_event "any((e.get('lastHealthyDigest') or '') for e in failed)" "failure keeps the last healthy digest"
  kill -0 "$LOOP_PID" 2>/dev/null && note "loop survived the package failure" || fail "loop died on package failure"

  # 2) 恢复。
  local built_before
  built_before=$(python3 -c "import json,sys;print(sum(1 for l in open('$LOG') if l.startswith('{') and json.loads(l).get('phase')=='built'))")
  mv /tmp/row6scenarios-backup "$PROBE_DIR/backend/plugin"
  touch_source
  sleep 6
  assert_event "count('built') > $built_before" "the package recovers after the file returns"

  # 3) 构建失败：作者前端构建脚本非零退出。
  mkdir -p "$PROBE_DIR/frontend/admin"
  printf '{"name":"row6-admin","private":true,"scripts":{"build":"exit 7"}}\n' > "$PROBE_DIR/frontend/admin/package.json"
  sleep 7
  assert_event "any('exit status 7' in (e.get('error') or '') for e in failed)" "a failing frontend build is reported with its exit status"
  kill -0 "$LOOP_PID" 2>/dev/null && note "loop survived the build failure" || fail "loop died on build failure"

  # 4) 恢复构建脚本，然后重启 API（文档化的注释探针）。
  printf '{"name":"row6-admin","private":true,"scripts":{"build":"node -e \\"require(0x27fs0x27).writeFileSync(0x27dist.mjs0x27,0x27export default {}\\\\n0x27)\\""}}\n' \
    | sed "s/0x27/'/g" > "$PROBE_DIR/frontend/admin/package.json"
  touch_source
  sleep 7
  local built_after_fix
  built_after_fix=$(python3 -c "import json;print(sum(1 for l in open('$LOG') if l.startswith('{') and json.loads(l).get('phase')=='built'))")

  local catalog="$REPO/apps/api/app/Support/Pages/catalog.go"
  cp "$catalog" /tmp/row6scenarios-catalog.bak
  local mtime_before mtime_now
  mtime_before=$(stat -c %Y "$REPO/apps/api/tmp/sforum-api" 2>/dev/null || echo 0)
  echo "// dev loop scenario restart probe $(date +%s)" >> "$catalog"
  for _ in $(seq 1 40); do
    sleep 2
    mtime_now=$(stat -c %Y "$REPO/apps/api/tmp/sforum-api" 2>/dev/null || echo 0)
    [ "$mtime_now" != "$mtime_before" ] && break
  done
  python3 - "$catalog" <<'PY'
import pathlib, sys
path = pathlib.Path(sys.argv[1])
body = path.read_text()
marker = '// dev loop scenario restart probe'
if marker in body:
    kept = [line for line in body.splitlines() if marker not in line]
    path.write_text('\n'.join(kept) + '\n')
PY
  if diff -q "$catalog" /tmp/row6scenarios-catalog.bak >/dev/null; then
    note "catalog.go restored byte-identical after the restart probe"
  else
    fail "restart probe left a diff in catalog.go"
  fi

  touch_source
  sleep 7
  assert_event "count('built') > $built_after_fix" "the loop keeps rebuilding after an API restart"
  kill -0 "$LOOP_PID" 2>/dev/null && note "loop survived the API restart" || fail "loop died on API restart"

  # 4.5) 重命名：delete+create 事件对被合并成一次成功事务（M8 任务 10 的 rename）。
  local built_before_rename
  built_before_rename=$(python3 -c "import json;print(sum(1 for l in open('$LOG') if l.startswith('{') and json.loads(l).get('phase')=='built'))")
  mv "$PROBE_DIR/backend/plugin" "$PROBE_DIR/backend/plugin-renamed"
  python3 -c "
import json, pathlib, sys
path = pathlib.Path(sys.argv[1])
manifest = json.loads(path.read_text())
manifest['backend']['entry'] = 'backend/plugin-renamed'
for entry in manifest.get('packageFiles') or []:
    if entry.get('path') == 'backend/plugin':
        entry['path'] = 'backend/plugin-renamed'
path.write_text(json.dumps(manifest, indent=2, ensure_ascii=False) + chr(10))
" "$PROBE_DIR/sforum.extension.json"
  sleep 7
  assert_event "count('built') > $built_before_rename" "a renamed source input rebuilds the package"

  # 5) SIGINT。
  kill -INT "$LOOP_PID"
  sleep 3
  if kill -0 "$LOOP_PID" 2>/dev/null; then
    fail "loop did not exit on SIGINT"
  else
    note "loop exited on SIGINT"
    assert_event "count('stopped') >= 1" "the loop reports a stopped phase"
  fi
  LOOP_PID=""

  if [ "$FAILURES" -gt 0 ]; then
    echo "dev loop scenarios: $FAILURES failure(s)"
    return 1
  fi
  echo "dev loop scenarios: OK"
}

main "$@"
