#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [ ! -f .env.production ]; then
  echo ".env.production is required."
  exit 1
fi

set -a
. ./.env.production
set +a

BACKUP_DIR="${SFORUM_BACKUP_DIR:-$ROOT_DIR/backups}"
mkdir -p "$BACKUP_DIR"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILE="$BACKUP_DIR/${POSTGRES_DB:-sforum}-$STAMP.sql"
COMPOSE=(docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml)

echo "Creating PostgreSQL backup: $FILE"
"${COMPOSE[@]}" exec -T postgres pg_dump -U "${POSTGRES_USER:-sforum}" -d "${POSTGRES_DB:-sforum}" > "$FILE"
echo "$FILE"
