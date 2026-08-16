# 生产部署

[← 中文文档首页](./README.md)

## 快速安装

支持 Linux `amd64` 和 `arm64`，不支持 Windows。服务器需要：

- Docker Engine 及 Docker Compose `2.24.4` 或更高版本；
- `curl` 和 `tar`；
- 能访问 GitHub 和 `ghcr.io`；
- loopback 的 `3000`、`18080` 端口可用，或在向导中选择其他端口。

不需要安装 Git 或克隆仓库。下面的命令下载**最新稳定版**的固定名称部署包
（`sforum-deploy.tar.gz`）：

```sh
(
  set -eu
  mkdir -p sforum
  cd sforum
  curl -fsSLo sforum-deploy.tar.gz \
    https://github.com/zhuchunshu/SForum/releases/latest/download/sforum-deploy.tar.gz
  curl -fsSLo SHA256SUMS \
    https://github.com/zhuchunshu/SForum/releases/latest/download/SHA256SUMS
  awk '$2 == "sforum-deploy.tar.gz" { print }' SHA256SUMS > sforum-deploy.sha256
  test "$(wc -l < sforum-deploy.sha256 | tr -d '[:space:]')" = 1
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c sforum-deploy.sha256
  else
    shasum -a 256 -c sforum-deploy.sha256
  fi
  if command -v gh >/dev/null 2>&1; then
    gh attestation verify sforum-deploy.tar.gz --repo zhuchunshu/SForum
  fi
  tar -xzf sforum-deploy.tar.gz --strip-components=1
  ./deploy.sh
)
```

这个 URL 始终指向最新稳定 Release 的部署资产，不需要维护版本号。部署包只包含
安装所需的 Compose 文件、`deploy.sh`、`upgrade.sh`、生产环境示例和
`deploy/` 下的必要脚本，不含源码与仓库历史。

### 校验下载（推荐）

每个 Release 都发布 `SHA256SUMS`（覆盖全部资产）和 GitHub 构建来源证明。

- **先下载压缩包，再下载校验文件**：不要用管道直接解压，否则后续校验
  找不到文件。
- **只校验 `SHA256SUMS` 中 `sforum-deploy.tar.gz` 的精确条目**：
  `awk '$2 == "sforum-deploy.tar.gz" { print }' SHA256SUMS` 按文件名字段匹配，
  而不是只匹配行尾；命令块还会要求结果恰好只有一条。
- 整段命令在 `set -eu` 子 shell 中运行；下载失败、条目缺失/重复、checksum
  或来源证明失败都会在解压前终止。
- `gh attestation verify` 为可选步骤，用于核对构建来源（需要 GitHub CLI 且
  已认证）。

### 通道语义（stable / prerelease）

- **默认通道是 `stable`**：`./deploy.sh` 不带版本、`./upgrade.sh` 的
  `latest` 都只解析最新**稳定** Release。
- **预发布版必须显式选择**：使用 `--channel prerelease`，或直接指定
  不可变版本 `--version v3.0.0-alpha.N`。
- 无论哪种选择，脚本都会在拉取镜像前解析为具体的 `vX.Y.Z` 标签，并运行
  对应的 GHCR 镜像；生产 Compose 不会直接运行浮动的 `latest` 镜像标签。
- 若当前还没有任何稳定 Release，`latest` 解析会失败并提示改用
  `--channel prerelease` 或显式版本；这是预期的 fail-closed 行为。

需要可重复部署时显式固定版本（用你要部署的版本替换 `$SFORUM_VERSION`）：

```sh
./deploy.sh --version $SFORUM_VERSION
./upgrade.sh --version $SFORUM_VERSION
```

## 目标形态

- Docker Compose 编排：`web`、`api`、`worker`、PostgreSQL、Redis
- 对外：仅 **loopback** 发布 Web（及可选 API WebSocket 入口），TLS 由**宿主机反代**负责
- 同域：浏览器只打到站点域名；普通 `/api/v1/*` 由 Nuxt 反代到 API；WebSocket Upgrade 可直打 API 端口

## 配置

推荐直接运行 `./deploy.sh`。首次安装会用中文解释每个选项，并自动生成
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
- `linux/amd64` 和 `linux/arm64` 后端运行包，包含 API、Worker、迁移、CLI
  以及从同一已扫描候选镜像提取的精确内置扩展；
- **固定名称部署包 `sforum-deploy.tar.gz`** 与独立 **`upgrade.sh`**：
  `releases/latest/download/` 始终指向最新稳定版的这两份资产；
- 覆盖全部压缩包的 `SHA256SUMS` 和 GitHub 构建来源证明。

Linux 后端包不包含 Nuxt Web、PostgreSQL 或 Redis，因此不是完整站点安装包；
生产部署仍推荐使用下方四个版本一致的 Docker 镜像。下载后可使用
`gh attestation verify <文件> --repo zhuchunshu/SForum` 验证构建来源，并按
`SHA256SUMS` 验证文件摘要。

## 部署入口

### 使用正式发布镜像（推荐）

正式版本发布到 GitHub Container Registry，包含以下镜像：

- `ghcr.io/zhuchunshu/sforum-api`
- `ghcr.io/zhuchunshu/sforum-worker`
- `ghcr.io/zhuchunshu/sforum-migrate`
- `ghcr.io/zhuchunshu/sforum-web`

每个发布版本同时提供 `linux/amd64` 和 `linux/arm64`。最简单的交互安装如下，
全部直接回车即可完成配置并部署：

```sh
./deploy.sh
```

也可以显式固定版本，或完全非交互地接受推荐配置（`$SFORUM_VERSION` 替换为
实际标签，例如 `v3.0.0-alpha.13` 这样的预发布版本）：

```sh
./deploy.sh --version $SFORUM_VERSION --lang zh
./deploy.sh --version $SFORUM_VERSION --lang en
./deploy.sh --version $SFORUM_VERSION --lang zh --yes --action deploy
```

预发布通道的非交互示例（解析最新发布，含预发布）：

```sh
./deploy.sh --channel prerelease --lang zh --yes --action deploy
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
前，会使用空的 Docker 凭据目录匿名拉取该版本的四个镜像；任一包不可公开
拉取都会阻断发布。首次创建包时，流水线可能先停在这个门禁：管理员将四个
包改为 Public 后重跑失败作业即可。写入仍使用仓库 `GITHUB_TOKEN`，不需要
长期 Registry 密钥。

`deploy.sh` 只走发布镜像，避免新手意外在服务器上源码构建。开发或定制源码
请使用开发文档中的 `scripts/dev.sh` 和构建命令。

## 零停机更新

已有 Compose 安装推荐使用 `upgrade.sh` 更新。可以在交互提示中输入版本，
也可以使用位置参数或 `--version`；直接回车时默认选择 **最新稳定版**：

```sh
./upgrade.sh
./upgrade.sh $SFORUM_VERSION
./upgrade.sh --version $SFORUM_VERSION
./upgrade.sh --yes                       # 无人值守：选择最新稳定版并跳过确认
./upgrade.sh --channel prerelease        # 明确允许预发布（解析最新发布，含预发布）
./upgrade.sh --channel prerelease --yes  # 无人值守的预发布通道
```

预发布版永远不会被隐式选中：要么显式传 `--channel prerelease`，要么直接
指定 `v3.0.0-alpha.N` 这样的不可变标签。无论哪种方式，脚本都会解析为具体
标签并运行对应镜像；`.deployrc` 保存的也是解析后的具体版本，不会保存
`latest`。

现有安装不需要重新克隆仓库。若要先刷新正式版更新脚本，再从**最新稳定
Release 资产**下载并校验（不要使用 `main` 分支的浮动内容）：

```sh
(
  set -eu
  cd /path/to/sforum
  curl -fsSLo upgrade.sh \
    https://github.com/zhuchunshu/SForum/releases/latest/download/upgrade.sh
  curl -fsSLo SHA256SUMS \
    https://github.com/zhuchunshu/SForum/releases/latest/download/SHA256SUMS
  awk '$2 == "upgrade.sh" { print }' SHA256SUMS > upgrade.sh.sha256
  test "$(wc -l < upgrade.sh.sha256 | tr -d '[:space:]')" = 1
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum -c upgrade.sh.sha256
  else
    shasum -a 256 -c upgrade.sh.sha256
  fi
  if command -v gh >/dev/null 2>&1; then
    gh attestation verify upgrade.sh --repo zhuchunshu/SForum
  fi
  chmod 0755 upgrade.sh
  ./upgrade.sh
)
```

下载后必须用 `SHA256SUMS` 中 `upgrade.sh` 的精确条目校验；校验失败不得
运行。需要固定到具体版本（含预发布）时，用对应标签的资产地址：

```sh
curl -fsSLo upgrade.sh \
  https://github.com/zhuchunshu/SForum/releases/download/<TAG>/upgrade.sh
# 再按上面的流程下载对应标签的 SHA256SUMS 并精确校验
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

后台任务不会同时由两个 Worker 消费：脚本先优雅停止旧 Worker，再启动新
Worker，因此队列消费会短暂停顿，但 River 中的持久任务不会丢失。更新前
脚本同时检查 SForum Core 与 River 的数据库迁移。只有目标迁移器声明支持
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
docker compose --env-file .env.production logs -f api worker web
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
| `api` | Fiber API；扩展运行时；可选 WS 入口 |
| `worker` | River 队列（生产默认与 API 分离） |
| `postgres` / `redis` | 状态与会话/缓存 |

开发里的 `EMBED_WORKER_IN_API` **不要**作为生产默认。

## 运行时内存与诊断

管理台 `/control-panel` 的资源卡通过
`GET /api/v1/admin/overview/resources` 读取 API、独立 Worker 和直属插件
进程的 CPU、RSS 与可用的 PSS。Linux 正式镜像直接读取 `/proc`，不依赖
BusyBox `ps`；生产 Compose 让 Worker 与对应 API 共享 PID namespace，使 API
能够发现并正确归因 Worker 与插件进程，同时不需要宿主机 PID namespace 或
Docker socket。资源请求最多每 5 秒共享一次采样，展示最近 60 秒中位数；
不支持 PSS 的系统不会伪造"有效占用"。插件明细按占用从高到低列出，并只
归因当前 API/Worker 直接拥有的插件进程。

开发环境默认将 Worker 内嵌到 API。此时 API 行明确标记"含 Worker"，Worker
行只显示内嵌并发槽位和运行任务数，不虚构一个独立 Worker 的 MiB。生产环境
若设置 `EMBED_WORKER_IN_API=false`，独立 Worker 会单独计量。

Go pprof 是显式 opt-in 的本机诊断面，默认关闭，且启动器拒绝非 loopback
监听地址：

```sh
# 临时开启 API 诊断（内嵌 Worker 已包含在同一 profile）
PPROF_ENABLED=true PPROF_ADDR=127.0.0.1:6060

# 只有独立 Worker 才需要单独开启
WORKER_PPROF_ENABLED=true WORKER_PPROF_ADDR=127.0.0.1:6061
```

启用后可用 `http://127.0.0.1:6060/debug/pprof/`（独立 Worker 为
`6061`）采集 profile。不要把该端口发布到公网或交给反向代理；采集完成后
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
