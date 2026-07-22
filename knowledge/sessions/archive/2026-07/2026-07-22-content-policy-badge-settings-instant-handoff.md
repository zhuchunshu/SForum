# 2026-07-22 Content policy badge settings instant effect

## Changed

- Host option `site.public_surface_revision` (public, default `1`): bumped on
  extension `UpdateSettings` / `ResetSettings` when manifest has
  `enabledBySetting` or public forum contributions
  (`forum.topic.badges|sidebar|list.badges`, `forum.nav.items`).
- Nuxt anonymous `/t/**` SWR `cache.varies` + middleware header
  `x-sforum-public-surface-revision` so bump misses old HTML.
- Admin extension settings Toast (P0): success description when public surface
  affected — refresh topic page, no theme reactivation, cache updated.
- `show_topic_badge` description clarifies save → refresh (no re-activate theme).

## Decisions

- Reuse `web_options` instead of a new table; actor cannot write revision via
  admin Update (only host `BumpPublicSurfaceRevision`).
- Do not disable global `/t/**` SWR; only vary cache key by revision.

## Next

- Optional: bump also on enable/disable of public-surface plugins if product
  needs that path (out of P0/P1 scope).

## Open Questions

- None for P0/P1.
