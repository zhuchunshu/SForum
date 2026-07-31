#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [ ! -f .env.production ]; then
  echo ".env.production is required."
  exit 1
fi
chmod 600 .env.production

set -a
. ./.env.production
set +a

BACKUP_DIR="${SFORUM_BACKUP_DIR:-$ROOT_DIR/backups}"
mkdir -p "$BACKUP_DIR"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILE="$BACKUP_DIR/${POSTGRES_DB:-sforum}-$STAMP.sql"
TEMP_FILE="$(mktemp "$BACKUP_DIR/.${POSTGRES_DB:-sforum}-$STAMP.XXXXXX")"
COMPOSE=(docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml)

cleanup() {
  local status=$?
  if [ -n "${TEMP_FILE:-}" ] && [ -f "$TEMP_FILE" ]; then
    rm -f "$TEMP_FILE"
  fi
  return "$status"
}
trap cleanup EXIT

echo "Creating PostgreSQL backup: $FILE"
"${COMPOSE[@]}" exec -T postgres pg_dump -U "${POSTGRES_USER:-sforum}" -d "${POSTGRES_DB:-sforum}" > "$TEMP_FILE"
if [ ! -s "$TEMP_FILE" ] || [ ! -r "$TEMP_FILE" ]; then
  echo "PostgreSQL backup was not created or is unreadable: $FILE" >&2
  exit 1
fi
mv "$TEMP_FILE" "$FILE"
TEMP_FILE=""
chmod 600 "$FILE"
echo "$FILE"
