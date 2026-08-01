# OAuth Provider Core Host Contracts

## Context

SForum OAuth Provider is an optional independently maintained plugin. Core
must provide stable Host capabilities without embedding OAuth client/token
models or allowing a plugin to reach Core identity/session storage.

## Decision

- Personal settings contributions use a dedicated `account-settings`
  Navigation Registry kind and a private Host endpoint. Contributions are
  authenticated by default, filtered by declared permission or a Host-owned
  retained-resource key, and returned as a redacted DTO with only safe local
  paths.
- Identity delegation is a separate focused package. It issues fresh random
  plugin-scoped subjects bound to the exact artifact and projects only
  username, display name, locale, truthful verified email, and `auth_time`.
  Core user IDs, roles, permissions, credentials, and session material remain
  in the Host store.
- Consent is a separate Host transaction bridge. Transactions bind actor,
  session fingerprint, exact artifact, client descriptor, redirect URI,
  scopes, CSRF, TTL, and optional recent-auth. Wrong-context decisions do not
  consume the transaction; a valid decision is one-use.

## Consequences

The independent plugin can implement OAuth/OIDC protocol behavior against
these contracts while Core remains provider-neutral. Persistent production
adapters and Protocol V2 exposure can evolve behind these narrow interfaces;
they must preserve exact-artifact and fail-closed lifecycle behavior.
