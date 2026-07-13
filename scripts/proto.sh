#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_ROOT="$ROOT/contracts/proto"
TOOLS_ROOT="$ROOT/tools/proto"
MODE="${1:-check}"

run_buf() {
  go -C "$TOOLS_ROOT" tool buf --cwd "$PROTO_ROOT" "$@"
}

lint() {
  run_buf lint
}

generate() {
  run_buf generate
}

check() {
  lint
  generate

  local changed
  changed="$(git -C "$ROOT" status --porcelain -- apps/api/sdk/plugin/v2/gen)"
  if [[ -n "$changed" ]]; then
    printf '%s\n' 'generated Protobuf Go SDK is out of date:'
    printf '%s\n' "$changed"
    git -C "$ROOT" diff -- apps/api/sdk/plugin/v2/gen
    return 1
  fi
}

breaking() {
  local against="${2:-.git#branch=main,subdir=contracts/proto}"
  run_buf breaking --against "$against"
}

case "$MODE" in
  lint)
    lint
    ;;
  generate)
    generate
    ;;
  check)
    check
    ;;
  breaking)
    breaking "$@"
    ;;
  *)
    printf 'usage: %s {lint|generate|check|breaking} [against]\n' "$0" >&2
    exit 2
    ;;
esac
