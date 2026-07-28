# 2026-07-29 Session Handoff

## Changed

- Added the supplied SForum V3 logo as the stable Core public asset
  `apps/web/public/brand/sforum-logo.svg`.
- Empty `site.logo_url` and `site.favicon_url` values now resolve through the
  focused `utils/settings/siteBrand.ts` helper consumed by `useWebOptions`;
  custom operator values still take precedence.
- The document head emits the resolved favicon while preserving automatic MIME
  detection for operator-configured PNG, WebP, or other supported formats.
- Added focused tests for absent, blank, and custom brand option values.

## Decisions

- Keep stored runtime option defaults empty. The Core fallback belongs to the
  Web presentation layer so clearing a custom value restores the bundled brand
  asset without writing deployment-specific URLs to the database.

## Next

- Operator manual QA: confirm the default navbar logo and browser favicon on
  desktop/mobile, then confirm an admin-configured asset URL overrides each
  fallback independently.

## Open Questions

- None.
