# 2026-08-01 Session Handoff

## Changed

- Added the `account-settings` navigation surface, Manifest V3 fields for
  visibility/retained-resource ownership, lifecycle projection, safe DTO, API
  route, OpenAPI contract, and Nuxt settings-sidebar integration.
- Added `Support/IdentityDelegation` for opaque plugin-scoped identity
  projections and `Support/ConsentBridge` for actor/session/artifact-bound
  one-use consent transactions.
- Added allowed/denied, owner-resource, safe-path, replay, CSRF, recent-auth,
  and exact-artifact tests.

## Decisions

- OAuth Provider remains an external optional plugin; Core contains only
  generic Host contracts and no OAuth business persistence.

## Next

- Expose identity delegation and consent bridge through the versioned Host API
  V2/runtime adapters and add production PostgreSQL/Redis stores where the
  plugin integration requires them.
- Start the independent plugin in a new conversation using the task prompt in
  `tmp/sforum-oauth-provider/2026-08-01-sforum-oauth-provider-development-taskbook.md`.

## Open Questions

- Exact Protocol V2 wire message shape for the two new Host capabilities should
  be finalized before publishing the first external plugin SDK release.
