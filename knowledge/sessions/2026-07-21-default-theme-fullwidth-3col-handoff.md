# 2026-07-21 Session Handoff

## Changed

- Default public shell is **full-width three-column flat** (demo
  `tmp/demos/grok/forum-fullwidth-3col/`).
- Theme package (primary): `extensions/builtin/themes/sforum-default/`
  - `assets/tokens.css`, `assets/theme.css` — flat tokens + full-bleed grid
    (`#f5f6f8`, sidebar `220px`, `--sf-public-shadow: none`)
  - `templates/home.html`, `templates/topic-show.html` —
    `data-layout=fullwidth-3col` + `sf-theme-shell--fullwidth-3col`
- Host islands/CSS (fail-closed presentation mount):
  - `SFTopicShowPage` — left `SFHomeNavigation` (route) + main + `SFTopicSideCard`
  - `SFHomePage` — `data-layout` marker
  - `sforum-theme.css` / `sforum-home.css` / `sforum-topic.css` baselines
- Tests: `defaultThemeHomepage.test.ts`, `defaultThemeTopicPage.test.ts` (19 pass)
- Docs: `knowledge/modules/frontend.md`, theme README

## Runtime activation (dev verified)

- Builtin root: `BUILTIN_EXTENSION_ROOT=storage/builtin-dev`
- After source edits: rsync → `storage/builtin-dev/themes/sforum-default/`, restart
  API so `SyncBuiltins` **stages** a new package digest (does not auto-promote an
  already-enabled theme).
- Super admin must `POST /api/v1/admin/extensions/sforum.default-theme/activate`
  with exact preview tuple + `approveCoreReplacements: true` to bind L1 pages.
- Verified active digest:
  `9e13d7349e39902a31f906461a801ca7a254d724762ac38bce0fac5d2845dbb2`
  - `/api/v1/site/active-theme/skin` → new digest
  - theme-assets tokens serve `#f5f6f8` / `220px` / `shadow: none`
  - `pages/resolve` `forum.home` + `forum.topic.show` → `provider=sforum.default-theme`
  - 19 `page_provider_bindings` rows; theme publication revision with
    `core_replacements_approved=true`
  - SSR `/`, `/t/...`, `/login`, `/c/general`, `/categories` →
    `data-provider=sforum.default-theme`, shell `fullwidth-3col`
  - Stress: sequential resolve×50 ok; concurrent skin/resolve/assets ok

## Decisions

- Collapse: hide right rail ≤1180px, single column ≤960px.
- No fake engagement UI; right rails remain API-backed.
- Accent still from appearance `--sf-accent`.
- Presentation code lives in the theme package; host islands keep the same
  fullwidth-3col layout for core fail-closed fallback.

## Next

- Optional: operator screenshot pass in browser at 1440/1024/720 widths.
- Nocturne theme left unchanged (non-goal).
- Fresh envs: after first boot SyncBuiltins stages/activates default theme;
  existing enabled digests need explicit re-activate when package content changes.

## Open Questions

- None for this shell ship.
