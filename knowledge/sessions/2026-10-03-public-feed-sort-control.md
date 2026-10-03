# 2026-10-03 首页公开列表排序控件与排序契约

## Changed

- **排序控件样式落到 Core 层**：新增 `apps/web/app/assets/css/sforum-sort-control.css`
  并在 `nuxt.config.ts` 注册（在 home / taxonomy / tags 之后、highlight-theme 之前）。
  首页 feed 排序、分类目录排序、标签页筛选统一为分段控件：容器 36px/圆角 6px/
  `--sf-public-surface-muted` 底 + 1px 边框，选项 28px 高（窄屏 32px）、选中态为
  「白色胶囊 + 强调色文字 + 1px 阴影」。全部几何与状态走 `--sf-sort-*` 变量。
- **样式必须覆盖两条渲染路径**：每个选择器都同时声明裸类名（Core fallback）与
  `.sf-page.sf-theme--default …`（主题皮肤），与 `sforum-home-pinned.css` 的既有约定
  一致。此前首页排序按钮只在 `hybrid-forum.css` 内定义且被 `.sf-theme--default` 限定，
  而当前实例所有公开页都走 Core fallback（`data-provider=core` / `data-template=0`），
  导致按钮退化成纯文字（无 padding、无选中态、点击目标仅 19px）。
- **排序改为服务端 sort + URL 状态**：`ForumHomeFilters` 增加 `sort`；
  `parse/build/forumHomeFeedKey` 全部带上 sort；`SFHomePage` 删除本地 `displayTopics`
  重排，选项集合由 `forumTopicSorts` 枚举派生（契约枚举），展示顺序由
  `FEED_SORT_PRIORITY` 决定（最新 → 最新帖子 → 热门，对应
  `GET /topics?sort=active|latest|hot`）。换 sort 即换 feedKey → 分页、cursor、
  `hasLoadedAllPages` 自动重置。选中站点默认排序（`forum.list.default_sort`）时清掉
  URL 参数；关键词搜索态不展示排序入口（搜索端点没有 sort）。
- **公开文案与默认值**：公开控件文案按运营要求互换命名——「最新」= `active`
  （最后活跃，默认选中）、「最新帖子」= `latest`（发布时间）、「热门」= `hot`。
  推荐默认排序由 `latest` 改为 `active`，三处同步：`Models/Forum/content_limits.go`、
  `Models/Options/community_policy_options.go`（+ 对应单测期望）、
  `web/utils/forum/forumTaxonomy.ts` 的论坛设置默认值；`SFHomePage` 的
  `webOption` 回退也改为 `active`，避免站点未配置时高亮与服务端排序不一致。
  本机实例的 `web_options.forum.list.default_sort` 已从 `latest` 写为 `active`
  （数据变更，非代码；`site.public_surface_revision` 是宿主内部选项，普通选项写入
  不 bump，故未动）。
- 移动端不再 `hidden`：窄屏改为横向可滑的分段控件。
- 空 `sort` 由 service 回填站点默认；`SFNavbar.submitSearch` 显式传 `sort: ''`。

## Decisions

- 排序只在服务端做，前端不做二次排序：本地重排只能作用于已加载页，且会破坏
  `is_pinned DESC` 第一维与 cursor（游标与 sort 绑定）续页语义。
- 推荐默认排序 = `active`（最后活跃）：一是公开列表没有 `(is_pinned, created_at, id)`
  索引，`latest` 在首页路径上得不到索引支持，而 `active` 对齐
  `topics_public_activity_idx`；二是论坛「最新」惯例就是最新活动。详见
  `decisions/2026-10-03-public-list-default-sort-active.md`。
- 不新增 API 枚举。「回复最多」语义下线，改用契约已支持的 `hot`；若将来要真正的
  「按回复数」排序，需新增 `replies` 枚举 + `(is_pinned, comment_count DESC, id DESC)`
  索引 + cursor 绑定，属于后端契约变更。
- 选中态采用「软胶囊」（面 vs 容器 ≈1.09:1），可读性由文字承担
  （选中 5.17:1 / 未选 5.13:1，均过 AA）。若要求 WCAG 1.4.11 的非文字 3:1，
  需改为强调色实心胶囊 + 白字。

## Verification

- `bun test`：922 pass / 0 fail；`bun run typecheck` 通过；`go build ./... && go test ./...`
  通过；`node tests/validate-architecture-boundaries.mjs` 通过。
- 运行时第 1 轮（Chrome CDP 真实点击）：点击排序项 → URL 变 `?sort=...`、
  请求 `GET /api/v1/topics?page=1&sort=...`、列表顺序与 API 完全一致。
- 运行时第 2 轮（改名 / 顺序 / 默认）：默认态 URL 无参数、高亮「最新」（第一项）、
  首 6 条 `[3,1,424,403,422,413]` 与 API `sort=active` 及无 sort 默认一致；
  点「最新帖子」→ `?sort=latest` + 顺序 `[3,1,426,425,424,423]`；点回「最新」→
  参数消失、请求退回无 sort；点「热门」→ `?sort=hot` + `[1,3,2,401,402,4]`；无控制台错误。
  桌面 1440 / 390×844 / 暗色态截图确认形态与选中态。


## Next

- 分类页 / 标签页话题列表仍没有排序入口（API 支持 category/tag + sort）；
  标签页「本周」筛选语义是「近 7 天新增标签」，不是活跃。
- 右栏「热门讨论（按回复）」仍是对已加载页按 `commentCount` 排序，未走 `sort=hot`。
- `forum.list.hot_window_days`（默认 7）在查询里从未被使用：要么落地为
  `hot` 的时间窗，要么从 admin 撤掉。
- 分类级 `categories.default_sort`（后台分类「默认排序」）只被存储与校验，
  公开列表查询从未读取它：要么落地为分类过滤的默认排序，要么撤掉入口。
- 主题包内的 `.sf-theme--default .sforum-home__feed-sort` 规则已冗余；清理需走
  `extension digest --write` + `validate` + `test` 与重新激活。

## Open Questions

- 排序控件目前只有三个契约枚举项；若要「精华 / 关注 / 我的帖子」，需要新的服务端
  过滤契约，不能只加按钮（假入口）。
