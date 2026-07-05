# 2026-07-05 SEO Full-Chain v1

## Status

Accepted.

## Context

SForum is an SEO-oriented forum, but the first admin SEO page only edited a
small set of frontend-side `seo.*` fields and the backend Options service did
not yet register those keys. Forum SEO also needs more than meta text: runtime
indexing control, robots.txt, sitemap.xml, structured data, canonical URLs,
social previews, and platform verification all need one coherent operator
surface.

## Decision

- SEO settings are first-class runtime `web_options` with typed backend
  validation and frontend-safe public reads.
- SEO management uses the independent `seo.manage` permission. General site
  settings continue to use `settings.manage`.
- Public pages use a shared `useSForumSeo()` composable for title templates,
  meta description, canonical URL, robots meta, Open Graph, Twitter Card,
  verification tags, and minimal JSON-LD.
- `@nuxtjs/seo` remains the umbrella module. Sitemap output uses a dynamic
  Nuxt server source, while robots.txt is extended through a Nitro hook.
- Local and preview site URLs such as `localhost` and `127.0.0.1` always emit
  `noindex, nofollow`, even when the admin indexing toggle is enabled.
- `nuxt-schema-org` stays disabled for now because a prior Bun runtime crash is
  recorded in the session handoff; SForum emits minimal JSON-LD manually until
  that module can be revalidated safely.

## Consequences

- A user can be granted SEO management without receiving broader system setting
  permissions.
- Future category, topic, post, and profile pages should call `useSForumSeo()`
  with their canonical path, public visibility, and structured data inputs.
- Forum content sitemap entries are intentionally reserved until public read
  models exist; the toggle is present but currently only static public pages are
  emitted.
