# 2026-07-12 PAT effective permissions are current ∩ scopes

## Context

Bearer PAT middleware stored scopes on the request, but many controllers only
read the cookie session. Separately, `RestrictActor` trusted stored scopes as
the full permission set without re-checking whether the user still holds those
permissions after token creation.

## Decision

1. All HTTP controllers that need an authenticated actor use the shared helpers
   `apphttp.ResolveUserID` / `LoadActor` / `OptionalActor` (Identity keeps an
   equivalent path that also calls `RestrictActor`).
2. Effective PAT permissions are always:

   **current user permissions ∩ token scopes**

   evaluated on every request after loading the live actor.
3. PAT results always strip `super_admin` role bypass so a token cannot act as an
   unbounded cookie session even when the user is super_admin; only listed
   scopes that the user currently holds remain.
4. Permission revoked from the user after token creation takes effect on the
   next request without rotating the token.

## Consequences

- Machines using PAT can call forum/admin routes that previously returned 401.
- Missing scope returns 403 even if the cookie session user would have the
  permission.
- CSRF cookie exemption for `Bearer sft_...` remains; invalid tokens still fail
  authentication before mutations run.
