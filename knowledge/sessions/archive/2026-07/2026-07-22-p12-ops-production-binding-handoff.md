# 2026-07-22 Session Handoff — P12 Ops Production Binding

## Changed

Reopened P12 ops packages that had been closed from Support-only / process-local
models and bound them to real production authority:

### 1. Compatibility Farm (gate green)

- Real executor: `apps/api/app/Support/CompatFarm/run.go`
- CLI: `tests/compat/run_matrix.go` (+ `tests/compat/go.mod`) and
  `apps/api/cmd/compat-farm`
- Wired into `./scripts/test.sh` (missing/skip/fail exit 1)
- Cells: current Host+Protocol V2+Manifest V3, Protocol V1 shim telemetry
  (APILTS), Manifest/SDK contract on builtin plugin

### 2. Marketplace (gate green)

- **Ed25519** signing (ADR: `decisions/2026-07-22-marketplace-ed25519-signing.md`)
- Full deep-copy after verify; List/Resolve isolation; nested-slice mutation tests
- Validates extension id, SemVer, SHA-256 / SBOM digests, channel, time windows
- Recursive dependency resolution (cycle/conflict/withdrawn/revocation/host range)
- Installer binding: Preflight/Stage/Activate/Rollback (not string lists)

### 3. RuntimeRollout

- `PlanStore` + `PostgresStore` (migration `202607220046`)
- Create/Canary/Health/Drain/Promote/Rollback durable; restart recovery
- Multi-API concurrent Create → one winner (advisory lock + unique active plan)
- Tests: missed notification reload, node lost, migration fail, health fail,
  rollback, race, Postgres integration

### 4. SystemTier

- Postgres `system_tier_members`; CLI `sforum extension system-tier`
  list/upsert/disable (no package code)
- Safe Mode returns nil LoadOrder before any system extension load
- Cross-process disable proven with shared store / Postgres integration

### 5. Privacy / Observability

- Privacy: permission gate, auditor, deadline, partial failure, external residual,
  `PublishContribution` for lifecycle/Protocol inventory (not only Go callbacks)
- Observability: `Process()` + ObserveRoute/Hook/SQL/Cache/RPC/Job; wired into
  real `ProtocolStarter.InvokeHook` and `ExecutePluginJob`

### 6. Bootstrap

- `bindProductionP12Ops` in API assembly (Postgres rollout/tier + marketplace + privacy)

## Evidence

```text
# Unit + race (no DB)
cd apps/api && go test ./app/Support/{CompatFarm,Marketplace,RuntimeRollout,SystemTier,Privacy,ExtensionObservability}/ -race -count=1

# Compat farm CI entry
cd tests/compat && go run .

# Postgres (compose maps 15432)
export DATABASE_URL=postgres://sforum:sforum@127.0.0.1:15432/sforum?sslmode=disable
cd apps/api && go test ./app/Support/RuntimeRollout/ ./app/Support/SystemTier/ -run TestPostgres -count=1 -v
```

## Decisions

- Marketplace production signing is Ed25519 (not HMAC).
- RuntimeRollout plan IDs are random (`rollout-` + 16 random bytes), not process seq.
- P12 ops share migration `202607220046` (rollout plans + system tier + privacy audit table).

## Status honesty

P12 rows that were Support-only were reopened during this work. They may be
re-closed only with the evidence above (production binding + restart/multi-node
where applicable + CI executor). Remaining product wiring (admin UI for
marketplace install, full Protocol V2 privacy RPCs, dual-API live e2e under
load) can continue without reopening the Support contracts.

## Next

- Apply migration `202607220046` in deployed environments before relying on
  Postgres rollout/tier tables.
- Optional: admin HTTP surfaces for marketplace resolve/stage and rollout plans.
- Optional: Protocol V2 Privacy service protobuf (inventory already publication-ready).

## Open Questions

- Whether marketplace public keys live in options table vs env-only config.
- Whether system-tier LoadOrder should run before or after builtin sync under
  non–Safe Mode (currently available via registry; boot order TBD product).
