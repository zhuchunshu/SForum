# 2026-07-22 Hybrid Demo Fidelity Polish

## Changed

- **Fonts / weights**: public shell forces `DM Sans` + `Noto Sans SC` (override host Inter); feed title uses `Noto Serif SC` 27/700 (Google import + Songti SC fallback); topic titles 14/600; nav/button weights aligned to demo (600/700, drop 650).
- **Topbar active**: `isDesktopNavActive` exact-match for home; `is-active` class + 2px accent underline; active text is ink (not accent red).
- **Topic list**: row matches hybrid demo — title/meta (`作者 · 时间`), category chip tones, reply count, **最近活动** column (`sf-home-topic-row__activity`); activity labels use `…前` / `…ago`.
- **Left nav categories**: management `icon` + `iconColor` rendered as colored icons (`sf-home-navigation__cat-icon`), not plain dots; nav icons sized 18px.

## Files

- `apps/web/app/components/SFNavbar.vue`
- `apps/web/app/components/SFHomeNavigation.vue`
- `apps/web/app/components/SFHomeTopicRow.vue`
- `apps/web/app/utils/forumListPresentation.ts`
- `apps/web/app/assets/css/sforum-theme.css`, `sforum-home.css`
- `extensions/builtin/themes/sforum-default/assets/hybrid-forum.css`, `theme.css`
- `apps/web/i18n/locales/zh-CN.json`, `en-US.json`
- tests: `defaultThemeHomepage`, `defaultThemeNavbar`

## Next

- Visual side-by-side with `tmp/demos/sforum-hybrid-topic-list` (served on `:8765` during session).
- If API theme asset cache serves stale package CSS, republish/restart active theme package so `hybrid-forum.css` picks up.
- Optional later: API `lastReplyAuthor` for true “最近回复” identity (currently author + `lastActivityAt`).

## Open Questions

- Whether left rail should also list tags with colored icons (demo only lists categories).
