#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C
export LANG=C

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sforum-release-assets-test.XXXXXX")"
VERSION="3.2.1-beta.2"
COMMIT="0123456789abcdef0123456789abcdef01234567"
BUILD_TIME="2026-07-30T10:20:30Z"
IMAGE_TAG="sha-$COMMIT"
OUTPUT_DIR="$TEMP_DIR/dist"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

fail() {
  printf 'release_assets_test.sh: %s\n' "$1" >&2
  exit 1
}

expect_failure() {
  local expected="$1"
  shift
  local output=""
  if output="$("$@" 2>&1)"; then
    fail "command unexpectedly succeeded: $*"
  fi
  [[ "$output" == *"$expected"* ]] || fail "failure output did not contain '$expected': $output"
}

mkdir -p "$TEMP_DIR/bin" "$OUTPUT_DIR"
cat > "$TEMP_DIR/bin/go" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  build)
    output=""
    while [[ $# -gt 0 ]]; do
      if [[ "$1" == "-o" ]]; then
        output="$2"
        break
      fi
      shift
    done
    [[ -n "$output" ]]
    mkdir -p "$(dirname "$output")"
    printf 'GOOS=%s\nGOARCH=%s\n' "$GOOS" "$GOARCH" > "$output"
    chmod +x "$output"
    ;;
  version)
    [[ "${2:-}" == "-m" && -f "${3:-}" ]]
    cat "$3"
    ;;
  *)
    exit 2
    ;;
esac
EOF
cat > "$TEMP_DIR/bin/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

command="$1"
shift
case "$command" in
  pull)
    ;;
  create)
    platform=""
    image=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --platform)
          platform="$2"
          shift 2
          ;;
        *)
          image="$1"
          shift
          ;;
      esac
    done
    service="${image##*/}"
    service="${service%%:*}"
    printf 'cid-%s-%s\n' "$service" "${platform#linux/}"
    ;;
  cp)
    source="$1"
    destination="$2"
    container="${source%%:*}"
    source_path="${source#*:}"
    arch="${container##*-}"
    if [[ "$source_path" == "/app/extensions/builtin" ]]; then
      mkdir -p "$destination/builtin/plugins/mock/backend"
      printf 'GOOS=linux\nGOARCH=%s\n' "$arch" > "$destination/builtin/plugins/mock/backend/plugin"
      chmod +x "$destination/builtin/plugins/mock/backend/plugin"
    else
      mkdir -p "$(dirname "$destination")"
      printf 'GOOS=linux\nGOARCH=%s\n' "$arch" > "$destination"
      chmod +x "$destination"
    fi
    ;;
  rm)
    ;;
  *)
    exit 2
    ;;
esac
EOF
chmod +x "$TEMP_DIR/bin/go" "$TEMP_DIR/bin/docker"
export PATH="$TEMP_DIR/bin:$PATH"

for target in \
  "linux amd64" \
  "linux arm64" \
  "darwin amd64" \
  "darwin arm64" \
  "windows amd64" \
  "windows arm64"; do
  read -r target_os target_arch <<< "$target"
  "$ROOT_DIR/scripts/ci/build-release-assets.sh" \
    "$VERSION" "$COMMIT" "$BUILD_TIME" "$IMAGE_TAG" "$target_os" "$target_arch" "$OUTPUT_DIR" >/dev/null
done

"$ROOT_DIR/scripts/ci/finalize-release-assets.sh" "$VERSION" "$OUTPUT_DIR" >/dev/null

[[ "$(find "$OUTPUT_DIR" -maxdepth 1 -type f | wc -l | tr -d ' ')" -eq 9 ]] || fail "unexpected release asset count"
[[ "$(wc -l < "$OUTPUT_DIR/SHA256SUMS" | tr -d ' ')" -eq 8 ]] || fail "unexpected checksum count"
grep -q "sforum-server_${VERSION}_linux_amd64.tar.gz" "$OUTPUT_DIR/SHA256SUMS" || fail "server checksum is missing"
grep -q "sforum-cli_${VERSION}_windows_arm64.zip" "$OUTPUT_DIR/SHA256SUMS" || fail "Windows CLI checksum is missing"

expect_failure "Usage:" "$ROOT_DIR/scripts/ci/build-release-assets.sh" \
  invalid "$COMMIT" "$BUILD_TIME" "$IMAGE_TAG" linux amd64 "$OUTPUT_DIR"

printf 'release_assets_test.sh: all checks passed\n'
