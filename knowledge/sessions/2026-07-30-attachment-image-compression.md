# 2026-07-30 Attachment Image Compression Handoff

## Changed

- Replaced the Attachment Configuration compression placeholder with a real
  Image Optimization tab, recommended defaults, adjacent JPEG estimate, queue
  and savings statistics, reset, save, and permission-aware history backfill.
- Added immutable `display` variants and durable compression tasks in migration
  `202607300003_attachment_image_compression.sql`.
- Added bounded JPEG/PNG processing with EXIF orientation, proportional resize,
  minimum-savings rejection, task dedupe, retry leases, and orphan cleanup.
- Registered `attachments.compress_image` and the one-minute
  `attachments.reconcile_compression` River schedule.
- Added settings, statistics, backfill, and authorized variant-content APIs and
  kept OpenAPI plus generated route/schedule catalogs in sync.
- Proxied JPEG/PNG attachment DTOs now return the display URL; missing, stale,
  disabled, or unreadable variants transparently return the original.

## Decisions

- Originals are immutable and remain authoritative for downloads and audits.
- V1 is Host-owned JPEG/PNG processing, not a plugin Media Pipeline ABI.
- Policy saves do not trigger an implicit full-library rewrite.
- Decision: `../decisions/2026-07-30-attachment-image-display-variants.md`.

## Verification

- Attachment, Options, Jobs, controller, worker schedule, and bootstrap focused
  Go tests pass, including PostgreSQL claim/dedupe/lease/variant coverage.
- OpenAPI references, catalog docs checks, 9 Bun attachment tests, and Nuxt
  production build pass.
- Chrome QA passed at `1920x992` and `390x844`: no framework overlay, relevant
  console errors, or horizontal overflow. Strength `55 -> 100` changed JPEG
  quality `81 -> 70` and the 5 MB estimate `3.0 MB -> 2.3 MB`.
- Save/reset/backfill buttons were not submitted during QA, so their live Toast
  feedback remains an operator interaction check.

## Next

- Re-run the full repository gate after the concurrent Identity, Appearance,
  route-catalog, localization, and architecture work is synchronized.
- Consider WebP/AVIF or named thumbnail variants only with a separate format and
  Media Pipeline contract decision.

## Open Questions

- Whether a future plugin ABI should stream bytes through Host-owned bounded
  storage handles or use a separately sandboxed transform service.
