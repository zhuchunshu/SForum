# Decision: Performance Hardening (Backend + Frontend)

## Status

Accepted (2026-07-08)

## Context

在 2026-07-08 搜索/缓存/深翻页加固之后，代码核查暴露了读路径之外的多个性能与安全短板：

1. **`ListComments` 全量加载**：`postgres_store.go` 的 `ListComments` 无 SQL `LIMIT/OFFSET`，
   先把某 topic 的全部 active 评论读入内存再内存分页。万楼热帖单请求即万行入内存，
   且该路径未走 `CachedStore`。
2. **Fiber 零超时/零压缩**：`server.go` 只设 `AppName`+`ErrorHandler`，无 `ReadTimeout`/
   `WriteTimeout`/`IdleTimeout`/`BodyLimit`，慢客户端攻击风险；无压缩中间件，带宽浪费。
3. **Redis 3 个独立连接池**：humanverify 与 cache 各持一个 go-redis client，session storage
   又是第三个，无显式 `PoolSize`/超时配置；`bootstrap/app.go` 注释声称复用但实际未复用；
   错误回退分支漏关 cacheClient（连接泄漏）。
4. **PG 连接池只设 MaxConns**：无 `MinConns`/`MaxConnIdleTime`/`MaxConnLifetime`/
   `ConnectTimeout`，突发负载下冷启动慢。
5. **Meilisearch 客户端无超时**：`meilisearch-go` 的 `Search` 不收 context，客户端无
   `http.Client.Timeout`，Meili 宕机时请求挂起 + 502/503/504 自动重试 3 次。
6. **无限流**：写接口（发帖/评论/登录/注册）无 IP 级限流，CC 攻击无保护。
7. **前端零 HTTP 缓存/零渲染模式控制**：所有页面全动态 SSR，无 `routeRules`/ISR/swr，
   无 `compressPublicAssets`，无静态资源 `Cache-Control`，无图片优化，重型组件同步加载。

## Decision

### 后端

#### 1. ListComments SQL 分页改造（根评论分页 + 子孙批量拉取）

- **flat 视图**：改为 SQL `LIMIT $2 OFFSET $3`（带 `status='active'` 过滤），`Total` 用
  独立 `SELECT count(*)` 查询。OFFSET 深翻页已被 `maxTopicPage=200` clamp 保护。
- **tree 视图**：三步查询——
  1. 对根评论（`parent_comment_id IS NULL`）按 `path_key ASC` `LIMIT/OFFSET` 分页
     （复用 `comments_topic_root_idx` 部分索引），`Total` = 根评论数。
  2. 取当页根评论 ID，用 `root_comment_id = ANY($rootIDs)` 批量拉这些根的全部子孙
     （同样命中 `comments_topic_root_idx`）。
  3. 内存 `buildCommentTree`（此时只在当页根+其子孙上建树，数据量 = perPage 个根讨论线）。
- **语义不变**：`Total`（flat=全部评论数，tree=根评论数）、`Items` 结构、`view` 参数
  完全不变，前端零改动。
- **Service 层**：移除重复的 `normalizePage`（Store 层已 normalize），补 view enum 校验。
- **OpenAPI**：评论 `page` 参数补 `maximum: 200`，与 `/topics`、`/search` 对齐。
- **清理**：删除不再被调用的 `pageComments` 死代码函数。

#### 2. Fiber 超时 + 压缩中间件

- `config.go` 新增 `HTTPReadTimeout`(10s)/`HTTPWriteTimeout`(20s)/`HTTPIdleTimeout`(120s)/
  `HTTPBodyLimit`(4MB)/`CompressLevel`(default=0)。
- `server.go` fiber.Config 补这些字段，注册 `compress.New` 中间件（brotli/gzip 自动协商）。
- `CompressLevel` 支持 env `COMPRESS_LEVEL` 配置（default/best_speed/best_compression/disabled）。

#### 3. Redis 客户端合并 + 显式池配置

- `humanverify.NewRedisClient` 签名扩展为接收 `RedisClientOptions{PoolSize, MinIdleConns,
  DialTimeout, ReadTimeout, WriteTimeout, ConnMaxIdleTime, ConnMaxLifetime}`。零值字段走
  go-redis 默认值，保持向后兼容。
- `bootstrap/app.go` 合并 humanverify 与 cache 为单一 `sharedRedisClient`，session storage
  保持独立（fiber Storage 接口，不合并）。
- 修复 close 链：所有错误回退分支统一关 `sharedRedisClient`（原漏关 cacheClient 的连接
  泄漏已修复）。
- `config.go` 新增 `RedisPoolSize`(20)/`RedisMinIdleConns`(5)/`RedisDialTimeout`(5s)/
  `RedisReadTimeout`(3s)/`RedisWriteTimeout`(3s)/`RedisConnMaxIdleTime`(30m)/
  `RedisConnMaxLifetime`(1h)。

#### 4. PG 连接池调优

- `pool.go` 新增 `PoolOptions{MaxConns, MinConns, MaxConnIdleTime, MaxConnLifetime,
  HealthCheckPeriod, ConnectTimeout}` 和 `NewPoolWithOptions`。旧 `NewPool`/`BuildPoolConfig`
  保留向后兼容（内部委托新函数）。
- `config.go` 新增 API 与 Worker 各自的 `DatabaseMinConns`(2)/`MaxConnIdleTime`(30m)/
  `MaxConnLifetime`(1h)/`ConnectTimeout`(10s)。
- `bootstrap/app.go` 和 `bootstrap/worker.go` 改用 `NewPoolWithOptions`。

#### 5. Meilisearch 客户端超时

- `search/client.go` 新增 `NewClientWithTimeout(host, apiKey, timeout)`，通过
  `meilisearch.WithCustomClient(&http.Client{Timeout: timeout})` 注入超时（meilisearch-go
  无 `WithTimeout` 选项，必须用 `WithCustomClient`）。
- `config.go` 新增 `MeiliTimeout`(5s)。
- `bootstrap/app.go` 和 `bootstrap/search_adapter.go` 改用 `NewClientWithTimeout`。
- `go mod tidy` 把 `meilisearch-go` 从 indirect 提升为 direct。

#### 6. Fiber limiter 写接口限流

- `server.go` 注册 `limiter.New`，`Next` 跳过 GET/HEAD/OPTIONS，只限写方法。
- `Storage` 注入到 `Dependencies`，复用 `redisStorage` 做分布式限流（多实例共享）；
  为 nil 时退化为进程内存限流。
- `config.go` 新增 `LimiterWriteMax`(30)/`LimiterWindow`(1m)。

### 前端

#### 7. routeRules + nitro 压缩

- `nuxt.config.ts` nitro 段新增 `compressPublicAssets: { brotli: true, gzip: true }`。
- `routeRules`：公开内容页（`/`、`/c/**`、`/tags/**`、`/u/**`、`/t/**`）走 swr（60-3600s）；
  表单/认证/设置/编辑页走 `ssr: false`（SPA）；admin 走 `ssr: false` + `robots: { index: false }`；
  图标目录走 `cache: { maxAge: 86400 }`。
- i18n `prefix_except_default` 下为 `/en/**` 镜像相同规则。

#### 8. 静态资源 + icon-collections 缓存头

- `server/middleware/seo-robots.ts` 静态资源分支加
  `cache-control: public, max-age=31536000, immutable`（带 hash 文件永久缓存）。
- `server/api/icon-collections/[collection].get.ts` 加
  `cache-control: public, max-age=86400, s-maxage=604800, stale-while-revalidate=86400`。

#### 9. 组件懒加载

- `SFEditor`（含 9 个 @tiptap 包）和 `SFIconPicker` 改为 `<LazySFEditor>`/
  `<LazySFIconPicker>`（Nuxt 自动 Lazy 前缀，零配置）。主要收益在帖子详情页的回复表单
  （首屏外的编辑器按需加载）。

#### 10. @nuxt/image + SFAvatar 改造

- 安装 `@nuxt/image`，`nuxt.config.ts` modules 注册 + `image` 配置（webp 格式、
  质量预设、头像尺寸 screens）。
- `SFAvatar.vue` 的 `<img>` 改为 `<NuxtImg>`，加 `loading="lazy"`/`decoding="async"`/
  `width`/`height`/`sizes`/`format="webp"`。src 为空时仍回退到 initials 逻辑。

## Consequences

- **内存**：`ListComments` 不再全量加载，万楼热帖单请求只读当页根讨论线数据。
- **网络**：Fiber 超时防慢客户端攻击；压缩降低带宽；静态资源/图标目录长缓存减轻回源。
- **连接**：Redis 从 3 池降到 2 池（humanverify+cache 合一），显式池参数；PG 池有 MinConns
  预热和连接生命周期管理；Meili 有超时兜底。
- **安全**：写接口限流防 CC 攻击。
- **前端**：公开页 swr 降低 SSR 开销；表单页 SPA 避免不必要的 SSR；编辑器懒加载减小初始包。
- **配置**：新增约 20 个 env 变量，全部有合理默认值，零配置即可运行（与 beginner-friendly
  原则一致）。
- **测试**：新增 ListComments view 校验/默认值/非法值测试、compress 生效测试、limiter 429 测试、
  Redis options 传递测试；现有测试全绿（3 个 useApiClient 失败为预先存在，与本次无关）。

## Follow-up（范围外，诚实标注）

- `view_count` 增长（Redis 计数 + 批量回写）——功能缺失，非性能，单独排期。
- 读写分离/水平扩展——部署架构层面，需单独决策。
- `getCachedData` / payload 主动复用——前端数据层较大重构，本次仅做 routeRules 层缓存。
- 评论列表 CachedStore 缓存——tree 分页后查询已轻量，暂不缓存。
- 正文 v-html 图片优化——XSS 风险边界，需单独设计 sanitized HTML 图片改写方案。
- `ListComments` tree 视图若单根子孙极多（如某根帖有 5000 回复），第二步子孙查询仍会拉取
  该根全部子孙——可后续考虑子孙分页或深度截断，但当前 perPage 控制根数已足够。

## Sources

- `knowledge/decisions/2026-07-08-search-cache-deep-pagination.md`（前置读路径加固）
- `apps/api/app/Models/Forum/postgres_store.go`（ListComments 改造）
- `apps/api/app/Http/server.go`（Fiber 配置 + compress + limiter）
- `apps/api/app/Support/HumanVerify/redis_store.go`（Redis client 合并）
- `apps/api/app/Support/Postgres/pool.go`（PG 池调优）
- `apps/api/app/Support/Search/client.go`（Meili 超时）
- `apps/web/nuxt.config.ts`（routeRules + compressPublicAssets + image）
- `apps/web/app/components/SFAvatar.vue`（NuxtImg 改造）
