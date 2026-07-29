# Runtime Site URL Owns OAuth Callback Base

Date: 2026-07-29

## Context

Operators can configure `site.url` in the admin area, but external-auth callback
URLs previously continued to use startup `APP_URL`. This made the displayed and
executed OAuth callback ignore the site's effective public address.

## Decision

- Host resolves the OAuth callback base from the effective runtime `site.url`.
- A stored non-empty `site.url` wins; an empty setting inherits environment
  `APP_URL` through the Options service.
- Request `Host` is never trusted as a callback authority.
- Production callback URLs must still use HTTPS.
- OAuth start stores the exact resolved callback URL in the one-use transaction;
  completion reuses that value, so later setting changes do not mutate an
  in-flight attempt.
- CSRF origins and cookie security remain environment/startup concerns; this
  decision changes only OAuth callback URL authority.

This supersedes only the APP_URL-only callback-source clauses in the 2026-07-27
APP URL handoff and GitHub social-login M0 freeze. Host ownership of state,
PKCE, callback transactions, and the reserved callback route is unchanged.

## Permission And Risk

The actor is an operator with `settings.site.manage`; the action is updating
`site.url`; the protected resource is the callback origin used for future
external-auth attempts. The existing server-side option permission and URL
validation remain authoritative. Option read failures fail closed rather than
silently accepting a request-derived address.

## Consequences

- Saving a new site address affects newly started OAuth flows without an API
  restart after the Options cache is invalidated.
- Clearing the setting restores the deployment `APP_URL` fallback.
- Provider consoles must register the exact callback URL displayed by Host.
