#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

REQUESTED_VERSION=""
TARGET_VERSION=""
ASSUME_YES=false
FORCE_BOOTSTRAP=false
DEPLOY_RC=".deployrc"
RUNTIME_DIR="$ROOT_DIR/deploy/runtime"
LOCK_DIR="$ROOT_DIR/.deploy.lock"
COMPOSE=()

usage() {
  cat <<'EOF'
Usage: ./upgrade.sh [VERSION] [options]

Zero-downtime HTTP update for an existing SForum Compose installation.

Options:
  VERSION             Target release number/tag, or latest (default: latest)
  --version VERSION   Same as the positional VERSION argument
  --bootstrap         Convert the legacy direct-port topology once
  --yes               Skip version and one-time topology confirmations
  -h, --help          Show this help

The updater refuses targets with pending Core or River database migrations.
Use ./deploy.sh for those maintenance-window upgrades.

Examples: latest, v3.0.0-alpha.11, or 3.0.0-alpha.11. The latest option uses
the newest published GitHub Release, including prereleases.
EOF
}

die() {
  printf '%s\n' "$1" >&2
  exit 1
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || die "--version requires a value"
      [ -z "$REQUESTED_VERSION" ] || die "Specify the target version only once."
      REQUESTED_VERSION="$2"
      shift 2
      ;;
    --version=*)
      [ -z "$REQUESTED_VERSION" ] || die "Specify the target version only once."
      REQUESTED_VERSION="${1#*=}"
      shift
      ;;
    --bootstrap) FORCE_BOOTSTRAP=true; shift ;;
    --yes|--defaults) ASSUME_YES=true; shift ;;
    -h|--help) usage; exit 0 ;;
    -*) die "Unknown option: $1" ;;
    *)
      [ -z "$REQUESTED_VERSION" ] || die "Specify the target version only once."
      REQUESTED_VERSION="$1"
      shift
      ;;
  esac
done

REQUESTED_VERSION="${REQUESTED_VERSION:-${SFORUM_VERSION:-}}"

rc_value() {
  local key="$1"
  [ -f "$DEPLOY_RC" ] || return 0
  sed -n "s/^${key}=//p" "$DEPLOY_RC" | tail -n 1
}

env_value() {
  local key="$1" fallback="${2:-}" value=""
  if [ -f .env.production ]; then
    value="$(sed -n "s/^${key}=//p" .env.production | tail -n 1)"
  fi
  printf '%s' "${value:-$fallback}"
}

lang() {
  local value
  value="$(rc_value lang)"
  printf '%s' "${value:-zh}"
}

say() {
  local zh="$1" en="$2"
  if [ "$(lang)" = "en" ]; then printf '%s\n' "$en"; else printf '%s\n' "$zh"; fi
}

validate_version() {
  [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]] || \
    die "Version must look like v3.0.0 or v3.0.0-alpha.11"
}

resolve_latest_version() {
  local api_url response resolved
  say "正在查询 GitHub Releases 的最新发布版本..." "Resolving the latest published GitHub Release..." >&2
  api_url="${SFORUM_RELEASES_API_URL:-https://api.github.com/repos/zhuchunshu/SForum/releases?per_page=1}"
  response="$(curl -fsSL \
    --connect-timeout 10 \
    --max-time 30 \
    -H 'Accept: application/vnd.github+json' \
    -H 'X-GitHub-Api-Version: 2022-11-28' \
    -H 'User-Agent: SForum-upgrade' \
    "$api_url")" || \
    die "Could not query the latest GitHub Release. Check the network and retry."
  # Public unauthenticated listings exclude drafts. Asking for one result keeps
  # the parser dependency-free while still including prereleases.
  resolved="$(printf '%s\n' "$response" | sed -n 's/^.*"tag_name":[[:space:]]*"\([^"]*\)".*$/\1/p' | sed -n '1p')"
  [ -n "$resolved" ] || die "GitHub returned no published SForum Release."
  validate_version "$resolved"
  printf '%s' "$resolved"
}

select_target_version() {
  local normalized
  if [ -z "$REQUESTED_VERSION" ]; then
    if [ "$ASSUME_YES" != true ]; then
      if [ "$(lang)" = en ]; then
        read -r -p "Release to install [latest]: " REQUESTED_VERSION || \
          die "Could not read a release. Use --version or --yes for unattended updates."
      else
        read -r -p "请输入要更新的版本 [latest]: " REQUESTED_VERSION || \
          die "Could not read a release. Use --version or --yes for unattended updates."
      fi
    fi
  fi
  REQUESTED_VERSION="${REQUESTED_VERSION:-latest}"
  if [ "$REQUESTED_VERSION" = latest ]; then
    TARGET_VERSION="$(resolve_latest_version)"
  else
    normalized="$REQUESTED_VERSION"
    if [[ "$normalized" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
      normalized="v$normalized"
    fi
    validate_version "$normalized"
    TARGET_VERSION="$normalized"
  fi
}

confirm_target_version() {
  local current_version="$1" confirmation
  say "当前实际版本：$current_version" "Current resolved version: $current_version"
  say "目标实际版本：$TARGET_VERSION" "Target resolved version: $TARGET_VERSION"
  [ "$ASSUME_YES" = true ] && return 0
  if [ "$(lang)" = en ]; then
    read -r -p "Update to $TARGET_VERSION? [Y/n]: " confirmation || \
      die "Could not read confirmation; use --yes to continue unattended."
  else
    read -r -p "确认更新到 $TARGET_VERSION？[Y/n]: " confirmation || \
      die "Could not read confirmation; use --yes to continue unattended."
  fi
  case "$confirmation" in
    ""|y|Y|yes|YES|Yes) ;;
    *) die "Canceled." ;;
  esac
}

acquire_lock() {
  if ! mkdir "$LOCK_DIR" 2>/dev/null; then
    die "Another deployment or update operation is running."
  fi
  trap 'rmdir "$LOCK_DIR" 2>/dev/null || true' EXIT HUP INT TERM
}

release_lock() {
  rmdir "$LOCK_DIR" 2>/dev/null || true
  trap - EXIT HUP INT TERM
}

configure_versions() {
  local active_slot="$1" active_version="$2" inactive_slot="$3" inactive_version="$4"
  if [ "$active_slot" = blue ]; then
    export SFORUM_BLUE_VERSION="$active_version"
    export SFORUM_GREEN_VERSION="$inactive_version"
  else
    export SFORUM_GREEN_VERSION="$active_version"
    export SFORUM_BLUE_VERSION="$inactive_version"
  fi
}

init_compose() {
  COMPOSE=(
    docker compose --env-file .env.production
    -f compose.yaml -f compose.prod.yaml -f compose.release.yaml
    -f compose.zero-downtime.yaml --profile zero-downtime
  )
}

pull_target_images() {
  local version="$1" image
  say "正在拉取目标版本镜像：$version" "Pulling target release images: $version"
  for image in api worker migrate web; do
    docker pull "ghcr.io/zhuchunshu/sforum-${image}:${version}"
  done
}

verify_target_images() {
  local version="$1" expected="${1#v}" image actual
  for image in api worker migrate web; do
    actual="$(docker image inspect "ghcr.io/zhuchunshu/sforum-${image}:${version}" \
      --format '{{index .Config.Labels "org.opencontainers.image.version"}}')"
    [ "$actual" = "$expected" ] || die "$image image version is '$actual', expected '$expected'"
  done
}

check_schema_is_online_safe() {
  local version="$1"
  say "正在确认目标版本不需要数据库迁移..." "Checking that the target requires no database migrations..."
  SFORUM_VERSION="$version" "${COMPOSE[@]}" run --rm -T --no-deps --pull never migrate sforum-migrate --check-no-pending >/dev/null || \
    die "Target schema differs from the live database. Use ./deploy.sh for a maintenance-window upgrade."
}

backup_database() {
  say "在线切换前创建 PostgreSQL 备份..." "Creating a PostgreSQL backup before the online switch..."
  ./deploy/scripts/backup-postgres.sh
}

slot_api() { printf 'api-%s' "$1"; }
slot_web() { printf 'web-%s' "$1"; }
slot_worker() { printf 'worker-%s' "$1"; }

wait_inside_slot() {
  local slot="$1" deadline=$((SECONDS + ${SFORUM_UPGRADE_HEALTH_TIMEOUT_SECONDS:-120}))
  local api web
  api="$(slot_api "$slot")"
  web="$(slot_web "$slot")"
  while [ "$SECONDS" -lt "$deadline" ]; do
    if "${COMPOSE[@]}" exec -T "$api" wget -q -O /dev/null http://127.0.0.1:8080/api/v1/ready \
      && "${COMPOSE[@]}" exec -T "$web" wget -q -O /dev/null http://127.0.0.1:3000/ \
      && "${COMPOSE[@]}" exec -T "$web" wget -q -O /dev/null http://127.0.0.1:3000/api/v1/ready; then
      return 0
    fi
    sleep 2
  done
  return 1
}

write_router_config() {
  local slot="$1" destination="$2"
  umask 077
  mkdir -p "$RUNTIME_DIR"
  {
    printf ':3000 {\n'
    printf '\theader X-SForum-Active-Slot %s\n' "$slot"
    printf '\treverse_proxy web-%s:3000\n' "$slot"
    printf '}\n\n'
    printf ':18080 {\n'
    printf '\theader X-SForum-Active-Slot %s\n' "$slot"
    printf '\treverse_proxy api-%s:8080\n' "$slot"
    printf '}\n'
  } > "$destination"
  chmod 600 "$destination"
}

wait_stable_ingress() {
  local slot="$1" web_port api_port deadline header_file
  web_port="$(env_value WEB_PORT 3000)"
  api_port="$(env_value API_PORT 18080)"
  deadline=$((SECONDS + ${SFORUM_UPGRADE_HEALTH_TIMEOUT_SECONDS:-120}))
  header_file="$(mktemp "${TMPDIR:-/tmp}/sforum-upgrade-headers.XXXXXX")"
  while [ "$SECONDS" -lt "$deadline" ]; do
    if curl -fsS -D "$header_file" -o /dev/null "http://127.0.0.1:${web_port}/" \
      && grep -Eiq "^X-SForum-Active-Slot:[[:space:]]*${slot}[[:space:]]*$" "$header_file" \
      && curl -fsS -D "$header_file" -o /dev/null "http://127.0.0.1:${api_port}/api/v1/ready" \
      && grep -Eiq "^X-SForum-Active-Slot:[[:space:]]*${slot}[[:space:]]*$" "$header_file"; then
      rm -f "$header_file"
      return 0
    fi
    sleep 1
  done
  rm -f "$header_file"
  return 1
}

switch_router() {
  local target_slot="$1" previous_slot="$2"
  local next="$RUNTIME_DIR/Caddyfile.next" current="$RUNTIME_DIR/Caddyfile" previous="$RUNTIME_DIR/Caddyfile.previous"
  write_router_config "$target_slot" "$next"
  "${COMPOSE[@]}" exec -T edge caddy validate --config /etc/sforum-router/Caddyfile.next --adapter caddyfile >/dev/null
  cp "$current" "$previous"
  mv -f "$next" "$current"
  if ! "${COMPOSE[@]}" exec -T edge caddy reload --config /etc/sforum-router/Caddyfile --adapter caddyfile >/dev/null; then
    mv -f "$previous" "$current"
    "${COMPOSE[@]}" exec -T edge caddy reload --config /etc/sforum-router/Caddyfile --adapter caddyfile >/dev/null || true
    return 1
  fi
  if wait_stable_ingress "$target_slot"; then
    rm -f "$previous"
    return 0
  fi
  mv -f "$previous" "$current"
  "${COMPOSE[@]}" exec -T edge caddy reload --config /etc/sforum-router/Caddyfile --adapter caddyfile >/dev/null || true
  wait_stable_ingress "$previous_slot" || true
  return 1
}

persist_state() {
  local active_slot="$1" active_version="$2" blue_version="$3" green_version="$4" temp_file
  validate_version "$active_version"
  validate_version "$blue_version"
  validate_version "$green_version"
  temp_file="$(mktemp "$ROOT_DIR/.deployrc.XXXXXX")"
  umask 077
  printf 'lang=%s\nversion=%s\nmode=release\ntopology=blue-green\nactive_slot=%s\nblue_version=%s\ngreen_version=%s\n' \
    "$(lang)" "$active_version" "$active_slot" "$blue_version" "$green_version" > "$temp_file"
  chmod 600 "$temp_file"
  mv -f "$temp_file" "$DEPLOY_RC"
}

bootstrap_topology() {
  local current_version="$1" target_version="$2" slot=blue
  if [ "$FORCE_BOOTSTRAP" != true ] && [ "$ASSUME_YES" != true ]; then
    say \
      "这是旧版直连端口拓扑。首次转换路由器会有一次短暂维护窗口；后续无迁移更新才是零停机。输入 BOOTSTRAP 继续：" \
      "This is the legacy direct-port topology. The one-time router conversion has a short maintenance window; later migration-free updates are zero-downtime. Type BOOTSTRAP to continue:"
    local confirmation
    read -r confirmation
    [ "$confirmation" = BOOTSTRAP ] || die "Canceled."
  fi

  configure_versions blue "$target_version" green "$target_version"
  pull_target_images "$target_version"
  docker pull caddy:2.10.2-alpine
  verify_target_images "$target_version"
  "${COMPOSE[@]}" up -d --wait postgres redis
  check_schema_is_online_safe "$target_version"
  backup_database
  "${COMPOSE[@]}" up -d --no-build --pull never api-blue web-blue
  wait_inside_slot blue || die "Blue candidate failed its internal API/Web checks."

  write_router_config blue "$RUNTIME_DIR/Caddyfile"
  say "正在执行一次性入口转换..." "Performing the one-time ingress conversion..."
  "${COMPOSE[@]}" stop worker web api
  if ! "${COMPOSE[@]}" up -d --no-build --pull never edge; then
    "${COMPOSE[@]}" start api worker web || true
    die "Could not start the stable router; legacy services were restarted."
  fi
  if ! wait_stable_ingress blue; then
    "${COMPOSE[@]}" stop edge || true
    "${COMPOSE[@]}" start api worker web || true
    die "Router health check failed; legacy services were restarted."
  fi
  "${COMPOSE[@]}" up -d --no-build --pull never worker-blue
  "${COMPOSE[@]}" ps --status running --services | grep -Fqx worker-blue || die "Blue worker did not start."
  persist_state blue "$target_version" "$target_version" "$target_version"
  say "双槽入口已启用。后续无迁移版本可零停机切换。" "Blue/green ingress is enabled. Later migration-free releases can switch without HTTP downtime."
}

online_update() {
  local active_slot="$1" current_version="$2" target_version="$3"
  local inactive_slot old_worker new_worker old_api old_web new_api new_web blue_version green_version
  if [ "$active_slot" = blue ]; then inactive_slot=green; else inactive_slot=blue; fi
  configure_versions "$active_slot" "$current_version" "$inactive_slot" "$target_version"
  blue_version="$SFORUM_BLUE_VERSION"
  green_version="$SFORUM_GREEN_VERSION"
  old_worker="$(slot_worker "$active_slot")"
  new_worker="$(slot_worker "$inactive_slot")"
  old_api="$(slot_api "$active_slot")"
  old_web="$(slot_web "$active_slot")"
  new_api="$(slot_api "$inactive_slot")"
  new_web="$(slot_web "$inactive_slot")"

  pull_target_images "$target_version"
  verify_target_images "$target_version"
  check_schema_is_online_safe "$target_version"
  backup_database

  say "正在启动并检查备用槽 ${inactive_slot}..." "Starting and checking standby slot ${inactive_slot}..."
  "${COMPOSE[@]}" up -d --no-build --pull never "$new_api" "$new_web"
  if ! wait_inside_slot "$inactive_slot"; then
    "${COMPOSE[@]}" stop "$new_web" "$new_api" || true
    die "Standby slot failed internal health checks; live traffic was not changed."
  fi

  say "正在原子切换 Web/API 流量..." "Atomically switching Web/API traffic..."
  if ! switch_router "$inactive_slot" "$active_slot"; then
    "${COMPOSE[@]}" stop "$new_web" "$new_api" || true
    die "Ingress switch failed and was rolled back to the active slot."
  fi

  say "正在优雅切换后台任务 Worker..." "Gracefully switching the background Worker..."
  "${COMPOSE[@]}" stop "$old_worker"
  if ! "${COMPOSE[@]}" up -d --no-build --pull never "$new_worker"; then
    switch_router "$active_slot" "$inactive_slot" || true
    "${COMPOSE[@]}" start "$old_worker" || true
    die "New Worker failed; HTTP traffic and the old Worker were restored."
  fi
  "${COMPOSE[@]}" ps --status running --services | grep -Fqx "$new_worker" || die "New Worker is not running."
  "${COMPOSE[@]}" stop "$old_web" "$old_api"
  persist_state "$inactive_slot" "$target_version" "$blue_version" "$green_version"
  say "零停机更新完成：$current_version -> $target_version" "Zero-downtime update completed: $current_version -> $target_version"
}

main() {
  [ -f .env.production ] || die ".env.production is missing; run ./deploy.sh first."
  [ -f "$DEPLOY_RC" ] || die ".deployrc is missing; run ./deploy.sh first."
  command -v curl >/dev/null 2>&1 || die "curl is required."

  local current_version active_slot
  current_version="$(rc_value version)"
  [ -n "$current_version" ] || die "The currently deployed version is unknown."
  validate_version "$current_version"
  select_target_version
  confirm_target_version "$current_version"

  command -v docker >/dev/null 2>&1 || die "Docker is required."
  docker info >/dev/null 2>&1 || die "Docker daemon is not running."
  export SFORUM_VERSION="$TARGET_VERSION"
  init_compose
  acquire_lock
  active_slot="$(rc_value active_slot)"
  if [ -z "$active_slot" ]; then
    bootstrap_topology "$current_version" "$TARGET_VERSION"
  else
    [ "$active_slot" = blue ] || [ "$active_slot" = green ] || die "Invalid active_slot in .deployrc"
    online_update "$active_slot" "$current_version" "$TARGET_VERSION"
  fi
  release_lock
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
