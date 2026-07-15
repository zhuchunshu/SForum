# 2026-07-16 Trusted Plugin And Theme Platform V3 P5 Closure

## Progress

- P5 is **17/17 (100%)** and contributes its full 8% phase weight.
- Overall weighted V3 progress is **61%** after flooring.
- The five previously open task/test rows were stale accounting. Production
  implementations and acceptance evidence were already committed; `7c00a7fff`
  adds the final entitlement concurrency/revision/revoke exit coverage.

## Accepted Contracts

- Database powers are additive: `own_schema`, `core_views`, `host_commands`,
  `raw_core`, and `kernel`. Legacy `database.authority` expands cumulatively.
- Every direct runtime receives a unique short-lived lease role and credential
  bound to extension, version, package, impact, runtime, and lease identity.
  Source and target may overlap only while both exact leases remain live.
- `raw_core` is the approved implementation of the task-book
  `database.core.full` concept. It grants disclosed Core DML, not DDL,
  ownership, role inheritance, River access, arbitrary function execution, or
  authority over foreign-owned objects.
- Actor-scoped Host Commands require short-lived Host-signed delegation from a
  core route/admin action. Background entitlement mutation is explicit
  actorless service authority and rejects actor delegation.
- Provider-neutral entitlements contain subject, resource/capability, state,
  source, validity, idempotency, and audit only; billing/provider behavior stays
  in plugins.

## Production Evidence

- Stable `sforum_core_v1` views and immutable Host Query catalogs are bound in
  API and worker processes before plugin brokers.
- Six Host Commands cover identity, topic visibility, entity metadata,
  moderation, entitlement, and attachment operations with one PostgreSQL
  transaction for domain writes, events, audit, and durable Host receipts.
- Real PostgreSQL gates cover allowed/denied calls, policy and schema failure,
  idempotent replay/conflict, receipt/storage/audit rollback, migration-once,
  raw-core ACL boundaries, compatibility blocking, source/target overlap,
  heartbeat/drain/reaper cleanup, forged identities, and physical role removal.
- Entitlement evidence additionally covers eight concurrent same-key calls,
  changed payload conflict, stale expected revision, revoke commit/replay, and
  actor-delegation rejection.

## Next

- Do not reopen P5 unless a regression or a new product contract changes these
  accepted boundaries.
- Continue active P6, P7, and P9 rows. P13 must rerun the complete production,
  security, race, browser, and performance gates before legacy removal.

## Open Questions

- None for P5.
