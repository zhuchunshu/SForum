# Client Version Policy Rides The Public Web Options

Date: 2026-10-02

## Context

A released native app cannot be un-shipped. Before the first App build ships,
the server needs one thing the browser never needed: a way for an operator to
tell old clients "this version is no longer supported" without waiting for an
app-store review cycle.

Three candidate shapes were on the table:

1. A dedicated `GET /api/v1/meta/client` endpoint returning an
   `updateRequired` verdict computed server-side.
2. Extending `/api/v1/health` with client-policy fields.
3. Registering the policy as public runtime options, so it arrives through the
   existing `GET /web-options` projection.

Facts that decided it:

- `GET /web-options` already returns the full public option projection
  (185 options in the dev database) and is already fetched at client cold start.
  Site identity, locales, password policy, content limits, and maintenance state
  (`site.maintenance.enabled` / `site.maintenance.message`) all live there.
- A new endpoint would be a second projection of operator configuration with a
  narrower, hand-maintained membership rule — every future option would need a
  decision about whether it also appears in `meta/client`. That is drift by
  construction, and it contradicts the reuse-first rule for integration
  surfaces.
- Computing `updateRequired` server-side would require tracking client version
  per request (`?platform=&version=`), i.e. a client-version reporting surface
  that does not exist and is not wanted. Comparing two version strings is
  client work.

## Decision

- Core owns three public options, all under `client.*`:
  `client.minimum_version`, `client.recommended_version`, `client.update_notice`.
  They are managed with `settings.site.manage` and delivered through the
  existing public options projection.
- Recommended defaults are all empty: empty means "no restriction". A default
  that blocks already-shipped clients would be a footgun, and the beginner-safe
  default is "nothing changes until an operator asks for it".
- Validation is deliberately loose but bounded: 1-4 numeric segments with an
  optional `-prerelease`/`+build` suffix, at most 32 characters; the notice is
  capped at 500 runes like the maintenance message. Malformed stored values are
  coerced back to empty on read rather than blocking startup.
- The server does not compute `updateRequired` and does not record client
  versions. Clients compare their own version against the policy they read.
- No `Deprecation` / `Sunset` response-header mechanism was introduced. There is
  no deprecated core route today, and inventing a header surface with zero
  declared users is speculative infrastructure. When a core route is actually
  deprecated, the header contract should be designed together with its first
  real consumer.
- The admin surface is a fixed settings tab (站点设置 → 客户端 /
  Site settings → Client apps) with the same field ownership rules as every
  other Core-owned tab: the tab component owns form state, validation feedback,
  save, and one-click restoration to the recommended defaults.

## Consequences

- The App update gate is operator-configurable with zero new routes, zero new
  permissions, and no browser contract change (the browser ignores `client.*`).
- The policy is readable unauthenticated. That is intentional: a forced-update
  screen must be reachable before login. Version strings and upgrade copy are
  not secrets.
- Cross-field consistency (`recommended_version >= minimum_version`) is not
  enforced; the operator sees the format hint and clients tolerate any ordering.
- If a future App needs per-platform minimums, the natural extension is
  additional `client.*` keys read by the same projection — not a new endpoint.
