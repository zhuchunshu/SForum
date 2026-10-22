# 2026-10-23 Public Feed Pager Removal

## Changed

- `apps/web/app/components/forum/SFHomePage.vue`：删除底部 `<SFPagination>` 块、
  `totalPages` computed、`homePageTo()`；底部状态区成为唯一加载入口，新增
  `role="status" aria-live="polite"` 与无 `IntersectionObserver` 时的「加载更多」
  降级按钮（`canAutoLoad`）。
- `apps/web/app/assets/css/sforum-home.css`：删除 `.sforum-home__pagination` 隐藏规则。
- `apps/web/i18n/locales/{zh-CN,en-US}.json`：新增 `home.feed.loadMore`。
- `apps/web/tests/forum/defaultThemeHomepage.test.ts`：反转 `:page-to="homePageTo"`
  断言为「信息流不含 SFPagination / homePageTo / sforum-home__pagination」，并锁定
  `aria-live`、`canAutoLoad`、`home.feed.loadMore`；分类页/标签页断言仍含
  `SFPagination`。
- 新增决策：`decisions/2026-10-23-public-feed-loading-model.md`。

## Decisions

- 公共信息流（首页 + 搜索）= 只追加，不渲染页码；`?page=N` 仍是锚点深链。
- 分类/标签/评论/审核台/后台保留分页（归档语义，无自动加载）。
- 不改 Go API / OpenAPI；不重建内置主题制品，主题里的
  `.sforum-home__pagination { display:flex }` 已成为死规则，留待下次制品重建清理。

## Verification

- `cd apps/web && bun test tests/forum/defaultThemeHomepage.test.ts tests/forum/publicPagination.test.ts` → 20 pass。
- `cd apps/web && bun run typecheck` → exit 0。
- `node tests/validate-architecture-boundaries.mjs` → passed（1912 files）。
- 浏览器实测（ego_*，PC 1633×810 / 移动 390×844）：首页 DOM 无 `.sf-pagination`，
  滚到底追加下一页，到底文案正确；`/?page=2` 首屏即第 2 页；分类/标签页码仍在。

## Next

- 下次重建 `sforum-default` 主题制品时删除 `assets/theme.css` 里的
  `.sforum-home__pagination` 规则，并跑 `extension digest --write` / `validate` /
  `test` + 重启 API 激活。
- 若运营需要「跳到第 N 页」，应作为独立的 feed 定位器（顶部页码 + 重置窗口），
  不要复用 `currentPage` 驱动追加。

## Open Questions

- 分类/标签页是否也要改成只追加？当前保留分页是有意区分（归档页需要稳定定位）。
- 通知（游标 + 按钮）与个人主页动态（页码 + 按钮）仍是另外两套模型，是否收敛未定。
