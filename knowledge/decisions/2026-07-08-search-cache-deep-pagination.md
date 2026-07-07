# Decision: Search, Cache, And Deep-Pagination Hardening

## Status

Accepted (2026-07-08)

## Context

SForum 在千万级数据量评估中暴露了三个读路径风险：

1. **搜索全表扫描**：`ListTopics` 用 `ILIKE '%kw%'` 同时匹配 `topics.title` 与
   `posts.plain_text`，前导通配符无法用 B-tree 索引，千万行下是数秒级全表扫描。
   Meilisearch 在架构与 compose 中已就绪，但 `meilisearch-go` 依赖未引入、
   `search.index_topic` job 从未被装配、forum.Service 没有 dispatcher/indexer 字段。
2. **公共读路径零缓存**：Redis 仅用于 session/限流/人机验证，首页/分类/主题详情
   每次刷新都打 PostgreSQL（含 JOIN + count），高并发下压垮数据库。
3. **OFFSET 深翻页**：`normalizePage` 只限 `perPage≤100`，不限 `page` 上限，
   爬虫/用户可请求 `page=100000` 让 PG 跳过千万行。

## Decision

### 1. Meilisearch 全文搜索完整接入

- 新增 `app/Support/Search` 包：`NewClient`/`Indexer`/`Service`/`EnsureIndex`。
- `Indexer` 同时实现 `forum.TopicSearchIndexer`（`EnqueueIndex`/`EnqueueDelete`，
  forum.Service 在写流程事务后调度）和 `searchjobs.TopicIndexer`（`IndexTopic`/
  `DeleteTopic`，River worker 执行）。
- 为避免 `forum → search` 循环依赖，forum 包定义窄接口 `TopicSearchIndexer`，
  search 包实现；search 依赖 searchjobs（单向），searchjobs 不依赖 search。
- `search.Service.Search` 直接查 Meilisearch，**完全不碰 PostgreSQL**，支持
  `categorySlug`/`tagSlug`/`status` 过滤，排序与 PG 列表一致（`isPinned:desc,
  lastActivityAt:desc`）。
- 新增 `GET /api/v1/search` 端点；`ListTopics` 的 `query` 非空分支改为返回
  `ErrUseSearchEndpoint`（400 `forum.use_search_endpoint`），引导前端走专用端点。
- 索引调度失败只记 slog 不中断主流程（搜索是可重建的派生数据）。
- 前端首页 `SFSearch` 搜索框：`searchQuery` 非空时调 `searchTopics`（`/search`），
  清空时回 `listTopics`（`/topics`）。

### 2. Redis 缓存层（CachedStore 装饰器）

- 新增 `app/Support/Cache` 包：`Cache` 接口 + `MemoryCache`（测试用）+
  `RedisCache`（生产，go-redis/v9）。定义 `Get/Set/Delete/Increment`。
- forum 包 `CachedStore` 通过**嵌入 Store 接口**装饰底层 store，只 override
  需缓存/失效的方法，未 override 的方法直接转发。
- **读缓存**：`ListCategories`/`ListCategoryGroups`/`ListTags`（TTL 60s）、
  `GetTopic`（TTL 30s）、`ListTopics`（TTL 15s）。
- **失效采用 generation 方案**：写方法（Create/Update/Delete Topic、CreateComment、
  taxonomy 写）成功后递增 generation 版本号，读 key 含 generation，旧 key 自然过期，
  避免 Redis SCAN。主题详情按 topicID 精确 Delete。
- bootstrap 复用 session 同款 Redis client（`humanverify.NewRedisClient`），
  `cache.NewRedisCache` 注入 forum provider，注册 close。

### 3. OFFSET 深翻页硬上限

- `normalizePage` 增加 `maxTopicPage = 200`，超出 clamp 到末页，消除
  `OFFSET=千万` 扫描。
- OpenAPI 的 `page` 参数加 `maximum: 200`。
- **不改前端组件/契约**：保留 `{items,total,page,perPage}` 与 `SFPagination`
  跳页语义——用户极少翻到 200 页之后，SEO 分页深度可由 sitemap 控制。

## Consequences

- **搜索**：关键词检索不再触发 PG 全表扫描；Meilisearch 不可达时搜索端点
  返回 503，主流程不受影响。索引数据可从 PG 完全重建。
- **缓存**：公共读 QPS 显著降低 PG 压力；写操作有 ≤15-60s 的最终一致性窗口
  （generation + TTL 双保险），对论坛场景可接受。
- **分页**：深翻页被 clamp，SEO 与用户体验无损。
- **依赖**：新增 `github.com/meilisearch/meilisearch-go`。
- **测试**：所有新组件用接口 + 内存 fake，保持"无外部依赖"的测试惯例。

## Follow-up（范围外，诚实标注）

- `ListComments` 全量进内存（万楼热帖）——需 path_key 范围查询 SQL 改造，单独排期。
- `view_count` 不增长（功能缺失，非性能）——需 Redis 计数 + 批量回写 job，单独排期。
- 读写分离/水平扩展——部署架构层面，需单独设计决策。

## Sources

- `knowledge/decisions/2026-07-04-performance-first-jobs-queues.md`（River 选型）
- `docs/architecture.md`（Meilisearch 可重建、PG 为源真相）
- `apps/api/app/Models/Forum/cached_store.go`、`service.go`、`app/Support/Search/`
