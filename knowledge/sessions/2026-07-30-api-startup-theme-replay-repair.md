# 2026-07-30 API Startup Theme Replay Repair

## Changed

- Notification `RevisionHub` now owns cancellable, awaitable shutdown; API
  failure and normal close paths stop it before closing the shared pgx pool.
- Page Registry binding writes preserve a historical approval only while its
  user row exists. Replay after user deletion stores `approved_by = NULL`,
  matching the table's `ON DELETE SET NULL` contract.
- Added regression coverage for deleted-approver replay and listener cleanup.

## Decisions

- Immutable theme runtime publications keep their historical actor ID. Only
  the mutable Page Registry foreign key is normalized when the actor no longer
  exists.

## Verification

- Focused Extensions PostgreSQL replay test passed against the dev database.
- Notifications, Pages, and bootstrap Go tests passed.
- Live `/api/v1/health` and `/api/v1/ready` returned 200 on port 8081.

## Next

- None.

## Open Questions

- None.
