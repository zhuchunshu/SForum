# 2026-07-14 Trusted Plugin And Theme Platform V3 P5 Progress

## Changed

- P5 is 65% complete at 11 of 17 authoritative rows. P6 is the active phase at
  56% (10 of 18 rows), and weighted V3 is 41%.
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
- Added the detached Route Inspector core and bounded redacted trace ring in
  `738b8a30b`. Inspection filters to the requested method/path, distinguishes
  selected, unselected, and stale providers without an implicit winner, and
  freezes exact guard/schema/fallback/artifact evidence at one Registry
  revision.
- Production-bound the same Route trace ring to the Dispatcher and Inspector in
  `3b017173c`, then fixed non-handler readonly fallback/commit attribution in
  `61da559d5`. The production Inspector row is now accepted.
- Replaced plugin self-reported side-effect authority with Host-observed
  transport evidence in `caa158402`. Safe pre-write GET fallback remains
  available, while accepted requests, partial responses, crashes, timeouts,
  cancellation, and unsafe methods can never execute Core as a second writer.
- Added the complete 23/23 Page ViewModel registry and sealed typed theme render
  output in `d9268872a`. P8 remains uncredited pending production construction
  and runtime publication.
- Added reviewed typed Core guards and exact inherited guard snapshots in
  `1fcfdbf05`; every returned permission slice is detached and mutation-tested.
- Added the exact-artifact Route Schema Catalog in `6667e630f`, including exact
  operation/status/media binding, HEAD response policy, duplicate/trailing JSON
  rejection, and bounded decode/validation. It remains uncredited until the
  lifecycle publishes it and bootstrap injects a non-nil catalog.

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
- Authenticated route-provider QA rendered once in Chrome, but API hot reload
  dropped during data loading and the session expired before retry. Do not
  treat route-provider desktop/mobile interaction QA as passed.

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
- Land the reviewed Fiber adapter only after pure core routes bypass buffering;
  then add exact schema catalogs, inherited/custom guard execution, Dispatcher
  trace publication, and Host-observed side-effect fencing.
- Land Inspector HTTP/OpenAPI separately, followed by its admin UI. Keep SSE,
  WebSocket, multipart, streaming, and backpressure open until real transports
  and disconnect tests pass.

## Open Questions

- Recommended: replace the single authority mode with additive grants while preserving the old field as a compatibility input.
- Recommended: actor-scoped commands use short-lived Host-signed delegation from a core route/admin invocation; background calls remain actorless service authority.
- Recommended entitlement minimum: subject, resource/capability, active/revoked/expired state, source reference, validity window, idempotency, and audit, without billing semantics.

## Resume Point

- Branch: `main`.
- Last implementation commit at this checkpoint: `6667e630f`.
- Frontend was started on port 3000 and API hot reload recovered on 8081, but
  process and login state are not durable; verify both before QA.
- Exact route schemas are clean and committed. Production publication is
  blocked on the missing Host-owned route policy source and the need to restore
  Route + Schema snapshots together at boot; do not inject a standalone catalog.
- Active dirty files are the isolated Dispatcher benchmark/report group. Parallel
  agents own production Core Guard authorization and P8 ViewModel publication.
  ThemeCompiler stays uncredited until production binding.
- Preserve any unrelated worktree changes; stage only V3-owned files/hunks.
