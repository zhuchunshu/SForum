# 2026-07-21 Forum settings search provider tab

## Changed

- Added admin search.provider APIs under `/admin/forum/search/providers|provider|provider/reset` (`search.manage`).
- Extended forum settings UI with **Search** tab: select provider, restore site search default, reindex shortcut.
- OpenAPI + catalog identities for the three new routes; core route catalog count 249 → 252.
- i18n zh-CN / en-US for the new tab.

## Decisions

- Search provider selection lives on forum settings (product UX), not only on generic provider-slots.
- Selection is independent of forum.* form save/restore (like mail providers).
- Only the selected engine is indexed/queried; switching requires reindex.

## Next

- Optional: surface probe/health detail for Meili on the same tab.
- Optional: after select, auto-prompt reindex modal instead of Toast hint only.

## Open Questions

- None for this slice.
