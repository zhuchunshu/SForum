# 2026-07-29 Stale Extension Role Suggestion Rejection

## Changed

- Pending extension permission recommendations can now be rejected after their
  exact artifact becomes inactive. Rejection remains CAS-protected,
  `role.manage`-authorized, audited, and unable to create grant evidence.
- Approval and legacy apply remain fail-closed on the exact active plugin,
  Registry declaration, permission catalog, target role, and grant evidence.
- Migration `202607290077` updates the PostgreSQL trigger authority without
  rewriting existing pending or terminal review history.
- OpenAPI and bilingual stale-artifact guidance now explain that inactive
  suggestions cannot be approved but may be rejected and closed.

## Decisions

- Lifecycle cleanup does not silently delete or auto-reject Host review history.
  An authorized human rejection is the explicit terminal action for a stale
  pending recommendation.

## Next

- Apply migrations, then manually confirm that the historical
  `sforum.admin-surface-reference.manage` suggestion cannot be approved, can be
  rejected, leaves `administrator` permissions unchanged, and moves from the
  pending filter to rejected history.
- Automated tests were added but not run in this session at the user's request.

## Open Questions

- None.
