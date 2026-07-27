# 2026-07-27 GitHub Social Login T1C Handoff

## Status

**T1C complete.** M1R remaining: T1D → T1E. Do not start M2 or the
GitHub plugin package / UI work.

Prior: `sessions/2026-07-27-github-social-login-t1b-handoff.md`.

## Changed

- Atomic optimistic activation CAS:
  `PostgresProviderActivationStore.Upsert` uses `SELECT … FOR UPDATE` then
  `UPDATE … WHERE provider_id AND revision = expected` with RowsAffected check;
  `MemoryProviderActivationStore` mirrors the same semantics under mutex.
- `ErrProviderActivationNoMutation` when next state equals current (no revision
  bump); reset is no-op when already defaults.
- Host-derived ownership: `PrepareActivationInput` reads live Registry only;
  browser `ownerExtensionId` / `ownerPackageDigest` →
  `auth.provider_ownership_rejected`; unsupported enable ops →
  `auth.provider_operation_unsupported`.
- Effective availability (`EvaluateOperationAvailability` /
  `IsEffectivelyAvailable` / `RequireActivated`): activation flag + live
  artifact digest/owner match + RuntimeInstanceID + supported operations +
  Safe Mode. Artifact drift / disable / uninstall / Safe Mode remove public
  availability until deliberate re-activation (rebinds live digest).
- Public `GET /auth/providers` uses `ListEffectivePublicCatalog`: only
  effectively available auth providers (plus live recovery); activation List
  failure returns 503 fail closed (no partial catalog).
- Admin probe: always `ok=false` with `probe_pending` or `probe_unavailable`;
  store forces pending/unavailable reasons to `last_probe_ok=false`. Never
  present pending as successful health.
- Actor-bound audit actions:
  `identity.provider.activation.update` /
  `identity.provider.activation.reset` /
  `identity.provider.probe`.
- Localization for new API reasons.

## Decisions

- Configuration checker (`IsProviderConfigured`) is optional until settings
  wiring (M2/M3); nil does not invent configured=true and does not block when
  unset. Probe remains the admin health signal and is not required for
  availability until a real probe RPC exists.
- Admin may still write activation rows while Safe Mode is on; effective
  public availability stays false until Safe Mode clears and live artifact
  still matches.
- No-mutation responses return current activation with HTTP 200 and skip
  audit (no durable change).

## Verification

Focused:

```text
cd apps/api && go test ./app/Models/Identity/ ./app/Http/Controllers/Identity/ ./app/Support/Localization/ -count=1
# all ok
```

Full:

```text
cd apps/api && go test ./... -count=1
# all packages ok (exit 0)
```

T1C-specific coverage: CAS allowed/stale/no-mutation, concurrent CAS,
probe never ok=true, artifact/Safe Mode/unsupported ops availability,
catalog fail-closed, ownership reject, activation audit.

## Next

Start a fresh conversation for **T1D only**: session-bound recent-auth,
unlink/password credentials, authoritative registration and canonical
session user. Do not start T1E, M2, plugin, or UI work.

## Open Questions

- None for T1D entry. If implementation evidence disproves a frozen rule, stop
  for a user decision.
