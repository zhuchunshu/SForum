# 2026-07-13 Trusted Plugin And Theme Platform V3 P0 Completion

## Progress

- Overall V3: **3%**.
- P0: **100%**, closed; P1 is next.
- Active branch: `main`; pre-V3 baseline `d72c9ac2c`.

## Changed

- Added deterministic V3 P0 catalog generation and drift validation.
- Generated a 99-row ADR traceability matrix, 204-route catalog, 113-surface UI
  catalog, current backend/admin inventories, and per-module Extension Surface
  Matrix.
- Added namespace/versioning, feature-gate, Safe Mode, custom-guard, raw core DB,
  conflict, and rollback governance.
- Added current v1 benchmarks for extension enable, theme resolve, loopback route
  proxy, and real subprocess net/rpc.
- Marked V3 active and added supersession notes to conflicting historical ADRs
  and module boundaries.
- Added the durable weighted percentage/context-compaction protocol requested by
  the user.
- Closed all P0 task-book and verification criteria without changing production
  behavior.

## Commits

- `eedfcb2d6 docs(extensions): freeze V3 P0 catalogs`
- `d21d7d90f test(extensions): record V3 P0 performance baseline`
- `3b98cfd88 docs(extensions): mark V3 as active platform direction`

## Verification

- Baseline Go benchmarks pass after making the benchmark peer explicitly close
  connections.
- Median results: enable 26.769 us; theme resolve 328.3 ns; route proxy 694.333
  us; plugin RPC 145.720 us.
- Existing live home SSR: 12 warm samples median 487.885 ms, all HTTP 200.
- Isolated seeded topic SSR: canonical 12-sample median 303.779 ms, 49,547-byte
  complete HTML; compatibility redirect median 511.779 ms. The isolated
  database, 13001/18081 processes, binaries, and generated build directory were
  removed afterward without touching the user's current database or ports.
- `node tests/validate-v3-p0-catalogs.mjs` passed: 204 routes, 113 UI surfaces,
  and 99 traceability rows.
- Focused extension/page/runtime Go tests and all four benchmark smoke cases
  passed.
- `./scripts/test.sh` passed, including all Go tests, 1,477 OpenAPI references,
  Nuxt typecheck, repo validators, and the V3 catalog gate.
- `cd apps/api && go build ./...` passed.
- `cd apps/web && bun run build` passed during P0, with existing warnings only.

## Decisions

- Generated catalogs derive from current Fiber/Nuxt/Go catalog sources and fail
  CI on drift; semantic governance remains reviewed documentation.
- New V3 registry migration gates are Host-owned and default-off. Safe mode
  always overrides them.
- P0 records the current RouteGateway connection-exhaustion risk and leaves the
  production fix to P6.
- Overall percentage uses fixed phase weights and verified exits only.

## Next

1. Audit current upload/install/enable services and OpenAPI/admin consumers for
   the P1 compatibility-first exact-artifact challenge insertion point.
2. Audit bootstrap sync order and implement `SFORUM_SAFE_MODE=1` before any
   third-party process start.
3. Audit `cmd/sforum` composition and add out-of-band list/disable recovery that
   does not boot HTTP or plugin code.
4. Add P1 persistence as an independent additive migration before service/UI
   implementation.

## Open Questions

- None requiring product input.
