# Browser Session And JWT Strategy

## Status

Accepted on 2026-07-05.

## Context

SForum is currently a first-party browser forum. The main authentication needs
are reliable logout, revocation, role/permission freshness, low-friction user
experience, and useful login audit history. A JWT-first browser design would
require access-token storage, refresh-token rotation, revocation lists, and CSRF
or XSS tradeoffs before the project has third-party API or mobile-client needs.

## Decision

Use Redis-backed server sessions for first-party browser authentication.

Default runtime policy:

- Session cookie is HTTP-only, SameSite=Lax, project-scoped as
  `sforum_session`, and Secure in production.
- Idle timeout is 30 days.
- Absolute timeout is 180 days.
- Session id renews every 24 hours during authenticated use.
- Login and registration reset the existing session before storing `user_id`.
- Successful registration auto-login and every successful login write an
  `audit_events` record with user id, IP address, User-Agent, action, and a
  salted session-id hash.

Do not introduce access/refresh JWT for the browser forum milestone. If SForum
later exposes mobile apps or third-party API access, add a separate OAuth-style
token model with short-lived access tokens, persisted rotating refresh tokens,
reuse detection, per-device revocation, and audit records.

## Rationale

- Redis-backed sessions support immediate logout and server-side revocation
  without token denylist complexity.
- Forum permissions and account status should take effect immediately after
  admin changes.
- Long but bounded sessions reduce repeated logins for normal forum use while
  preserving an absolute lifetime.
- Periodic session-id renewal limits the impact of long-lived id exposure.
- Salted session hashes make audit correlation possible without logging raw
  session secrets.

## Consequences

- The API remains cookie-authenticated for browser flows, so CSRF protection is
  still required before unsafe cookie-authenticated writes are considered
  production-ready.
- Production deployments must provide a strong `SESSION_HASH_SECRET` and use
  HTTPS so the Secure cookie is delivered.
- Future token-based clients must be designed as a separate auth surface, not by
  reusing browser session cookies.
