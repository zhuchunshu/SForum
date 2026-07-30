# Decision: Offer a Host Choice for Unlinked External Login

## Status

Accepted

## Context

An external provider can authenticate a subject that has no local identity
link. Some users already have a local account and should bind it; others need
to register. Silently choosing either effect, matching by provider email, or
asking the provider to authenticate twice would weaken the Host-owned account
and registration boundaries.

The product already has one local login flow, one external registration flow,
and one `ExternalIdentityLinkStore`. A new choice page must orchestrate those
authorities instead of introducing a second registration form or a second link
write implementation.

## Decision

- An unlinked `login.complete` issues one opaque, one-use, browser-bound
  continuation ticket and redirects to `/auth/continue?ticket=...`.
- The choice page rechecks the two effects independently. Existing-account
  binding requires effective provider login and link operations. Registration
  requires effective provider login and registration operations plus open site
  registration policy. Closing registration must not disable binding.
- "Log in and bind" returns through the existing local login page. The source
  provider is excluded from that page for this continuation so the user cannot
  restart the same unlinked OAuth loop. After local login, the Host consumes
  the ticket and writes through the existing `ExternalIdentityLinkStore`.
- The reused provider assertion remains truthfully identified as
  `login.complete`; the Host does not claim that a second `link.complete`
  provider call occurred. Current active actor, session-bound recent auth,
  exact live artifact, subject uniqueness, and link activation are rechecked
  before and inside the link commit fence.
- "Register and bind" enters the existing `/register?ticket=...` flow. Final
  submission remains `POST /auth/external-registration`, with its existing
  validation and atomic user/default-role/link/audit transaction.
- `POST /auth/external-registration/prepare` inspects but does not consume the
  ticket. It returns only editable username, display-name, and explicitly
  verified email hints; it never returns subject, digest, artifact identity,
  state, verifier, or OAuth material.
- The callback state and login continuation carry only a digest of a Host-set
  HttpOnly, SameSite=Lax browser secret. Copying a ticket to another browser
  cannot bind the attacker's provider identity to a victim's signed-in local
  account.
- Provider email is never used to find, merge, or bind an existing local
  account. Existing registration conflicts remain field errors; account
  ownership is established only by local login.

## Consequences

- Users explicitly choose whether to prove ownership of an existing account or
  create a new one, while both paths reuse existing Host workflows.
- Disabling registration removes only the registration choice. Disabling link
  removes only the existing-account choice. Artifact drift, disabled login,
  unavailable runtime authority, expiry, replay, or browser mismatch invalidate
  the continuation before account effects.
- Providers that cannot assert verified email may still suggest a username or
  display name, but the user must enter email through the normal form.
- `/auth/continue` is a Page Registry surface with a Host-owned continuation
  island. Themes may own presentation but cannot replace ticket inspection,
  login/session authority, registration, or link persistence.
