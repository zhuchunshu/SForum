# 2026-07-13 Session Handoff — Runtime Page Registry P0 Inventory

## Changed

- Added author-facing inventory:
  `docs/extensions/page-catalog.md`
  - Live page ids from `sforum-default` layer + host pages
  - Reserved path prefixes (incl. default admin `/control-panel`)
  - Layer / Web Release / `extension.theme_activate` touchpoints
  - Dual-stack option key names + defaults (docs only)
- Updated plan P0 checklist + catalog summary
- Linked inventory from `knowledge/modules/{extensions,frontend,options}.md`
  and `knowledge/index.md`

## Decisions

- Runtime catalog SOT in **P1 = Go** (this doc is inventory only)
- Flag keys: `pages.registry_enabled`, `themes.runtime_l0_enabled`,
  `themes.runtime_l1_enabled`, `themes.layer_activation_enabled` (default on)
- Extra catalog ids vs plan draft: settings profile/security, notifications,
  moderation review

## Next

1. P1: `feat(pages): add core page catalog definitions` (Go)
2. Registry resolve always-core + `pages.registry_enabled`
3. Admin read-only pages list (API and/or UI)
4. `SFPageOutlet` + migrate **only** `forum.home`
5. Tests + author docs sync

## Explicitly Not Done

- No Options registration for flags yet
- No SFPageOutlet / registry service code
- No Layer deprecation or deletion

## Open Questions

- Whether admin list is always visible or only when
  `pages.registry_enabled` (decide in P1)
- L1 engine still deferred to P3
)
