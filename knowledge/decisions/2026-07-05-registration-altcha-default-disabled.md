# Decision: Registration ALTCHA Default Off

## Status

Accepted

## Context

Registration already has an ALTCHA-backed human-verification path, but early
local setup and first-run registration should stay low-friction unless an
operator explicitly enables anti-automation verification.

## Decision

Set `HUMAN_VERIFICATION_PROVIDER=disabled` as the default runtime value.
Deployments that need registration human verification should set
`HUMAN_VERIFICATION_PROVIDER=altcha`.

The Nuxt registration page reads a public, non-secret runtime provider and only
renders/submits ALTCHA when that provider is `altcha`. Backend verification
remains authoritative.

## Consequences

- Fresh local and production example environments do not show ALTCHA on the
  registration page.
- ALTCHA challenge generation, server verification, replay protection, and rate
  limiting remain available when explicitly enabled.
- Future provider settings must keep secrets in environment configuration; only
  the selected public provider name is exposed to Nuxt.
