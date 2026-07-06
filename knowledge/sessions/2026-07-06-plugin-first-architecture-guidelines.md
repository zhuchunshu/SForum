# 2026-07-06 Plugin-First Architecture Guidelines

## Changed

- Updated architecture docs to state that SForum core is the host framework,
  not a home for every optional product vertical.
- Updated development guidelines so payments, outbound mail delivery,
  notification channels, analytics, external integrations, and vendor-specific
  provider behavior default to plugins.
- Clarified that core may add events, filters, provider slots, policy checks,
  admin selection/reset flows, SDK helpers, scaffolding, no-op defaults,
  development adapters, and protected built-in plugins when those make plugin
  development practical.
- Updated roadmap, product notes, deployment docs, backend module notes,
  extension module notes, and the knowledge index to match the plugin-first
  boundary.

## Decisions

- Payment and mail systems must not be implemented as direct core verticals.
- Real provider/vendor behavior belongs in extension packages by default.
- Core-owned events, provider slots, plugin routes, and extension admin pages
  are the supported extension points; arbitrary route override and
  monkey-patching remain disallowed.

## Next

- Before building mail, notification, payment, analytics, or integration work,
  define the host contract first and add plugin author documentation/examples.
- Consider which low-risk provider slot should become the first plugin-first
  vertical example.

## Open Questions

- Should SForum ship a protected built-in transactional mail plugin, or only a
  development/no-op mail adapter until a specific provider is selected?
- If payments enter scope, does core need only entitlement checks, or also a
  generic payment intent/order contract for plugins?
