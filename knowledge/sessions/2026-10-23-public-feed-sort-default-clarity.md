# 2026-10-23 首页默认排序文案与兜底

## Changed

- 公开首页排序文案改为 最近活跃 (`active`) / 最新发表 (`latest`) / 热门
  (`hot`)，i18n 键名不变，只改值；默认仍是 `active`，`FEED_SORT_PRIORITY` 与
  URL 状态逻辑未动。
- 后台「列表默认排序」改为按 `forumListSortOptions` 渲染，推荐项 `active`
  排第一并带「（推荐）」；新增帮助文案，说明活跃排序与活动索引、以及
  「最新发表」不因回复改变位置。
- `Service.ListTopics` 的排序解析抽成 `resolveTopicListSort`：显式合法值 >
  站点默认 > 推荐默认 `active`。空/非法 `sort` 不再落到 `latest`；空/非法站点
  设置在 service 层被 `resolvedSettings` 拒绝，helper 仍保留 `active` 兜底。
- 架构基线：`Forum/service.go` 从 1083 行降到 1074 行，baseline 同步下调。

## Verification

- `bun test`：1567 pass / 0 fail；`bun run typecheck` 通过。
- `go test ./app/Models/Forum -run 'TestResolveTopicListSort|TestServiceListTopicsResolvesSortToSiteDefault|TestTopicListOrderBy'` 通过。
- `node tests/validate-architecture-boundaries.mjs` 通过。
- 运行时（真实 API）：无 `sort` 与 `sort=active` 同序；`sort=latest`、`sort=hot` 各自不同；
  `sort=bogus` 回到站点默认（active）。
- 浏览器（1633×810 / 390×844）：默认 `最近活跃` 高亮；点「最新发表」→
  `?sort=latest` 且列表顺序与 API 一致；点回「最近活跃」→ 参数清空。英文文案
  经语言切换验证为 `Recently active / Newest threads / Hot`。
- 后台 `/control-panel/forum/settings` 常规 tab：桌面与 390 宽度均显示
  `最近活跃（推荐）` 与帮助文案，无横向溢出。

## Decisions

- 不改默认值：`active` 已是推荐默认，且与 `topics_public_activity_idx` 对齐；
  `latest` 在公开列表没有 `(is_pinned, created_at, id)` 索引。
- 已有实例存 `latest` 时不自动改写，后台标记推荐值并保留「恢复推荐默认值」。
- URL 仍是无 `sort` 即站点默认；显式 `?sort=` 保持可分享并如实高亮。

## Next

- 若未来把 `latest` 设为默认，先补 `(is_pinned, created_at DESC, id DESC)` 索引。
- 分类级 `categories.default_sort` 仍未参与公开查询（既有问题）。
