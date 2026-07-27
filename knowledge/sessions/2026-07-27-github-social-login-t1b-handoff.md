# 2026-07-27 GitHub Social Login T1B Handoff

## Status

**T1B complete.** M1R remaining: T1C → T1D → T1E. Do not start M2 or the
GitHub plugin package / UI work.

Prior: `sessions/2026-07-27-github-social-login-t1a-handoff.md`.

## Changed

- `config.Config.IdentitySubjectHMACSecret` loads `IDENTITY_SUBJECT_HMAC_SECRET`
  with stable dev default `IdentitySubjectHMACDevDefault`.
- Production validation uses real `APP_ENV=production` path
  (`validateProductionSecrets`): missing, weak (<32 bytes), placeholder, or
  dev-default secrets panic at startup.
- Bootstrap `wireAPICoreStack` calls
  `identity.ConfigureIdentitySubjectHMAC(cfg.IdentitySubjectHMACSecret)` before
  domain wiring. Digest service never uses process-random material.
- In-memory callback/ticket stores: `sync.Mutex`, TTL fill/clamp matching Redis,
  storage keys = `sha256(opaque browser token)`, concurrent single-consume.
- Redis callback/ticket stores: same hash keys, atomic Lua consume, used-
  tombstone for callback replay (`ErrCallbackStateReplayed`); registration
  ticket replay maps to public `invalid` reason.
- Registration tickets: Save/Consume enforce CreatedAt/ExpiresAt and
  operation=`registration` + provider id + owner extension + package digest +
  subject material.

## Decisions

- Stable dev HMAC default is a fixed ≥32-byte string shared by config and
  identity packages (literal match asserted in tests). Rotation/dual-read is
  deferred.
- Callback replay is distinguishable (replayed vs invalid/expired). Ticket
  replay stays `auth.external_registration_ticket_invalid` (no new public
  reason).
- Raw browser tokens never appear as Redis/memory map keys; errors do not log
  raw state/ticket material.

## Verification

Focused:

```text
cd apps/api && go test ./app/Models/Identity/ ./app/Http/Controllers/Identity/ ./app/Providers/ ./bootstrap/ ./config/ -count=1
# all ok
```

Full:

```text
cd apps/api && go test ./... -count=1
# all packages ok (exit 0)
```

T1B-specific coverage includes: production HMAC reject tests in `config`,
stable-default/reconfigure digest tests, concurrent consume, hash-key storage,
TTL clamp, registration ticket binding enforcement.

## Next

Start a fresh conversation for **T1C only**: atomic audited activation,
effective catalog, Safe Mode filtering, truthful probe. Do not start T1D–T1E,
M2, plugin, or UI work.

## Open Questions

- None for T1C entry. If implementation evidence disproves a frozen rule, stop
  for a user decision.
