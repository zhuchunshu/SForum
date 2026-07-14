# 2026-07-14 Trusted Plugin And Theme Platform V3 P5 Progress

## Changed

- P5 is 47% complete at 8 of 17 authoritative rows; weighted V3 remains 34%.
- Added read-only exact migration preflight and independent-process migration-once proof.
- Enforced per-plugin PostgreSQL connection and slow-query budgets.
- Added stable read-only core views and production Host Query runtime.
- Added durable Host Command receipts, server-attested runtime identity, and a
  real PostgreSQL transactional backend.
- Added explicit admin backup/export, compatibility, retention, digest, and
  non-transactional migration disclosure.

## Verification

- Real PostgreSQL migration, registry, stable-view, Host Query, and Host Command suites passed.
- Focused race and vet gates passed; `go build ./...` passed.
- Database migration packages passed in full with versions 016 and 017.
- Nuxt typecheck, focused trust UI tests, all 284 web tests, and admin validator passed.
- Authenticated `/control-panel/extensions` and the production risk component
  passed desktop plus 390px Browser QA with no overflow or app console errors.

## Decisions

- Migration preflight must remain read-only and share the exact execution parser.
- Stable views are Host-owned, versioned, non-updatable, and grant no base-table access.
- Host API handlers trust only broker-attested identity context, never request protobuf identity.
- Audit retention may delete audit rows; durable ledgers retain numeric audit references without FKs.
- P4 owns process/schedule drain; P5 owns database admission, query, and command semantics.

## Next

- Decide and implement additive database grants plus exact-artifact operation catalogs, or a safe per-runtime direct credential model.
- Implement plugin-owned bounded transactions/DatabaseService.
- Implement six production Host Commands for user/content/meta/moderation/entitlement/attachment.
- Implement exact `database.core.full`, core-upgrade compatibility block, and raw-authority tests.
- Add Host Query/DatabaseService tracing to complete the observability row.

## Open Questions

- Recommended: replace the single authority mode with additive grants while preserving the old field as a compatibility input.
- Recommended: actor-scoped commands use short-lived Host-signed delegation from a core route/admin invocation; background calls remain actorless service authority.
- Recommended entitlement minimum: subject, resource/capability, active/revoked/expired state, source reference, validity window, idempotency, and audit, without billing semantics.

## Resume Point

- Branch: `main`.
- Last implementation commit at this checkpoint: `368e48e4d`.
- Frontend/API dev servers are running on ports 3000/8081 for continued QA.
- Preserve any unrelated worktree changes; stage only V3-owned files/hunks.
