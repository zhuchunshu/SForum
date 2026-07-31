#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sforum-release-notes-test.XXXXXX")"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

fail() {
  printf 'generate-release-notes_test.sh: %s\n' "$1" >&2
  exit 1
}

git -C "$TEMP_DIR" init -q
git -C "$TEMP_DIR" config user.name SForum
git -C "$TEMP_DIR" config user.email release@sforum.test
printf 'base\n' > "$TEMP_DIR/history.txt"
git -C "$TEMP_DIR" add history.txt
git -C "$TEMP_DIR" commit -qm 'chore: previous release'
git -C "$TEMP_DIR" tag -a v3.0.0-alpha.10 -m 'SForum 3.0.0-alpha.10'
printf 'metrics\n' >> "$TEMP_DIR/history.txt"
git -C "$TEMP_DIR" commit -qam 'fix: restore production metrics'
printf 'upgrade\n' >> "$TEMP_DIR/history.txt"
git -C "$TEMP_DIR" commit -qam 'feat: add zero-downtime updates'
git -C "$TEMP_DIR" tag -a v3.0.0-alpha.11 -m $'SForum 3.0.0-alpha.11\n\n- Production fixes'

(
  cd "$TEMP_DIR"
  GITHUB_REPOSITORY=zhuchunshu/SForum \
    "$ROOT_DIR/scripts/ci/generate-release-notes.sh" v3.0.0-alpha.11 "$TEMP_DIR/notes.md"
)

grep -Fq -- '- Production fixes' "$TEMP_DIR/notes.md" || fail "annotated tag notes are missing"
grep -Fq '> Maintainable, plugin-first open-source forum framework.' "$TEMP_DIR/notes.md" || fail "product summary is missing"
grep -Fq 'docker pull ghcr.io/zhuchunshu/sforum-api:v3.0.0-alpha.11' "$TEMP_DIR/notes.md" || fail "Docker pull instructions are missing"
grep -Fq './deploy.sh --yes --version v3.0.0-alpha.11' "$TEMP_DIR/notes.md" || fail "Compose installation instructions are missing"
grep -Fq './upgrade.sh v3.0.0-alpha.11' "$TEMP_DIR/notes.md" || fail "update instructions are missing"
grep -Fq 'docs/zh-CN/deployment.md' "$TEMP_DIR/notes.md" || fail "deployment documentation is missing"
grep -Fq 'v3.0.0-alpha.10...v3.0.0-alpha.11' "$TEMP_DIR/notes.md" || fail "full changelog comparison is missing"

git -C "$TEMP_DIR" tag -a v3.0.0-alpha.12 -m 'SForum 3.0.0-alpha.12'
(
  cd "$TEMP_DIR"
  GITHUB_REPOSITORY=zhuchunshu/SForum \
    "$ROOT_DIR/scripts/ci/generate-release-notes.sh" v3.0.0-alpha.12 "$TEMP_DIR/automatic.md"
)
grep -Fq -- '- add zero-downtime updates' "$TEMP_DIR/automatic.md" || fail "automatic commit summary is missing"
grep -Fq -- '- restore production metrics' "$TEMP_DIR/automatic.md" || fail "automatic fix summary is missing"
if grep -Eq '^- (feat|fix):' "$TEMP_DIR/automatic.md"; then
  fail "conventional commit prefix was not removed"
fi

printf 'generate-release-notes_test.sh: all checks passed\n'
