#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sforum-upgrade-version-test.XXXXXX")"
TEST_ROOT="$TEMP_DIR/repo"
MOCK_BIN="$TEMP_DIR/bin"
MOCK_LOG="$TEMP_DIR/mock.log"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

fail() {
  printf 'upgrade-version-selection_test.sh: %s\n' "$1" >&2
  exit 1
}

assert_contains() {
  local file="$1" expected="$2"
  grep -Fq -- "$expected" "$file" || fail "missing expected text: $expected"
}

assert_not_contains() {
  local file="$1" unexpected="$2"
  if grep -Fq -- "$unexpected" "$file"; then
    fail "found unexpected text: $unexpected"
  fi
}

mkdir -p "$TEST_ROOT/deploy/scripts" "$MOCK_BIN"
cp "$ROOT_DIR/upgrade.sh" "$TEST_ROOT/"
touch \
  "$TEST_ROOT/compose.yaml" \
  "$TEST_ROOT/compose.prod.yaml" \
  "$TEST_ROOT/compose.release.yaml" \
  "$TEST_ROOT/compose.zero-downtime.yaml"
printf '%s\n' 'WEB_PORT=3000' 'API_PORT=18080' > "$TEST_ROOT/.env.production"
printf '%s\n' \
  'lang=en' \
  'version=v3.0.0-alpha.10' \
  'mode=release' \
  'topology=blue-green' \
  'active_slot=blue' \
  'blue_version=v3.0.0-alpha.10' \
  'green_version=v3.0.0-alpha.9' > "$TEST_ROOT/.deployrc"

cat > "$MOCK_BIN/curl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'curl %s\n' "$*" >> "$MOCK_UPGRADE_LOG"
if [ "${MOCK_NO_RELEASE:-}" = "1" ]; then
  printf 'GitHub not found\n' >&2
  exit 22
fi
case " $* " in
  *"/releases/latest"*)
    printf '%s\n' '{"tag_name":"v3.0.0","prerelease":false,"draft":false}'
    ;;
  *)
    printf '%s\n' '[{"tag_name":"v3.0.0-beta.1","draft":false,"prerelease":true}]'
    ;;
esac
EOF

cat > "$MOCK_BIN/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
printf 'docker %s\n' "$*" >> "$MOCK_UPGRADE_LOG"
if [ "${1:-}" = info ]; then
  exit 0
fi
if [ "${1:-}" = pull ]; then
  exit 42
fi
exit 43
EOF
chmod +x "$MOCK_BIN/curl" "$MOCK_BIN/docker" "$TEST_ROOT/upgrade.sh"

run_canceled() {
  local name="$1"
  local input="$2"
  shift
  shift
  local output="$TEMP_DIR/${name}.out"
  : > "$MOCK_LOG"
  (
    cd "$TEST_ROOT"
    printf '%b' "$input" | PATH="$MOCK_BIN:$PATH" MOCK_UPGRADE_LOG="$MOCK_LOG" ./upgrade.sh "$@"
  ) > "$output" 2>&1 || true
  assert_contains "$output" 'Current resolved version: v3.0.0-alpha.10'
  assert_contains "$output" 'Canceled.'
  assert_not_contains "$MOCK_LOG" 'docker pull '
}

run_canceled positional 'n\n' v3.0.0-beta.1
assert_contains "$TEMP_DIR/positional.out" 'Target resolved version: v3.0.0-beta.1'
assert_not_contains "$MOCK_LOG" 'curl '

run_canceled flag 'n\n' --version v3.0.0-alpha.12
assert_contains "$TEMP_DIR/flag.out" 'Target resolved version: v3.0.0-alpha.12'
assert_not_contains "$MOCK_LOG" 'curl '

run_canceled no-prefix 'n\n' 3.0.0-alpha.13
assert_contains "$TEMP_DIR/no-prefix.out" 'Target resolved version: v3.0.0-alpha.13'
assert_not_contains "$MOCK_LOG" 'curl '

run_canceled explicit-prerelease-channel 'n\n' --channel prerelease v3.0.0-alpha.12
assert_contains "$TEMP_DIR/explicit-prerelease-channel.out" 'Target resolved version: v3.0.0-alpha.12'
assert_not_contains "$MOCK_LOG" 'curl '

# Default channel is stable: "latest" must resolve only the stable Release.
run_canceled explicit-latest 'n\n' latest
assert_contains "$TEMP_DIR/explicit-latest.out" 'Target resolved version: v3.0.0'
assert_contains "$MOCK_LOG" '/releases/latest'
assert_not_contains "$MOCK_LOG" 'per_page=1'

# --channel prerelease allows the newest published Release, including prereleases.
run_canceled prerelease-channel 'n\n' --channel prerelease latest
assert_contains "$TEMP_DIR/prerelease-channel.out" 'Target resolved version: v3.0.0-beta.1'
assert_contains "$MOCK_LOG" 'per_page=1'
assert_not_contains "$MOCK_LOG" '/releases/latest'

run_canceled default-latest '\nn\n'
assert_contains "$TEMP_DIR/default-latest.out" 'Target resolved version: v3.0.0'
assert_contains "$MOCK_LOG" '/releases/latest'

# Invalid channel is rejected before any network access.
: > "$MOCK_LOG"
if (
  cd "$TEST_ROOT"
  PATH="$MOCK_BIN:$PATH" MOCK_UPGRADE_LOG="$MOCK_LOG" ./upgrade.sh --channel beta </dev/null
) > "$TEMP_DIR/invalid-channel.out" 2>&1; then
  fail "an invalid channel was accepted"
fi
assert_contains "$TEMP_DIR/invalid-channel.out" '--channel must be stable or prerelease'
assert_not_contains "$MOCK_LOG" 'curl '

# No stable Release available: default resolution fails with a channel hint.
: > "$MOCK_LOG"
if (
  cd "$TEST_ROOT"
  PATH="$MOCK_BIN:$PATH" MOCK_UPGRADE_LOG="$MOCK_LOG" MOCK_NO_RELEASE=1 ./upgrade.sh --yes </dev/null
) > "$TEMP_DIR/no-release.out" 2>&1; then
  fail "a missing stable Release was not detected"
fi
assert_contains "$TEMP_DIR/no-release.out" '--channel prerelease'

# Invalid explicit version is rejected without network access.
: > "$MOCK_LOG"
if (
  cd "$TEST_ROOT"
  PATH="$MOCK_BIN:$PATH" MOCK_UPGRADE_LOG="$MOCK_LOG" ./upgrade.sh not-a-version </dev/null
) > "$TEMP_DIR/invalid-version.out" 2>&1; then
  fail "an invalid explicit version was accepted"
fi
assert_contains "$TEMP_DIR/invalid-version.out" 'Version must look like'
assert_not_contains "$MOCK_LOG" 'curl '

# Unattended mode resolves the stable channel and proceeds to the image pull.
: > "$MOCK_LOG"
if (
  cd "$TEST_ROOT"
  PATH="$MOCK_BIN:$PATH" MOCK_UPGRADE_LOG="$MOCK_LOG" ./upgrade.sh --yes </dev/null
) > "$TEMP_DIR/yes.out" 2>&1; then
  fail "the fake image pull should stop the --yes update"
fi
assert_contains "$TEMP_DIR/yes.out" 'Current resolved version: v3.0.0-alpha.10'
assert_contains "$TEMP_DIR/yes.out" 'Target resolved version: v3.0.0'
assert_contains "$MOCK_LOG" 'docker pull ghcr.io/zhuchunshu/sforum-api:v3.0.0'
assert_contains "$MOCK_LOG" '/releases/latest'

# Unattended prerelease channel resolves the newest release including prereleases.
: > "$MOCK_LOG"
if (
  cd "$TEST_ROOT"
  PATH="$MOCK_BIN:$PATH" MOCK_UPGRADE_LOG="$MOCK_LOG" ./upgrade.sh --yes --channel prerelease </dev/null
) > "$TEMP_DIR/yes-prerelease.out" 2>&1; then
  fail "the fake image pull should stop the --yes prerelease update"
fi
assert_contains "$TEMP_DIR/yes-prerelease.out" 'Target resolved version: v3.0.0-beta.1'
assert_contains "$MOCK_LOG" 'docker pull ghcr.io/zhuchunshu/sforum-api:v3.0.0-beta.1'
assert_contains "$MOCK_LOG" 'per_page=1'

printf 'upgrade-version-selection_test.sh: all checks passed\n'
