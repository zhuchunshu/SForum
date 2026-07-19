# 2026-07-19 V3 Session Policy Selection Checkpoint

## Progress

- Weighted progress remains **67.8295%** (display **67.0%**).
- P7 remains **18/22**. The committed Store is infrastructure and receives no
  row credit before lifecycle, Core consumer, and joined fixture gates pass.

## Changed

- `7dec20a32` freezes exact Core and seven-field plugin event evidence in
  additive migration 041 and rejects ambiguous or secret-bearing tuples.
- `3b52ef041` adds the Host-owned exact PostgreSQL Session Policy selection
  Store with implicit Core, exact Select/Reset CAS, atomic audit/event evidence,
  Safe Mode effective-Core resolution, restart rebind, and commit readback.
- `88a674da5` adds real PostgreSQL, race, Safe Mode, restart/rebind, authority,
  rollback, and migration behavior gates.

## Decisions

- Reset updates the singleton to explicit Core and never deletes it, preserving
  monotonic revisions.
- CAS is checked before equality, so stale identical requests conflict.
- `Current` returns durable desired state; `Resolve` returns effective state.
- Safe Mode evaluates Core without silently rewriting desired state. Explicit
  `super_admin` Reset remains available as Host recovery.
- Missing or stale selected providers fail closed outside Safe Mode.

## Next

1. Invalidate a retiring selected provider inside the existing Serializable
   Identity Registry reconciliation transaction before unpublication.
2. Add atomic `ProviderResolution` and `InvokeExact`, including selection-
   revision recheck immediately before the Host session effect.
3. Wire Safe Mode/Core evaluation into session issue and renew; keep revocation
   Host-local.
4. Continue `GetUser`, Host Command/automation, membership fixture, and joined
   P7 normal/race/restart/Safe Mode/ForceDrain/no-implicit-grant gates.

## Worktree

- Preserve all existing unowned route, PageViewModels, bootstrap, Web inspector,
  public-runtime, content-policy, OpenAPI, ADR/taskbook, and P9 policy changes.
- Work directly on `main`; do not create a branch or worktree.
