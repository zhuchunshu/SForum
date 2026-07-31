# 生产部署

[← 中文文档首页](./README.md)

## 目标形态

- Docker Compose 编排：`web`、`api`、`worker`、PostgreSQL、Redis  
- 对外：仅 **loopback** 发布 Web（及可选 API WebSocket 入口），TLS 由**宿主机反代**负责  
- 同域：浏览器只打到站点域名；普通 `/api/v1/*` 由 Nuxt 反代到 API；WebSocket Upgrade 可直打 API 端口  

## 配置

1. 复制生产环境文件：

   ```sh
cp .env.production.example .env.production
```

`deploy.sh` 及数据库辅助脚本会把 `.env.production` 权限收紧为 `0600`，
避免同机其他用户读取生产密钥。

2. **必须**修改：

   - `POSTGRES_PASSWORD` / `DATABASE_URL`  
   - `REDIS_PASSWORD`  
   - `APP_URL` / `APP_DOMAIN`  
   - 会话与 CSRF 等相关密钥（以 example 注释为准）  
   - `MARKETPLACE_ED25519_PUBLIC_KEY_HEX`（签名 Marketplace 索引的 32 字节公钥，64 个十六进制字符）及对应的 `MARKETPLACE_ED25519_KEY_ID`。生产/预发布 API 缺少公钥会在就绪前拒绝启动；私钥不得写入仓库或容器。

3. 可选 Meilisearch：仅在使用 Meili 插件时启用对应服务与配置  

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

- `linux/amd64`、`linux/arm64`、`darwin/amd64`、`darwin/arm64`、
  `windows/amd64`、`windows/arm64` 的 `sforum` 管理 CLI；
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

每个正式版本同时提供 `linux/amd64` 和 `linux/arm64`。部署时固定完整版本，
不要在生产环境固定使用 `latest`：

```sh
./deploy.sh --version v2.8.0
./deploy.sh --version v2.8.0 --lang zh
./deploy.sh --version v2.8.0 --lang en
```

该模式组合 `compose.yaml`、`compose.prod.yaml` 与
`compose.release.yaml`，先拉取指定版本，再备份、迁移并启动相同版本的
API、Worker 和 Web。需要 Docker Compose 2.24.4 或更高版本，以支持
`!reset` 覆盖构建配置。启动后脚本会等待 API `/api/v1/ready` 与 Web 首页
同时成功；任一服务超时都不会打印部署完成地址。

等价的非交互命令：

```sh
export SFORUM_VERSION=v2.8.0
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

### 从源码构建

开发版本或需要自定义源码时仍可使用原有入口：

```sh
./deploy.sh
./deploy.sh --lang zh
./deploy.sh --lang en
```

首次会选择语言并写入 `.deployrc`。菜单通常包括安装/更新、迁移、备份等（以脚本实际菜单为准）。

等价 Compose 命令：

```sh
docker compose --env-file .env.production -f compose.yaml -f compose.prod.yaml up -d --build
```

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
进程的 RSS。资源请求最多每 5 秒共享一次采样，展示最近 60 秒中位数；Linux
可同时显示完整进程族的 PSS，macOS 等不支持 PSS 的系统不会伪造“有效占用”。
插件明细按占用从高到低列出，并只归因当前 API/Worker 直接拥有的插件进程。

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
