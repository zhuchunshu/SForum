# 2026-07-27 GitHub Social Login T1A Handoff

## Status

**T1A complete.** M1R remaining: T1B → T1C → T1D → T1E. Do not start M2 or the
GitHub plugin package.

Prior review baseline:
`sessions/2026-07-27-github-social-login-m1-review-handoff.md`.

## Changed

- `AuthProviderFlow.Complete` is assertion-only: no external-link write on
  `link.complete`. Persistence only via `ExternalAuthService.CompleteLink`
  after authorization gates.
- Public `POST /auth/providers/{id}/{operation}/complete` for
  login/registration/link returns `410 auth.provider_callback_required`.
  Recovery complete still works. Business effects only through reserved
  `GET …/callback`.
- Callback order: consume state → live Registry re-resolve + artifact match
  (provider, operation, owner extension/version, package digest, contract) →
  re-check activation → (link) session actor + recent-auth → complete with
  Host PKCE verifier + absolute callback URL → effect.
- Absolute callback URL from trusted `APP_URL` (`WithPublicAppURL`); production
  requires HTTPS. Never uses request Host.
- `redirectHint` validated (`ValidateSafeRedirectPath`) before store/plugin.
  Registration continuation is fixed `/register?ticket=…&redirect=…`.
- `CompleteLink` re-checks activation, recent-auth, and live artifact before
  `Link` as defense in depth.

## Decisions

- Legacy public auth complete stays as a fail-closed compatibility stub rather
  than being deleted (OpenAPI/Core Route Catalog cleanup is T1E).
- Recent-auth remains user-scoped for T1A; session-bound recent-auth is T1D.
- In-memory callback/ticket concurrency/TTL hardening is T1B.

## Verification

Focused:

```text
cd apps/api && go test ./app/Models/Identity/ -count=1
# ok

cd apps/api && go test ./app/Http/Controllers/Identity/ ./app/Providers/ ./bootstrap/ -count=1
# ok
```

Full:

```text
cd apps/api && go test ./... -count=1
# all packages ok (exit 0)
```

T1A-specific coverage includes:
`external_auth_t1a_safety_test.go`, callback URL/redirect tests,
`TestAuthProviderFlowLinkCompleteIsAssertionOnly`, PKCE pass-through.

## Next

Start a fresh conversation for **T1B only**: stable HMAC config, concurrent/
expiring hashed state and ticket stores. Do not start T1C–T1E, M2, plugin, or
UI work.

## Open Questions

- None for T1B entry. If implementation evidence disproves a frozen rule, stop
  for a user decision.
