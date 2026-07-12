# 2026-07-12 Session Handoff — Personalization + Site Chrome merge

## Changed

- Merged admin **品牌与前台壳** (`/site-chrome`) into **个性化设置**
  (`/personalization`) as page-level tabs.
- Extracted chrome UI into
  `apps/web/app/components/admin/SFAdminSiteChromePanel.vue`.
- Personalization tabs (larger `size="md"` buttons, bottom border):
  - 外观与页脚 (`appearance`) — `settings.appearance.manage`
  - 品牌 / 导航 / 公告 / 法律页 / 友情链接 — `settings.site.manage`
- Page registry: `/personalization` uses `permissionMode: 'any'` for
  appearance or site manage; tabs filter by permission.
- Sidebar: removed `/site-chrome` from System folder.
- `/site-chrome` route kept as redirect to `/personalization?tab=…`.
- i18n zh-CN / en-US updated (intro, toolbar, tab labels, no-access copy).
- Framework test: system folder order unchanged; asserts no sidebar
  site-chrome and personalization permission merge.

## Decisions

- Keep one personalization hub rather than nested settings under site settings.
- Preserve old `/site-chrome` URLs via redirect for bookmarks.
- Do not change API permissions: chrome CRUD still `settings.site.manage`.

## Next

- Optional: deep-link restore of open admin tab when landing from redirect.
- Optional polish from Wave 2 (attachment URL resolve, Markdown legal render).

## Open Questions

- Whether operators want a dedicated “Brand” top-level name instead of
  “Personalization” after the merge (product copy only).
