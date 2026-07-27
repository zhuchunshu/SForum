# 2026-07-28 Architecture Debt M7 Handoff

## Changed

- Closed M6 with a green full repository gate and desktop/mobile browser QA
  for attachments and mail settings.
- Moved frontend product components, composables, utilities, and tests from
  crowded roots into explicit domain directories.
- Reduced frontend roots to exact approved allowlists: 28 components, 4
  composables, 7 utilities, and no root tests.
- Updated stable UI identity source paths and regenerated the V3 P0 catalogs.
- Added an architecture check requiring explicit runtime imports for nested
  Vue components and fixed the unresolved usages it exposed.

## Decisions

- Root directory constraints are exact allowlists rather than count caps.
- Domain moves preserve stable identity IDs and contracts; only their source
  paths change.
- Do not repeatedly run the full Bun discovery suite. The six Bun tests wired
  into `scripts/test.sh` are the M7 checkpoint unless a later milestone touches
  another frontend area.

## Evidence

- Architecture boundary validation passed: 1374 production files scanned.
- V3 P0 catalog validation passed: 280 routes, 228 UI surfaces, 99
  traceability rows.
- Nuxt typecheck and production build passed.
- Repo-gate Bun tests passed: 25 tests.
- Relevant admin, identity, framework, theme, error, notification, legal, SEO,
  moderation, and staged-extension validators passed.

One diagnostic full Bun discovery run also found stale expectations outside
`scripts/test.sh`: three topic-path tests expect no default page, two
moderation CSS assertions expect older values, and one live plugin proxy check
returned 502. These were not caused by M7 path relocation and should only be
revisited when their owning behavior is in scope.

## Next

- Complete M8 as same-package backend splits before introducing new Go package
  boundaries.
- Split API assembly, Identity Controller/PostgresStore, Forum PostgresStore,
  and Options normalization by responsibility.
- Replace Forum constructor permutations with one configuration constructor
  while preserving compatibility and close/rollback order.
- Use focused Go tests plus architecture validation for the M8 checkpoint.

## Open Questions

- None for M8. Pause for an ADR only if the same-package split exposes a real
  contract or transaction-ownership ambiguity.
