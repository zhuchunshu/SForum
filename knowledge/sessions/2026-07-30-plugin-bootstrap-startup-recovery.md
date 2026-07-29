# 2026-07-30 Plugin Bootstrap Startup Recovery

## Changed

- Restored the shared HashiCorp go-plugin Bootstrap ABI v1 cookie for the Host
  and SDK while retaining strict Manifest V3, application Protocol V2, and Host
  API V2 enforcement.
- Added a cross-built fixture that hard-codes the historical bootstrap contract
  so Host and fixture constants cannot drift together undetected.
- Classified bounded local subprocess startup diagnostics without exposing raw
  stderr in public readiness data.
- Initial API plugin convergence failure now enters recovery-only mode: Core
  liveness/readiness remain available, product routes return `503`, plugin
  runtime and route publications reconcile to Core-only process state, and the
  database's active/staged artifact authority is unchanged. Recovery readiness
  does not enumerate extension health contributions, and the extension guard
  catalog remains empty without initial or periodic artifact refresh.
- Added exact version/digest quarantine for protected built-in/system artifacts
  with desired-set locking, immutable recovery publication, and audit evidence.
- Added a CI release gate that requires a real SemVer increase when built-in
  backend source or package contracts change and rejects version regression.

## Decisions

- Bootstrap ABI and SForum application protocol are independent version axes.
  A future bootstrap change requires its own migration contract and cross-build
  evidence; changing Protocol V2 alone must not change the bootstrap cookie.
- The API exposes Host-owned recovery endpoints after initial convergence
  failure. A standalone worker has no recovery HTTP surface and remains
  fail-closed for its supervisor to restart or alert.
- Recovery does not auto-promote staged built-ins or reinterpret active state.

## Verification

- Automated tests, builds, service startup, and browser checks were not run at
  the user's request. The user owns runtime verification for this change.

## Next

- Run `./scripts/api-dev.sh`; the existing active V3/P2 artifact should start
  with Bootstrap ABI v1.
- For a deliberately incompatible artifact, confirm `/api/v1/health` returns
  `200` with `recovery_required`, `/api/v1/ready` returns `503` with desired
  revision/artifacts, and all other routes return `503`.
- If the failing exact artifact is protected, run
  `go run ./cmd/sforum extension quarantine <id> --expect-version <version> --expect-digest <digest>`
  from `apps/api` with the exact values reported by readiness, then restart the
  API.

## Open Questions

- None. Manual runtime verification remains outstanding.
