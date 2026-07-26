# 2026-07-27 GitHub Social Login Plan Handoff

## Changed

- Replaced the broad four-provider social-login task book with the focused
  ready plan `plans/2026-07-27-github-social-login-builtin-plugin.md`.
- Added the accepted decision to ship GitHub V1 under
  `extensions/builtin/plugins/sforum-auth-github`.
- Archived the superseded 2026-07-22 plan and handoff.

## Decisions

- V1 ships GitHub only. Other OAuth/OIDC and regional providers are deferred.
- Built-in means shipped, built, boot-discovered, and staged; it does not mean
  automatically trusted, enabled, configured, or publicly activated.
- Core owns callback state, subject HMAC, registration tickets, users, links,
  risk/session policy, Redis sessions, audit, and last-method protection.
- Plugin output carries the raw GitHub subject only through the internal typed
  response. Core stores a dedicated-secret HMAC and exposes neither form.
- The delivery order is contract freeze, Host closure, headless real GitHub
  vertical slice, admin UI, public/account UI, then lifecycle hardening.

## Next

- Implement M0 only: inspect current contracts, verify current official GitHub
  OAuth App behavior, finish the library survey, and freeze the additive
  schemas/config without changing production behavior.

## Open Questions

- M0 must verify GitHub's current PKCE, scope, callback, token-format, and email
  behavior against official documentation.
- M0 must confirm whether any non-fixture durable external-link rows exist
  before changing the subject assertion contract.

