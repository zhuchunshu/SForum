# 2026-07-06 Plugin-First Architecture Guidelines

## Changed

- Updated architecture docs to state that SForum core is the host framework,
  not a home for every optional product vertical.
- Updated development guidelines so payments, outbound mail delivery,
  notification channels, analytics, external integrations, and vendor-specific
  provider behavior default to plugins.
- Clarified that core should expose events, filters, provider slots, policy
  checks, typed payloads, admin selection/reset flows, SDK helpers, scaffolding,
  no-op defaults, development adapters, and protected built-in plugins when
  those make plugin development practical.
- Clarified the payment example: core should own provider-neutral payment
  architecture when payments enter scope, including intents, canonical
  transactions/refunds, webhook idempotency, entitlement checks, events,
  provider interfaces, and admin provider selection. Plugins extend providers,
  provider-specific transaction behavior, checkout/session flows, invoice
  rendering, webhook parsing, and vendor settings.
- Updated roadmap, product notes, deployment docs, backend module notes,
  extension module notes, and the knowledge index to match the plugin-first
  boundary.

## Decisions

- Payment provider/vendor behavior and mail delivery must not be implemented as
  hard-coded core verticals.
- Core should expose stable interfaces and canonical state where plugins need a
  shared architecture, especially for payments.
- Real provider/vendor behavior belongs in extension packages by default.
- Core-owned events, provider slots, plugin routes, and extension admin pages
  are the supported extension points; arbitrary route override and
  monkey-patching remain disallowed.

## Next

- Before building mail, notification, payment, analytics, or integration work,
  define the host contract first and add plugin author documentation/examples.
- For payment work, design the core payment framework before writing any
  provider plugin.
- Consider which low-risk provider slot should become the first plugin-first
  vertical example.

## Open Questions

- Should SForum ship a protected built-in transactional mail plugin, or only a
  development/no-op mail adapter until a specific provider is selected?
- What is the minimum payment framework v1: intent plus transaction only, or
  intent, transaction, refund, entitlement, and webhook-delivery records?
