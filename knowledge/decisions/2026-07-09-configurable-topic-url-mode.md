# Decision: Configurable Topic URL Mode (Permalink Switching)

## Status

Accepted

## Context

The original design fixed the topic detail URL at `/t/:topicID/:topicSlug`
(ID + slug). Operators asked to make the URL shape configurable from the SEO
admin page, with sensible defaults and 301 redirects so old links keep
working when the shape changes.

Three shapes are useful in practice:
- `id_slug` (`/t/123/hello-world`) — stable + readable, the safest default.
- `id` (`/t/123`) — most permanent, no readability.
- `slug` (`/t/hello-world`) — shortest and cleanest, but requires the slug
  to be globally unique.

The hard constraint: Nuxt file routing cannot change shape at runtime —
`/t/123`, `/t/hello`, and `/t/123/hello` are three different file structures.
A catch-all `/t/[...path]` route is the only way to serve all shapes from a
single component while the configured mode is a runtime option.

A second constraint: a sibling `t/[...path]/edit.vue` edit page becomes a
**nested child route** of the catch-all detail page, and Nuxt only renders
nested children when the parent provides a `<NuxtPage />` outlet — but the
985-line detail page renders a full layout, not an outlet.

## Decision

1. **Catch-all detail page** `t/[...path].vue` parses `route.params.path`
   with `parseTopicPath(segments, mode)` and serves all three shapes.
2. **`seo.topic_url_mode`** option (`id_slug` | `id` | `slug`, default
   `id_slug`) drives both URL generation (`forumTopicPath(topic, mode)`,
   the single canonical link outlet) and detail-page parsing. Public so SSR
   and the admin page both read it.
3. **Canonical 301 in SSR-first position**: the detail page computes the
   mode-canonical path and `navigateTo(target, { redirectCode: 301 })` on
   mismatch. Covers mode switches, slug renames, and stray slug segments.
   This is **same-topic canonicalization only** — no cross-mode redirect
   history table (e.g. `topic_redirects`) in v1.
4. **Slug uniqueness for `slug` mode**: migration `202607090001` upgrades
   `topics.slug` to a UNIQUE index (de-duplicating existing rows first).
   `Service.ensureUniqueTopicSlug` appends `-2`/`-3` suffixes on collision
   at create/rename time. `id_slug`/`id` modes do not depend on the
   constraint and remain unaffected.
5. **Edit via `?edit=1` query**, not `/edit` path. The catch-all detail page
   conditionally renders `SFTopicEditor.vue` when `route.query.edit` is set.
   This avoids the nested-child-router outlet problem entirely and keeps the
   detail page under the 1000-line guideline by extracting the editor into
   its own component. (Deviation from the approved plan's
   `t/[...path]/edit.vue` — documented here for the same reason.)

## Consequences

- `forumTopicPath` gains a `mode` parameter; all 8 call sites pass the
  current `topicUrlMode` so links respect the configured shape everywhere.
- Moderation links use explicit `id` mode (only the ID is known there) and
  rely on the detail page canonicalization for non-`id` modes.
- Switching modes after launch is safe: ID-keyed shapes never break, and the
  301 keeps search-indexed URLs valid. Operators should still avoid frequent
  switching.
- Adding a 4th shape (`category_slug` `/c/general/hello-world`) is deferred
  to v2 — it needs category coupling and disambiguation against the
  `/c/:slug` category-list route.
