# 2026-07-23 Forum Content Revisions V1 M6 Authenticated QA Handoff

## Changed

- Removed the never-published `source_format=json` schema option. Migration
  `202607230054` fails clearly if manually inserted legacy rows exist, then
  narrows `posts` to `markdown`, `html`, and `editor-document`. Runtime,
  frontend types, and OpenAPI now use the same set; Tiptap JSON remains stored
  as accepted `raw_content` under the explicit `editor-document` contract.
- M6 is complete. Fixed the Nuxt UI/Reka `USelect` contract in
  `apps/web/app/pages/admin/forum/content.vue`: the all-status option now uses
  the non-empty `__all__` sentinel and normalizes it to an omitted API filter.
  Both selects declare `value-key`/`label-key` explicitly.
- Reused the existing local `super_admin` Chrome session at
  `http://127.0.0.1:3000/control-panel/forum/content`; no account or credential
  was read, reset, or changed. It completed topic edit v1→v2, lazy history
  detail/diff/preview, v1 restore to v3, super-admin redaction of v2, a second
  current v4 save plus stale-v3 conflict UI, and comment edit v1→v2/restore-v3.
  A 390x844 viewport rendered the topic preview/diff without overlap.
- Fixed the discovered comment-update 500: `GetCommentSummary` now qualifies
  `comments.id` in its joined query. Added PostgreSQL regression coverage.
- Guarded the revision-action modal content by `revisionAction` and switched to
  the existing `admin.common.cancel` copy, removing the M6 null-key and cancel
  i18n warnings. New desktop session console capture has no app warnings/errors.

## Decisions

- M0-M6 ledger, permission, expected-revision, 409, and no-force-overwrite
  decisions remain frozen for M7.
- The status sentinel is UI-only: `requestFilters()` maps it to `''`, which the
  existing query builder omits. No API, permission, expected-revision, or 409
  conflict semantics changed.

## Next

- Begin M7 only from `knowledge/plans/2026-07-22-forum-content-revisions-v1.md`.
  Keep using only the existing revision/restore/redaction API surface and do not
  add force overwrite or public history.

## Open Questions

- The ephemeral local QA topic `M6 QA revision lifecycle` and its comment remain
  in the development database as explicit test evidence; topic revision 2 is
  intentionally redacted. This is test data only, not a product migration.
