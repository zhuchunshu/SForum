#!/usr/bin/env bash
# Sequential k6 suite for M0 baseline. Requires BASE_URL and seeded hot slug.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

export BASE_URL="${BASE_URL:-http://127.0.0.1:8081}"
export HOT_SLUG="${HOT_SLUG:-perf-hot-thread}"
export CATEGORY_SLUG="${CATEGORY_SLUG:-general}"
# Default LIGHT=1 for Compose PG safety; set LIGHT=0 for full stages on stronger hosts.
export LIGHT="${LIGHT:-1}"

K6_BIN="${K6_BIN:-k6}"
if ! command -v "$K6_BIN" >/dev/null 2>&1; then
  echo "k6 not found; set K6_BIN or install k6 (see tests/perf/README.md)" >&2
  exit 1
fi

echo "BASE_URL=$BASE_URL HOT_SLUG=$HOT_SLUG CATEGORY_SLUG=$CATEGORY_SLUG LIGHT=$LIGHT"
for script in \
  home_topics.js \
  category_topics.js \
  topic_by_slug.js \
  comments_flat.js \
  comments_tree.js \
  mixed_read_write.js \
  view_flood.js \
  deep_scroll.js
do
  echo "======== $script ========"
  "$K6_BIN" run "tests/perf/$script"
done
echo "all scenarios finished"
