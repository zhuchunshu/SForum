#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

DEPLOY_RC=".deployrc"
DEFAULT_RELEASE_VERSION="${SFORUM_DEFAULT_VERSION:-latest}"
LANGUAGE=""
ACTION=""
ASSUME_DEFAULTS=false
RELEASE_VERSION="${SFORUM_VERSION:-}"
COMPOSE=()

usage() {
  cat <<'EOF'
Usage: ./deploy.sh [options]

Options:
  --lang zh|en          Interface language
  --version VERSION     Release tag or latest stable release (default: latest)
  --action ACTION       deploy, backup, restore, status, logs, restart, stop
  --yes, --defaults     Accept recommended configuration defaults
  -h, --help            Show this help
EOF
}

die() {
  printf '%s\n' "$1" >&2
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --lang)
      [ "$#" -ge 2 ] || die "--lang requires a value"
      LANGUAGE="$2"
      shift 2
      ;;
    --lang=*) LANGUAGE="${1#*=}"; shift ;;
    --version)
      [ "$#" -ge 2 ] || die "--version requires a value"
      RELEASE_VERSION="$2"
      shift 2
      ;;
    --version=*) RELEASE_VERSION="${1#*=}"; shift ;;
    --action)
      [ "$#" -ge 2 ] || die "--action requires a value"
      ACTION="$2"
      shift 2
      ;;
    --action=*) ACTION="${1#*=}"; shift ;;
    --yes|--defaults) ASSUME_DEFAULTS=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "Unknown option: $1" ;;
  esac
done

rc_value() {
  local key="$1"
  [ -f "$DEPLOY_RC" ] || return 0
  sed -n "s/^${key}=//p" "$DEPLOY_RC" | tail -n 1
}

env_file_value() {
  local file="$1"
  local key="$2"
  local fallback="${3:-}"
  local value=""
  if [ -f "$file" ]; then
    value="$(sed -n "s/^${key}=//p" "$file" | tail -n 1)"
  fi
  printf '%s' "${value:-$fallback}"
}

load_language() {
  if [ -z "$LANGUAGE" ]; then
    LANGUAGE="$(rc_value lang)"
  fi
  if [ -z "$LANGUAGE" ]; then
    if [ "$ASSUME_DEFAULTS" = true ]; then
      LANGUAGE="zh"
    else
      echo "Choose language / 选择语言:"
      echo "1) English"
      echo "2) 简体中文（推荐）"
      read -r -p "> [2] " choice
      case "$choice" in
        1) LANGUAGE="en" ;;
        *) LANGUAGE="zh" ;;
      esac
    fi
  fi
  [ "$LANGUAGE" = "zh" ] || [ "$LANGUAGE" = "en" ] || die "--lang must be zh or en"
}

t() {
  local key="$1"
  if [ "$LANGUAGE" = "en" ]; then
    case "$key" in
      title) echo "SForum deployment console" ;;
      install) echo "Install or update (recommended)" ;;
      migrate) echo "Run migrations" ;;
      backup) echo "Create PostgreSQL backup" ;;
      restore) echo "Restore PostgreSQL backup" ;;
      status) echo "View service status" ;;
      logs) echo "View logs" ;;
      restart) echo "Restart services" ;;
      stop) echo "Stop services" ;;
      exit) echo "Exit" ;;
      preflight) echo "Checking Docker and production configuration..." ;;
      no_docker) echo "Docker is required. Install Docker Desktop or Docker Engine first." ;;
      no_daemon) echo "Docker is installed but the daemon is not running. Start Docker and retry." ;;
      no_compose) echo "Docker Compose 2.24.4 or newer is required." ;;
      no_curl) echo "curl is required for the final health check." ;;
      env_missing) echo ".env.production is missing. Run the install/update action first." ;;
      invalid_public_api_base) echo "NUXT_PUBLIC_API_BASE_URL must be /api/v1." ;;
      invalid_env) echo "The production configuration is incomplete or unsafe. Move it aside and rerun the installer, or fix the named value." ;;
      port_busy) echo "A required loopback port is already used by another process:" ;;
      deploy_locked) echo "Another deployment operation is running. Wait for it to finish and retry." ;;
      pulling_release) echo "Pulling the four SForum release images before changing the database:" ;;
      verifying_images) echo "Verifying API, Worker, and migrator build identities..." ;;
      starting_infra) echo "Starting the managed PostgreSQL and Redis services..." ;;
      fresh_database) echo "Fresh database detected; no backup is needed." ;;
      backup_first) echo "Existing installation detected; creating a PostgreSQL backup..." ;;
      stopping_app) echo "Stopping the previous API, worker, and Web containers..." ;;
      migrations_running) echo "Running database migrations with the target release image..." ;;
      starting_app) echo "Starting SForum from release images..." ;;
      verifying) echo "Checking API, Web, and all managed services..." ;;
      recovery_required) echo "Deployment did not complete after the old app stopped. Services were not declared healthy; inspect .deployrc and Docker logs before retrying." ;;
      version) echo "Release version" ;;
      web_url) echo "SForum is running at:" ;;
      success) echo "Deployment completed successfully." ;;
      rollback_later) echo "To roll back, deploy an earlier immutable version only after checking migration compatibility." ;;
      confirm_restore) echo "Type RESTORE to confirm database restore:" ;;
      backup_path) echo "Backup file path:" ;;
      canceled) echo "Canceled." ;;
      *) echo "$key" ;;
    esac
  else
    case "$key" in
      title) echo "SForum 部署控制台" ;;
      install) echo "安装或更新（推荐）" ;;
      migrate) echo "运行数据库迁移" ;;
      backup) echo "创建 PostgreSQL 备份" ;;
      restore) echo "恢复 PostgreSQL 备份" ;;
      status) echo "查看服务状态" ;;
      logs) echo "查看日志" ;;
      restart) echo "重启服务" ;;
      stop) echo "停止服务" ;;
      exit) echo "退出" ;;
      preflight) echo "正在检查 Docker 和生产配置..." ;;
      no_docker) echo "需要先安装 Docker Desktop 或 Docker Engine。" ;;
      no_daemon) echo "Docker 已安装，但服务尚未运行；请启动 Docker 后重试。" ;;
      no_compose) echo "需要 Docker Compose 2.24.4 或更高版本。" ;;
      no_curl) echo "最终健康检查需要 curl。" ;;
      env_missing) echo "缺少 .env.production，请先选择 '安装或更新'。" ;;
      invalid_public_api_base) echo "NUXT_PUBLIC_API_BASE_URL 必须是 /api/v1。" ;;
      invalid_env) echo "生产配置不完整或不安全。请移走旧文件后重新运行安装，或修复提示中的配置项。" ;;
      port_busy) echo "以下本机端口已被其他程序占用：" ;;
      deploy_locked) echo "另一个部署操作正在运行，请等待其结束后重试。" ;;
      pulling_release) echo "先拉取四个 SForum 发布镜像，成功后才会改动数据库：" ;;
      verifying_images) echo "正在核对 API、Worker 和迁移器的构建身份..." ;;
      starting_infra) echo "正在启动内置 PostgreSQL 和 Redis..." ;;
      fresh_database) echo "检测到全新数据库，本次无需备份。" ;;
      backup_first) echo "检测到已有安装，正在创建 PostgreSQL 备份..." ;;
      stopping_app) echo "正在停止旧版 API、Worker 和 Web 容器..." ;;
      migrations_running) echo "正在使用目标版本镜像运行数据库迁移..." ;;
      starting_app) echo "正在从发布镜像启动 SForum..." ;;
      verifying) echo "正在检查 API、Web 和全部内置服务..." ;;
      recovery_required) echo "旧应用停止后部署未能完成。当前服务未被声明为健康；重试前请检查 .deployrc 和 Docker 日志。" ;;
      version) echo "发布版本" ;;
      web_url) echo "SForum 已运行：" ;;
      success) echo "部署成功完成。" ;;
      rollback_later) echo "如需回滚，请先确认数据库迁移兼容，再部署较早的不可变版本。" ;;
      confirm_restore) echo "请输入 RESTORE 确认恢复数据库：" ;;
      backup_path) echo "备份文件路径：" ;;
      canceled) echo "已取消。" ;;
      *) echo "$key" ;;
    esac
  fi
}

resolve_release_version() {
  local response resolved
  if [ -z "$RELEASE_VERSION" ]; then
    RELEASE_VERSION="$(rc_value version)"
  fi
  if [ -z "$RELEASE_VERSION" ]; then
    RELEASE_VERSION="$(env_file_value .env.production SFORUM_VERSION)"
  fi
  RELEASE_VERSION="${RELEASE_VERSION:-$DEFAULT_RELEASE_VERSION}"
  if [ "$RELEASE_VERSION" = "latest" ]; then
    response="$(curl -fsSL \
      --connect-timeout 10 \
      --max-time 30 \
      -H 'Accept: application/vnd.github+json' \
      -H 'X-GitHub-Api-Version: 2022-11-28' \
      -H 'User-Agent: SForum-deploy' \
      "${SFORUM_LATEST_RELEASE_API_URL:-https://api.github.com/repos/zhuchunshu/SForum/releases/latest}")" || \
      die "Could not query the latest stable GitHub Release. Check the network and retry."
    resolved="$(printf '%s\n' "$response" | sed -n 's/^.*"tag_name":[[:space:]]*"\([^"]*\)".*$/\1/p' | sed -n '1p')"
    [ -n "$resolved" ] || die "GitHub returned no stable SForum Release."
    RELEASE_VERSION="$resolved"
  fi
  if [[ ! "$RELEASE_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
    die "--version must be latest or look like v3.0.0"
  fi
  export SFORUM_VERSION="$RELEASE_VERSION"
  COMPOSE=(docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml -f compose.release.yaml)
}

configure_if_needed() {
  if [ -f .env.production ]; then
    chmod 600 .env.production
    return
  fi
  local wizard=(./deploy/scripts/configure-production.sh --lang "$LANGUAGE" --version "$RELEASE_VERSION")
  if [ "$ASSUME_DEFAULTS" = true ]; then
    wizard+=(--defaults)
  fi
  "${wizard[@]}"
}

compose_version_supported() {
  local raw major minor patch
  raw="$(docker compose version --short 2>/dev/null || true)"
  raw="${raw#v}"
  raw="${raw%%-*}"
  IFS='.' read -r major minor patch <<< "$raw"
  [[ "$major" =~ ^[0-9]+$ && "$minor" =~ ^[0-9]+$ && "$patch" =~ ^[0-9]+$ ]] || return 1
  (( major > 2 || (major == 2 && minor > 24) || (major == 2 && minor == 24 && patch >= 4) ))
}

validate_env_contract() {
  local public_api_base app_env app_url postgres_db postgres_user postgres_password database_url
  local redis_addr redis_password session_secret identity_secret option_key altcha_secret marketplace_key
  public_api_base="$(env_file_value .env.production NUXT_PUBLIC_API_BASE_URL /api/v1)"
  [ "$public_api_base" = "/api/v1" ] || die "$(t invalid_public_api_base)"

  app_env="$(env_file_value .env.production APP_ENV)"
  app_url="$(env_file_value .env.production APP_URL)"
  postgres_db="$(env_file_value .env.production POSTGRES_DB)"
  postgres_user="$(env_file_value .env.production POSTGRES_USER)"
  postgres_password="$(env_file_value .env.production POSTGRES_PASSWORD)"
  database_url="$(env_file_value .env.production DATABASE_URL)"
  redis_addr="$(env_file_value .env.production REDIS_ADDR)"
  redis_password="$(env_file_value .env.production REDIS_PASSWORD)"
  session_secret="$(env_file_value .env.production SESSION_HASH_SECRET)"
  identity_secret="$(env_file_value .env.production IDENTITY_SUBJECT_HMAC_SECRET)"
  option_key="$(env_file_value .env.production APP_OPTION_ENC_KEY)"
  altcha_secret="$(env_file_value .env.production ALTCHA_SECRET)"
  marketplace_key="$(env_file_value .env.production MARKETPLACE_ED25519_PUBLIC_KEY_HEX)"

  [ "$app_env" = "production" ] || die "$(t invalid_env) APP_ENV"
  [[ "$app_url" =~ ^https?://[^[:space:]]+$ ]] || die "$(t invalid_env) APP_URL"
  [[ "$postgres_db" =~ ^[A-Za-z0-9_.-]+$ ]] || die "$(t invalid_env) POSTGRES_DB"
  [[ "$postgres_user" =~ ^[A-Za-z0-9_.-]+$ ]] || die "$(t invalid_env) POSTGRES_USER"
  [[ "$postgres_password" =~ ^[A-Za-z0-9._~-]{32,}$ ]] || die "$(t invalid_env) POSTGRES_PASSWORD"
  [[ "$database_url" == postgres://*"@postgres:5432/${postgres_db}"* ]] || die "$(t invalid_env) DATABASE_URL"
  [ "$redis_addr" = "redis:6379" ] || die "$(t invalid_env) REDIS_ADDR"
  [[ "$redis_password" =~ ^[A-Za-z0-9._~-]{32,}$ ]] || die "$(t invalid_env) REDIS_PASSWORD"
  [ "${#session_secret}" -ge 32 ] || die "$(t invalid_env) SESSION_HASH_SECRET"
  [ "${#identity_secret}" -ge 32 ] || die "$(t invalid_env) IDENTITY_SUBJECT_HMAC_SECRET"
  [[ "$option_key" =~ ^[0-9A-Fa-f]{64}$ ]] || die "$(t invalid_env) APP_OPTION_ENC_KEY"
  [ "${#altcha_secret}" -ge 32 ] || die "$(t invalid_env) ALTCHA_SECRET"
  [[ "$marketplace_key" =~ ^[0-9A-Fa-f]{64}$ ]] || die "$(t invalid_env) MARKETPLACE_ED25519_PUBLIC_KEY_HEX"
}

service_is_running() {
  local service="$1"
  "${COMPOSE[@]}" ps --status running --services | grep -Fqx "$service"
}

port_is_listening() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
  elif command -v ss >/dev/null 2>&1; then
    ss -ltn | awk -v suffix=":$port" 'NR > 1 && $4 ~ suffix "$" { found=1 } END { exit !found }'
  else
    return 1
  fi
}

validate_ports_available() {
  local web_port api_port
  web_port="$(env_file_value .env.production WEB_PORT 3000)"
  api_port="$(env_file_value .env.production API_PORT 18080)"
  if ! service_is_running web && port_is_listening "$web_port"; then
    die "$(t port_busy) 127.0.0.1:${web_port} (Web)"
  fi
  if ! service_is_running api && port_is_listening "$api_port"; then
    die "$(t port_busy) 127.0.0.1:${api_port} (API)"
  fi
}

preflight() {
  local check_ports="${1:-false}"
  t preflight
  command -v docker >/dev/null 2>&1 || die "$(t no_docker)"
  docker info >/dev/null 2>&1 || die "$(t no_daemon)"
  docker compose version >/dev/null 2>&1 || die "$(t no_compose)"
  compose_version_supported || die "$(t no_compose)"
  command -v curl >/dev/null 2>&1 || die "$(t no_curl)"
  [ -f .env.production ] || die "$(t env_missing)"
  chmod 600 .env.production
  validate_env_contract
  "${COMPOSE[@]}" config --quiet
  if [ "$check_ports" = true ]; then
    validate_ports_available
  fi
}

database_has_installation() {
  local postgres_user postgres_db result
  postgres_user="$(env_file_value .env.production POSTGRES_USER sforum)"
  postgres_db="$(env_file_value .env.production POSTGRES_DB sforum)"
  if ! result="$("${COMPOSE[@]}" exec -T postgres psql -U "$postgres_user" -d "$postgres_db" -tAc "SELECT to_regclass('public.goose_db_version') IS NOT NULL")"; then
    die "Could not inspect the PostgreSQL migration state."
  fi
  [ "${result//[[:space:]]/}" = "t" ]
}

run_migrations_command() {
  t migrations_running
  "${COMPOSE[@]}" run --rm -T --pull never migrate
}

wait_for_deployment() {
  local api_port web_port timeout_seconds
  api_port="$(env_file_value .env.production API_PORT 18080)"
  web_port="$(env_file_value .env.production WEB_PORT 3000)"
  timeout_seconds="${SFORUM_DEPLOY_HEALTH_TIMEOUT_SECONDS:-120}"
  ./deploy/scripts/wait-for-health.sh "http://127.0.0.1:${api_port}/api/v1/ready" "$timeout_seconds" || return 1
  ./deploy/scripts/wait-for-health.sh "http://127.0.0.1:${web_port}/" "$timeout_seconds" || return 1
}

verify_services_running() {
  local running service
  running="$("${COMPOSE[@]}" ps --status running --services)"
  for service in postgres redis api worker web; do
    if ! grep -Fqx "$service" <<< "$running"; then
      echo "Service is not running after deployment: $service" >&2
      return 1
    fi
  done
}

verify_services_stable() {
  local stability_seconds="${SFORUM_DEPLOY_STABILITY_SECONDS:-3}"
  if [ "$stability_seconds" -gt 0 ]; then
    sleep "$stability_seconds"
  fi
  verify_services_running
}

verify_release_identities() {
  local service binary image actual expected_version
  expected_version="${RELEASE_VERSION#v}"
  t verifying_images
  for service in api worker migrate; do
    binary="sforum-$service"
    image="${SFORUM_REGISTRY:-ghcr.io/zhuchunshu}/sforum-${service}:${RELEASE_VERSION}"
    if ! actual="$(docker run --rm "$image" "$binary" --version)"; then
      echo "Could not inspect the $service image identity." >&2
      return 1
    fi
    if [[ "$actual" != "SForum $expected_version ("* ]]; then
      echo "$service image identity mismatch: $actual" >&2
      return 1
    fi
  done
}

persist_successful_state() {
  local temp_file
  umask 077
  temp_file="$(mktemp "$ROOT_DIR/.deployrc.XXXXXX")"
  printf 'lang=%s\nversion=%s\nmode=release\n' "$LANGUAGE" "$RELEASE_VERSION" > "$temp_file"
  chmod 600 "$temp_file"
  mv -f "$temp_file" "$DEPLOY_RC"
}

persist_recovery_state() {
  local reason="$1"
  local backup_file="${2:-}"
  local previous_version="$3"
  local temp_file
  umask 077
  temp_file="$(mktemp "$ROOT_DIR/.deployrc.XXXXXX")"
  printf 'lang=%s\nstatus=recovery_required\nattempted_version=%s\nprevious_version=%s\nreason=%s\nbackup=%s\n' \
    "$LANGUAGE" "$RELEASE_VERSION" "$previous_version" "$reason" "$backup_file" > "$temp_file"
  chmod 600 "$temp_file"
  mv -f "$temp_file" "$DEPLOY_RC"
}

DEPLOY_LOCK_DIR="$ROOT_DIR/.deploy.lock"
acquire_deploy_lock() {
  if ! mkdir "$DEPLOY_LOCK_DIR" 2>/dev/null; then
    die "$(t deploy_locked)"
  fi
  trap 'rmdir "$DEPLOY_LOCK_DIR" 2>/dev/null || true' EXIT HUP INT TERM
}

release_deploy_lock() {
  rmdir "$DEPLOY_LOCK_DIR" 2>/dev/null || true
  trap - EXIT HUP INT TERM
}

print_web_url() {
  local web_port
  web_port="$(env_file_value .env.production WEB_PORT 3000)"
  echo "$(t web_url) http://127.0.0.1:${web_port}"
}

deploy_update() {
  local backup_output="" backup_file="" previous_version previous_running
  acquire_deploy_lock
  configure_if_needed
  preflight true
  previous_version="$(rc_value version)"
  previous_running="$("${COMPOSE[@]}" ps --status running --services | grep -E '^(api|worker|web)$' || true)"
  printf '%s %s\n' "$(t version):" "$RELEASE_VERSION"

  echo "$(t pulling_release) $RELEASE_VERSION"
  "${COMPOSE[@]}" pull migrate api worker web
  verify_release_identities || die "Release image verification failed before any database change."

  t starting_infra
  "${COMPOSE[@]}" up -d --wait postgres redis

  if database_has_installation; then
    t backup_first
    backup_output="$(./deploy/scripts/backup-postgres.sh)"
    printf '%s\n' "$backup_output"
    backup_file="$(tail -n 1 <<< "$backup_output")"
  else
    t fresh_database
  fi

  t stopping_app
  if ! "${COMPOSE[@]}" stop api worker web; then
    if [ -n "$previous_running" ]; then
      # shellcheck disable=SC2086
      "${COMPOSE[@]}" start $previous_running || true
    fi
    die "Could not stop the previous application services."
  fi
  if ! run_migrations_command; then
    persist_recovery_state migration_failed "$backup_file" "$previous_version"
    t recovery_required >&2
    return 1
  fi

  t starting_app
  if ! "${COMPOSE[@]}" up -d --no-build --pull never postgres redis api worker web; then
    persist_recovery_state startup_failed "$backup_file" "$previous_version"
    t recovery_required >&2
    return 1
  fi

  t verifying
  if ! wait_for_deployment || ! verify_services_running || ! verify_services_stable; then
    persist_recovery_state health_failed "$backup_file" "$previous_version"
    t recovery_required >&2
    "${COMPOSE[@]}" ps >&2 || true
    return 1
  fi
  "${COMPOSE[@]}" ps || true
  persist_successful_state
  release_deploy_lock
  print_web_url
  t success
}

create_backup() {
  acquire_deploy_lock
  preflight
  "${COMPOSE[@]}" up -d --wait postgres
  ./deploy/scripts/backup-postgres.sh
  release_deploy_lock
}

restore_backup() {
  acquire_deploy_lock
  preflight
  t backup_path
  read -r backup_file
  t confirm_restore
  read -r confirmation
  if [ "$confirmation" != "RESTORE" ]; then
    t canceled
    release_deploy_lock
    return
  fi
  SFORUM_CONFIRM_RESTORE=RESTORE ./deploy/scripts/restore-postgres.sh "$backup_file"
  release_deploy_lock
}

show_status() { preflight; "${COMPOSE[@]}" ps; }
show_logs() { preflight; "${COMPOSE[@]}" logs -f --tail=200; }
restart_services() {
  acquire_deploy_lock
  preflight true
  "${COMPOSE[@]}" restart
  wait_for_deployment
  verify_services_running
  "${COMPOSE[@]}" ps
  print_web_url
  release_deploy_lock
}
stop_services() { acquire_deploy_lock; preflight; "${COMPOSE[@]}" stop; release_deploy_lock; }

run_action() {
  case "$1" in
    deploy|install|update) deploy_update ;;
    backup) create_backup ;;
    restore) restore_backup ;;
    status) show_status ;;
    logs) show_logs ;;
    restart) restart_services ;;
    stop) stop_services ;;
    rollback) preflight; t rollback_later ;;
    *) die "Unknown action: $1" ;;
  esac
}

main_menu() {
  while true; do
    echo
    echo "== $(t title) =="
    echo "1) $(t install)"
    echo "2) $(t backup)"
    echo "3) $(t restore)"
    echo "4) $(t status)"
    echo "5) $(t logs)"
    echo "6) $(t restart)"
    echo "7) $(t stop)"
    echo "0) $(t exit)"
    read -r -p "> [1] " menu_action
    case "$menu_action" in
      ""|1) deploy_update ;;
      2) create_backup ;;
      3) restore_backup ;;
      4) show_status ;;
      5) show_logs ;;
      6) restart_services ;;
      7) stop_services ;;
      0) exit 0 ;;
      *) echo "?" ;;
    esac
  done
}

load_language
resolve_release_version
if [ -n "$ACTION" ]; then
  run_action "$ACTION"
else
  main_menu
fi
