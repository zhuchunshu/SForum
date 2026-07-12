# 2026-07-13 Session Handoff

## Changed

- Platform: themes may declare `frontend.admin` and settings-page contributions only (`admin.extension.settings.{page,header,footer}`).
- Web Release planner includes active theme admin frontend in composition.
- Default theme: expanded `manifest/settings.json` (~19 keys), custom multi-tab `ThemeSettingsPage`, public consumers wired for empty copy, rail limits, nav toggles, footer/announcement layout.
- Public read path remains `GET /site/active-theme/settings` + `useActiveThemeSettings`.

## Decisions

- See `knowledge/decisions/2026-07-13-theme-admin-settings-page.md`.

## Next

- Manual rebuild: **扩展 → Web 发布 → 立即重建**.
- Typecheck: always runs in release log; default **non-blocking**. Toggle
  hard-fail on the same page (`web_release.typecheck_fail`). CI still forces
  typecheck via `scripts/test.sh`.
- After editing builtin packages on disk: restart API (`SyncBuiltins`) →
  立即重建 → wait active → refresh admin.
- Optional: more theme settings tabs; clear remaining `apps/web` typecheck debt.

## Open Questions

- Whether non-active installed themes should ever snapshot admin frontends for offline preview (currently only active theme).
