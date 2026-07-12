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
2. Forum references (`topic`, `comment`, and shared `post`) authorize against
   the referenced resource on every request. Public-category active/locked
   content follows `forum.guest.read`; pending content is visible to its author
   and reviewers; hidden/deleted content and hidden categories are reviewer-only.
3. Unreferenced public uploads are owner/admin-only. Missing resources and
   reference query failures fail closed.
4. Forum and unreferenced media always use the API content proxy, including in
   public guest-read mode. Only explicit site-public assets may expose a
   permanent provider/CDN URL.
5. Denied anonymous reads return `401` with reason `forum.guest_login_required`
   (same as forum guest gate).
6. Forum create/edit payloads carry explicit `content.attachmentIds`; the
   PostgreSQL topic/comment transaction validates ownership and active/public
   attachment state, replaces references, and maintains `reference_count`.
   Omitted IDs preserve references on edit; an explicit empty array clears them.
   Soft deletion removes owned content references and decrements counts.
7. Logo, favicon, and apple-touch-icon option updates replace their `site`
   reference in the same database transaction. A migration backfills valid
   historical option values.

## Consequences

- Operators enabling login-required reading must accept that post images load
  only for signed-in users (and via API proxy when remote storage is used).
- Clients that upload inline media must retain the returned numeric attachment
  ID and submit it through `content.attachmentIds`; parsing rendered URLs is not
  an ownership or reference source.

## Alternatives considered

- Gate every public attachment: breaks avatars/SEO.
- Frontend-only hide: not authoritative.
