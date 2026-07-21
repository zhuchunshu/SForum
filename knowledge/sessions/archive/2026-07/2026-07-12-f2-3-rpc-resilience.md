# 2026-07-12 Session Handoff — F2.3 Plugin RPC Resilience

## Changed

- `app/Support/Extensions/resilience.go`: per-extension gate (semaphore +
  consecutive-failure circuit breaker)
- Defaults: max concurrent 4, failure threshold 5, open 30s, mail timeout 15s
- Manager `invoke` + `SendMail` enter/release the gate; observe/fail_open skips
  on open circuit; fail_closed fails with `extension.circuit_open`
- ProtocolStarter: net/rpc calls wrapped with ctx select for real deadlines
- Runtime status: new `degraded` state + circuit/failure observability fields
- Admin plugins list: degraded badge, circuit open badge, failure summary
- OpenAPI `ExtensionRuntimeStatus` enum + fields
- Tests: gate unit + manager circuit/degraded/recovery

## Verification

- `go test ./app/Support/Extensions/` green
- Models/Extensions, Controllers/Extensions, bootstrap green
- OpenAPI refs OK

## Next

- F2.4 lifecycle (upgrade / uninstall / migrations / disable drain)
- Optional: runtime options for resilience knobs; Host API gateway rate limits
