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

read_production_env_value() {
  local key="$1"
  local default_value="$2"
  local line value=""

  while IFS= read -r line || [ -n "$line" ]; do
    line="${line%$'\r'}"
    case "$line" in
      "$key="*) value="${line#*=}" ;;
    esac
  done < .env.production

  if [ -z "$value" ]; then
    value="$default_value"
  elif [ "${value:0:1}" = '"' ] || [ "${value:0:1}" = "'" ]; then
    if [ "${value: -1}" != "${value:0:1}" ]; then
      echo "Invalid quoted value for $key in .env.production." >&2
      return 1
    fi
    value="${value:1:${#value}-2}"
  fi

  if [ -z "$value" ]; then
    value="$default_value"
  fi

  case "$value" in
    *[!A-Za-z0-9_.-]*)
      echo "Invalid value for $key in .env.production." >&2
      return 1
      ;;
  esac
  printf '%s\n' "$value"
}

POSTGRES_DB="$(read_production_env_value POSTGRES_DB sforum)"
POSTGRES_USER="$(read_production_env_value POSTGRES_USER sforum)"

BACKUP_DIR="${SFORUM_BACKUP_DIR:-$ROOT_DIR/backups}"
mkdir -p "$BACKUP_DIR"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILE="$BACKUP_DIR/$POSTGRES_DB-$STAMP.sql"
TEMP_FILE="$(mktemp "$BACKUP_DIR/.$POSTGRES_DB-$STAMP.XXXXXX")"
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
"${COMPOSE[@]}" exec -T postgres pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" > "$TEMP_FILE"
if [ ! -s "$TEMP_FILE" ] || [ ! -r "$TEMP_FILE" ]; then
  echo "PostgreSQL backup was not created or is unreadable: $FILE" >&2
  exit 1
fi
mv "$TEMP_FILE" "$FILE"
TEMP_FILE=""
chmod 600 "$FILE"
echo "$FILE"
