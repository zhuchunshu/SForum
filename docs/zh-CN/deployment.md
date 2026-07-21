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

2. **必须**修改：

   - `POSTGRES_PASSWORD` / `DATABASE_URL`  
   - `REDIS_PASSWORD`  
   - `APP_URL` / `APP_DOMAIN`  
   - 会话与 CSRF 等相关密钥（以 example 注释为准）  

3. 可选 Meilisearch：仅在使用 Meili 插件时启用对应服务与配置  

## 部署入口

交互式（支持中英菜单）：

```sh
./deploy.sh
./deploy.sh --lang zh
./deploy.sh --lang en
```

首次会选择语言并写入 `.deployrc`。菜单通常包括安装/更新、迁移、备份等（以脚本实际菜单为准）。

等价 Compose 思路：

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

## 备份

仓库提供 `deploy/scripts/` 下的 PostgreSQL 备份/恢复辅助脚本。请结合站点策略设定保留周期与异地备份（产品层仍开放「备份策略」问题）。

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
