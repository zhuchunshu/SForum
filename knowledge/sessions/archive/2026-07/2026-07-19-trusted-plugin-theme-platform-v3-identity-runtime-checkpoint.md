# 2026-07-19 V3 Identity Runtime Checkpoint

## Progress

- Verified weighted progress remains **67.8295%** (display **67.0%**).
- P7 remains **18/22**. The work below is required production infrastructure;
  it does not close an authoritative P7 row before real Core consumers and the
  joined membership-plugin gates pass.

## Changed

- `97fe90d12` negotiates the exact `identity.runtime@1` Protocol V2 feature and
  freezes the runtime Manifest declaration.
- `449c1653b` adds the Host `ProviderCall` transport with allowlisted actor
  context, exact declaration matching, dual Schema references, and response
  validation.
- `be692be7b`, `d02c51f03`, and `098bc21d7` add the external identity link,
  user-field value, and session-policy selection schemas. All three migrations
  passed a real PostgreSQL startup `Up` gate.
- `8f411ed0a` adds the exact Identity provider resolver. It binds one Manager to
  one authoritative Registry, rejects Core/catalog-only/generic/active-runtime
  fallback, holds one exact provider lease through Host acceptance, validates
  bounded input and output against the immutable Registry Schemas, enforces the
  declared timeout at the Manager boundary, and fences Safe Mode, replacement,
  quarantine, ForceDrain, artifact, version id, Manifest digest, Registry
  revision, and Registry digest drift.
- The commit fence is deliberately documented as process-local final admission
  validation, not a cross-system CAS. Production stores must perform durable
  exact-artifact checks inside the same rollback-capable Host transaction as the
  effect and audit. Accept success is terminal; a post-commit Registry read must
  not convert a committed effect into a retryable error.

## Verification

- Focused Identity provider normal and race x3 gates pass.
- Complete `app/Support/Extensions` tests and `go vet` pass.
- Non-cooperative invokers receive the Host deadline while retaining their exact
  lease until they actually return.
- Go build/test cache and `/private/tmp/sforum*` were cleaned after the disk had
  accumulated about 183 GB of regenerable data. Continue monitoring these roots
  after heavy race and integration gates.

## Next

1. Add a narrow external-link store with exact durable provider-tip locking,
   idempotency fingerprint replay, revision CAS, redacted events, audit, and
   real PostgreSQL races.
2. Add a narrow user-field value store with live Registry Schema validation,
   exact durable field-tip locking, absent-row serialization, revision CAS, and
   digest-only events.
3. Add a narrow session-policy selection store. Reset and invalidate must CAS
   update the singleton to explicit Core provenance rather than delete it, so
   the revision remains monotonic.
4. Wire registration, login, recovery, profile, account management, external
   linking, session, and risk Core consumers through the resolver and stores.
5. Add trusted `extensions.read/call/manage` automation, then prove all four P7
   rows with a real Protocol V2 membership plugin and joined normal/race/
   restart/Safe Mode/ForceDrain/no-implicit-grant gates.

## Worktree Ownership

- Preserve all pre-existing dirty files. In particular, do not stage the dirty
  `bootstrap/app.go`, `go.mod`, PageViewModel, route/web inspector, public
  frontend policy, ADR, or V3 task-book changes with Identity commits.
