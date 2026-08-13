# 2026-08-14 Public Attachment Variant Guard Fix

## Changed

- Added `core.route.attachments.variant_content` to the production attachment
  read guard's accepted route set. This is the API route used by the stable
  `/media/attachments/{publicId}` browser alias.
- Added regression coverage for anonymous public display variants and for the
  `forum.guest.read=login_required` denial path.

## Decisions

- Display variants retain the same attachment/reference authorization as
  metadata and original content. The fix does not make private, pending,
  hidden-category, disabled, unreferenced, or login-required media public.

## Verification

- `go test ./app/Http -run 'TestProductionAttachmentReadGuard' -count=1`
- `go test ./app/Http ./app/Models/Attachments -count=1`
- Anonymous runtime requests to active public comment attachments through
  `/media/attachments/{publicId}` returned `200 image/png`.
- `node tests/validate-architecture-boundaries.mjs`

## Next

- None for this repair.

## Open Questions

- None.
