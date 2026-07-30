#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

LANGUAGE="zh"
LANGUAGE_SET=false
MODE="auto"
VERSION_INPUT=""
RELEASE_TYPE=""
RELEASE_TYPE_SET=false
ASSUME_YES=false
DRY_RUN=false
LOCAL_CHECKS=false
WAIT_MODE="no-wait"
SHOW_HELP=false

if [[ -t 1 ]]; then
  COLOR_RED=$'\033[31m'
  COLOR_GREEN=$'\033[32m'
  COLOR_YELLOW=$'\033[33m'
  COLOR_BLUE=$'\033[34m'
  COLOR_BOLD=$'\033[1m'
  COLOR_RESET=$'\033[0m'
else
  COLOR_RED=""
  COLOR_GREEN=""
  COLOR_YELLOW=""
  COLOR_BLUE=""
  COLOR_BOLD=""
  COLOR_RESET=""
fi

usage_zh() {
  cat <<'EOF'
SForum 一键发布脚本

用法：
  ./scripts/release.sh
  ./scripts/release.sh 2.8.0 [选项]
  ./scripts/release.sh --version 2.8.0 [选项]

选项：
  --version VERSION       发布版本，例如 2.8.0 或 2.8.0-beta.1
  --type TYPE             发布类型：alpha、beta 或 stable
  --lang zh|en            界面语言（默认：zh）
  --interactive           强制使用交互模式，并自动建议下一个版本
  --non-interactive       禁止提问，版本必须通过参数提供
  --yes, -y               交互模式下跳过最终标签确认
  --dry-run               仅检查并显示计划，不创建或推送标签
  --local-checks          推送前额外执行本地测试和构建（需要本地数据库等依赖）
  --wait                  推送后等待 GitHub Actions 发布流程完成
  --no-wait               推送后立即返回（默认）
  --help, -h              显示帮助

发布原理：
  脚本验证 main 与 origin/main 完全同步后，创建并推送带说明的
  vX.Y.Z 标签。GitHub Actions 负责测试、构建镜像和跨平台发布资产、
  安全扫描、冒烟验证和创建 GitHub Release。交互模式根据最近的远端发布标签建议
  下一版本；脚本不会修改源码中的 dev 版本。

示例：
  ./scripts/release.sh
  ./scripts/release.sh 2.8.0 --dry-run
  ./scripts/release.sh --version 2.8.0 --non-interactive --no-wait
EOF
}

usage_en() {
  cat <<'EOF'
SForum release helper

Usage:
  ./scripts/release.sh
  ./scripts/release.sh 2.8.0 [options]
  ./scripts/release.sh --version 2.8.0 [options]

Options:
  --version VERSION       Release version, such as 2.8.0 or 2.8.0-beta.1
  --type TYPE             Release type: alpha, beta, or stable
  --lang zh|en            Interface language (default: zh)
  --interactive           Force interactive mode and suggest the next version
  --non-interactive       Never prompt; the version must be an argument
  --yes, -y               Skip the final tag confirmation in interactive mode
  --dry-run               Check and show the plan without creating or pushing a tag
  --local-checks          Also run local tests/builds before pushing (requires local services)
  --wait                  Wait for the GitHub Actions release workflow after pushing
  --no-wait               Return immediately after pushing (default)
  --help, -h              Show help

How it works:
  After verifying that main exactly matches origin/main, this script creates
  and pushes an annotated vX.Y.Z tag. GitHub Actions owns testing, image and
  cross-platform asset builds, security scans, smoke tests, and the GitHub Release. Interactive mode suggests
  the next version from the latest remote release tag. The script never changes
  the dev version stored in source code.

Examples:
  ./scripts/release.sh
  ./scripts/release.sh 2.8.0 --dry-run
  ./scripts/release.sh --version 2.8.0 --non-interactive --no-wait
EOF
}

say() {
  local key="$1"
  shift || true
  local message=""

  if [[ "$LANGUAGE" == "en" ]]; then
    case "$key" in
      unknown_option) message="Unknown option: $1" ;;
      missing_value) message="Missing value for $1." ;;
      duplicate_version) message="Specify the version only once." ;;
      invalid_language) message="Unsupported language: $1 (use zh or en)." ;;
      version_required) message="A version is required in non-interactive mode. Use --version 2.8.0." ;;
      select_release_type) message="Select the release type:" ;;
      alpha_description) message="1) alpha  - internal testing; features may be incomplete" ;;
      beta_description) message="2) beta   - public testing; release candidate validation" ;;
      stable_description) message="3) stable - official production release" ;;
      release_type_prompt) message="Enter 1, 2, or 3: " ;;
      invalid_release_type) message="Invalid release type: $1. Choose alpha, beta, or stable." ;;
      release_type_mismatch) message="Release type $1 does not match version $2." ;;
      latest_release) message="Latest release: $1" ;;
      no_previous_release) message="No previous release tag was found; using the initial recommended version." ;;
      enter_base_version) message="Base version [$1] (press Enter to use the default): " ;;
      enter_prerelease_number) message="$1 prerelease number [$2] (press Enter to use the default): " ;;
      invalid_base_version) message="Invalid base version: $1. Use X.Y.Z without a prerelease suffix." ;;
      invalid_prerelease_number) message="Invalid prerelease number: $1. Use a positive integer." ;;
      invalid_version) message="Invalid release version: $1. Use X.Y.Z or X.Y.Z-prerelease; do not use dev, latest, or a branch name." ;;
      error_prefix) message="Error" ;;
      git_required) message="git is required to release SForum." ;;
      not_repo) message="Run this script from an SForum Git working tree." ;;
      origin_required) message="The Git remote 'origin' is missing." ;;
      git_identity_required) message="Git user.name and user.email are required for an annotated release tag. Configure them and try again." ;;
      branch_main) message="Releases must be created from main. Current branch: $1" ;;
      operation_in_progress) message="A merge, rebase, cherry-pick, or revert is in progress. Finish or abort it first." ;;
      dirty_tree) message="The working tree is not clean. Commit, stash, or remove all changes, including untracked files." ;;
      dirty_details) message="Current changes:" ;;
      fetch_main) message="Refreshing origin/main..." ;;
      fetch_failed) message="Could not fetch origin/main. Check the network and remote access." ;;
      upstream_missing) message="origin/main was not found after fetching." ;;
      head_mismatch) message="Local main does not exactly match origin/main. Pull, rebase, or push your commits before releasing." ;;
      local_tag_exists) message="Tag $1 already exists locally." ;;
      remote_tag_exists) message="Tag $1 already exists on origin." ;;
      remote_tag_check_failed) message="Could not check whether tag $1 exists on origin." ;;
      remote_tags_failed) message="Could not read release tags from origin." ;;
      checks_changed_repo) message="The repository changed while release checks were running. Review the changes and run the release again." ;;
      checking_tools) message="Checking release prerequisites" ;;
      checking_git) message="Checking Git state and remote synchronization" ;;
      running_checks) message="Running local release checks" ;;
      checks_delegated) message="Tests, builds, database compatibility, and image validation will run in GitHub Actions." ;;
      command_failed) message="Release check failed: $1" ;;
      summary) message="Release plan" ;;
      summary_version) message="Version" ;;
      summary_tag) message="Git tag" ;;
      summary_channel) message="Channel" ;;
      summary_stable) message="stable" ;;
      summary_prerelease) message="prerelease" ;;
      summary_alpha) message="alpha prerelease" ;;
      summary_beta) message="beta prerelease" ;;
      summary_branch) message="Branch" ;;
      summary_commit) message="Commit" ;;
      summary_checks) message="Release gate" ;;
      summary_run) message="local checks + GitHub Actions" ;;
      summary_github) message="GitHub Actions" ;;
      summary_action) message="Action" ;;
      action_dry_run) message="check only; no tag or push" ;;
      action_release) message="create annotated tag and push it to origin" ;;
      confirm) message="Type $1 to confirm the release: " ;;
      confirmation_failed) message="Confirmation did not match. Nothing was changed." ;;
      dry_run_done) message="Dry run completed. No tag was created or pushed." ;;
      creating_tag) message="Creating annotated tag $1..." ;;
      pushing_tag) message="Pushing $1 to origin..." ;;
      push_failed) message="The tag push failed. The local tag $1 was kept so you can inspect the state safely." ;;
      release_triggered) message="Release $1 has been triggered." ;;
      release_continues) message="Release continues in GitHub Actions. Use --wait only when synchronous status is required." ;;
      actions_url) message="GitHub Actions: $1" ;;
      wait_unavailable) message="GitHub CLI is unavailable or not authenticated, so the script cannot wait for the workflow." ;;
      finding_run) message="Waiting for the GitHub Actions release run to appear..." ;;
      run_not_found) message="The release run did not appear in time. Check GitHub Actions with the link above." ;;
      watching_run) message="Watching GitHub Actions release run $1..." ;;
      release_complete) message="GitHub Actions completed release $1 successfully." ;;
      *) message="$key" ;;
    esac
  else
    case "$key" in
      unknown_option) message="未知选项：$1" ;;
      missing_value) message="$1 缺少参数值。" ;;
      duplicate_version) message="版本只能指定一次。" ;;
      invalid_language) message="不支持的语言：$1（请使用 zh 或 en）。" ;;
      version_required) message="非交互模式必须指定版本，请使用 --version 2.8.0。" ;;
      select_release_type) message="请选择发布类型：" ;;
      alpha_description) message="1) alpha  - 内部测试，功能可能尚未完整" ;;
      beta_description) message="2) beta   - 公开测试，用于发布候选验证" ;;
      stable_description) message="3) stable - 正式生产版本" ;;
      release_type_prompt) message="请输入 1、2 或 3：" ;;
      invalid_release_type) message="发布类型无效：$1。请选择 alpha、beta 或 stable。" ;;
      release_type_mismatch) message="发布类型 $1 与版本 $2 不匹配。" ;;
      latest_release) message="最近发布版本：$1" ;;
      no_previous_release) message="未找到历史发布标签，将使用推荐的初始版本。" ;;
      enter_base_version) message="请输入基础版本 [$1]（直接回车使用默认值）：" ;;
      enter_prerelease_number) message="请输入 $1 预发布编号 [$2]（直接回车使用默认值）：" ;;
      invalid_base_version) message="基础版本无效：$1。请使用不带预发布后缀的 X.Y.Z。" ;;
      invalid_prerelease_number) message="预发布编号无效：$1。请输入正整数。" ;;
      invalid_version) message="发布版本无效：$1。请使用 X.Y.Z 或 X.Y.Z-预发布标识，不要填写 dev、latest 或分支名。" ;;
      error_prefix) message="错误" ;;
      git_required) message="发布 SForum 需要安装 git。" ;;
      not_repo) message="请在 SForum Git 工作区中运行此脚本。" ;;
      origin_required) message="缺少 Git 远端 origin。" ;;
      git_identity_required) message="带说明的发布标签需要 Git user.name 和 user.email。请先配置它们，然后重试。" ;;
      branch_main) message="只能从 main 分支发布，当前分支：$1" ;;
      operation_in_progress) message="当前有合并、变基、拣选或还原操作尚未结束，请先完成或中止它。" ;;
      dirty_tree) message="工作区不干净。请先提交、暂存或清理全部改动，包括未跟踪文件。" ;;
      dirty_details) message="当前改动：" ;;
      fetch_main) message="正在刷新 origin/main..." ;;
      fetch_failed) message="无法获取 origin/main，请检查网络和远端访问权限。" ;;
      upstream_missing) message="刷新后仍找不到 origin/main。" ;;
      head_mismatch) message="本地 main 与 origin/main 不完全一致。请先拉取、变基或推送本地提交。" ;;
      local_tag_exists) message="本地已存在标签 $1。" ;;
      remote_tag_exists) message="远端 origin 已存在标签 $1。" ;;
      remote_tag_check_failed) message="无法检查远端是否存在标签 $1。" ;;
      remote_tags_failed) message="无法从 origin 读取发布标签。" ;;
      checks_changed_repo) message="执行发布检查期间仓库发生了变化。请检查这些改动，然后重新运行发布。" ;;
      checking_tools) message="检查发布工具" ;;
      checking_git) message="检查 Git 状态和远端同步情况" ;;
      running_checks) message="执行本地发布检查" ;;
      checks_delegated) message="测试、构建、数据库兼容验证和镜像验证将统一在 GitHub Actions 中执行。" ;;
      command_failed) message="发布检查失败：$1" ;;
      summary) message="发布计划" ;;
      summary_version) message="版本" ;;
      summary_tag) message="Git 标签" ;;
      summary_channel) message="发布通道" ;;
      summary_stable) message="正式版" ;;
      summary_prerelease) message="预发布版" ;;
      summary_alpha) message="alpha 预发布版" ;;
      summary_beta) message="beta 预发布版" ;;
      summary_branch) message="分支" ;;
      summary_commit) message="提交" ;;
      summary_checks) message="发布门禁" ;;
      summary_run) message="本地检查 + GitHub Actions" ;;
      summary_github) message="GitHub Actions" ;;
      summary_action) message="将要执行" ;;
      action_dry_run) message="仅检查，不创建或推送标签" ;;
      action_release) message="创建带说明的标签并推送到 origin" ;;
      confirm) message="请输入 $1 确认发布：" ;;
      confirmation_failed) message="确认内容不匹配，未执行任何修改。" ;;
      dry_run_done) message="预演完成，没有创建或推送任何标签。" ;;
      creating_tag) message="正在创建带说明的标签 $1..." ;;
      pushing_tag) message="正在向 origin 推送 $1..." ;;
      push_failed) message="标签推送失败。为便于安全排查，本地标签 $1 已保留。" ;;
      release_triggered) message="已触发 $1 发布。" ;;
      release_continues) message="发布流程将在 GitHub Actions 中继续运行；仅在需要同步确认结果时使用 --wait。" ;;
      actions_url) message="GitHub Actions：$1" ;;
      wait_unavailable) message="GitHub CLI 不可用或尚未登录，脚本无法等待工作流。" ;;
      finding_run) message="正在等待 GitHub Actions 发布任务出现..." ;;
      run_not_found) message="暂未发现发布任务，请通过上方链接查看 GitHub Actions。" ;;
      watching_run) message="正在查看 GitHub Actions 发布任务 $1..." ;;
      release_complete) message="GitHub Actions 已成功完成 $1 发布。" ;;
      *) message="$key" ;;
    esac
  fi

  printf '%s' "$message"
}

die() {
  printf '%s%s:%s %s\n' "$COLOR_RED" "$(say error_prefix)" "$COLOR_RESET" "$(say "$@")" >&2
  exit 1
}

step() {
  printf '\n%s==>%s %s%s%s\n' "$COLOR_BLUE" "$COLOR_RESET" "$COLOR_BOLD" "$(say "$1")" "$COLOR_RESET"
}

warn() {
  printf '%s%s%s\n' "$COLOR_YELLOW" "$(say "$@")" "$COLOR_RESET" >&2
}

success() {
  printf '%s%s%s\n' "$COLOR_GREEN" "$(say "$@")" "$COLOR_RESET"
}

ensure_clean_worktree() {
  local worktree_status
  worktree_status="$(git status --porcelain --untracked-files=all)"
  if [[ -n "$worktree_status" ]]; then
    warn dirty_tree
    printf '%s\n%s\n' "$(say dirty_details)" "$worktree_status" >&2
    return 1
  fi
}

ensure_remote_tag_available() {
  local remote_tag_status

  if git show-ref --verify --quiet "refs/tags/$TAG"; then
    die local_tag_exists "$TAG"
  fi

  set +e
  git ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1
  remote_tag_status=$?
  set -e
  case "$remote_tag_status" in
    0) die remote_tag_exists "$TAG" ;;
    2) ;;
    *) die remote_tag_check_failed "$TAG" ;;
  esac
}

validate_and_set_version() {
  local candidate="${VERSION_INPUT#v}"
  local inferred_type="stable"

  if [[ "$candidate" == dev || "$candidate" == dev-* || "$candidate" == latest ]]; then
    die invalid_version "$VERSION_INPUT"
  fi
  if [[ ! "$candidate" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
    die invalid_version "$VERSION_INPUT"
  fi

  if [[ "$candidate" =~ -alpha\.([1-9][0-9]*)$ ]]; then
    inferred_type="alpha"
  elif [[ "$candidate" =~ -beta\.([1-9][0-9]*)$ ]]; then
    inferred_type="beta"
  elif [[ "$candidate" == *-* ]]; then
    inferred_type="prerelease"
  fi

  if [[ "$RELEASE_TYPE_SET" == true ]]; then
    case "$RELEASE_TYPE" in
      stable)
        [[ "$inferred_type" == "stable" ]] || die release_type_mismatch "$RELEASE_TYPE" "$candidate"
        ;;
      alpha|beta)
        [[ "$inferred_type" == "$RELEASE_TYPE" ]] || die release_type_mismatch "$RELEASE_TYPE" "$candidate"
        ;;
    esac
  else
    RELEASE_TYPE="$inferred_type"
  fi

  VERSION="$candidate"
  TAG="v$VERSION"
  case "$RELEASE_TYPE" in
    alpha) CHANNEL="$(say summary_alpha)" ;;
    beta) CHANNEL="$(say summary_beta)" ;;
    stable) CHANNEL="$(say summary_stable)" ;;
    *) CHANNEL="$(say summary_prerelease)" ;;
  esac
}

resolve_release_history() {
  local _object=""
  local ref=""
  local tag=""
  local released_version=""
  local base_version=""
  local major=""
  local minor=""
  local patch=""

  if ! REMOTE_RELEASE_TAGS="$(git ls-remote --tags --refs --sort=-version:refname origin 'refs/tags/v*')"; then
    die remote_tags_failed
  fi

  LATEST_RELEASE=""
  while IFS=$'\t' read -r _object ref; do
    tag="${ref#refs/tags/}"
    if [[ "$tag" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$ ]]; then
      LATEST_RELEASE="$tag"
      break
    fi
  done <<< "$REMOTE_RELEASE_TAGS"

  if [[ -z "$LATEST_RELEASE" ]]; then
    SUGGESTED_BASE_VERSION="0.1.0"
    return
  fi

  released_version="${LATEST_RELEASE#v}"
  base_version="${released_version%%-*}"
  IFS='.' read -r major minor patch <<< "$base_version"

  if [[ "$released_version" != *-* ]]; then
    SUGGESTED_BASE_VERSION="$major.$minor.$((10#$patch + 1))"
  else
    SUGGESTED_BASE_VERSION="$base_version"
  fi
}

suggest_prerelease_number() {
  local base_version="$1"
  local release_type="$2"
  local _object=""
  local ref=""
  local tag=""
  local prefix="v$base_version-$release_type."
  local number=""
  local highest=0

  while IFS=$'\t' read -r _object ref; do
    tag="${ref#refs/tags/}"
    if [[ "$tag" == "$prefix"* ]]; then
      number="${tag#"$prefix"}"
      if [[ "$number" =~ ^[1-9][0-9]*$ ]] && ((10#$number > highest)); then
        highest=$((10#$number))
      fi
    fi
  done <<< "$REMOTE_RELEASE_TAGS"

  SUGGESTED_PRERELEASE_NUMBER=$((highest + 1))
}

select_release_type() {
  local choice=""

  printf '\n%s\n' "$(say select_release_type)"
  printf '  %s\n' "$(say alpha_description)"
  printf '  %s\n' "$(say beta_description)"
  printf '  %s\n' "$(say stable_description)"
  while true; do
    say release_type_prompt
    read -r choice
    case "$choice" in
      1|alpha) RELEASE_TYPE="alpha"; break ;;
      2|beta) RELEASE_TYPE="beta"; break ;;
      3|stable) RELEASE_TYPE="stable"; break ;;
      *) warn invalid_release_type "$choice" ;;
    esac
  done
  RELEASE_TYPE_SET=true
}

set_version() {
  if [[ -n "$VERSION_INPUT" ]]; then
    die duplicate_version
  fi
  VERSION_INPUT="$1"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      [[ $# -ge 2 ]] || die missing_value --version
      set_version "$2"
      shift 2
      ;;
    --lang)
      [[ $# -ge 2 ]] || die missing_value --lang
      LANGUAGE="$2"
      LANGUAGE_SET=true
      shift 2
      ;;
    --type)
      [[ $# -ge 2 ]] || die missing_value --type
      RELEASE_TYPE="$2"
      RELEASE_TYPE_SET=true
      shift 2
      ;;
    --interactive)
      MODE="interactive"
      shift
      ;;
    --non-interactive)
      MODE="non-interactive"
      shift
      ;;
    --yes|-y)
      ASSUME_YES=true
      shift
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --local-checks)
      LOCAL_CHECKS=true
      shift
      ;;
    --skip-checks)
      LOCAL_CHECKS=false
      shift
      ;;
    --wait)
      WAIT_MODE="wait"
      shift
      ;;
    --no-wait)
      WAIT_MODE="no-wait"
      shift
      ;;
    --help|-h)
      SHOW_HELP=true
      shift
      ;;
    --*)
      die unknown_option "$1"
      ;;
    *)
      set_version "$1"
      shift
      ;;
  esac
done

case "$LANGUAGE" in
  zh|en) ;;
  *) die invalid_language "$LANGUAGE" ;;
esac

if [[ "$RELEASE_TYPE_SET" == true ]]; then
  case "$RELEASE_TYPE" in
    alpha|beta|stable) ;;
    *) die invalid_release_type "$RELEASE_TYPE" ;;
  esac
fi

if [[ "$SHOW_HELP" == true ]]; then
  if [[ "$LANGUAGE" == "en" ]]; then
    usage_en
  else
    usage_zh
  fi
  exit 0
fi

if [[ "$MODE" == "auto" ]]; then
  if [[ -t 0 && -t 1 ]]; then
    MODE="interactive"
  else
    MODE="non-interactive"
  fi
fi

if [[ "$MODE" == "interactive" && "$LANGUAGE_SET" == false ]]; then
  printf '请选择界面语言 / Choose language [1=中文, 2=English]（默认/Default: 1）：'
  read -r language_choice
  case "$language_choice" in
    ""|1|zh) LANGUAGE="zh" ;;
    2|en) LANGUAGE="en" ;;
    *) die invalid_language "$language_choice" ;;
  esac
fi

if [[ -z "$VERSION_INPUT" ]]; then
  if [[ "$MODE" == "non-interactive" ]]; then
    die version_required
  fi
  if [[ "$RELEASE_TYPE_SET" == false ]]; then
    select_release_type
  fi
else
  validate_and_set_version
fi

cd "$ROOT_DIR"

step checking_tools
command -v git >/dev/null 2>&1 || die git_required
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die not_repo
git remote get-url origin >/dev/null 2>&1 || die origin_required
[[ -n "$(git config user.name || true)" && -n "$(git config user.email || true)" ]] || die git_identity_required

step checking_git
BRANCH="$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)"
[[ "$BRANCH" == "main" ]] || die branch_main "${BRANCH:-DETACHED_HEAD}"

GIT_DIR="$(git rev-parse --git-dir)"
if git rev-parse -q --verify MERGE_HEAD >/dev/null 2>&1 \
  || git rev-parse -q --verify CHERRY_PICK_HEAD >/dev/null 2>&1 \
  || git rev-parse -q --verify REVERT_HEAD >/dev/null 2>&1 \
  || [[ -d "$GIT_DIR/rebase-merge" || -d "$GIT_DIR/rebase-apply" ]]; then
  die operation_in_progress
fi

ensure_clean_worktree || exit 1

say fetch_main
printf '\n'
if ! git fetch --prune origin main; then
  die fetch_failed
fi
git show-ref --verify --quiet refs/remotes/origin/main || die upstream_missing

HEAD_COMMIT="$(git rev-parse HEAD)"
ORIGIN_COMMIT="$(git rev-parse refs/remotes/origin/main)"
[[ "$HEAD_COMMIT" == "$ORIGIN_COMMIT" ]] || die head_mismatch

if [[ -z "$VERSION_INPUT" ]]; then
  resolve_release_history
  if [[ -n "$LATEST_RELEASE" ]]; then
    say latest_release "$LATEST_RELEASE"
  else
    say no_previous_release
  fi
  printf '\n'
  say enter_base_version "$SUGGESTED_BASE_VERSION"
  read -r base_version_input
  base_version_input="${base_version_input:-$SUGGESTED_BASE_VERSION}"
  if [[ ! "$base_version_input" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    die invalid_base_version "$base_version_input"
  fi

  if [[ "$RELEASE_TYPE" == "stable" ]]; then
    VERSION_INPUT="$base_version_input"
  else
    suggest_prerelease_number "$base_version_input" "$RELEASE_TYPE"
    say enter_prerelease_number "$RELEASE_TYPE" "$SUGGESTED_PRERELEASE_NUMBER"
    read -r prerelease_number_input
    prerelease_number_input="${prerelease_number_input:-$SUGGESTED_PRERELEASE_NUMBER}"
    if [[ ! "$prerelease_number_input" =~ ^[1-9][0-9]*$ ]]; then
      die invalid_prerelease_number "$prerelease_number_input"
    fi
    VERSION_INPUT="$base_version_input-$RELEASE_TYPE.$prerelease_number_input"
  fi
  validate_and_set_version
fi

ensure_remote_tag_available

if [[ "$LOCAL_CHECKS" == true ]]; then
  step running_checks
  CHECK_COMMANDS=(
    "./scripts/test.sh"
    "cd apps/api && go build ./..."
    "cd apps/web && bun test"
    "cd apps/web && bun run build"
  )
  for check_command in "${CHECK_COMMANDS[@]}"; do
    printf '%s$%s %s\n' "$COLOR_BLUE" "$COLOR_RESET" "$check_command"
    if ! bash -c "$check_command"; then
      die command_failed "$check_command"
    fi
  done
else
  printf '\n%s\n' "$(say checks_delegated)"
fi

if ! ensure_clean_worktree || [[ "$(git rev-parse HEAD)" != "$HEAD_COMMIT" ]]; then
  die checks_changed_repo
fi

say fetch_main
printf '\n'
if ! git fetch --prune origin main; then
  die fetch_failed
fi
ORIGIN_COMMIT="$(git rev-parse refs/remotes/origin/main)"
[[ "$HEAD_COMMIT" == "$ORIGIN_COMMIT" ]] || die head_mismatch
ensure_remote_tag_available

step summary
printf '  %-18s %s\n' "$(say summary_version):" "$VERSION"
printf '  %-18s %s\n' "$(say summary_tag):" "$TAG"
printf '  %-18s %s\n' "$(say summary_channel):" "$CHANNEL"
printf '  %-18s %s\n' "$(say summary_branch):" "$BRANCH"
printf '  %-18s %s\n' "$(say summary_commit):" "$HEAD_COMMIT"
if [[ "$LOCAL_CHECKS" == true ]]; then
  printf '  %-18s %s\n' "$(say summary_checks):" "$(say summary_run)"
else
  printf '  %-18s %s\n' "$(say summary_checks):" "$(say summary_github)"
fi
if [[ "$DRY_RUN" == true ]]; then
  printf '  %-18s %s\n' "$(say summary_action):" "$(say action_dry_run)"
else
  printf '  %-18s %s\n' "$(say summary_action):" "$(say action_release)"
fi

if [[ "$DRY_RUN" == true ]]; then
  success dry_run_done
  exit 0
fi

if [[ "$MODE" == "interactive" && "$ASSUME_YES" == false ]]; then
  printf '\n'
  say confirm "$TAG"
  read -r confirmation
  [[ "$confirmation" == "$TAG" ]] || die confirmation_failed
fi

say creating_tag "$TAG"
printf '\n'
git tag -a "$TAG" -m "SForum $VERSION"

say pushing_tag "$TAG"
printf '\n'
if ! git push origin "refs/tags/$TAG"; then
  die push_failed "$TAG"
fi

success release_triggered "$TAG"

REMOTE_URL="$(git remote get-url origin)"
REPOSITORY_URL=""
case "$REMOTE_URL" in
  https://github.com/*.git) REPOSITORY_URL="${REMOTE_URL%.git}" ;;
  https://github.com/*) REPOSITORY_URL="$REMOTE_URL" ;;
  git@github.com:*.git) REPOSITORY_URL="https://github.com/${REMOTE_URL#git@github.com:}"; REPOSITORY_URL="${REPOSITORY_URL%.git}" ;;
  git@github.com:*) REPOSITORY_URL="https://github.com/${REMOTE_URL#git@github.com:}" ;;
esac

if [[ -n "$REPOSITORY_URL" ]]; then
  ACTIONS_URL="$REPOSITORY_URL/actions/workflows/release.yml"
  say actions_url "$ACTIONS_URL"
  printf '\n'
fi

if [[ "$WAIT_MODE" != "wait" ]]; then
  success release_continues
  exit 0
fi

if [[ -z "$REPOSITORY_URL" ]] || ! command -v gh >/dev/null 2>&1 || ! gh auth status >/dev/null 2>&1; then
  warn wait_unavailable
  exit 0
fi

say finding_run
printf '\n'
RUN_ID=""
for _attempt in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20; do
  RUN_ID="$(gh run list --workflow release.yml --branch "$TAG" --event push --limit 1 --json databaseId --jq '.[0].databaseId // empty' 2>/dev/null || true)"
  if [[ -n "$RUN_ID" ]]; then
    break
  fi
  sleep 3
done

if [[ -z "$RUN_ID" ]]; then
  warn run_not_found
  exit 0
fi

say watching_run "$RUN_ID"
printf '\n'
gh run watch "$RUN_ID" --exit-status
success release_complete "$TAG"
