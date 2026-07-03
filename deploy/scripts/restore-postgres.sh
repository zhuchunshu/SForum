#!/usr/bin/env bash
set -euo pipefail

if [ "${SFORUM_CONFIRM_RESTORE:-}" != "RESTORE" ]; then
  echo "Set SFORUM_CONFIRM_RESTORE=RESTORE to run a database restore."
  exit 1
fi

if [ "${1:-}" = "" ]; then
  echo "Usage: deploy/scripts/restore-postgres.sh path/to/backup.sql"
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [ ! -f .env.production ]; then
  echo ".env.production is required."
  exit 1
fi

BACKUP_FILE="$1"
if [ ! -f "$BACKUP_FILE" ]; then
  echo "Backup file not found: $BACKUP_FILE"
  exit 1
fi

set -a
. ./.env.production
set +a

COMPOSE=(docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml)

echo "Restoring PostgreSQL backup: $BACKUP_FILE"
"${COMPOSE[@]}" exec -T postgres psql -U "${POSTGRES_USER:-sforum}" -d "${POSTGRES_DB:-sforum}" < "$BACKUP_FILE"
