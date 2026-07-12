# 2026-07-12 Attachment access follows forum guest-read policy

## Status

Accepted

## Context

`forum.guest.read=login_required` protected forum JSON endpoints but not
`GET /api/v1/attachments/{publicId}` or `/content` for attachments with
`visibility=public`. Copied post media URLs were anonymously readable.

Site assets (avatars, SEO images, logos) must remain anonymously loadable so
login/register and branding pages work.

## Decision

1. **Site-public references** stay anonymous when active+public:
   - `resource_type=user` + `context=avatar`
   - `resource_type=seo` (any context)
   - `resource_type=site` + logo/favicon/brand/chrome contexts
2. **All other public attachments** (including unreferenced and post/topic
   media) require an active session when `forum.guest.read=login_required`.
3. Fail closed if reference listing fails or no references exist under
   login_required.
4. Under login_required, forum media `URL` decoration prefers the API content
   proxy path instead of permanent CDN public URLs so session policy cannot be
   bypassed by a cached object URL. Site-public assets may still use CDN URLs.
5. Denied anonymous reads return `401` with reason `forum.guest_login_required`
   (same as forum guest gate).

## Consequences

- Operators enabling login-required reading must accept that post images load
  only for signed-in users (and via API proxy when remote storage is used).
- Full post↔attachment reference binding remains incomplete for inline media
  until forum content pipelines write `attachment_references`; until then
  unreferenced public uploads also require login under login_required
  (fail closed).

## Alternatives considered

- Gate every public attachment: breaks avatars/SEO.
- Frontend-only hide: not authoritative.
