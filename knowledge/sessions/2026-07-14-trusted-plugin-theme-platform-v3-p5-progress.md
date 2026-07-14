# 2026-07-14 Trusted Plugin And Theme Platform V3 P5 Progress

## Changed

- P5 is 65% complete at 11 of 17 authoritative rows; weighted V3 is 36%.
- Added read-only exact migration preflight and independent-process migration-once proof.
- Enforced per-plugin PostgreSQL connection and slow-query budgets.
- Added stable read-only core views and production Host Query runtime.
- Added durable Host Command receipts, server-attested runtime identity, and a
  real PostgreSQL transactional backend.
- Added explicit admin backup/export, compatibility, retention, digest, and
  non-transactional migration disclosure.
- Added a pre-Goose raw-authority core-version compatibility gate and bounded,
  redacted Host Query tracing with slow classification.
- Production-bound exact-artifact own-schema DatabaseService transactions in
  API and standalone worker bootstrap. Active, disabled/installed, staged,
  rollback, and upgrade artifacts load immutable digest-checked SQL catalogs;
  running brokers retain their old snapshot while future brokers capture the
  newly published snapshot.
- Added bounded, redacted DatabaseService query/execute tracing with exact
  artifact and operation attribution, slow classification, and no SQL,
  parameter, result, credential, idempotency-key, or error-text disclosure.
- Added durable exact route-provider selection Store/API with revision CAS,
  enabled active artifact validation, append-only events, target/provider
  contract drift rejection, and V2 disable/uninstall invalidation before
  physical cleanup. Bootstrap, admin UI/API, and Fiber dispatch remain open.
- Hardened P6 foundations with exact runtime instance fencing, wildcard method
  conflict visibility, the production 212-route core snapshot, Safe Mode
  filtering, and immutable execution plans. P6 remains uncredited until the
  request dispatcher consumes the Registry.

## Verification

- Real PostgreSQL migration, registry, stable-view, Host Query, and Host Command suites passed.
- Real PostgreSQL core-upgrade blocking and DatabaseService transaction,
  idempotent replay, core-isolation, and revoke tests passed.
- Database catalog bootstrap focused/race/vet/build gates passed across
  bootstrap, HostAPI, and Extensions; exact enable/upgrade broker snapshots and
  trace redaction are covered.
- Route provider selection focused, race, vet, and real PostgreSQL CAS/stale/
  audit/invalidation tests passed. V1 provider cleanup remains compatible.
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

- Decide and implement additive database grants while preserving the legacy
  authority input.
- Implement six production Host Commands for user/content/meta/moderation/entitlement/attachment.
- Implement exact `database.core.full` and raw-authority tests.
- Production-bind route provider selection, expose the permissioned admin
  conflict workflow, and make the Fiber dispatcher consume selected plans.

## Open Questions

- Recommended: replace the single authority mode with additive grants while preserving the old field as a compatibility input.
- Recommended: actor-scoped commands use short-lived Host-signed delegation from a core route/admin invocation; background calls remain actorless service authority.
- Recommended entitlement minimum: subject, resource/capability, active/revoked/expired state, source reference, validity window, idempotency, and audit, without billing semantics.

## Resume Point

- Branch: `main`.
- Last implementation commit at this checkpoint: `dc49dcf6d`.
- Frontend/API server state is not durable; start both explicitly before QA.
- Preserve any unrelated worktree changes; stage only V3-owned files/hunks.
