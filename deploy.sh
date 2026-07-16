#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

DEPLOY_RC=".deployrc"
LANGUAGE=""
COMPOSE=(docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml)

while [ $# -gt 0 ]; do
  case "$1" in
    --lang)
      LANGUAGE="${2:-}"
      shift 2
      ;;
    --lang=*)
      LANGUAGE="${1#*=}"
      shift
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

load_language() {
  if [ "$LANGUAGE" != "" ]; then
    return
  fi

  if [ -f "$DEPLOY_RC" ]; then
    LANGUAGE="$(grep "^lang=" "$DEPLOY_RC" | cut -d "=" -f 2 || true)"
  fi

  if [ "$LANGUAGE" = "" ]; then
    echo "Choose language / 选择语言:"
    echo "1) English"
    echo "2) 简体中文"
    read -r -p "> " choice
    case "$choice" in
      1) LANGUAGE="en" ;;
      *) LANGUAGE="zh" ;;
    esac
    printf "lang=%s\n" "$LANGUAGE" > "$DEPLOY_RC"
  fi

  if [ "$LANGUAGE" != "en" ] && [ "$LANGUAGE" != "zh" ]; then
    LANGUAGE="zh"
  fi
}

t() {
  key="$1"
  if [ "$LANGUAGE" = "en" ]; then
    case "$key" in
      title) echo "SForum deployment console" ;;
      menu) echo "Choose an action:" ;;
      install) echo "Install / first-time setup" ;;
      deploy) echo "Deploy or update" ;;
      migrate) echo "Run migrations" ;;
      backup) echo "Create PostgreSQL backup" ;;
      restore) echo "Restore PostgreSQL backup" ;;
      status) echo "View service status" ;;
      logs) echo "View logs" ;;
      restart) echo "Restart services" ;;
      stop) echo "Stop services" ;;
      rollback) echo "Rollback" ;;
      exit) echo "Exit" ;;
      env_missing) echo ".env.production is missing. Creating it from .env.production.example." ;;
      edit_env) echo "Please edit .env.production before deploying." ;;
      preflight) echo "Running preflight checks..." ;;
      no_docker) echo "Docker is required." ;;
      no_compose) echo "Docker Compose plugin is required." ;;
      invalid_public_api_base) echo "NUXT_PUBLIC_API_BASE_URL must be /api/v1 so ordinary API traffic stays same-origin through Nuxt." ;;
      backup_first) echo "Creating backup before deploy..." ;;
      migrations_running) echo "Running database migrations..." ;;
      rollback_later) echo "Rollback metadata is not available yet. This will be enabled when release image tags are introduced." ;;
      confirm_restore) echo "Type RESTORE to confirm database restore:" ;;
      backup_path) echo "Backup file path:" ;;
      web_url) echo "Web is bound to:" ;;
      done) echo "Done." ;;
      *) echo "$key" ;;
    esac
  else
    case "$key" in
      title) echo "SForum 部署控制台" ;;
      menu) echo "请选择操作：" ;;
      install) echo "安装 / 首次初始化" ;;
      deploy) echo "部署或更新" ;;
      migrate) echo "运行数据库迁移" ;;
      backup) echo "创建 PostgreSQL 备份" ;;
      restore) echo "恢复 PostgreSQL 备份" ;;
      status) echo "查看服务状态" ;;
      logs) echo "查看日志" ;;
      restart) echo "重启服务" ;;
      stop) echo "停止服务" ;;
      rollback) echo "回滚" ;;
      exit) echo "退出" ;;
      env_missing) echo "缺少 .env.production，已从 .env.production.example 创建。" ;;
      edit_env) echo "部署前请先编辑 .env.production。" ;;
      preflight) echo "正在执行预检..." ;;
      no_docker) echo "需要先安装 Docker。" ;;
      no_compose) echo "需要 Docker Compose 插件。" ;;
      invalid_public_api_base) echo "NUXT_PUBLIC_API_BASE_URL 必须是 /api/v1，确保普通 API 流量继续同源经过 Nuxt。" ;;
      backup_first) echo "部署前正在创建备份..." ;;
      migrations_running) echo "正在运行数据库迁移..." ;;
      rollback_later) echo "暂未记录可回滚版本；引入发布镜像标签后会启用。" ;;
      confirm_restore) echo "请输入 RESTORE 确认恢复数据库：" ;;
      backup_path) echo "备份文件路径：" ;;
      web_url) echo "Web 已绑定到：" ;;
      done) echo "完成。" ;;
      *) echo "$key" ;;
    esac
  fi
}

ensure_env() {
  if [ ! -f .env.production ]; then
    cp .env.production.example .env.production
    echo "$(t env_missing)"
    echo "$(t edit_env)"
    exit 1
  fi
}

env_file_value() {
  local file="$1"
  local key="$2"
  local fallback="$3"
  local value

  value="$(grep "^${key}=" "$file" | tail -n 1 | cut -d "=" -f 2- || true)"
  if [ "$value" = "" ]; then
    value="$fallback"
  fi
  printf "%s" "$value"
}

validate_env_contract() {
  local public_api_base
  public_api_base="$(env_file_value .env.production NUXT_PUBLIC_API_BASE_URL /api/v1)"

  if [ "$public_api_base" != "/api/v1" ]; then
    echo "$(t invalid_public_api_base)"
    exit 1
  fi
}

print_web_url() {
  local web_port
  web_port="$(env_file_value .env.production WEB_PORT 3000)"
  echo "$(t web_url) http://127.0.0.1:${web_port}"
}

preflight() {
  echo "$(t preflight)"
  if ! command -v docker >/dev/null 2>&1; then
    echo "$(t no_docker)"
    exit 1
  fi
  if ! docker compose version >/dev/null 2>&1; then
    echo "$(t no_compose)"
    exit 1
  fi
  ensure_env
  validate_env_contract
}

install() {
  ensure_env
  echo "$(t edit_env)"
}

deploy_update() {
  preflight
  echo "$(t backup_first)"
  ./deploy/scripts/backup-postgres.sh || true
  run_migrations_command
  "${COMPOSE[@]}" up -d --build
  "${COMPOSE[@]}" ps
  print_web_url
}

run_migrations_command() {
  echo "$(t migrations_running)"
  "${COMPOSE[@]}" run --rm -T --build migrate
}

run_migrations() {
  preflight
  run_migrations_command
}

create_backup() {
  preflight
  ./deploy/scripts/backup-postgres.sh
}

restore_backup() {
  preflight
  echo "$(t backup_path)"
  read -r backup_file
  echo "$(t confirm_restore)"
  read -r confirmation
  if [ "$confirmation" != "RESTORE" ]; then
    echo "Canceled."
    return
  fi
  SFORUM_CONFIRM_RESTORE=RESTORE ./deploy/scripts/restore-postgres.sh "$backup_file"
}

show_status() {
  preflight
  "${COMPOSE[@]}" ps
}

show_logs() {
  preflight
  "${COMPOSE[@]}" logs -f --tail=200
}

restart_services() {
  preflight
  "${COMPOSE[@]}" restart
}

stop_services() {
  preflight
  "${COMPOSE[@]}" stop
}

rollback() {
  preflight
  echo "$(t rollback_later)"
}

main_menu() {
  while true; do
    echo
    echo "== $(t title) =="
    echo "1) $(t install)"
    echo "2) $(t deploy)"
    echo "3) $(t migrate)"
    echo "4) $(t backup)"
    echo "5) $(t restore)"
    echo "6) $(t status)"
    echo "7) $(t logs)"
    echo "8) $(t restart)"
    echo "9) $(t stop)"
    echo "10) $(t rollback)"
    echo "0) $(t exit)"
    read -r -p "> " action

    case "$action" in
      1) install ;;
      2) deploy_update ;;
      3) run_migrations ;;
      4) create_backup ;;
      5) restore_backup ;;
      6) show_status ;;
      7) show_logs ;;
      8) restart_services ;;
      9) stop_services ;;
      10) rollback ;;
      0) exit 0 ;;
      *) echo "?" ;;
    esac
  done
}

load_language
main_menu
