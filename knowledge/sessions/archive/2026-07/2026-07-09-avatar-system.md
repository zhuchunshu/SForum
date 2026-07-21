# 2026-07-09 Avatar System Handoff

## Changed

- Implemented profile avatar strategy: uploaded avatar first, then configured
  fallback provider (`initials`, `gravatar`, or `static`).
- Added `avatar.*` runtime options with recommended defaults, admin settings
  UI under System configuration, OpenAPI coverage, and frontend option helpers.
- Added `POST /api/v1/profile/avatar` and `DELETE /api/v1/profile/avatar`.
  Upload requires login, active user, `attachment.upload`, and enabled
  `avatar.allow_upload`.
- Avatar uploads reuse attachment storage with avatar-specific size, dimension,
  GIF, and compression rules. JPEG/PNG compression uses
  `github.com/disintegration/imaging`; GIF remains uncompressed when enabled.
- Profile responses now expose `avatar` (`kind`, `url`, `attachmentId`, `alt`)
  while preserving `avatarAttachmentId` for compatibility.
- Profile settings page now shows avatar preview/upload/remove controls.
  `SFAvatar` accepts `AvatarView` and renders external URLs with plain `img`.
- Topic detail route cache was changed from SWR to `cache: false` for `/t/**`
  and `/en/t/**`, because the current edit mode uses `?edit=1` and Nitro
  routeRules cannot isolate query-based user states.

## Decisions

- Recommended avatar fallback is local initials, not Gravatar, so a new install
  does not depend on an external service.
- Gravatar uses SHA-256 by default, with MD5 kept for legacy mirror
  compatibility.
- `imaging` was chosen over `bimg` to avoid a libvips/C toolchain requirement
  in default deployments.
- Avatar attachments are forced public and tracked through
  `attachment_references` resource `user`, context `avatar`.

## Verification

- `ruby scripts/validate-openapi-refs.rb`
- `go test ./...` from `apps/api`
- `bun test` from `apps/web`
- `bun run typecheck` from `apps/web` (passes with the existing Nuxt robots
  warning about `/api/**`)

## Next

- If topic edit mode later moves away from query parameters to a dedicated
  protected route, `/t/**` can be reconsidered for SWR caching.
- Browser QA is still useful for the new admin avatar settings page and profile
  avatar controls once the local dev server is running.
