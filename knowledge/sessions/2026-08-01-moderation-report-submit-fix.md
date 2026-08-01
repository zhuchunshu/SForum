# 2026-08-01 Moderation Report Submit Fix

## Changed

- Fixed report creation and legacy report-status updates returning 500 after
  PostgreSQL had already committed the mutation.
- Create, update, list, and get now share the same enriched 16-column report
  projection.
- Added a real PostgreSQL regression covering create, duplicate detection, and
  resolved-state update results.

## Decisions

- Kept the existing one-open-report uniqueness policy and public API contract.
- No frontend change was required.

## Next

- Manually confirm the public dialog returns success for a fresh target and a
  second submission returns the existing duplicate-report message.

## Open Questions

- None.
