# Decision: Continue Unlinked External Login Through Host Registration

## Status

Accepted

## Context

An external provider can authenticate a subject that has no local identity
link. Rejecting that login leaves a legitimate new user without a clear next
step, while silently creating, matching, or binding an account would weaken the
Host-owned registration and account-takeover boundaries.

The product already has one Host registration page and one external
registration submission endpoint. Maintaining a second provider-specific form
or registration path would duplicate validation and drift from the local
registration policy.

## Decision

- An unlinked `login.complete` may issue the existing opaque external
  registration ticket and redirect to `/register?ticket=...`.
- The ticket records `registration` as its target effect and preserves
  `login` or `registration` as the source provider assertion operation.
- Login-to-registration continuation is allowed only while site registration,
  the source operation, provider registration, the exact extension artifact,
  both complete operations, and normal non-Safe-Mode runtime authority all
  remain valid. Zero-user bootstrap remains password-only.
- `POST /auth/external-registration/prepare` inspects but does not consume the
  ticket. It returns only editable username, display-name, and explicitly
  verified email hints; it never returns subject, digest, artifact identity,
  state, verifier, or OAuth material.
- Registration submission continues through
  `POST /auth/external-registration`. Ticket consumption, local field checks,
  registration policy, user/default-role creation, external link creation, and
  audit keep the existing atomic boundary.
- Provider email is never used to find, merge, or bind an existing local
  account. Existing conflicts remain field errors and require the user to log
  in and use the explicit link flow.

## Consequences

- The user gets one coherent registration experience with provider-assisted
  prefill and no second form or endpoint for account creation.
- Disabling registration, either provider operation, the extension, trust, or
  the exact artifact invalidates outstanding continuation before any account
  effect.
- Providers that cannot assert verified email may still suggest a username or
  display name, but the user must enter email through the normal form.
- The ticket contract and provider complete output now carry source operation,
  username hint, and email verification metadata; provider schemas and exact
  package digests must be updated together.
