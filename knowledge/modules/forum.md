# Forum Module

## Purpose

Owns the core discussion model: categories, user-facing topics/posts, tree
comments, shared content records, revisions, topic states, slugs, and public
read models.

## Current Status

Backend foundation implemented on 2026-07-06. Real taxonomy slice implemented
on 2026-07-07.

- `categories` owns public forum sections. The first seed category is
  `general` / `综合讨论`.
- `topics` owns user-facing posts/threads: title, slug, category, author,
  state, counters, and a `content_id`.
- `comments` owns tree-shaped replies under topics: parent/root references,
  stable path keys, depth, reply counters, and state.
- `posts` is the shared content table for both topics and comments. It stores
  raw content, sanitized HTML, plain text, excerpt, source format, editor type,
  editor version, render version, and content hash.
- `post_revisions` stores previous shared-content snapshots when comments are
  edited.
- Taxonomy now uses two levels: `category_groups` contain ordered
  `categories`. v1 category access is only `public` or `hidden`; role-scoped
  category access is deferred.
- Core tags live in `tags` with `active`, `pending`, and `disabled` statuses.
  Topic/tag joins live in `topic_tags`, and topic summaries/details expose
  active tag summaries. Tag slugs accept Unicode letters/numbers plus hyphens,
  so Chinese tags can be created and filtered directly; category and topic URL
  slugs remain ASCII-oriented.
- Runtime forum settings live in `web_options`: default category slug, tag
  creation mode, public tag pages, and max tags per topic. Recommended defaults
  are configurable and resettable.
- Tag creation modes are `controlled`, `review`, and `open`. Controlled mode
  only allows approved tags, review mode creates pending tags, and open mode
  creates active tags directly.
- Public Nuxt pages now consume real forum data for the homepage, category
  pages, and tag pages. Tag pages can be disabled through public runtime
  options.
- Topic and comment author summaries include the shared `AvatarView` contract,
  including reply reference authors when present. Frontend forum surfaces should
  pass `author.avatar` into `SFAvatar` directly.
- Admin UI includes category group/category management, tag management, and
  forum settings under the low-code admin module registry.
- Admin-managed categories and tags have optional `icon` and `iconColor`
  visual fields for backend configuration and admin-list previews. Public theme
  pages do not consume these fields yet.
- Go domain logic lives under `apps/api/app/Models/Forum`; HTTP routes live
  under `apps/api/app/Http/Controllers/Forum`.

## Domain Shape

- Category: groups topics and defines visibility defaults.
- Category group: groups categories for navigation and operator management.
- Topic: user-facing post/thread with title, slug, author, category, state,
  counters, and latest activity.
- Tag: reusable topic label controlled by core settings and admin review.
- Comment: arbitrary-depth tree reply under a topic.
- Post: shared content record used by topics and comments. It is not the
  frontend "帖子" concept.
- Post revision: audit history for edited shared content.
- Topic state: active, locked, hidden, or deleted.

## SEO URL Shape

- Category: `/c/:categorySlug`
- Tag: `/tags/:tagSlug`
- Topic: `/t/<path>` — 详情页为 catch-all 路由（`t/[...path].vue`），具体形态由
  `seo.topic_url_mode` 选项控制，管理员可在 SEO 设置页切换：

| 模式 | 形态 | 后端查找 |
|---|---|---|
| `id_slug`（默认） | `/t/123/hello-world` | `GET /topics/:topicID`（按 ID） |
| `id` | `/t/123` | `GET /topics/:topicID`（按 ID） |
| `slug` | `/t/hello-world` | `GET /topics/by-slug/:slug`（按 slug，需全局唯一） |

URL 规范化：详情页在 SSR 入口按当前 mode 计算 canonical 路径，与请求路径
不符时直接 301（SSR）/ replace（客户端）。详情页先通过
`topicPathLookupCandidates` 生成有序查询候选，兼容当前模式和切换模式前遗留的
`id`、`id_slug`、`slug` 旧链接；只有 404 会尝试下一个候选，API/网络错误不被吞掉。
触发场景包括：模式切换后的旧 URL、slug 变更后的旧 slug、`id` 模式下多余的
slug 段。编辑入口为 `?edit=1` query（避免 catch-all 嵌套子路由的渲染出口问题）。

`slug` 模式要求 slug 全局唯一：迁移 `202607090001` 把 `topics.slug` 升级为
UNIQUE 索引（先去重），创建/改标题时 `Service.ensureUniqueTopicSlug` 在冲突时
追加 `-2`/`-3` 后缀。`id_slug`/`id` 模式不依赖唯一约束。


## Content Rules

- v1 stores accepted content in the shared `posts` table as raw content,
  sanitized HTML, extracted plain text, and excerpt.
- Markdown and HTML source formats are accepted by the backend in v1.
- `json` is reserved in the schema for future structured editors, but the API
  rejects JSON publishing until a Tiptap/native-JSON acceptance contract exists.
- Render Markdown with `goldmark`; sanitize display HTML with `bluemonday`.
- goldmark runs with the **GFM extension set** (tables, strikethrough,
  autolinks, task lists), matching the editor's `gfm: true`. The sanitizer
  keeps `class="language-<lang>"` on `<code>` and read-only
  `<input type="checkbox">` for task lists; both are regex-gated so non-
  checkbox inputs and event-handler attributes are still stripped.
- The frontend highlights code blocks with `highlight.js` via a `v-highlight`
  directive on the topic body, comment bodies, and editor preview.
- `RenderVersion` is `goldmark-bluemonday-v2`; existing posts keep their old
  HTML until next edit (no batch re-render).
- Keep edit history through `post_revisions` for comment edits. Topic editing
  endpoints are deferred.
- Hide deleted or moderation-only content from public SSR pages, sitemap, and
  Meilisearch indexes.
- Category labels, moderation labels, and system-authored forum text must be
  localizable, defaulting to Simplified Chinese.
- User-authored topics and comments are stored as written and are not
  translated by default.

## API Surface

- `GET /api/v1/categories`
- `GET /api/v1/category-groups`
- `GET /api/v1/tags`
- `GET /api/v1/topics`
- `POST /api/v1/topics`
- `GET /api/v1/topics/{topicID}`
- `GET /api/v1/topics/{topicID}/comments?view=tree|flat`
- `POST /api/v1/topics/{topicID}/comments`
- `GET /api/v1/comments/{commentID}/replies`
- `PATCH /api/v1/comments/{commentID}`
- `DELETE /api/v1/comments/{commentID}`
- `GET /api/v1/admin/forum/category-groups`
- `POST /api/v1/admin/forum/category-groups`
- `PATCH /api/v1/admin/forum/category-groups/{groupID}`
- `GET /api/v1/admin/forum/categories`
- `POST /api/v1/admin/forum/categories`
- `PATCH /api/v1/admin/forum/categories/{categoryID}`
- `GET /api/v1/admin/forum/tags`
- `POST /api/v1/admin/forum/tags`
- `PATCH /api/v1/admin/forum/tags/{tagID}`
- Admin category/tag create and update payloads accept optional `icon` and
  `iconColor` fields for backend visual configuration.
- `GET /api/v1/admin/forum/settings`
- `PUT /api/v1/admin/forum/settings`
- `POST /api/v1/admin/forum/settings/reset`

## Permission Boundaries

- Create topic: login required plus `topic.create`.
- Create comment: login required plus existing `post.create`.
- Manage category groups and categories: `category.manage`.
- Manage tags and tag policy settings: `tag.manage`.
- Manage forum settings: `category.manage` or `tag.manage`, with writes
  limited to the permission that owns the changed values.
- Edit comment: author with `post.edit_own`, or any user with
  `post.edit_any`.
- Delete comment: author with `post.delete_own`, or any user with
  `post.delete_any`.
- Future topic lock/pin/hide/delete endpoints should reuse the existing
  `topic.lock`, `topic.pin`, `topic.edit_any`, and `topic.delete_any`
  permissions.

## Plugin Boundary

- Core owns category groups, categories, tag semantics, topic-tag joins, public
  routes, admin routes, runtime settings, and permission checks.
- Plugins may react through explicit forum events and future provider slots, but
  must not override core forum routes, mutate core taxonomy tables directly, or
  bypass API policy checks.
- Full search indexing, notification fanout, external analytics, and provider
  integrations remain plugin/provider work unless core needs a stable contract.

## Comment Display Decision

The backend stores full tree comments. The intended public UI should render
desktop comments with the A-style reading-flow/connection-line layout and
mobile comments with the D-style flat list plus "replying to" context labels.

## Open Questions

- Edit grace period and revision visibility rules.
- Whether votes/reactions exist in MVP.
- When to add topic editing, deletion, locking, hiding, and pinning endpoints.
- How to reconcile the accepted future Tiptap/native-JSON decision with the v1
  shared `posts` table when the rich editor becomes a backend write path.
- When to add tag merge history and taxonomy moderation workflow.
- When to add role-scoped category permissions.

## Next Steps

- Implement topic detail/comment Nuxt consumers for the backend topic and
  comment APIs.
- Add topic moderation/admin endpoints when the moderation UI starts.
- Wire topic/comment writes into the future Meilisearch indexer.

## Search, Cache, And Deep-Pagination (2026-07-08)

千万级数据读路径加固已实现：

- **Meilisearch 全文搜索**：`app/Support/Search` 包提供
  `Indexer`（实现 `forum.TopicSearchIndexer` 与 `searchjobs.TopicIndexer`）
  与 `Service.Search`。新增 `GET /api/v1/search`（query/page/perPage/categorySlug/
  tagSlug），直接查 Meilisearch，不过 PG。`ListTopics` 的 `query` 非空分支改为
  返回 `ErrUseSearchEndpoint`（400），引导前端走专用端点。主题写流程（Create/
  Update/Delete/ApplyTopicAction/CreateComment）事务后 `EnqueueIndex`/
  `EnqueueDelete`，hide/delete 删索引，其余重索引；调度失败只记 slog 不中断。
  依赖解耦：forum 定义窄接口 `TopicSearchIndexer`，search 实现；search → searchjobs
  单向依赖。前端首页 `SFSearch` 搜索框关键词非空时调 `searchTopics`。
  Meilisearch 客户端通过 `NewClientWithTimeout` 注入 `http.Client.Timeout`（默认 5s），
  避免 Meili 宕机时请求挂起。
  **后台重建**：`search.manage` 权限，`POST/GET /admin/forum/search/reindex`
  （触发/进度）+ `GET /admin/forum/search/reindex/runs`（历史）。
  `search.ReindexManager` 扫描 `forum.Store.ListAllTopicIDs` → 分批
  `Dispatcher.EnqueueMany`（River InsertMany）→ `search_reindex_runs` 状态表记录。
  进度实时查 `river_job` 剩余数；并发重建被拒（保证进度精确）；后台
  `/admin/search` 页面带进度条 + 2s 轮询 + 历史。
- **Redis 读缓存**：`app/Support/Cache` 提供 `Cache` 接口 + `MemoryCache`/
  `RedisCache`。`forum.CachedStore` 嵌入 `Store` 接口装饰，缓存分类/分组/标签
  （60s）、主题详情（30s）、主题列表（15s）。失效用 generation 方案（写时递增版本号，
  读 key 含 generation），主题详情按 topicID 精确 Delete。Redis 客户端由
  bootstrap 合并为单一 `sharedRedisClient`（humanverify 与 cache 共用），显式配置
  PoolSize/超时（见 `decisions/2026-07-08-performance-hardening.md`）。
- **深翻页 clamp**：`normalizePage` 增加 `maxTopicPage=200`，消除深分页 OFFSET 扫描。
  评论端点 `page` 参数 OpenAPI 也补了 `maximum: 200`。
- **ListComments SQL 分页改造**（2026-07-08 性能加固）：原实现全量加载 topic 全部
  active 评论再内存分页，已改为：flat 视图直接 SQL `LIMIT/OFFSET`；tree 视图三步
  查询（根评论分页 → `root_comment_id = ANY(...)` 批量拉子孙 → 内存建树）。语义
  不变（Total/Items/view），前端零改动。
- 测试覆盖：`cached_store_test.go`（hit/miss/失效/降级）、
  `service_index_test.go`（各写流程索引调度 + nil/失败降级 + query 拒绝 + 分页 clamp）、
  `service_test.go` ListComments view 校验/默认值/非法值、controller search 端点 +
  query 引导错误测试。决策记录见
  `decisions/2026-07-08-search-cache-deep-pagination.md`。

## Topic Lifecycle (Core Forum V1)

Topic lifecycle is implemented in `apps/api/app/Models/Forum` and exposed via
the public API:

- `PATCH /api/v1/topics/{topicID}` updates title/category/tags/content. Title
  changes regenerate the slug. Content changes preserve the triple-storage
  rule: prior content is copied to `post_revisions` before the `posts` row is
  overwritten.
- `DELETE /api/v1/topics/{topicID}` soft-deletes (status `deleted`, sets
  `deleted_at`, decrements category `topic_count`).
- `POST /api/v1/topics/{topicID}/{hide|restore|lock|unlock|pin|unpin}` apply
  status/pin transitions. `restore` clears `deleted_at`/`locked_at`.

Permission model reuses existing keys: own edit needs `post.edit_own`,
any edit needs `topic.edit_any`, own delete needs `post.delete_own`,
any delete/hide/restore needs `topic.delete_any`, lock/unlock needs
`topic.lock`, pin/unpin needs `topic.pin`.

Public reads are limited to `active` and `locked` topics; hidden/deleted
topics return 404 on public detail and are excluded from public lists. Locked
topics remain readable but reject new comments with `forum.topic_closed`.

Events: `topic.updated`, `topic.deleted`, `topic.hidden`, `topic.restored`,
`topic.locked`, `topic.unlocked`, `topic.pinned`, `topic.unpinned` are emitted
as observe events (see `app/Support/Events/catalog.go`).

`GetTopicForAction` loads a topic summary without public visibility filtering
for permission checks. `ScanTopicSummary`/`RowScanner` are exported so the
Profile model reuses the same SELECT column layout for recent-topic lists.

Frontend: `/t/[...path]` renders topic detail (catch-all, shape controlled by
`seo.topic_url_mode`) with canonical redirect (301 SSR / replace client),
sanitized HTML, comment tree/flat views, reply editor, and permission-aware
action buttons. `/topics/new` provides the composer flow; topic editing is
entered via `?edit=1` query on the detail page and renders `SFTopicEditor`.
Composer and editor use `SFEditor` (submits markdown with
`sourceFormat=markdown`, `editorType=tiptap`).
