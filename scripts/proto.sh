#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_ROOT="$ROOT/contracts/proto"
TOOLS_ROOT="$ROOT/tools/proto"
MODE="${1:-check}"
BIN_ROOT="$TOOLS_ROOT/.bin"
BUF_BIN="$BIN_ROOT/buf"

ensure_tools() {
  if [[ -x "$BUF_BIN" && -x "$BIN_ROOT/protoc-gen-go" && -x "$BIN_ROOT/protoc-gen-go-grpc" ]]; then
    return
  fi
  mkdir -p "$BIN_ROOT"
  go -C "$TOOLS_ROOT" build -o "$BUF_BIN" github.com/bufbuild/buf/cmd/buf
  go -C "$TOOLS_ROOT" build -o "$BIN_ROOT/protoc-gen-go" google.golang.org/protobuf/cmd/protoc-gen-go
  go -C "$TOOLS_ROOT" build -o "$BIN_ROOT/protoc-gen-go-grpc" google.golang.org/grpc/cmd/protoc-gen-go-grpc
}

run_buf() {
  ensure_tools
  (
    cd "$PROTO_ROOT"
    "$BUF_BIN" "$@"
  )
}

lint() {
  run_buf lint
}

generate() {
  ensure_tools
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
  local against="${1:-.git#branch=main,subdir=contracts/proto}"
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
    shift
    breaking "${1:-}"
    ;;
  *)
    printf 'usage: %s {lint|generate|check|breaking} [against]\n' "$0" >&2
    exit 2
    ;;
esac
