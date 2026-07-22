# 2026-07-22 Forum Content Revisions V1 M5 Handoff

## Changed

- Added `/admin/forum/content` as a permission-aware Forum navigation page.
  Topic and comment tabs are hidden independently unless the actor has the
  matching `*_edit_any` or `*.revision.view_any` permission.
- Added server-side status, author, topic/title/category, updated-time, and
  opaque-cursor pagination controls backed only by the existing admin content
  read models. Non-public content loads through its admin detail endpoint.
- Topic editing reuses `SFTopicEditor`; comment editing reuses `SFEditor`.
  Both canonical PATCH paths send the loaded `expectedRevision`; cross-author
  edits require a reason, and deleted content remains inspection-only.
- A `forum.revision_conflict` response leaves a persistent reload/history
  choice with no force overwrite. M5 does not add timeline, diff, restore, or
  redaction controls.
- Regenerated V3 catalogs. This also records the pre-existing M2-M4 revision
  and admin read routes in the stable identity ledger and classifies their
  route guards without changing service behavior.

## Decisions

- Restore and redaction APIs remain canonical M4 routes but intentionally have
  no M5 UI; M6 owns their confirmation and irreversible-action flows.
- The workbench converts `datetime-local` filter input to RFC3339 before the
  request and preserves a hidden current category slug during topic editing.

## Verification

- `cd apps/web && bun run typecheck`
- `cd apps/web && bun test tests/adminForumContent.test.ts tests/adminForum.test.ts tests/adminRegistryCatalogs.test.ts`
- `node tests/validate-admin-framework.ts`
- `node scripts/v3-catalog/generate.mjs --check`
- `cd apps/api && go test ./app/Support/Routes ./app/Support/ComponentCatalog`
- `git diff --check`

## Next

- Start M6 only: staff revision timeline, lazy detail, diff, historical preview,
  restore confirmation, and super-admin redaction UI.

## Open Questions

- Browser QA could only verify the current 502 API-error boundary because the
  local API was unavailable during M5 validation. Re-run authenticated desktop
  and mobile workbench flows when the API is running.
