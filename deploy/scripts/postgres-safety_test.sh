#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/sforum-postgres-safety-test.XXXXXX")"
TEST_ROOT="$TEMP_DIR/repo"
MOCK_BIN="$TEMP_DIR/bin"
MOCK_LOG="$TEMP_DIR/docker.log"

cleanup() {
  rm -rf "$TEMP_DIR"
}
trap cleanup EXIT

fail() {
  printf 'postgres-safety_test.sh: %s\n' "$1" >&2
  exit 1
}

file_mode() {
  if stat -f '%Lp' "$1" >/dev/null 2>&1; then
    stat -f '%Lp' "$1"
  else
    stat -c '%a' "$1"
  fi
}

mkdir -p "$TEST_ROOT/deploy/scripts" "$MOCK_BIN"
cp "$ROOT_DIR/deploy/scripts/backup-postgres.sh" "$TEST_ROOT/deploy/scripts/"
cp "$ROOT_DIR/deploy/scripts/restore-postgres.sh" "$TEST_ROOT/deploy/scripts/"
touch "$TEST_ROOT/compose.yaml" "$TEST_ROOT/compose.prod.yaml"
printf '%s\n' \
  'POSTGRES_DB=sforum_test' \
  'POSTGRES_USER=sforum_test' \
  "UNTRUSTED_VALUE=\$(touch '$TEMP_DIR/env-command-was-executed')" > "$TEST_ROOT/.env.production"
chmod 644 "$TEST_ROOT/.env.production"

cat > "$MOCK_BIN/docker" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

[[ "${1:-}" == "compose" ]] || exit 90
shift
while [[ $# -gt 0 ]]; do
  case "$1" in
    --env-file|-f|--project-name)
      shift 2
      ;;
    *)
      break
      ;;
  esac
done

command="${1:-}"
shift || true
case "$command" in
  ps)
    printf 'compose ps\n' >> "$MOCK_DOCKER_LOG"
    printf 'api\nworker\n'
    ;;
  stop|start|up)
    printf 'compose %s %s\n' "$command" "$*" >> "$MOCK_DOCKER_LOG"
    ;;
  exec)
    [[ "${1:-}" == "-T" ]] && shift
    service="${1:-}"
    shift
    [[ "$service" == "postgres" ]] || exit 91
    inner="${1:-}"
    shift
    case "$inner" in
      pg_dump)
        printf 'exec pg_dump %s\n' "$*" >> "$MOCK_DOCKER_LOG"
        printf '%s\n' '-- SForum test dump' 'CREATE TABLE restored_table (id bigint);'
        if [[ "${MOCK_PG_DUMP_FAIL:-false}" == "true" ]]; then
          printf '%s\n' '-- partial output before failure'
          exit 9
        fi
        ;;
      pg_isready)
        printf 'exec pg_isready %s\n' "$*" >> "$MOCK_DOCKER_LOG"
        ;;
      createdb)
        printf 'exec createdb %s\n' "$*" >> "$MOCK_DOCKER_LOG"
        ;;
      dropdb)
        printf 'exec dropdb %s\n' "$*" >> "$MOCK_DOCKER_LOG"
        ;;
      psql)
        args=" $* "
        if [[ "$args" == *" -c "* ]]; then
          printf 'exec validation %s\n' "$*" >> "$MOCK_DOCKER_LOG"
          printf '3\n'
        elif [[ "$args" == *" -d sforum_restore_"* ]]; then
          cat >/dev/null
          printf 'exec restore %s\n' "$*" >> "$MOCK_DOCKER_LOG"
          if [[ "${MOCK_PSQL_FAIL:-false}" == "true" ]]; then
            printf 'restore failed\n' >> "$MOCK_DOCKER_LOG"
            exit 7
          fi
        elif [[ "$args" == *" -d postgres "* ]]; then
          swap_sql="$(cat)"
          printf 'exec swap %s\n%s\n' "$*" "$swap_sql" >> "$MOCK_DOCKER_LOG"
        else
          exit 92
        fi
        ;;
      *)
        exit 93
        ;;
    esac
    ;;
  *)
    exit 94
    ;;
esac
EOF
chmod +x "$MOCK_BIN/docker"

export PATH="$MOCK_BIN:$PATH"
export MOCK_DOCKER_LOG="$MOCK_LOG"

SUCCESS_BACKUP_DIR="$TEMP_DIR/backups-success"
backup_output="$(cd "$TEST_ROOT" && SFORUM_BACKUP_DIR="$SUCCESS_BACKUP_DIR" ./deploy/scripts/backup-postgres.sh)"
backup_file="$(tail -n 1 <<< "$backup_output")"
[[ -s "$backup_file" ]] || fail "successful backup is missing"
[[ ! -e "$TEMP_DIR/env-command-was-executed" ]] || fail "backup executed shell content from .env.production"
[[ "$(file_mode "$backup_file")" == "600" ]] || fail "backup file mode is not 0600"
[[ "$(file_mode "$TEST_ROOT/.env.production")" == "600" ]] || fail ".env.production mode is not 0600"
if find "$SUCCESS_BACKUP_DIR" -maxdepth 1 -name '.*' -type f | grep -q .; then
  fail "successful backup left a temporary file"
fi

FAILED_BACKUP_DIR="$TEMP_DIR/backups-failed"
if (cd "$TEST_ROOT" && MOCK_PG_DUMP_FAIL=true SFORUM_BACKUP_DIR="$FAILED_BACKUP_DIR" ./deploy/scripts/backup-postgres.sh >/dev/null 2>&1); then
  fail "failed pg_dump unexpectedly produced a successful backup"
fi
if find "$FAILED_BACKUP_DIR" -maxdepth 1 -type f | grep -q .; then
  fail "failed pg_dump left a partial backup"
fi

printf '%s\n' 'CREATE TABLE restored_table (id bigint);' > "$TEMP_DIR/restore.sql"
: > "$MOCK_LOG"
(cd "$TEST_ROOT" && SFORUM_CONFIRM_RESTORE=RESTORE ./deploy/scripts/restore-postgres.sh "$TEMP_DIR/restore.sql" >/dev/null)

[[ ! -e "$TEMP_DIR/env-command-was-executed" ]] || fail "restore executed shell content from .env.production"

grep -q '^compose stop api$' "$MOCK_LOG" || fail "restore did not stop API"
if grep -q '^compose stop .*worker' "$MOCK_LOG"; then
  fail "restore attempted to stop a legacy standalone Worker"
fi
grep -q 'exec restore .*--set=ON_ERROR_STOP=1.*--single-transaction' "$MOCK_LOG" || fail "restore is missing fail-fast single-transaction flags"
grep -q 'ALTER DATABASE :"target_db" WITH ALLOW_CONNECTIONS false;' "$MOCK_LOG" || fail "restore did not close target database admission"
grep -q 'ALTER DATABASE :"target_db" RENAME TO :"previous_db";' "$MOCK_LOG" || fail "restore did not retain the old target until swap"
grep -q 'ALTER DATABASE :"restore_db" RENAME TO :"target_db";' "$MOCK_LOG" || fail "restore did not publish the validated database"
grep -q '^compose start api$' "$MOCK_LOG" || fail "restore did not restart the previously running API"
if grep -q '^compose start .*worker' "$MOCK_LOG"; then
  fail "restore attempted to restart a legacy standalone Worker"
fi

: > "$MOCK_LOG"
if (cd "$TEST_ROOT" && MOCK_PSQL_FAIL=true SFORUM_CONFIRM_RESTORE=RESTORE ./deploy/scripts/restore-postgres.sh "$TEMP_DIR/restore.sql" >/dev/null 2>&1); then
  fail "restore unexpectedly succeeded after a psql failure"
fi
grep -q '^restore failed$' "$MOCK_LOG" || fail "psql failure path was not exercised"
if grep -q '^exec swap ' "$MOCK_LOG"; then
  fail "failed restore attempted to publish the temporary database"
fi
grep -q 'exec dropdb .*sforum_restore_' "$MOCK_LOG" || fail "failed restore did not remove its temporary database"
grep -q '^compose start api$' "$MOCK_LOG" || fail "failed restore did not restart the previously running API"
if grep -q '^compose start .*worker' "$MOCK_LOG"; then
  fail "failed restore attempted to restart a legacy standalone Worker"
fi

printf '%s\n' \
  "POSTGRES_DB=\$(touch '$TEMP_DIR/database-value-was-executed')" \
  'POSTGRES_USER=sforum_test' > "$TEST_ROOT/.env.production"
if (cd "$TEST_ROOT" && SFORUM_BACKUP_DIR="$TEMP_DIR/backups-invalid-env" ./deploy/scripts/backup-postgres.sh >/dev/null 2>&1); then
  fail "backup accepted an unsafe POSTGRES_DB value"
fi
[[ ! -e "$TEMP_DIR/database-value-was-executed" ]] || fail "backup executed an unsafe POSTGRES_DB value"

if (cd "$TEST_ROOT" && SFORUM_CONFIRM_RESTORE=RESTORE ./deploy/scripts/restore-postgres.sh "$TEMP_DIR/restore.sql" >/dev/null 2>&1); then
  fail "restore accepted an unsafe POSTGRES_DB value"
fi
[[ ! -e "$TEMP_DIR/database-value-was-executed" ]] || fail "restore executed an unsafe POSTGRES_DB value"

grep -q 'wait-for-health.sh "http://127.0.0.1:${api_port}/api/v1/ready"' "$ROOT_DIR/deploy.sh" || fail "deploy does not wait for API readiness"
grep -q 'wait-for-health.sh "http://127.0.0.1:${web_port}/health"' "$ROOT_DIR/deploy.sh" || fail "deploy does not wait for Web readiness"
grep -q 'chmod 600 .env.production' "$ROOT_DIR/deploy.sh" || fail "deploy does not protect production configuration"

printf 'postgres-safety_test.sh: all checks passed\n'
