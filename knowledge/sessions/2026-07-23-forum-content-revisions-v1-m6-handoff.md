# 2026-07-23 Forum Content Revisions V1 M6 Handoff

## Changed

- Added staff revision timeline headers with lazy detail reads, source and
  metadata comparison, sanitized historical preview, and explicit legacy or
  redacted states in the admin content workbench.
- Restore uses the existing topic/comment revision routes, sends the loaded
  `expectedRevision`, requires a reason, preserves 409 reload/history behavior,
  and reports the newly allocated revision in a 10-second success Toast.
- Redaction is rendered only for active `super_admin` actors, requires a reason
  plus typed `REDACT`, and keeps the irreversible warning visible in its modal.
- Added direct BSD-3-Clause `diff@9`, focused revision UI tests, explicit page
  imports for admin child components, and reviewed V3 component identities.

## Decisions

- No admin mutation route was added. Timeline, restore, and redaction use only
  the canonical revision API surface; no force overwrite or public history was
  introduced.
- The runtime admin route is `/control-panel/forum/content`; `/admin/...` is a
  legacy source/catalog path and reaches the public dynamic registry at runtime.

## Next

- With a supplied testable admin session, run desktop and mobile topic/comment
  edit-history-restore flows and two-tab stale-CAS checks before M7. Do not
  reset or alter the existing super-admin account for QA.

## Open Questions

- The local API health endpoint is available. Unauthenticated desktop/mobile
  route guard was verified; authenticated workbench browser QA remains blocked
  only by unavailable test credentials.
