# 生产部署

[← 中文文档首页](./README.md)

## 快速安装

支持 Linux `amd64` 和 `arm64`，不支持 Windows。服务器需要：

- Docker Engine 及 Docker Compose `2.24.4` 或更高版本；
- `curl`、`tar` 和标准 `install` 命令；
- 能访问 GitHub 和 `ghcr.io`；
- loopback 的 `3000`、`18080` 端口可用，或在向导中选择其他端口。

不需要安装 Git 或克隆仓库。下面的命令下载并校验**最新稳定版**的
`sforum-bootstrap.sh`；bootstrap 随后会刷新同一目标版本的完整部署工具包：

```sh
(
  set -eu
  mkdir -p sforum
  cd sforum
  bootstrap_dir="$(mktemp -d .sforum-bootstrap.XXXXXX)"
  trap 'rm -rf "$bootstrap_dir"' EXIT HUP INT TERM
  curl -fsSLo "$bootstrap_dir/sforum-bootstrap.sh" \
    https://github.com/zhuchunshu/SForum/releases/latest/download/sforum-bootstrap.sh
  curl -fsSLo "$bootstrap_dir/SHA256SUMS" \
    https://github.com/zhuchunshu/SForum/releases/latest/download/SHA256SUMS
  (
    cd "$bootstrap_dir"
    awk '$2 == "sforum-bootstrap.sh" { print }' SHA256SUMS > sforum-bootstrap.sha256
    test "$(wc -l < sforum-bootstrap.sha256 | tr -d '[:space:]')" = 1
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum -c sforum-bootstrap.sha256
    else
      shasum -a 256 -c sforum-bootstrap.sha256
    fi
    if command -v gh >/dev/null 2>&1; then
      gh attestation verify sforum-bootstrap.sh --repo zhuchunshu/SForum
    fi
  )
  install -m 0755 "$bootstrap_dir/sforum-bootstrap.sh" ./sforum-bootstrap.sh
  rm -rf "$bootstrap_dir"
  trap - EXIT HUP INT TERM
  ./sforum-bootstrap.sh install
)
```

这个 URL 始终指向最新稳定 Release 的 bootstrap，不需要维护版本号。它会先
解析不可变目标标签，再校验该标签的 `sforum-deploy.tar.gz`，最后只安装匹配的
Compose 文件、入口脚本、生产环境示例和 `deploy/` 辅助脚本。

### 校验下载（必需）

每个 Release 都发布 `SHA256SUMS`（覆盖全部资产）和 GitHub 构建来源证明。

- **禁止把远程响应通过管道直接交给 shell**：必须先保存 bootstrap，再下载
  校验文件。
- **只校验 `SHA256SUMS` 中 `sforum-bootstrap.sh` 的精确条目**，并要求结果
  恰好只有一条。bootstrap 内部会用同样规则校验完整部署包。
- 整段命令在 `set -eu` 子 shell 中运行；下载失败、条目缺失/重复、checksum
  或来源证明失败都会在任何远程脚本执行前终止。
- `gh attestation verify` 为可选步骤，用于核对构建来源（需要 GitHub CLI 且
  已认证）。

### 通道语义（stable / prerelease）

- **默认通道是 `stable`**：bootstrap 的 install/upgrade 都只解析最新
  **稳定** Release。
- **预发布版必须显式选择**：使用 `--channel prerelease`，或直接指定
  不可变版本 `--version v3.0.0-alpha.N`。
- 无论哪种选择，脚本都会在拉取镜像前解析为具体的 `vX.Y.Z` 标签，并运行
  对应的 GHCR 镜像；生产 Compose 不会直接运行浮动的 `latest` 镜像标签。
- 若当前还没有任何稳定 Release，`latest` 解析会失败并提示改用
  `--channel prerelease` 或显式版本；这是预期的 fail-closed 行为。

需要可重复部署时显式固定版本（用你要部署的版本替换 `$SFORUM_VERSION`）：

```sh
./sforum-bootstrap.sh install --version $SFORUM_VERSION
./sforum-bootstrap.sh upgrade --version $SFORUM_VERSION
```

## 目标形态

- Docker Compose 编排：`web`、始终内嵌 Worker 的 `api`、PostgreSQL、Redis
- 对外：仅 **loopback** 发布 Web（及可选 API WebSocket 入口），TLS 由**宿主机反代**负责
- 同域：浏览器只打到站点域名；普通 `/api/v1/*` 由 Nuxt 反代到 API；WebSocket Upgrade 可直打 API 端口

## 配置

推荐运行 `./sforum-bootstrap.sh install`。刷新完整工具包后，安装向导会用
中文解释每个选项，并自动生成
PostgreSQL、Redis、会话、验证码、身份 HMAC、敏感选项加密和 Marketplace
验签密钥。所有问题直接回车即可得到一套能启动的本机配置；生成的
`.env.production` 权限为 `0600`，秘密不会打印到终端。

入门向导固定使用 Compose 内置 PostgreSQL 和 Redis，不需要另外准备数据库
或填写连接字符串。外部数据库属于高级部署，这一版向导暂不开放，避免更新
时错误备份内置空库。

默认 `APP_URL` 是本机验证地址。公网部署应重新运行配置向导或编辑
`.env.production`，配置真实 HTTPS 地址和宿主机 Caddy/Nginx。向导生成的
`deployment-local-untrusted` Marketplace 公钥只用于让未知索引保持锁定；
接入官方 Marketplace 时必须替换为官方公钥与 key ID，私钥不得进入仓库或
容器。

## HTTPS 反向代理

SForum 只监听 loopback 端口，TLS 由宿主机反向代理终止。宿主机
Caddy/Nginx 示例见部署包内的 `deploy/caddy/Caddyfile`：

- 代理 Web 目标 `http://127.0.0.1:${WEB_PORT:-3000}`（同域 HTTP API 由 Nuxt 转发）；
- 可选：把 `/api/v1/` 的 WebSocket Upgrade 直打 API 目标
  `http://127.0.0.1:${API_PORT:-18080}`；
- 配置 `TRUST_PROXY` 与精确的 `TRUSTED_PROXIES` 以便信任反代传递的客户端 IP
  （不要无脑信任所有网段）。

## 维护者发布

维护者从已同步到 `origin/main` 的干净 `main` 工作区创建版本：

```sh
./scripts/release.sh
```

脚本推送版本标签后默认立即返回，发布状态继续由 GitHub Actions 管理。只有
需要占用当前终端同步确认结果时才使用 `--wait`；`--no-wait` 是默认行为。
交互发布可填写一行面向用户的"发布重点"，直接回车则只使用 GitHub 自动
生成的完整变更记录。多行 Markdown 可通过
`./scripts/release.sh 2.8.0 --notes-file /tmp/release-notes.md` 提供；手写重点会
显示在自动变更记录之前。
Release 会在 GitHub 端等待并复用相同提交已经触发的 `main` CI，只有精确
SHA 的 `push` CI 成功后才构建、扫描和提升发布镜像，不会再为标签重复运行
整套仓库门禁。镜像通过扫描和 Compose 冒烟后，GitHub Release 同时发布：

- `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64` 的
  `sforum` 管理 CLI；Windows 不在 SForum 的支持平台范围内；
- `linux/amd64` 和 `linux/arm64` 后端运行包，包含 API、迁移、CLI
  以及从同一已扫描候选镜像提取的精确内置扩展；
- 固定名称 **`sforum-bootstrap.sh`**、部署包 **`sforum-deploy.tar.gz`** 与
  旧安装兼容用 **`upgrade.sh`**；
- 覆盖全部压缩包的 `SHA256SUMS` 和 GitHub 构建来源证明。

Linux 后端包不包含 Nuxt Web、PostgreSQL 或 Redis，因此不是完整站点安装包；
生产部署仍推荐使用下方三个版本一致的 Docker 镜像。下载后可使用
`gh attestation verify <文件> --repo zhuchunshu/SForum` 验证构建来源，并按
`SHA256SUMS` 验证文件摘要。

## 部署入口

### 使用正式发布镜像（推荐）

正式版本发布到 GitHub Container Registry，包含以下镜像：

- `ghcr.io/zhuchunshu/sforum-api`
- `ghcr.io/zhuchunshu/sforum-migrate`
- `ghcr.io/zhuchunshu/sforum-web`

每个发布版本同时提供 `linux/amd64` 和 `linux/arm64`。最简单的交互安装如下，
全部直接回车即可完成配置并部署：

```sh
./sforum-bootstrap.sh install
```

也可以显式固定版本，或完全非交互地接受推荐配置（`$SFORUM_VERSION` 替换为
实际标签，例如 `v3.0.0-alpha.13` 这样的预发布版本）：

```sh
./sforum-bootstrap.sh install --version $SFORUM_VERSION --lang zh
./sforum-bootstrap.sh install --version $SFORUM_VERSION --lang en
./sforum-bootstrap.sh install --version $SFORUM_VERSION --lang zh --yes
```

预发布通道的非交互示例（解析最新发布，含预发布）：

```sh
./sforum-bootstrap.sh install --channel prerelease --lang zh --yes
```

该模式组合 `compose.yaml`、`compose.prod.yaml` 与
`compose.release.yaml`。脚本先拉取同版本的四个 GHCR 镜像，成功后才启动
内置 PostgreSQL/Redis；全新库跳过备份，已有安装先备份，再停止旧应用、
迁移并以 `--no-build` 启动。需要 Docker Compose 2.24.4 或更高版本。
API `/api/v1/ready`、Web 首页和五个常驻服务全部通过后，脚本才记录成功
版本，并打印 Web 反代目标、API/WebSocket 反代目标、站点访问地址和管理后台地址。
首次配置时，向导还会询问管理后台路径前缀（默认 `/control-panel`）。

脚本会在改动数据库前检查配置占位密钥、端口冲突和三个 Go 镜像的构建版本，
并使用部署锁阻止两个更新并发执行。预拉取完成后，迁移和启动都使用
`--pull never`，因此旧应用停止后不会再次依赖 Registry。若迁移或健康检查
失败，`.deployrc` 会记录 `status=recovery_required`、目标版本、前一版本和
备份路径；脚本不会把该实例误报为部署成功。

等价的非交互命令（`<VERSION>` 必须是解析后的具体标签，例如
`v3.0.0-alpha.13`）：

```sh
SFORUM_VERSION=<VERSION>
export SFORUM_VERSION
docker compose --env-file .env.production \
  -f compose.yaml -f compose.prod.yaml -f compose.release.yaml pull
docker compose --env-file .env.production \
  -f compose.yaml -f compose.prod.yaml -f compose.release.yaml run --rm -T migrate
docker compose --env-file .env.production \
  -f compose.yaml -f compose.prod.yaml -f compose.release.yaml up -d --no-build
```

四个 GHCR 包必须公开可读并关联到本仓库。发布流水线在创建 GitHub Release
前，会使用空的 Docker 凭据目录匿名拉取该版本的三个镜像；任一包不可公开
拉取都会阻断发布。首次创建包时，流水线可能先停在这个门禁：管理员将三个
包改为 Public 后重跑失败作业即可。写入仍使用仓库 `GITHUB_TOKEN`，不需要
长期 Registry 密钥。

`deploy.sh` 只走发布镜像，避免新手意外在服务器上源码构建。开发或定制源码
请使用开发文档中的 `scripts/dev.sh` 和构建命令。

## 零停机更新

已有 Compose 安装推荐使用 bootstrap 更新。它每次都会先把自身刷新到目标
Release，再校验并刷新该 Release 的完整部署工具包，最后才交给
`upgrade.sh`。默认选择 **最新稳定版**：

```sh
./sforum-bootstrap.sh upgrade
./sforum-bootstrap.sh upgrade --version $SFORUM_VERSION
./sforum-bootstrap.sh upgrade --yes
./sforum-bootstrap.sh upgrade --channel prerelease
./sforum-bootstrap.sh upgrade --channel prerelease --yes
```

预发布版永远不会被隐式选中：要么显式传 `--channel prerelease`，要么直接
指定 `v3.0.0-alpha.N` 这样的不可变标签。无论哪种方式，脚本都会解析为具体
标签并运行对应镜像；`.deployrc` 保存的也是解析后的具体版本，不会保存
`latest`。

现有安装不需要重新克隆仓库。第一次接入 bootstrap 时，在现有安装目录从
**最新稳定 Release 资产**下载并校验（不要使用 `main` 分支的浮动内容）：

```sh
(
  set -eu
  cd /path/to/sforum
  bootstrap_dir="$(mktemp -d .sforum-bootstrap.XXXXXX)"
  trap 'rm -rf "$bootstrap_dir"' EXIT HUP INT TERM
  curl -fsSLo "$bootstrap_dir/sforum-bootstrap.sh" \
    https://github.com/zhuchunshu/SForum/releases/latest/download/sforum-bootstrap.sh
  curl -fsSLo "$bootstrap_dir/SHA256SUMS" \
    https://github.com/zhuchunshu/SForum/releases/latest/download/SHA256SUMS
  (
    cd "$bootstrap_dir"
    awk '$2 == "sforum-bootstrap.sh" { print }' SHA256SUMS > sforum-bootstrap.sha256
    test "$(wc -l < sforum-bootstrap.sha256 | tr -d '[:space:]')" = 1
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum -c sforum-bootstrap.sha256
    else
      shasum -a 256 -c sforum-bootstrap.sha256
    fi
    if command -v gh >/dev/null 2>&1; then
      gh attestation verify sforum-bootstrap.sh --repo zhuchunshu/SForum
    fi
  )
  install -m 0755 "$bootstrap_dir/sforum-bootstrap.sh" ./sforum-bootstrap.sh
  rm -rf "$bootstrap_dir"
  trap - EXIT HUP INT TERM
  ./sforum-bootstrap.sh upgrade
)
```

这个过程不会修改 `.env.production`、`.deployrc`、`deploy/runtime/` 或数据卷；
刷新前的受管工具文件会备份到 `.sforum/tooling-backups/`。固定具体版本（含
预发布）时，将上面的两个滚动地址改为对应标签：

```sh
https://github.com/zhuchunshu/SForum/releases/download/<TAG>/sforum-bootstrap.sh
https://github.com/zhuchunshu/SForum/releases/download/<TAG>/SHA256SUMS
```

> 历史兼容说明：`v3.0.0-alpha.12` 及更早版本附带的 `upgrade.sh` 无法正确
> 启动数据库兼容预检，且旧的 `latest` 语义包含预发布；请使用
> `v3.0.0-alpha.13` 或更高版本附带的脚本（历史安装目录需要先固定下载修复后
> 的脚本，再执行更新；该不可变标签是历史事实，不会改成 main/latest）。

```sh
(
  set -eu
  curl -fsSLo upgrade.sh \
    https://raw.githubusercontent.com/zhuchunshu/SForum/v3.0.0-alpha.13/upgrade.sh
  printf '%s  %s\n' ae186e13ca9551014e21ce7f77a7335413791268d97fb0b72f6be9820dedfe13 upgrade.sh > upgrade.sh.sha256
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c upgrade.sh.sha256
  else
    shasum -a 256 -c upgrade.sh.sha256
  fi
  chmod 0755 upgrade.sh
  ./upgrade.sh v3.0.0-alpha.13
)
```

这里的 `latest` 不是浮动的容器镜像标签。脚本在稳定通道查询 GitHub
`/releases/latest`（只返回稳定 Release），预发布通道查询 Release 列表，
再解析为具体的 `vX.Y.Z` tag。执行前会输出当前实际版本和目标实际版本，并
询问是否使用该版本；只有显式传入 `--yes` 才跳过输入与确认。

第一次从旧版直连端口拓扑运行时，脚本会要求确认一次性 blue/green 入口
转换。该转换需要停止旧服务并启动稳定 Caddy 入口，因此会有一次短暂维护
窗口。转换完成后，对于数据库不变或只包含已声明向后兼容 Core 迁移的版本，
脚本会保持当前槽服务，先备份数据库并执行受限在线迁移，再启动备用槽检查
API/Web，经 Caddy 原子切换流量后才停止旧槽，从而保持 HTTP 服务连续。
WebSocket 长连接在切换时可能需要自动重连。

蓝绿升级的每个 API 槽都内嵌 Worker。备用 API 健康检查期间可能与当前槽短暂
同时消费队列，任务领取和重试由 River 的数据库锁保证；切换后脚本停止旧 API，
不会启动或保留独立 Worker。更新前脚本同时检查 SForum Core 与 River 的数据库
迁移。只有目标迁移器声明支持
在线检查、所有待执行 Core SQL 都带有 Host 审核的 `-- +sforum OnlineSafe`
声明，并且 River 迁移完全一致时，才会在旧槽继续服务期间迁移。在线 SQL
必须在事务中设置有限的 `lock_timeout` 与 `statement_timeout`；执行失败时旧槽
继续提供服务，脚本不会切流量。未声明 Core 迁移、任何 River 迁移和旧版迁移
镜像仍会拒绝在线更新，此时使用 `./deploy.sh --version <版本>` 接受维护窗口。
维护部署能够识别已有 blue/green `edge` 与槽位服务，先备份，再统一停止旧槽、
迁移并启动直连目标服务，不会把受管入口端口误判为外部占用。

## 健康检查、日志与故障恢复

### 健康检查

| 检查 | 地址 | 用途 |
| --- | --- | --- |
| Web 健康 | `http://127.0.0.1:${WEB_PORT:-3000}/health` | 容器存活与启动完成 |
| API 存活 | `http://127.0.0.1:${API_PORT:-18080}/api/v1/health` | 进程存活 |
| API 就绪 | `http://127.0.0.1:${API_PORT:-18080}/api/v1/ready` | PostgreSQL 依赖就绪（Redis/Meili 降级就绪） |

`deploy.sh` 与 `upgrade.sh` 在成功记录前都会执行这些检查；
`SFORUM_DEPLOY_HEALTH_TIMEOUT_SECONDS` / `SFORUM_UPGRADE_HEALTH_TIMEOUT_SECONDS`
可调整等待时间。

### 日志

```sh
./deploy.sh --action logs          # 跟踪全部服务日志
./deploy.sh --action status        # 查看服务状态
docker compose --env-file .env.production logs -f api web
```

### 故障恢复

- **部署失败**：`.deployrc` 记录 `status=recovery_required`、尝试版本、之前
  版本与备份路径；先查看 `./deploy.sh --action logs`，修复原因后重试。
- **数据库回滚**：恢复用 `SFORUM_CONFIRM_RESTORE=RESTORE` 显式确认（见下）。
- **版本回退**：确认迁移兼容后，用较早的不可变版本重新部署
  `./deploy.sh --version <旧版本>`。
- **Safe Mode / 带外恢复**：扩展导致启动失败时，用
  `sforum extension disable` / `disable-all` / `quarantine` 带外恢复，
  见[开发者 CLI](./development/cli.md)与[扩展与主题](./usage/extensions.md)。

## 端口（生产示例）

| 用途 | 默认（见 `.env.production.example`） |
| --- | --- |
| Web（loopback） | `127.0.0.1:${WEB_PORT:-3000}` |
| API WebSocket 入口（loopback） | `127.0.0.1:${API_PORT:-18080}` |

宿主机 Caddy/Nginx 示例见 `deploy/caddy/Caddyfile`。

若信任反代传递客户端 IP：配置 `TRUST_PROXY` 与精确的 `TRUSTED_PROXIES`（不要无脑信任所有网段）。

## 进程职责

| 服务 | 职责 |
| --- | --- |
| `web` | Nuxt 生产输出；同域代理 HTTP API |
| `api` | Fiber API；扩展运行时；内嵌 River Worker；可选 WS 入口 |
| `postgres` / `redis` | 状态与会话/缓存 |

生产和开发均由 API 始终承载 Worker，Worker 所有权没有环境变量开关。API 与
后台任务复用 PostgreSQL 连接池、Redis、SettingsLifecycle、SecretStore 和扩展
runtime，避免两套进程读取不同配置。更新旧版本时，部署脚本会按 Compose 标签
删除遗留的 `worker`、`worker-blue` 与 `worker-green` 容器。

蓝绿槽位同样由各自 API 内嵌 Worker。候选槽启动后 River 允许短暂存在两个
消费者；流量切换完成后停止旧 API，也会同时停止旧槽的 Worker。

内嵌模式下 HTTP 与后台任务共用 `DATABASE_MAX_CONNS`。队列持续繁忙、需要
任务延迟开始影响请求延迟时，应提高该值并观察 PostgreSQL，或降低对应队列
并发，而不是无边界提高 River 并发。

## 运行时内存与诊断

管理台 `/control-panel` 的资源卡通过
`GET /api/v1/admin/overview/resources` 读取 API 与直属插件进程的 CPU、RSS
和可用的 PSS。Linux 正式镜像直接读取 `/proc`，不依赖 BusyBox `ps`，也不
需要宿主机 PID namespace 或 Docker socket。资源请求最多每 5 秒共享一次采样，
RSS 与完整 PSS 都展示最近 60 秒中位数；不支持 PSS 的系统不会伪造"有效占用"。
Linux 同时报告当前 `AnonHugePages`，便于确认插件是否仍有透明大页放大。插件
明细按 RSS 从高到低列出，并只归因当前 API/Worker 直接拥有的插件进程。

API 行明确标记"含 Worker"；Worker 行只显示内嵌并发槽位和运行任务数，不
虚构一个独立 Worker 的 MiB。

Go pprof 是显式 opt-in 的本机诊断面，默认关闭，且启动器拒绝非 loopback
监听地址：

```sh
# 临时开启 API 诊断（Worker 已包含在同一 profile）
PPROF_ENABLED=true PPROF_ADDR=127.0.0.1:6060
```

启用后可用 `http://127.0.0.1:6060/debug/pprof/` 采集 API 与 Worker profile。
不要把该端口发布到公网或交给反向代理；采集完成后
删除开关并重启进程。`GOMEMLIMIT` 可作为 Go runtime 的软堆上限，例如
`GOMEMLIMIT=512MiB`，但它不是插件进程或整个容器的硬 RSS 上限；插件和
容器仍应配置各自的资源限制。

## 备份

仓库提供 `deploy/scripts/` 下的 PostgreSQL 备份/恢复辅助脚本。备份先写入
临时文件，成功后才发布为 `0600` 的 `.sql` 文件；失败不会留下可被误用的
半截备份。

```sh
./deploy.sh --action backup
```

恢复需要显式设置 `SFORUM_CONFIRM_RESTORE=RESTORE`。脚本会停止原先运行的
API/Worker，在独立临时数据库中使用 `ON_ERROR_STOP=1` 和单事务完整恢复，
校验应用表后才原子切换数据库名；任一 SQL 错误都会返回失败，原目标库不会
被半恢复内容替换。结束后只重启恢复前原本运行的 API/Worker。

```sh
SFORUM_CONFIRM_RESTORE=RESTORE ./deploy.sh --action restore
```

请结合站点策略设定保留周期与异地备份（产品层仍开放「备份策略」问题）。

## 上线后清单

1. 注册首个超级管理员（若空库）
2. 站点名称、HTTPS URL、邮件 SMTP
3. 附件存储提供方
4. 搜索：默认站点搜索即可；需要再切 Meili
5. 扩展信任策略与 Safe Mode 演练
6. 健康检查：`/health`、`/api/v1/health`、`/api/v1/ready`

## 相关

- 开发环境：[快速开始](./getting-started.md)
- 站长：[使用说明](./usage/README.md)
- 历史长文归档：`docs/archive/legacy-root/development-and-deployment.md`
