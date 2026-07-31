#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

DEPLOY_RC=".deployrc"
DEFAULT_RELEASE_VERSION="${SFORUM_DEFAULT_VERSION:-v3.0.0-alpha.9}"
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
  --version VERSION     Immutable GHCR release tag (for example v3.0.0-alpha.9)
  --action ACTION       deploy, migrate, backup, restore, status, logs, restart, stop
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
      pulling_release) echo "Pulling the four SForum release images before changing the database:" ;;
      starting_infra) echo "Starting the managed PostgreSQL and Redis services..." ;;
      fresh_database) echo "Fresh database detected; no backup is needed." ;;
      backup_first) echo "Existing installation detected; creating a PostgreSQL backup..." ;;
      stopping_app) echo "Stopping the previous API, worker, and Web containers..." ;;
      migrations_running) echo "Running database migrations with the target release image..." ;;
      starting_app) echo "Starting SForum from release images..." ;;
      verifying) echo "Checking API, Web, and all managed services..." ;;
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
      pulling_release) echo "先拉取四个 SForum 发布镜像，成功后才会改动数据库：" ;;
      starting_infra) echo "正在启动内置 PostgreSQL 和 Redis..." ;;
      fresh_database) echo "检测到全新数据库，本次无需备份。" ;;
      backup_first) echo "检测到已有安装，正在创建 PostgreSQL 备份..." ;;
      stopping_app) echo "正在停止旧版 API、Worker 和 Web 容器..." ;;
      migrations_running) echo "正在使用目标版本镜像运行数据库迁移..." ;;
      starting_app) echo "正在从发布镜像启动 SForum..." ;;
      verifying) echo "正在检查 API、Web 和全部内置服务..." ;;
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
  if [ -z "$RELEASE_VERSION" ]; then
    RELEASE_VERSION="$(rc_value version)"
  fi
  if [ -z "$RELEASE_VERSION" ]; then
    RELEASE_VERSION="$(env_file_value .env.production SFORUM_VERSION)"
  fi
  RELEASE_VERSION="${RELEASE_VERSION:-$DEFAULT_RELEASE_VERSION}"
  if [[ ! "$RELEASE_VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
    die "--version must look like v3.0.0 or v3.0.0-alpha.9"
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
  local public_api_base
  public_api_base="$(env_file_value .env.production NUXT_PUBLIC_API_BASE_URL /api/v1)"
  [ "$public_api_base" = "/api/v1" ] || die "$(t invalid_public_api_base)"
}

preflight() {
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
  "${COMPOSE[@]}" run --rm -T migrate
}

wait_for_deployment() {
  local api_port web_port timeout_seconds
  api_port="$(env_file_value .env.production API_PORT 18080)"
  web_port="$(env_file_value .env.production WEB_PORT 3000)"
  timeout_seconds="${SFORUM_DEPLOY_HEALTH_TIMEOUT_SECONDS:-120}"
  ./deploy/scripts/wait-for-health.sh "http://127.0.0.1:${api_port}/api/v1/ready" "$timeout_seconds"
  ./deploy/scripts/wait-for-health.sh "http://127.0.0.1:${web_port}/" "$timeout_seconds"
}

verify_services_running() {
  local running service
  running="$("${COMPOSE[@]}" ps --status running --services)"
  for service in postgres redis api worker web; do
    grep -Fqx "$service" <<< "$running" || die "Service is not running after deployment: $service"
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

print_web_url() {
  local web_port
  web_port="$(env_file_value .env.production WEB_PORT 3000)"
  echo "$(t web_url) http://127.0.0.1:${web_port}"
}

deploy_update() {
  configure_if_needed
  preflight
  printf '%s %s\n' "$(t version):" "$RELEASE_VERSION"

  echo "$(t pulling_release) $RELEASE_VERSION"
  "${COMPOSE[@]}" pull migrate api worker web

  t starting_infra
  "${COMPOSE[@]}" up -d --wait postgres redis

  if database_has_installation; then
    t backup_first
    ./deploy/scripts/backup-postgres.sh
  else
    t fresh_database
  fi

  t stopping_app
  "${COMPOSE[@]}" stop api worker web
  run_migrations_command

  t starting_app
  "${COMPOSE[@]}" up -d --no-build postgres redis api worker web

  t verifying
  wait_for_deployment
  verify_services_running
  persist_successful_state
  "${COMPOSE[@]}" ps
  print_web_url
  t success
}

run_migrations() {
  preflight
  "${COMPOSE[@]}" pull migrate
  "${COMPOSE[@]}" up -d --wait postgres redis
  run_migrations_command
}

create_backup() {
  preflight
  "${COMPOSE[@]}" up -d --wait postgres
  ./deploy/scripts/backup-postgres.sh
}

restore_backup() {
  preflight
  t backup_path
  read -r backup_file
  t confirm_restore
  read -r confirmation
  if [ "$confirmation" != "RESTORE" ]; then
    t canceled
    return
  fi
  SFORUM_CONFIRM_RESTORE=RESTORE ./deploy/scripts/restore-postgres.sh "$backup_file"
}

show_status() { preflight; "${COMPOSE[@]}" ps; }
show_logs() { preflight; "${COMPOSE[@]}" logs -f --tail=200; }
restart_services() {
  preflight
  "${COMPOSE[@]}" restart
  wait_for_deployment
  verify_services_running
  "${COMPOSE[@]}" ps
  print_web_url
}
stop_services() { preflight; "${COMPOSE[@]}" stop; }

run_action() {
  case "$1" in
    deploy|install|update) deploy_update ;;
    migrate) run_migrations ;;
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
    echo "2) $(t migrate)"
    echo "3) $(t backup)"
    echo "4) $(t restore)"
    echo "5) $(t status)"
    echo "6) $(t logs)"
    echo "7) $(t restart)"
    echo "8) $(t stop)"
    echo "0) $(t exit)"
    read -r -p "> [1] " menu_action
    case "$menu_action" in
      ""|1) deploy_update ;;
      2) run_migrations ;;
      3) create_backup ;;
      4) restore_backup ;;
      5) show_status ;;
      6) show_logs ;;
      7) restart_services ;;
      8) stop_services ;;
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
