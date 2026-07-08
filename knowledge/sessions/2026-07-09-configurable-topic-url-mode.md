# 2026-07-09 Session Handoff

## Changed

后台可配置的帖子 URL 模式（`seo.topic_url_mode`）已落地，分 5 个逻辑提交：

- **后端**：新增 `seo.topic_url_mode` option（`id_slug`|`id`|`slug`，默认
  `id_slug`，SEO 管理权限）；`Forum.Store` 新增 `GetTopicBySlug` +
  `TopicSlugExists`；新增 `GET /topics/by-slug/:slug` 公开端点；`Service`
  新增 `ensureUniqueTopicSlug`（冲突追加 -2/-3 后缀）；迁移
  `202607090001` 把 `topics.slug` 升级为 UNIQUE 索引（先去重）；OpenAPI
  新增 `topicBySlug` path + `TopicSlug` 参数 + 选项键。
- **前端**：`useWebOptions` 接线 `topicUrlMode`（含
  `normalizeTopicUrlMode`/`recommendedTopicUrlMode`/`topicUrlModes`）；
  `forumTopicPath(topic, mode)` 成为唯一链接出口；新增 `parseTopicPath` +
  `previewTopicSlug`；8 个调用点（首页/分类/标签/用户/新建/编辑/详情/审核）
  全部传入当前 mode。
- **详情页**：从 `t/[topicID]/[topicSlug].vue` 迁移到 catch-all
  `t/[...path].vue`，按 mode 解析 + 规范化 301（SSR `redirectCode:301` /
  客户端 `replace`）；评论查询改用 `topic.value?.id`（slug 模式下
  `topicID` 可能为 0）。
- **编辑**：从 `t/[topicID]/[topicSlug]/edit.vue` 改为 `?edit=1` query +
  独立组件 `SFTopicEditor.vue`（避免 catch-all 嵌套子路由的渲染出口问题）。
- **SEO 管理页**：新增「链接结构」tab，radio-card 三选项 + 预览 URL +
  slug 模式警告；zh-CN/en-US i18n 已补。
- **测试**：`forumTopic.test.ts` 补 mode/parseTopicPath 用例（18 pass）；
  后端 `TestServiceCreateTopicDeduplicatesSlugOnCollision` +
  `seo.topic_url_mode` 校验用例。

## Decisions

- **编辑入口用 `?edit=1` 而非 `/edit` 路径**（偏离原计划
  `t/[...path]/edit.vue`）：catch-all 父路由不渲染 `<NuxtPage/>`，嵌套子
  路由无法显示；`?edit` + 独立组件更干净且详情页不超 1000 行。详见
  `knowledge/decisions/2026-07-09-configurable-topic-url-mode.md`。
- **v1 的 301 只做同帖规范化**（模式切换/slug 变更），不做跨模式历史映射
  表（`topic_redirects`）；ID 稳定 + slug 全局唯一已足够兜底。
- **v1 不含 `category_slug` 模式**（`/c/general/hello-world`）：需分类联动
  且与 `/c/:slug` 列表路由消歧，留 v2。

## Next

- 端到端验证：手动切三种模式，验证首页/列表链接形态、详情页 301、编辑
  保存后跳转、sitemap 输出（sitemap 当前未生成 topic URL，确认无影响）。
- 若要支持 `category_slug` 模式：需后端按 `categorySlug` + topicSlug 联查，
  前端 `parseTopicPath` 扩展，并处理与 `/c/:slug` 的路由消歧。
- 考虑 slug 模式下的 sitemap 接入（`_sitemap-urls.ts` 当前早返回，未生成
  topic URL；若开启 `sitemapIncludeForumContent` 需按 mode 产出）。

## Open Questions

- `slug` 模式下，已 hidden/deleted 的主题 slug 仍占用唯一约束（`TopicSlugExists`
  统计所有状态），避免删除后 slug 被复用造成 URL 歧义——是否符合预期？
- 编辑页 `?edit=1` 对未登录用户会渲染「权限拒绝」提示而非重定向到登录；
  全局 auth 中间件仅在 `requiresAuth` 时拦截，当前 `public:true`，需确认
  是否要给编辑态加登录重定向。
