#!/usr/bin/env bash
# sforum-latest.sh — 每次都从 GitHub Release 取回官方部署工具链，再安装或更新。
#
# 与直接执行 ./sforum-bootstrap.sh upgrade 的差别：bootstrap 的“自刷新”由本地
# 旧脚本先解析版本、再替换自己；本脚本不信任本地脚本，先联网下载目标 Release
# 的 sforum-bootstrap.sh，校验 SHA-256（以及可用时的构建来源），然后用它接管。
# 因此本地残留的旧 bootstrap 不参与任何版本或工具链决策。
set -euo pipefail
export LC_ALL=C

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPOSITORY="${SFORUM_RELEASE_REPOSITORY:-zhuchunshu/SForum}"
API_ROOT="https://api.github.com/repos/${REPOSITORY}/releases"
DOWNLOAD_ROOT="${SFORUM_RELEASE_DOWNLOAD_BASE_URL:-https://github.com/${REPOSITORY}/releases/download}"
LOCK_DIR="$ROOT_DIR/.sforum-latest.lock"
BOOTSTRAP="$ROOT_DIR/sforum-bootstrap.sh"

CHANNEL="stable"
TARGET_TAG=""
CHECK_ONLY=false
ACTION=""
BOOTSTRAP_SHA=""
ATTESTATION_STATUS="skipped-no-gh"
LOCK_HELD=false
PASSTHROUGH=()
STAGE=""

usage() {
  cat <<'EOF'
用法: ./sforum-latest.sh [选项]

始终下载 GitHub 上最新的官方 sforum-bootstrap.sh，校验后交给它执行。当前目录
已有 .env.production 与 .deployrc 时执行 upgrade，两者都不存在时执行 install。

选项:
  --check, --dry-run        只解析并校验目标工具链，不替换、不执行
  --channel stable|prerelease
                            选择解析渠道（默认 stable；prerelease 取最新发布版）
  --version VERSION         固定到不可变标签（如 v3.0.9），跳过联网解析
  --yes, --defaults         跳过 bootstrap/upgrade 的确认（无人值守）
  --bootstrap               已安装目录首次做蓝绿入口转换（upgrade 专用）
  --lang zh|en              全新安装时的安装器语言（install 专用）
  -h, --help                显示本帮助

示例:
  ./sforum-latest.sh --check            先看会用什么版本、本地是否一致
  ./sforum-latest.sh                    交互式更新到最新稳定版
  ./sforum-latest.sh --yes              无人值守更新到最新稳定版
  ./sforum-latest.sh --channel prerelease
  ./sforum-latest.sh --version v3.0.9

环境变量:
  SFORUM_RELEASE_REPOSITORY        覆盖仓库（默认 zhuchunshu/SForum）
  SFORUM_RELEASE_DOWNLOAD_BASE_URL 覆盖下载基址（镜像场景）
  SFORUM_SKIP_ATTESTATION=true     跳过 gh 构建来源校验（bootstrap 同样识别）
  https_proxy / http_proxy         访问 GitHub 需要时自行设置

维护窗口迁移由安装目录的 upgrade.sh 直接处理，不经过本脚本：
  ./upgrade.sh --version <TAG> --confirm-maintenance
EOF
}

die() {
  printf 'sforum-latest: %s\n' "$1" >&2
  exit 1
}

cleanup() {
  [ -z "$STAGE" ] || rm -rf "$STAGE"
  if [ "$LOCK_HELD" = true ]; then
    rmdir "$LOCK_DIR" 2>/dev/null || true
    LOCK_HELD=false
  fi
}

acquire_lock() {
  mkdir "$LOCK_DIR" 2>/dev/null || die "同目录已有另一个 sforum-latest 正在运行"
  LOCK_HELD=true
}

passthrough_has() {
  local needle="$1" item
  for item in ${PASSTHROUGH[@]+"${PASSTHROUGH[@]}"}; do
    [ "$item" = "$needle" ] && return 0
  done
  return 1
}

normalize_tag() {
  local tag="$1"
  case "$tag" in
    v[0-9]*) ;;
    [0-9]*) tag="v$tag" ;;
  esac
  [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]] || \
    die "版本标签格式不正确: $tag"
  printf '%s' "$tag"
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --check|--dry-run) CHECK_ONLY=true; shift ;;
      --channel)
        [ "$#" -ge 2 ] || die "--channel 需要一个值"
        CHANNEL="$2"
        shift 2
        ;;
      --version)
        [ "$#" -ge 2 ] || die "--version 需要一个值"
        [ -z "$TARGET_TAG" ] || die "只指定一次目标版本"
        TARGET_TAG="$(normalize_tag "$2")"
        shift 2
        ;;
      --version=*)
        [ -z "$TARGET_TAG" ] || die "只指定一次目标版本"
        TARGET_TAG="$(normalize_tag "${1#*=}")"
        shift
        ;;
      --lang)
        [ "$#" -ge 2 ] || die "--lang 需要一个值"
        PASSTHROUGH+=("$1" "$2")
        shift 2
        ;;
      --lang=*) PASSTHROUGH+=("$1"); shift ;;
      --yes|--defaults|--bootstrap) PASSTHROUGH+=("$1"); shift ;;
      --confirm-maintenance|--confirm-maintenance=*)
        die "维护窗口参数由 upgrade.sh 直接处理：./upgrade.sh --version <TAG> --confirm-maintenance"
        ;;
      -h|--help) usage; exit 0 ;;
      -*) die "未知选项: $1" ;;
      *)
        [ -z "$TARGET_TAG" ] || die "只指定一次目标版本"
        TARGET_TAG="$(normalize_tag "$1")"
        shift
        ;;
    esac
  done
  [ "$CHANNEL" = stable ] || [ "$CHANNEL" = prerelease ] || \
    die "--channel 只能是 stable 或 prerelease"
}

detect_action() {
  if [ -f "$ROOT_DIR/.env.production" ] && [ -f "$ROOT_DIR/.deployrc" ]; then
    ACTION=upgrade
  elif [ ! -f "$ROOT_DIR/.env.production" ] && [ ! -f "$ROOT_DIR/.deployrc" ]; then
    ACTION=install
  else
    ACTION=incomplete
  fi
}

resolve_tag_from_redirect() {
  local location tag
  location="$(curl -fsSI --connect-timeout 10 --max-time 30 \
      "https://github.com/${REPOSITORY}/releases/latest/download/sforum-bootstrap.sh" 2>/dev/null \
    | tr -d '\r' \
    | sed -n 's/^[Ll]ocation:[[:space:]]*//p' \
    | sed -n '1p')"
  [ -n "$location" ] || return 1
  tag="$(printf '%s\n' "$location" | sed -n 's|.*/releases/download/\([^/]*\)/.*|\1|p')"
  [ -n "$tag" ] || return 1
  printf '%s' "$tag"
}

resolve_tag() {
  local api_url json tag
  if [ -n "$TARGET_TAG" ]; then
    printf '%s' "$TARGET_TAG"
    return 0
  fi

  if [ "$CHANNEL" = stable ]; then
    # 未显式指定 API 时，先从 releases/latest 的 302 Location 取标签，
    # 这样一次更新完全不消耗 GitHub API 配额。取不到再回退 API。
    if [ -z "${SFORUM_LATEST_RELEASE_API_URL:-}" ]; then
      if tag="$(resolve_tag_from_redirect)"; then
        normalize_tag "$tag"
        return 0
      fi
    fi
    api_url="${SFORUM_LATEST_RELEASE_API_URL:-$API_ROOT/latest}"
  else
    api_url="${SFORUM_RELEASES_API_URL:-$API_ROOT?per_page=1}"
  fi

  json="$(curl -fsSL --connect-timeout 10 --max-time 30 \
    -H 'Accept: application/vnd.github+json' \
    -H 'X-GitHub-Api-Version: 2022-11-28' \
    -H 'User-Agent: SForum-latest' \
    "$api_url")" || die "查询 GitHub Release 失败：检查网络或代理，或用 --version 指定标签"

  tag="$(printf '%s\n' "$json" \
    | grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' \
    | head -n 1 \
    | sed 's/.*"\([^"]*\)"$/\1/' || true)"
  [ -n "$tag" ] || die "GitHub 没有返回可用的 Release 标签"
  normalize_tag "$tag"
}

stage_bootstrap() {
  local tag="$1" sha
  STAGE="$(mktemp -d "$ROOT_DIR/.sforum-latest.XXXXXX")"

  curl -fsSLo "$STAGE/sforum-bootstrap.sh" --connect-timeout 10 --max-time 120 \
    "$DOWNLOAD_ROOT/$tag/sforum-bootstrap.sh" || \
    die "下载 sforum-bootstrap.sh 失败（$tag）"
  curl -fsSLo "$STAGE/SHA256SUMS" --connect-timeout 10 --max-time 120 \
    "$DOWNLOAD_ROOT/$tag/SHA256SUMS" || \
    die "下载 SHA256SUMS 失败（$tag）"

  (
    cd "$STAGE"
    awk '$2 == "sforum-bootstrap.sh" { print }' SHA256SUMS > sforum-bootstrap.sha256
    [ "$(wc -l < sforum-bootstrap.sha256 | tr -d '[:space:]')" = 1 ]
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum -c sforum-bootstrap.sha256
    else
      shasum -a 256 -c sforum-bootstrap.sha256
    fi
  ) >/dev/null || die "sforum-bootstrap.sh 的 SHA-256 校验失败（$tag）"

  bash -n "$STAGE/sforum-bootstrap.sh" || die "下载的 bootstrap 语法检查失败（$tag）"

  ATTESTATION_STATUS="skipped-no-gh"
  if [ "${SFORUM_SKIP_ATTESTATION:-false}" = true ]; then
    ATTESTATION_STATUS="skipped-env"
  elif command -v gh >/dev/null 2>&1; then
    if gh attestation --help >/dev/null 2>&1; then
      if gh attestation verify "$STAGE/sforum-bootstrap.sh" --repo "$REPOSITORY" >/dev/null; then
        ATTESTATION_STATUS="verified"
      else
        die "bootstrap 构建来源校验失败（$tag）"
      fi
    else
      # 旧版 gh 没有 attestation 子命令；同步给 bootstrap，避免它内部的同样调用失败。
      ATTESTATION_STATUS="skipped-unsupported-gh"
      export SFORUM_SKIP_ATTESTATION=true
      printf '提示: 本机 gh 不支持 attestation，已跳过构建来源校验（SHA-256 已通过）\n' >&2
    fi
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    sha="$(sha256sum "$STAGE/sforum-bootstrap.sh" | awk '{print $1}')"
  else
    sha="$(shasum -a 256 "$STAGE/sforum-bootstrap.sh" | awk '{print $1}')"
  fi
  BOOTSTRAP_SHA="$sha"
}

print_attestation() {
  case "$ATTESTATION_STATUS" in
    verified) printf '构建来源: gh attestation 校验通过\n' ;;
    skipped-env) printf '构建来源: 按 SFORUM_SKIP_ATTESTATION 跳过\n' ;;
    skipped-no-gh) printf '构建来源: 未安装 gh，仅做 SHA-256 校验\n' ;;
    *) printf '构建来源: 本机 gh 不支持 attestation，仅做 SHA-256 校验\n' ;;
  esac
}

report() {
  local tag="$1"
  printf '目标 Release: %s\n' "$tag"
  printf 'bootstrap SHA-256: %s\n' "$BOOTSTRAP_SHA"
  print_attestation
  if [ ! -f "$BOOTSTRAP" ]; then
    printf '本地 bootstrap: 缺失，执行前写入\n'
  elif cmp -s "$STAGE/sforum-bootstrap.sh" "$BOOTSTRAP"; then
    printf '本地 bootstrap: 与目标一致\n'
  else
    printf '本地 bootstrap: 与目标不同，执行前替换（旧文件备份到 .sforum/bootstrap-backups/）\n'
  fi
  case "$ACTION" in
    install) printf '安装状态: 未安装，计划动作 install\n' ;;
    upgrade) printf '安装状态: 已安装，计划动作 upgrade\n' ;;
    *) printf '安装状态: 不完整（.env.production 与 .deployrc 需同时存在），需要先人工修复\n' ;;
  esac
}

promote_bootstrap() {
  local tag="$1" backup_dir tmp_file
  if [ -f "$BOOTSTRAP" ] && cmp -s "$STAGE/sforum-bootstrap.sh" "$BOOTSTRAP"; then
    return 0
  fi
  if [ -f "$BOOTSTRAP" ]; then
    backup_dir="$ROOT_DIR/.sforum/bootstrap-backups/$(date -u +%Y%m%dT%H%M%SZ)-${tag}"
    mkdir -p "$backup_dir"
    cp -p "$BOOTSTRAP" "$backup_dir/sforum-bootstrap.sh"
    printf '旧 bootstrap 备份: %s\n' "$backup_dir/sforum-bootstrap.sh"
  fi
  # 同目录临时文件 + mv，保证替换是原子的；不依赖额外工具。
  tmp_file="$(mktemp "$ROOT_DIR/.sforum-bootstrap.XXXXXX")"
  cp "$STAGE/sforum-bootstrap.sh" "$tmp_file"
  chmod 0755 "$tmp_file"
  mv -f "$tmp_file" "$BOOTSTRAP"
  printf '已写入 %s\n' "$BOOTSTRAP"
}

main() {
  trap cleanup EXIT
  trap 'exit 130' HUP INT TERM

  parse_args "$@"
  detect_action

  if [ "$ACTION" = upgrade ] && passthrough_has --lang; then
    die "--lang 只用于 install"
  fi
  if [ "$ACTION" = install ] && passthrough_has --bootstrap; then
    die "--bootstrap 只用于 upgrade"
  fi
  if [ "$ACTION" = incomplete ] && [ "$CHECK_ONLY" != true ]; then
    die "安装状态不完整：先执行 ./deploy.sh --action deploy --version <TAG> 修复"
  fi

  command -v curl >/dev/null 2>&1 || die "需要 curl"
  command -v tar >/dev/null 2>&1 || die "需要 tar"

  TARGET_TAG="$(resolve_tag)"
  stage_bootstrap "$TARGET_TAG"

  if [ "$CHECK_ONLY" = true ]; then
    report "$TARGET_TAG"
    exit 0
  fi

  printf '目标 Release: %s\n' "$TARGET_TAG"
  printf 'bootstrap SHA-256: %s\n' "$BOOTSTRAP_SHA"
  print_attestation
  acquire_lock
  promote_bootstrap "$TARGET_TAG"

  cleanup
  trap - EXIT HUP INT TERM
  printf '执行: ./sforum-bootstrap.sh %s --version %s\n' "$ACTION" "$TARGET_TAG"
  exec "$BOOTSTRAP" "$ACTION" --version "$TARGET_TAG" ${PASSTHROUGH[@]+"${PASSTHROUGH[@]}"}
}

main "$@"
