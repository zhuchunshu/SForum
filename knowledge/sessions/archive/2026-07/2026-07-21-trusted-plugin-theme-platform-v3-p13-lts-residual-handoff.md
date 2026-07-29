# 2026-07-21 Session Handoff — V3 P13 LTS residual only (archived)

## Status

- P0–P12 complete. P13 implementable work closed at **~99.6%**.
- Three task-book rows remain open by **published LTS policy**, not by missing
  implementation capacity.

## Open rows (must stay open)

1. Remove request-time template loader / legacy Page Outlet behavior after
   parity — **Fail-closed `SFPageOutlet` never fully removed**; loader residual
   retained until LTS checklist.
2. Remove Protocol V1 paths — seeded `RemoveAfter` ≈ **2026-11-28** (DeprecatedAt
   2026-06-01 + 180d) **and** `CanRemoveWithZeroShim` on live API/worker process.
3. Compatibility path removal — same checklist in
   `docs/extensions/v3/p13-migration-and-lts.md` items 1–7.

## Closed this session (beyond presentation)

- Route inspector `invocationStage` UI/OpenAPI/i18n
- Public L2 descriptor cross-owner graph hardening
- Host-owned Link header docs + ADR note
- Page view-model disabled-search fail-closed test
- WebSocket safe-mode/trust-revoke integration tests
- content-policy digest refresh, GOWORK plugin build isolation

## Operator inspect

```bash
cd apps/api && go run ./cmd/sforum extension api-lts
cd apps/api && go run ./cmd/sforum extension api-lts --json
```

## Do not

- Delete LoadTemplate / Protocol V1 / SFPageOutlet early
- Claim V3 100% complete while the three LTS rows remain open
- Stage unrelated future WIP without verification

## Exact next

Wait for LTS window + zero shim telemetry on production processes, then execute
deletions under the published checklist with fine-grained commits and full gates.
