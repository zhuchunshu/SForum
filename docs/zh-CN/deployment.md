# 生产部署

[← 中文文档首页](./README.md)

## 快速安装

支持 Linux `amd64` 和 `arm64`，不支持 Windows。服务器需要：

- Docker Engine 及 Docker Compose `2.24.4` 或更高版本；
- `curl` 和 `tar`；
- 能访问 GitHub 和 `ghcr.io`；
- loopback 的 `3000`、`18080` 端口可用，或在向导中选择其他端口。

不需要安装 Git 或克隆仓库。下面的命令下载正式版部署文件并启动交互安装：

```sh
mkdir -p sforum
curl -fsSL https://github.com/zhuchunshu/SForum/archive/refs/tags/v3.0.1.tar.gz \
  | tar -xz --strip-components=1 -C sforum
cd sforum
./deploy.sh
```

向导已提供推荐值。不了解某个选项时直接回车即可；脚本会使用 Docker Compose
内置的 PostgreSQL 和 Redis，生成所需密钥，拉取同版本的 GitHub 容器镜像，
执行迁移并等待 API 和 Web 健康。安装完成后默认访问
`http://127.0.0.1:3000`。公网使用前仍须按下文配置 HTTPS 反向代理。

`./deploy.sh` 默认查询 GitHub 最新正式 Release，并在部署前解析为具体版本；
它不会直接运行浮动的 `latest` 镜像。需要可重复部署时可显式执行
`./deploy.sh --version v3.0.1`。

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

## 维护者发布

维护者从已同步到 `origin/main` 的干净 `main` 工作区创建版本：

```sh
./scripts/release.sh
```

脚本推送版本标签后默认立即返回，发布状态继续由 GitHub Actions 管理。只有
需要占用当前终端同步确认结果时才使用 `--wait`；`--no-wait` 是默认行为。
交互发布可填写一行面向用户的“发布重点”，直接回车则只使用 GitHub 自动
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

也可以显式固定版本，或完全非交互地接受推荐配置：

```sh
./deploy.sh --version v3.0.1 --lang zh
./deploy.sh --version v3.0.1 --lang en
./deploy.sh --version v3.0.1 --lang zh --yes --action deploy
```

该模式组合 `compose.yaml`、`compose.prod.yaml` 与
`compose.release.yaml`。脚本先拉取同版本的四个 GHCR 镜像，成功后才启动
内置 PostgreSQL/Redis；全新库跳过备份，已有安装先备份，再停止旧应用、
迁移并以 `--no-build` 启动。需要 Docker Compose 2.24.4 或更高版本。
API `/api/v1/ready`、Web 首页和五个常驻服务全部通过后，脚本才记录成功
版本并打印访问地址。

脚本会在改动数据库前检查配置占位密钥、端口冲突和三个 Go 镜像的构建版本，
并使用部署锁阻止两个更新并发执行。预拉取完成后，迁移和启动都使用
`--pull never`，因此旧应用停止后不会再次依赖 Registry。若迁移或健康检查
失败，`.deployrc` 会记录 `status=recovery_required`、目标版本、前一版本和
备份路径；脚本不会把该实例误报为部署成功。

等价的非交互命令：

```sh
export SFORUM_VERSION=v3.0.1
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
也可以使用位置参数或 `--version`；直接回车时默认选择 `latest`：

```sh
./upgrade.sh
./upgrade.sh v3.0.1
./upgrade.sh --version v3.0.1
./upgrade.sh --yes                       # 无人值守：选择最新发布并跳过确认
```

现有安装不需要重新克隆仓库。若要先刷新正式版更新脚本，再进入交互更新：

```sh
cd /path/to/sforum
curl -fsSLo upgrade.sh \
  https://raw.githubusercontent.com/zhuchunshu/SForum/v3.0.1/upgrade.sh
chmod 0755 upgrade.sh
./upgrade.sh
```

请使用 `v3.0.0-alpha.13` 或更高版本附带的 `upgrade.sh`。`v3.0.0-alpha.12`
附带的脚本无法正确启动数据库兼容预检；若当前安装目录来自该版本，先固定下载
修复后的脚本，再执行更新：

```sh
curl -fsSLo upgrade.sh \
  https://raw.githubusercontent.com/zhuchunshu/SForum/v3.0.0-alpha.13/upgrade.sh
chmod 0755 upgrade.sh
./upgrade.sh v3.0.0-alpha.13
```

这里的 `latest` 不是浮动的容器镜像标签。脚本查询 GitHub Release 列表，选择
最新的已发布 Release（包括预发布版），再解析为具体的 `vX.Y.Z` tag。执行前
会输出当前实际版本和目标实际版本，并询问是否使用该版本；只有显式传入
`--yes` 才跳过输入与确认。更新成功后 `.deployrc` 保存的也是解析后的具体
版本，不会保存 `latest`。

第一次从旧版直连端口拓扑运行时，脚本会要求确认一次性 blue/green 入口
转换。该转换需要停止旧服务并启动稳定 Caddy 入口，因此会有一次短暂维护
窗口。转换完成后，对于不包含数据库迁移的版本，脚本会先在备用槽启动并
检查 API/Web，经 Caddy 原子切换流量后才停止旧槽，从而保持 HTTP 服务连续。
WebSocket 长连接在切换时可能需要自动重连。

后台任务不会同时由两个 Worker 消费：脚本先优雅停止旧 Worker，再启动新
Worker，因此队列消费会短暂停顿，但 River 中的持久任务不会丢失。更新前
脚本同时检查 SForum Core 与 River 的数据库迁移；只要目标版本存在待执行
迁移，就会拒绝零停机更新。此时请改用 `./deploy.sh --version <版本>`，接受
维护窗口完成备份、迁移和部署。

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
不支持 PSS 的系统不会伪造“有效占用”。插件明细按占用从高到低列出，并只
归因当前 API/Worker 直接拥有的插件进程。

开发环境默认将 Worker 内嵌到 API。此时 API 行明确标记“含 Worker”，Worker
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

恢复需要显式设置 `SFORUM_CONFIRM_RESTORE=RESTORE`。脚本会停止原先运行的
API/Worker，在独立临时数据库中使用 `ON_ERROR_STOP=1` 和单事务完整恢复，
校验应用表后才原子切换数据库名；任一 SQL 错误都会返回失败，原目标库不会
被半恢复内容替换。结束后只重启恢复前原本运行的 API/Worker。

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
- 历史长文归档：`docs/archive/legacy-root/development-and-deployment.md`  
