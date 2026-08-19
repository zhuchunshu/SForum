#!/usr/bin/env bash
set -euo pipefail
umask 077

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
chmod 600 .env.production

BACKUP_FILE="$1"
if [ ! -f "$BACKUP_FILE" ]; then
  echo "Backup file not found: $BACKUP_FILE"
  exit 1
fi

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

COMPOSE=(docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml)
TARGET_DB="$POSTGRES_DB"
POSTGRES_ACTOR="$POSTGRES_USER"

case "$TARGET_DB" in
  ""|postgres|template0|template1)
    echo "Refusing to restore protected PostgreSQL database: ${TARGET_DB:-<empty>}" >&2
    exit 1
    ;;
esac

RESTORE_SUFFIX="$(date -u +%Y%m%d%H%M%S)-$$"
RESTORE_DB="sforum_restore_${RESTORE_SUFFIX}"
PREVIOUS_DB="sforum_before_restore_${RESTORE_SUFFIX}"
SWAPPED=false
APP_SERVICES_TO_RESTART=()

restart_application_services() {
  if [ "${#APP_SERVICES_TO_RESTART[@]}" -gt 0 ]; then
    "${COMPOSE[@]}" start "${APP_SERVICES_TO_RESTART[@]}"
  fi
}

cleanup() {
  local status=$?
  local restart_status=0
  trap - EXIT

  if [ "$SWAPPED" != true ] && [ -n "${RESTORE_DB:-}" ]; then
    "${COMPOSE[@]}" exec -T postgres dropdb --if-exists --force -U "$POSTGRES_ACTOR" "$RESTORE_DB" >/dev/null 2>&1 || true
  fi
  restart_application_services || restart_status=$?
  if [ "$status" -eq 0 ] && [ "$restart_status" -ne 0 ]; then
    status="$restart_status"
  fi
  exit "$status"
}
trap cleanup EXIT

running_services="$("${COMPOSE[@]}" ps --status running --services)"
for service in api; do
  if grep -qx "$service" <<< "$running_services"; then
    APP_SERVICES_TO_RESTART+=("$service")
  fi
done

if [ "${#APP_SERVICES_TO_RESTART[@]}" -gt 0 ]; then
  "${COMPOSE[@]}" stop "${APP_SERVICES_TO_RESTART[@]}"
fi

"${COMPOSE[@]}" up -d postgres
"${COMPOSE[@]}" exec -T postgres pg_isready -U "$POSTGRES_ACTOR" -d "$TARGET_DB"
"${COMPOSE[@]}" exec -T postgres createdb -U "$POSTGRES_ACTOR" --owner "$POSTGRES_ACTOR" --template template0 "$RESTORE_DB"

echo "Restoring PostgreSQL backup: $BACKUP_FILE"
"${COMPOSE[@]}" exec -T postgres psql \
  -X \
  --set=ON_ERROR_STOP=1 \
  --single-transaction \
  -U "$POSTGRES_ACTOR" \
  -d "$RESTORE_DB" < "$BACKUP_FILE"

relation_count="$("${COMPOSE[@]}" exec -T postgres psql \
  -XAt \
  --set=ON_ERROR_STOP=1 \
  -U "$POSTGRES_ACTOR" \
  -d "$RESTORE_DB" \
  -c "SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace WHERE c.relkind IN ('r', 'p') AND n.nspname NOT IN ('pg_catalog', 'information_schema') AND n.nspname !~ '^pg_toast';")"
case "$relation_count" in
  ""|*[!0-9]*)
    echo "Restored database validation returned an invalid relation count: $relation_count" >&2
    exit 1
    ;;
esac
if [ "$relation_count" -eq 0 ]; then
  echo "Restored database validation found no application tables." >&2
  exit 1
fi

# 先在独立数据库完整恢复；只有全部 SQL 与校验通过后，才原子切换数据库名。
"${COMPOSE[@]}" exec -T postgres psql \
  -X \
  --set=ON_ERROR_STOP=1 \
  --single-transaction \
  --set=target_db="$TARGET_DB" \
  --set=restore_db="$RESTORE_DB" \
  --set=previous_db="$PREVIOUS_DB" \
  -U "$POSTGRES_ACTOR" \
  -d postgres <<'SQL'
ALTER DATABASE :"target_db" WITH ALLOW_CONNECTIONS false;
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = :'target_db' AND pid <> pg_backend_pid();
ALTER DATABASE :"target_db" RENAME TO :"previous_db";
ALTER DATABASE :"restore_db" RENAME TO :"target_db";
SQL

SWAPPED=true
RESTORE_DB=""
"${COMPOSE[@]}" exec -T postgres dropdb --if-exists --force -U "$POSTGRES_ACTOR" "$PREVIOUS_DB"
PREVIOUS_DB=""
echo "PostgreSQL restore completed and validated: $TARGET_DB"
