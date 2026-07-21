# 2026-07-21 Session Handoff

## Changed

- Removed first-party `/my` and `/my/content-review` routes, Host body islands,
  theme L1 templates/replacements, Page Registry entries, and related nav
  entry points (navbar, home right rail, profile self actions).
- Pending topic creation still toasts and returns home; notification deep-links
  without a topic target fall back to `/notifications`.
- Retired UI catalog identities (5) into the append-only ledger; regenerated
  V3 catalogs. Inventory gate is now **253 routes / 145 UI surfaces / 99**.
- Kept backend `GET /api/v1/me/content-review` for API/clients (no first-party
  page). Frontend `listAuthorReviewItems` / `AuthorContentReviewStatus` removed.

## Decisions

- Self-center hub is not a core product surface; public profile + settings +
  notifications cover the needed entry points.
- Catalog identity retirement (not hard-delete) for the five UI surfaces.

## Next

- None required for this removal. Optional later: drop or document the unused
  author content-review API if no clients remain.

## Open Questions

- None.
