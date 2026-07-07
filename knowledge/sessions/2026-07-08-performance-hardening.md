# 2026-07-08 Performance Hardening Handoff

## Changed

### 后端（Go/Fiber）

- **ListComments SQL 分页改造**（`app/Models/Forum/postgres_store.go`）：
  flat 视图改为 SQL `LIMIT/OFFSET` + 独立 count；tree 视图三步查询（根评论分页 →
  `root_comment_id = ANY(...)` 拉子孙 → 内存建树）。删除死代码 `pageComments`。
  Service 层移除重复 normalizePage，补 view enum 校验（`service.go`）。
  OpenAPI 评论 `page` 补 `maximum: 200`（`contracts/openapi/paths/forum.yaml`）。
  新增测试：`service_test.go` ListComments 默认 tree/flat 透传/非法 view 拒绝。
- **Fiber 超时 + 压缩**（`app/Http/server.go` + `config/config.go`）：
  fiber.Config 补 ReadTimeout(10s)/WriteTimeout(20s)/IdleTimeout(120s)/BodyLimit(4MB)；
  注册 compress 中间件。新增测试：压缩生效测试。
- **Redis 客户端合并**（`app/Support/HumanVerify/redis_store.go` + `bootstrap/app.go`）：
  `NewRedisClient` 接收 `RedisClientOptions`；humanverify 与 cache 合并为 `sharedRedisClient`；
  修复 close 链漏关 cacheClient 的连接泄漏。新增测试：options 传递测试。
- **PG 连接池调优**（`app/Support/Postgres/pool.go` + `config/config.go` + bootstrap）：
  新增 `PoolOptions` + `NewPoolWithOptions`；API/Worker 各自 MinConns(2)/IdleTime(30m)/
  Lifetime(1h)/ConnectTimeout(10s)。
- **Meilisearch 超时**（`app/Support/Search/client.go` + bootstrap）：
  `NewClientWithTimeout` 用 `WithCustomClient(&http.Client{Timeout})`；`go mod tidy`
  提升 meilisearch-go 为 direct。
- **Fiber limiter 写接口限流**（`app/Http/server.go` + `bootstrap/app.go`）：
  limiter 跳过 GET/HEAD/OPTIONS，写接口按 IP 限流（默认 30/min），Storage 注入
  redisStorage 做分布式限流。新增测试：429 限流测试。`Dependencies` 加 `Storage` 字段。

### 前端（Nuxt）

- **routeRules + compressPublicAssets**（`apps/web/nuxt.config.ts`）：
  公开页 swr（60-3600s），表单/认证/编辑页 `ssr: false`，admin SPA + noindex，
  图标目录长缓存。i18n `/en/**` 镜像。
- **静态资源缓存头**（`server/middleware/seo-robots.ts` + `server/api/icon-collections/...`）：
  静态资源 `immutable` 长缓存；图标目录 `s-maxage=604800`。
- **组件懒加载**：`SFEditor`/`SFIconPicker` 改 `<LazySFEditor>`/`<LazySFIconPicker>`
  （6 处引用：components.vue × 3、[topicSlug].vue × 3、edit.vue、topics/new.vue）。
- **@nuxt/image + SFAvatar**：安装 @nuxt/image，`<img>` 改 `<NuxtImg>`（webp/lazy/
  decoding/width/height/sizes）。

### 知识库

- 新增 `decisions/2026-07-08-performance-hardening.md`。
- 更新 `modules/forum.md`、`modules/backend.md`。
- 本 handoff。

## Decisions

- ListComments tree 视图用"根评论分页 + 子孙批量拉取"，保留 Total/Items/view 语义不变。
- Redis 合并 humanverify + cache 为一个 client；session storage 保持独立。
- limiter 只限写接口，读接口有缓存挡。
- @nuxt/image 引入新依赖，SFAvatar 用 NuxtImg。

## Next

- `view_count` 增长（Redis 计数 + 批量回写）——功能缺失，单独排期。
- 读写分离/水平扩展——部署架构决策。
- `getCachedData` / payload 主动复用——前端数据层重构。
- 评论 tree 视图单根子孙极多时的子孙分页（当前 perPage 控制根数已够用）。
- 正文 v-html 图片优化（需 sanitized HTML 改写方案）。

## Open Questions

- 生产环境 `COMPRESS_LEVEL` / `LIMITER_WRITE_MAX` / `LIMITER_WINDOW` 的调参需根据
  实际流量观察后调整。
- routeRules swr 在多实例部署下需确认 Nitro storage backend（默认内存，多实例不共享）。

## Verification

- `go test ./...`：全绿（apps/api）。
- `ruby scripts/validate-openapi-refs.rb`：834 refs OK。
- `bun run typecheck`：EXIT 0。
- `bun test`：81 pass / 3 fail（3 个失败为预先存在，git stash 验证与本次无关，
  是 useApiClient.test.ts 读 `app/pages/register.vue` 但页面在主题层）。
