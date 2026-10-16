#!/usr/bin/env bash
# M8 的「管理端激活自动化」：驱动**受批准的管理端流程**，不做任何信任绕过。
#
# 为什么是这个形状：M0 ADR 明确禁止自动化绕过精确制品信任确认。所以本脚本走的是
# 真实 HTTP 路径——安装（惰性）→ 申请一次性信任挑战 → 带确认 token 启用——并且
# 对**可执行**包必须拿到 token 才继续；对不含可执行制品的包，服务端返回
# extension.trust_not_required，脚本如实接受并继续（这是服务端的正确判定）。
#
# 环境变量：
#   SFORUM_API_BASE        默认 http://127.0.0.1:8080/api/v1
#   SFORUM_PACKAGE_ZIP     必填：要安装的 .sforum.zip
#   SFORUM_SESSION_COOKIE  可选：super_admin 的 sforum_session 值
#   或 SFORUM_ADMIN_EMAIL + SFORUM_ADMIN_PASSWORD（脚本自己登录取会话）
#   SFORUM_UNINSTALL=1     安装+启用成功后卸载并删除包（用于一次性验证）
#
# 缺少凭据或 zip 时打印 SKIP 并退出 0（离线/CI 友好）；SFORUM_REQUIRE_LIVE=1 时改为失败。
set -uo pipefail

API="${SFORUM_API_BASE:-http://127.0.0.1:8080/api/v1}"
ORIGIN="$(printf '%s' "$API" | sed -E 's#(https?://[^/]+).*#\1#')"
ZIP="${SFORUM_PACKAGE_ZIP:-}"
COOKIE_JAR="$(mktemp)"
FAILURES=0
INSTALLED_ID=""

cleanup() { rm -f "$COOKIE_JAR"; }
trap cleanup EXIT

note() { printf '  %s\n' "$*"; }
fail() { printf '  FAIL %s\n' "$*"; FAILURES=$((FAILURES + 1)); }
skip() {
  printf 'SKIP %s\n' "$*"
  [ "${SFORUM_REQUIRE_LIVE:-0}" = "1" ] && exit 1
  exit 0
}

csrf_token() { grep -m1 'csrf_' "$COOKIE_JAR" | awk '{print $7}'; }

api() {
  local method="$1" path="$2" body="${3:-}"
  local args=(-s -o /tmp/activation-step.json -w '%{http_code}' -X "$method"
    -b "$COOKIE_JAR" -c "$COOKIE_JAR" -H "Origin: $ORIGIN" -H "X-CSRF-Token: $(csrf_token)")
  [ -n "$body" ] && args+=(-H 'Content-Type: application/json' -d "$body")
  curl "${args[@]}" "$API$path"
}

step() {
  local label="$1" code="$2" expected="$3"
  local reason
  reason="$(python3 -c "
import json
try:
    d = json.load(open('/tmp/activation-step.json'))
except Exception:
    d = {}
data = d.get('data') or {}
print(d.get('data',{}).get('reason') if isinstance(data, dict) else '')" 2>/dev/null)"
  if [ "$code" = "$expected" ]; then
    note "ok   $label → $code ${reason:+[$reason]}"
  else
    fail "$label → $code (want $expected) ${reason:+[$reason]}"
    head -c 200 /tmp/activation-step.json | sed 's/^/       /'
    printf '\n'
  fi
}

main() {
  echo "admin extension activation (api=$API)"
  [ -n "$ZIP" ] || skip "set SFORUM_PACKAGE_ZIP to a .sforum.zip to run this"
  [ -f "$ZIP" ] || skip "package archive not found: $ZIP"

  # 1) 取得 super_admin 会话（csrf_ cookie + 可选登录）。
  curl -s -o /dev/null -c "$COOKIE_JAR" "$API/health" || skip "api unreachable at $API"
  if [ -n "${SFORUM_SESSION_COOKIE:-}" ]; then
    printf '127.0.0.1\tFALSE\t/\tFALSE\t0\tsforum_session\t%s\n' "$SFORUM_SESSION_COOKIE" >> "$COOKIE_JAR"
    note "ok   session cookie supplied by environment"
  elif [ -n "${SFORUM_ADMIN_EMAIL:-}" ] && [ -n "${SFORUM_ADMIN_PASSWORD:-}" ]; then
    local login_code
    login_code="$(api POST /auth/login "{\"login\":\"$SFORUM_ADMIN_EMAIL\",\"password\":\"$SFORUM_ADMIN_PASSWORD\"}")"
    step "login as $SFORUM_ADMIN_EMAIL" "$login_code" 200
  else
    skip "set SFORUM_SESSION_COOKIE or SFORUM_ADMIN_EMAIL+SFORUM_ADMIN_PASSWORD"
  fi

  # 2) 安装：惰性，绝不执行包代码。
  local code
  code="$(curl -s -o /tmp/activation-step.json -w '%{http_code}' -b "$COOKIE_JAR" -H "Origin: $ORIGIN" \
    -H "X-CSRF-Token: $(csrf_token)" -F "file=@$ZIP;filename=$(basename "$ZIP")" -X POST "$API/admin/extensions")"
  step "install $(basename "$ZIP")" "$code" 201
  INSTALLED_ID="$(python3 -c "
import json
try:
    d = json.load(open('/tmp/activation-step.json'))
except Exception:
    d = {}
print(((d.get('data') or {}).get('extension') or {}).get('id') or '')" 2>/dev/null)"
  [ -n "$INSTALLED_ID" ] || { fail "install response did not name the extension"; return 1; }
  note "installed id: $INSTALLED_ID"
  local status
  status="$(python3 -c "
import json
d=json.load(open('/tmp/activation-step.json'))
print(((d.get('data') or {}).get('extension') or {}).get('status') or '')")"
  [ "$status" = "installed" ] && note "ok   package is inert (status=installed)" || fail "install status=$status (want installed)"

  # 3) 信任挑战：可执行包必须拿到一次性 token；不含可执行制品的包由服务端判定 trust_not_required。
  code="$(api POST "/admin/extensions/$INSTALLED_ID/trust/challenge")"
  local token=""
  if [ "$code" = "200" ]; then
    token="$(python3 -c "
import json
d=json.load(open('/tmp/activation-step.json'))
print((d.get('data') or {}).get('token') or '')")"
    [ -n "$token" ] && note "ok   exact-artifact challenge issued (token ${#token} chars)" \
      || fail "challenge 200 without a token"
  else
    local reason
    reason="$(python3 -c "
import json
d=json.load(open('/tmp/activation-step.json'))
print(((d.get('data') or {}).get('reason')) or '')")"
    if [ "$reason" = "extension.trust_not_required" ]; then
      note "ok   trust_not_required (package ships no executable artifact)"
    else
      fail "trust challenge → $code [$reason]"
    fi
  fi

  # 4) 启用：带 token 或不带 token（后者仅对非可执行包合法）。
  if [ -n "$token" ]; then
    code="$(api POST "/admin/extensions/$INSTALLED_ID/enable" "{\"confirmCapabilities\":true,\"confirmationToken\":\"$token\"}")"
  else
    code="$(api POST "/admin/extensions/$INSTALLED_ID/enable" '{"confirmCapabilities":true}')"
  fi
  step "enable $INSTALLED_ID" "$code" 200
  status="$(python3 -c "
import json
d=json.load(open('/tmp/activation-step.json'))
print((d.get('data') or {}).get('status') or '')" 2>/dev/null)"
  [ "$status" = "enabled" ] && note "ok   extension is enabled" || fail "enable status=$status (want enabled)"

  # 5) 可选：卸载并删除包（一次性验证用）。
  if [ "${SFORUM_UNINSTALL:-0}" = "1" ]; then
    code="$(api POST "/admin/extensions/$INSTALLED_ID/disable" '{"confirmCapabilities":true}')"
    step "disable $INSTALLED_ID" "$code" 200
    code="$(api DELETE "/admin/extensions/$INSTALLED_ID")"
    step "uninstall $INSTALLED_ID" "$code" 200
  fi

  if [ "$FAILURES" -gt 0 ]; then
    echo "admin extension activation: $FAILURES failure(s)"
    return 1
  fi
  echo "admin extension activation: OK"
}

main "$@"
