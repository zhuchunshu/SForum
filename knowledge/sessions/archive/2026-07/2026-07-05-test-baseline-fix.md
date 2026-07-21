# 2026-07-05 Test Baseline Fix Handoff

## Changed

- Synced admin framework validation with registered `/seo` and `/attachments`
  admin pages.
- Synced identity UI validation with runtime ALTCHA scenario and widget
  settings instead of older hard-coded registration-page expectations.
- Moved the local attachment provider root from process config to the
  admin-only runtime option `attachment.local.root`, defaulting to
  `storage/app/attachments`.
- Updated local storage probing to verify write access by creating and removing
  a temporary probe file.

## Decisions

- Local attachment root is now governed by `attachment.settings.manage` through
  runtime options. Deployments should mount storage at the configured path.

## Verification

- `./scripts/test.sh` passed.

## Next

- Keep future attachment storage docs, Compose mounts, OpenAPI contracts, and
  admin UI copy aligned with `attachment.local.root`.
