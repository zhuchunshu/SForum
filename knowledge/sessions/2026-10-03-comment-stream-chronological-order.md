# 2026-10-03 评论流改为时间流排序（最底楼 = 新回复）

## Changed

- **flat 评论流排序键换成发表时间**：`listCommentsFlat` 从
  `ORDER BY path_key ASC, id ASC`（树的前序展开，回复被插在被回复评论下面）
  改为 `ORDER BY created_at ASC, id ASC`。新评论恒追加在列表末尾，既有评论
  的楼层号不再因他人回复而漂移。抽取到
  `apps/api/app/Models/Forum/postgres_store_comments_flat.go`（列表 / keyset /
  楼层计数三件套同文件对照），`postgres_store_ops.go` 1604 → 1490 行。
- **位置口径全部对齐同一排序键**：`commentListCursor` 载荷改为
  `{v, k: RFC3339Nano(created_at), i: id}`（旧 path_key 游标统一 400
  `forum.cursor_invalid`，客户端重新取第一页）；`CountCommentsBefore` 改比较
  `ROW(created_at, id)`；`ResolveCommentPage` 传 `summary.CreatedAt`；个人主页
  公开动态的 `commentPage` 子查询同步改 `ROW(siblings.created_at, siblings.id)`。
- **迁移 `202610030001_forum_comment_flat_created_index.sql`**：新增
  `comments_topic_created_idx (topic_id, created_at, id)`（`CONCURRENTLY`，
  `-- +goose NO TRANSACTION`），同时服务分页 ORDER BY 与 keyset seek。本机开发库
  已应用（API 启动迁移）。
- **回复后落到新评论**：`useTopicCommentSubmission` 新增
  `focusCreatedComment` 钩子（仅发布成功后调用，待审回复不跳）；
  新 composable `apps/web/app/composables/forum/useTopicCommentAnchor.ts` 承接
  深链滚动/闪烁与 `focusCreatedComment`（按服务端反查页码跳到
  `/page/N#comment-<id>`，复用既有滚动 + 微光高亮）；快速回复与高级回复抽屉共用。
  `SFTopicShowPage.vue` 1249 → 1168 行，两个文件的架构基线同步下调。
- 契约描述更新：`contracts/openapi/paths/forum.yaml`（flat = 时间流 + cursor =
  created_at + id）、`schemas/forum.yaml`（`nextCursor`、`Comment.pathKey` 说明）。

## Verification

- 真库集成测试 `TestPostgresFlatCommentStreamIsChronological`（新，fixture 自带
  最小 schema）：树序 `[1 3 5 6 2 4]` vs 时间流 `[1 2 3 4 5 6]`——把 SQL 临时改回
  path_key 会让该测试失败（已做变异验证），同时覆盖 `CountCommentsBefore` 逐条对齐、
  keyset 续页不重不漏、tree 视图父子结构不变。
- 运行时（开发 API :8080，topic 403）：flat 返回 `#1..#10` 按时间升序，
  最新回复 1676 在最后一楼；`perPage=3` 的 page1/2/3/4 与 `after` 游标续页一致；
  旧 path_key 游标返回 400；`/comments/{id}/page` 正常。
- SSR HTML（Nuxt :3000）：DOM 顺序 `1562,1563,1564,1565,1566,1672…1676`，
  楼层标签 `#1..#10` 升序，7 个回复行仍带 replyTo 引用卡。
- `go test ./...` 全量通过；`bun test` 923 pass；`bun run typecheck` 通过；
  `node tests/validate-architecture-boundaries.mjs` 通过；OpenAPI `$ref` 校验
  2726 refs / 56 files 通过（本机无 ruby，用等价的 Node 脚本执行同一算法）。
- **未做**：渲染态 Browser QA（本会话 BrowserSkill CLI 缺失、桌面 computer_* 工具
  报 `shell.run is not a function`），所以「点回复 → 视口滚到新楼层」只做了代码 +
  SSR + 接口层证据，未做真实点击截图。

## Decisions

- flat = 时间流（楼层稳定），tree 视图保留 path_key 层级，供需要嵌套的消费者使用。
- 不新增运营选项；`view=tree` 已经是嵌套路径，选择排序的运营开关等真实需求出现再说。
- 见 `decisions/2026-10-03-comment-stream-chronological-order.md`。

## Next

- 上线提醒：`CachedStore.ListComments` 有 20–40s Redis TTL（第一页 40s），部署后
  最多 40s 内可能仍读到旧顺序的缓存 payload；写入 bump 或 TTL 到期即恢复，无需手工清缓存。
- flat 读路径基准（深 `page` + keyset）按新排序键重跑，落到 `knowledge/reports/`；
  M3/M5 报告的数字是 path_key 时代的。
- `SFCommentStreamControls.vue`（树/平铺切换）已无生产引用，可连同 i18n 键一起清理。

## Open Questions

- 回复卡是否要在人名前补父评论楼层（`回复 #12 张三`）？时间流下父评论可能跨页，
  需要后端在 `replyTo` 里给出楼层或前端只对已加载页显示。
