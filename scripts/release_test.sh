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

git init --bare "$ORIGIN_DIR" >/dev/null 2>&1
git init -b main "$WORK_DIR" >/dev/null 2>&1
git -C "$WORK_DIR" config user.name "SForum Release Test"
git -C "$WORK_DIR" config user.email "release-test@example.invalid"
mkdir -p "$WORK_DIR/scripts"
cp "$ROOT_DIR/scripts/release.sh" "$WORK_DIR/scripts/release.sh"
printf 'release fixture\n' > "$WORK_DIR/README.md"
git -C "$WORK_DIR" add README.md scripts/release.sh
git -C "$WORK_DIR" commit -m "test: initialize release fixture" >/dev/null 2>&1
git -C "$WORK_DIR" remote add origin "$ORIGIN_DIR"
git -C "$WORK_DIR" push -u origin main >/dev/null 2>&1
git -C "$WORK_DIR" tag -a v2.7.7 -m "SForum 2.7.7"
git -C "$WORK_DIR" push origin v2.7.7 >/dev/null 2>&1

ZH_HELP="$(cd "$WORK_DIR" && bash scripts/release.sh --help)"
[[ "$ZH_HELP" == *"SForum 一键发布脚本"* ]] || fail "Chinese help is not the default"
[[ "$ZH_HELP" == *"--no-wait"*"默认"* ]] || fail "Chinese help does not describe asynchronous release as the default"
[[ "$ZH_HELP" == *"--notes-file"*"自动"* ]] || fail "Chinese help does not describe hybrid release notes"

EN_HELP="$(cd "$WORK_DIR" && bash scripts/release.sh --lang en --help)"
[[ "$EN_HELP" == *"SForum release helper"* ]] || fail "English help was not selected"
[[ "$EN_HELP" == *"--no-wait"*"default"* ]] || fail "English help does not describe asynchronous release as the default"
[[ "$EN_HELP" == *"--notes-file"*"generated"* ]] || fail "English help does not describe hybrid release notes"

expect_failure "不支持的语言" bash "$WORK_DIR/scripts/release.sh" --lang fr --help
expect_failure "非交互模式必须指定版本" bash "$WORK_DIR/scripts/release.sh" --non-interactive
expect_failure "发布版本无效" bash "$WORK_DIR/scripts/release.sh" dev-12345 --non-interactive
expect_failure "发布版本无效" bash "$WORK_DIR/scripts/release.sh" 2.8 --non-interactive
expect_failure "与版本 3.0.0-beta.1 不匹配" bash "$WORK_DIR/scripts/release.sh" 3.0.0-beta.1 --type stable --non-interactive
expect_failure "只能使用其中一个" bash "$WORK_DIR/scripts/release.sh" 2.8.0 --notes one --notes-file "$WORK_DIR/README.md" --non-interactive

INTERACTIVE_DEFAULT_OUTPUT="$(cd "$WORK_DIR" && printf '3\n\n\n' | bash scripts/release.sh --interactive --lang en --dry-run --no-wait 2>&1)"
[[ "$INTERACTIVE_DEFAULT_OUTPUT" == *"Select the release type:"* ]] || fail "interactive mode did not ask for the release type"
[[ "$INTERACTIVE_DEFAULT_OUTPUT" == *"Latest release: v2.7.7"* ]] || fail "interactive mode did not show the latest release"
[[ "$INTERACTIVE_DEFAULT_OUTPUT" == *"Base version [2.7.8]"* ]] || fail "interactive mode did not suggest the next patch version"
[[ "$INTERACTIVE_DEFAULT_OUTPUT" == *"Release gate:"*"GitHub Actions"* ]] || fail "GitHub Actions is not the default release gate"
[[ "$INTERACTIVE_DEFAULT_OUTPUT" == *"Release highlights:"*"generated automatically"* ]] || fail "interactive empty notes did not retain generated notes"
[[ "$INTERACTIVE_DEFAULT_OUTPUT" == *"2.7.8"* ]] || fail "interactive mode did not use the default version"
if git -C "$WORK_DIR" show-ref --verify --quiet refs/tags/v2.7.8; then
  fail "interactive dry run created its suggested tag"
fi

INTERACTIVE_OVERRIDE_OUTPUT="$(cd "$WORK_DIR" && printf '3\n2.9.0\nA concise operator-facing highlight\n' | bash scripts/release.sh --interactive --lang en --dry-run --no-wait 2>&1)"
[[ "$INTERACTIVE_OVERRIDE_OUTPUT" == *"2.9.0"* ]] || fail "interactive input did not override the suggested version"
[[ "$INTERACTIVE_OVERRIDE_OUTPUT" == *"A concise operator-facing highlight"* ]] || fail "interactive release highlight was not shown"

git -C "$WORK_DIR" tag -a v3.0.0-beta.1 -m "SForum 3.0.0-beta.1"
git -C "$WORK_DIR" push origin v3.0.0-beta.1 >/dev/null 2>&1
PRERELEASE_DEFAULT_OUTPUT="$(cd "$WORK_DIR" && printf '2\n\n\n\n' | bash scripts/release.sh --interactive --lang en --dry-run --no-wait 2>&1)"
[[ "$PRERELEASE_DEFAULT_OUTPUT" == *"Latest release: v3.0.0-beta.1"* ]] || fail "interactive mode did not select the latest prerelease"
[[ "$PRERELEASE_DEFAULT_OUTPUT" == *"Base version [3.0.0]"* ]] || fail "interactive mode did not retain the prerelease base version"
[[ "$PRERELEASE_DEFAULT_OUTPUT" == *"beta prerelease number [2]"* ]] || fail "interactive mode did not increment the prerelease number"
[[ "$PRERELEASE_DEFAULT_OUTPUT" == *"3.0.0-beta.2"* ]] || fail "interactive mode did not assemble the beta version"

DRY_RUN_OUTPUT="$(cd "$WORK_DIR" && bash scripts/release.sh 2.8.0 --non-interactive --no-wait --dry-run 2>&1)"
[[ "$DRY_RUN_OUTPUT" == *"预演完成"* ]] || fail "dry run did not complete"
if git -C "$WORK_DIR" show-ref --verify --quiet refs/tags/v2.8.0; then
  fail "dry run created a local tag"
fi
if git --git-dir="$ORIGIN_DIR" show-ref --verify --quiet refs/tags/v2.8.0; then
  fail "dry run pushed a remote tag"
fi

NOTES_FILE="$TEMP_DIR/release-notes.md"
printf '## Highlights\n\n- Safer upgrades\n- Faster releases\n' > "$NOTES_FILE"
NOTES_FILE_OUTPUT="$(cd "$WORK_DIR" && bash scripts/release.sh 2.8.0 --lang en --non-interactive --dry-run --notes-file "$NOTES_FILE" 2>&1)"
[[ "$NOTES_FILE_OUTPUT" == *"## Highlights"*"- Safer upgrades"*"- Faster releases"* ]] || fail "multi-line release notes file was not shown"

DEFAULT_ASYNC_OUTPUT="$(cd "$WORK_DIR" && bash scripts/release.sh 2.8.0 --lang en --interactive --yes --notes "Alpha release highlights" 2>&1)"
[[ "$DEFAULT_ASYNC_OUTPUT" == *"Release continues in GitHub Actions"* ]] || fail "interactive release did not return asynchronously by default"
[[ "$DEFAULT_ASYNC_OUTPUT" != *"cannot wait for the workflow"* ]] || fail "interactive release unexpectedly entered wait mode"
[[ "$(git -C "$WORK_DIR" cat-file -t v2.8.0)" == "tag" ]] || fail "local release tag is not annotated"
[[ "$(git --git-dir="$ORIGIN_DIR" cat-file -t v2.8.0)" == "tag" ]] || fail "remote release tag was not pushed"
TAG_BODY="$(git -C "$WORK_DIR" for-each-ref --format='%(contents:body)' refs/tags/v2.8.0)"
[[ "$TAG_BODY" == "Alpha release highlights" ]] || fail "release highlights were not stored in the annotated tag"

expect_failure "already exists locally" bash "$WORK_DIR/scripts/release.sh" 2.8.0 --lang en --non-interactive --no-wait

git -C "$WORK_DIR" tag -d v2.8.0 >/dev/null 2>&1
expect_failure "already exists on origin" bash "$WORK_DIR/scripts/release.sh" 2.8.0 --lang en --non-interactive --no-wait

printf 'untracked\n' > "$WORK_DIR/untracked.txt"
expect_failure "working tree is not clean" bash "$WORK_DIR/scripts/release.sh" 2.8.1 --lang en --non-interactive --no-wait
rm -f "$WORK_DIR/untracked.txt"

UPDATER_DIR="$TEMP_DIR/updater"
git clone "$ORIGIN_DIR" "$UPDATER_DIR" >/dev/null 2>&1
git -C "$UPDATER_DIR" switch main >/dev/null 2>&1
git -C "$UPDATER_DIR" config user.name "SForum Release Test"
git -C "$UPDATER_DIR" config user.email "release-test@example.invalid"
printf 'upstream change\n' >> "$UPDATER_DIR/README.md"
git -C "$UPDATER_DIR" add README.md
git -C "$UPDATER_DIR" commit -m "test: advance origin main" >/dev/null 2>&1
git -C "$UPDATER_DIR" push origin main >/dev/null 2>&1
expect_failure "does not exactly match origin/main" bash "$WORK_DIR/scripts/release.sh" 2.8.1 --lang en --non-interactive --no-wait

printf 'release_test.sh: all checks passed\n'
