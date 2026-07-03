#!/usr/bin/env bash
set -euo pipefail

URL="${1:-}"
TIMEOUT_SECONDS="${2:-60}"

if [ "$URL" = "" ]; then
  echo "Usage: deploy/scripts/wait-for-health.sh URL [timeout_seconds]"
  exit 1
fi

started_at="$(date +%s)"

while true; do
  if curl -fsS "$URL" >/dev/null 2>&1; then
    echo "Healthy: $URL"
    exit 0
  fi

  now="$(date +%s)"
  if [ $((now - started_at)) -ge "$TIMEOUT_SECONDS" ]; then
    echo "Timed out waiting for health: $URL"
    exit 1
  fi

  sleep 2
done
