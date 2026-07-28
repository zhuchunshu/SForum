# 2026-07-29 Session Handoff

## Changed

- Added `POST /admin/site/brand-assets` for logo, favicon, and Apple Touch icon
  uploads. It uses `settings.site.manage`, reuses attachment storage and MIME
  policy, and forces active public images.
- Replaced the Brand tab's bare URL/ID fields with compact click-or-drop upload
  controls. Uploads auto-fill URL and numeric attachment ID, show a small
  preview, and support replace/remove plus local error/loading states.
- Updated the modular OpenAPI contract and bilingual UI copy.
- Added focused permission, context, and public-visibility unit test source.
- Added brand-only SVG upload support through bounded server-side rasterization
  to PNG; raw SVG source is never stored or served.

## Decisions

- Brand uploads do not use the general `/attachments` endpoint because site
  operators do not necessarily hold `attachment.upload`, and general uploads
  may inherit private visibility.
- Upload remains a draft action. The existing brand settings save establishes
  or replaces the authoritative `site` attachment reference.
- SVG stays prohibited in general attachments. Brand SVG uses BSD-3-Clause
  `oksvg` + `rasterx`, a non-browser parser/render path that discards executable
  source and emits bounded PNG bytes.

## Next

- Manually refresh `/control-panel/personalization?tab=brand`, test click and
  drag uploads (including SVG) for all three assets, then save and confirm
  public logo/favicon rendering.

## Open Questions

- None.
