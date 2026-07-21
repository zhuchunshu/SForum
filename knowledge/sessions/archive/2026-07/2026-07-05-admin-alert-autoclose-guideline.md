# 2026-07-05 Admin Alert Auto-Close Guideline

## Changed

- Added a frontend development convention: admin alerts, toasts, and equivalent
  immediate feedback should auto-dismiss after 10 seconds for non-error states.
- Recorded that `error` feedback must remain visible until the user dismisses
  it or the page resolves the blocking issue.
- Updated the admin layout rules decision and frontend module notes so future
  admin pages inherit the same feedback behavior.

## Decisions

- Treat 10-second auto-dismiss as the default behavior for admin success,
  info, neutral, and warning feedback.
- Keep errors persistent because they often require reading details, retrying,
  or changing configuration before continuing.

## Next

- When implementing or refactoring a shared admin feedback helper/component,
  encode this rule as the default so individual pages do not repeat timeout
  logic.

## Open Questions

- None.
