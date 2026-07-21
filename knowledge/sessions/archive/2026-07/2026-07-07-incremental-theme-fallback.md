# 2026-07-07 Incremental Theme Fallback

## Changed

- Documented the uploaded theme layer order in `apps/web/nuxt.config.ts`.
- Added theme runtime validation that locks uploaded themes before the default
  fallback layer.
- Added an extension service test showing a minimal uploaded layer can verify
  without public pages, layouts, or assets.
- Recorded the incremental theme fallback architecture in the knowledge base.

## Decisions

- Uploaded themes are incremental overlays.
- `sforum.default-theme` must remain the final Nuxt Layer fallback.
- Missing files inside an uploaded layer inherit from the default theme, while
  a missing layer directory remains invalid.

## Next

- Keep future theme author docs aligned with the incremental-overlay contract.

## Open Questions

- None for this slice.
