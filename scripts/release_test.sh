#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sforum-release-test.XXXXXX")"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

fail() {
  printf 'release_test.sh: %s\n' "$1" >&2
  exit 1
}

expect_failure() {
  local expected="$1"
  shift
  local output
  if output="$("$@" 2>&1)"; then
    fail "command unexpectedly succeeded: $*"
  fi
  [[ "$output" == *"$expected"* ]] || fail "failure output did not contain '$expected': $output"
}

ORIGIN_DIR="$TEMP_DIR/origin.git"
WORK_DIR="$TEMP_DIR/work"

git init --bare "$ORIGIN_DIR" >/dev/null
git init -b main "$WORK_DIR" >/dev/null
git -C "$WORK_DIR" config user.name "SForum Release Test"
git -C "$WORK_DIR" config user.email "release-test@example.invalid"
mkdir -p "$WORK_DIR/scripts"
cp "$ROOT_DIR/scripts/release.sh" "$WORK_DIR/scripts/release.sh"
printf 'release fixture\n' > "$WORK_DIR/README.md"
git -C "$WORK_DIR" add README.md scripts/release.sh
git -C "$WORK_DIR" commit -m "test: initialize release fixture" >/dev/null
git -C "$WORK_DIR" remote add origin "$ORIGIN_DIR"
git -C "$WORK_DIR" push -u origin main >/dev/null
git -C "$WORK_DIR" tag -a v2.7.7 -m "SForum 2.7.7"
git -C "$WORK_DIR" push origin v2.7.7 >/dev/null

ZH_HELP="$(cd "$WORK_DIR" && bash scripts/release.sh --help)"
[[ "$ZH_HELP" == *"SForum 一键发布脚本"* ]] || fail "Chinese help is not the default"

EN_HELP="$(cd "$WORK_DIR" && bash scripts/release.sh --lang en --help)"
[[ "$EN_HELP" == *"SForum release helper"* ]] || fail "English help was not selected"

expect_failure "不支持的语言" bash "$WORK_DIR/scripts/release.sh" --lang fr --help
expect_failure "非交互模式必须指定版本" bash "$WORK_DIR/scripts/release.sh" --non-interactive
expect_failure "发布版本无效" bash "$WORK_DIR/scripts/release.sh" dev-12345 --non-interactive
expect_failure "发布版本无效" bash "$WORK_DIR/scripts/release.sh" 2.8 --non-interactive

INTERACTIVE_DEFAULT_OUTPUT="$(cd "$WORK_DIR" && printf '\n' | bash scripts/release.sh --interactive --lang en --dry-run --skip-checks --no-wait 2>&1)"
[[ "$INTERACTIVE_DEFAULT_OUTPUT" == *"Latest release: v2.7.7"* ]] || fail "interactive mode did not show the latest release"
[[ "$INTERACTIVE_DEFAULT_OUTPUT" == *"Release version [2.7.8]"* ]] || fail "interactive mode did not suggest the next patch version"
[[ "$INTERACTIVE_DEFAULT_OUTPUT" == *"2.7.8"* ]] || fail "interactive mode did not use the default version"
if git -C "$WORK_DIR" show-ref --verify --quiet refs/tags/v2.7.8; then
  fail "interactive dry run created its suggested tag"
fi

INTERACTIVE_OVERRIDE_OUTPUT="$(cd "$WORK_DIR" && printf '2.9.0\n' | bash scripts/release.sh --interactive --lang en --dry-run --skip-checks --no-wait 2>&1)"
[[ "$INTERACTIVE_OVERRIDE_OUTPUT" == *"2.9.0"* ]] || fail "interactive input did not override the suggested version"

git -C "$WORK_DIR" tag -a v3.0.0-beta.1 -m "SForum 3.0.0-beta.1"
git -C "$WORK_DIR" push origin v3.0.0-beta.1 >/dev/null
PRERELEASE_DEFAULT_OUTPUT="$(cd "$WORK_DIR" && printf '\n' | bash scripts/release.sh --interactive --lang en --dry-run --skip-checks --no-wait 2>&1)"
[[ "$PRERELEASE_DEFAULT_OUTPUT" == *"Latest release: v3.0.0-beta.1"* ]] || fail "interactive mode did not select the latest prerelease"
[[ "$PRERELEASE_DEFAULT_OUTPUT" == *"Release version [3.0.0-beta.2]"* ]] || fail "interactive mode did not increment the prerelease number"

DRY_RUN_OUTPUT="$(cd "$WORK_DIR" && bash scripts/release.sh 2.8.0 --non-interactive --skip-checks --no-wait --dry-run 2>&1)"
[[ "$DRY_RUN_OUTPUT" == *"预演完成"* ]] || fail "dry run did not complete"
if git -C "$WORK_DIR" show-ref --verify --quiet refs/tags/v2.8.0; then
  fail "dry run created a local tag"
fi
if git --git-dir="$ORIGIN_DIR" show-ref --verify --quiet refs/tags/v2.8.0; then
  fail "dry run pushed a remote tag"
fi

(cd "$WORK_DIR" && bash scripts/release.sh 2.8.0 --lang en --non-interactive --skip-checks --no-wait >/dev/null)
[[ "$(git -C "$WORK_DIR" cat-file -t v2.8.0)" == "tag" ]] || fail "local release tag is not annotated"
[[ "$(git --git-dir="$ORIGIN_DIR" cat-file -t v2.8.0)" == "tag" ]] || fail "remote release tag was not pushed"

expect_failure "already exists locally" bash "$WORK_DIR/scripts/release.sh" 2.8.0 --lang en --non-interactive --skip-checks --no-wait

git -C "$WORK_DIR" tag -d v2.8.0 >/dev/null
expect_failure "already exists on origin" bash "$WORK_DIR/scripts/release.sh" 2.8.0 --lang en --non-interactive --skip-checks --no-wait

printf 'untracked\n' > "$WORK_DIR/untracked.txt"
expect_failure "working tree is not clean" bash "$WORK_DIR/scripts/release.sh" 2.8.1 --lang en --non-interactive --skip-checks --no-wait
rm -f "$WORK_DIR/untracked.txt"

UPDATER_DIR="$TEMP_DIR/updater"
git clone "$ORIGIN_DIR" "$UPDATER_DIR" >/dev/null 2>&1
git -C "$UPDATER_DIR" switch main >/dev/null 2>&1
git -C "$UPDATER_DIR" config user.name "SForum Release Test"
git -C "$UPDATER_DIR" config user.email "release-test@example.invalid"
printf 'upstream change\n' >> "$UPDATER_DIR/README.md"
git -C "$UPDATER_DIR" add README.md
git -C "$UPDATER_DIR" commit -m "test: advance origin main" >/dev/null
git -C "$UPDATER_DIR" push origin main >/dev/null
expect_failure "does not exactly match origin/main" bash "$WORK_DIR/scripts/release.sh" 2.8.1 --lang en --non-interactive --skip-checks --no-wait

printf 'release_test.sh: all checks passed\n'
