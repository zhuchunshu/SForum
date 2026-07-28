# 2026-07-29 Announcement Editor Handoff

## Changed

- Replaced bare announcement inputs with labeled bilingual fields, help text,
  validation, style/link/order controls, active-window inputs, and explicit
  dismissible/enabled switches.
- Added `SFEditor`'s `basic-field` preset: Markdown-backed Tiptap with only
  undo/redo, bold, italic, links, and lists. It excludes trusted L2 modules and
  non-announcement nodes.
- Kept stored announcement bodies as Markdown and added server-rendered,
  sanitized `bodyHtmlZhCN` / `bodyHtmlEnUS` response fields for admin/public
  presentation, with a temporary plain-text response fallback.

## Decisions

- No database migration or new dependency. Existing Goldmark and Bluemonday
  libraries own Markdown rendering and the restricted HTML allowlist.
- Existing create, enable/disable, and delete workflows remain; editing an
  existing announcement is outside this change.

## Next

- Operator manually verifies `/control-panel/personalization?tab=announcements`
  at desktop/mobile widths and checks a created announcement in public chrome.
- Automated tests, typecheck, OpenAPI validation, and Browser QA were not run
  at the operator's request.

## Open Questions

- None.
