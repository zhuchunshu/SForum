# 2026-07-24 Session Handoff

## Changed

- Backend: new `GET /topics/:topicID/comments/:commentID/page` endpoint resolves
  the flat-view page holding a comment. `Service.ResolveCommentPage` +
  `Store.CountActiveCommentsBefore` (COUNT active comments ordered before
  `(path_key, id)`); only active comments resolve, 404 otherwise without
  leaking status. Route, controller, OpenAPI path+schema added.
- Frontend: `SFTopicShowPage` now resolves the comment page server-side for
  `#comment-<id>` anchors (zero-flash: target comment is in first-paint HTML,
  browser scrolls natively). `commentsAsync` drops `lazy` only when an anchor
  is present. Added a client-side hash-scroll watch for client navigations.
- Comment pagination moved from `?page=N` to path segment `/page/N`
  (`forumTaxonomy.ts`: `forumTopicPath`/`parseTopicPath`/
  `topicPathLookupCandidates` + new `splitTopicPageSegment`). Old `?page=N`
  still resolves, normalized client-side after hydration (never SSR redirect).
- URL normalization split: slug/mode mismatch still 301/replace; page-segment
  normalization is client-only replace (preserves resolved first paint).
- Profile public-activity "public replies" links: `SFProfileShowPage` changed
  the activity `<a :href>` to `<NuxtLink :to>` so in-app clicks do SPA
  navigation, preserving `route.hash` and triggering the client-side page
  resolve. The notifications page already used `router.push` (no change
  needed). Same-page anchors in `SFTopicSideCard`/`SFComment` keep native
  `<a href="#comment-X">` (in-page scroll, correct).

## Decisions

- SSR page resolve chosen over client-side resolve to eliminate page-1 → blank
  → page-2 flash. See
  `decisions/2026-07-24-comment-cross-page-anchor-ssr-resolve.md`.
- Lazy-generation links (profile/notifications/copy) keep emitting only
  `#comment-<id>` — page recomputed per visit, no drift as comments grow.
- `commentView` hard-pinned to `flat`; tree-view cross-page anchoring not
  supported (no tree UI ships).

## Next

- Manual QA: seed a topic with 25 comments (perPage=20), open
  `/t/<id>/<slug>#comment-21` from profile/notifications — expect page 2
  rendered on first paint, scrolled to comment-21, no flash.
- If a tree view UI ships later, extend `ResolveCommentPage` to walk a child
  comment up to its root and page by root `path_key`.
- Consider a Playwright test under `tests/` covering the cross-page anchor flow
  once the dev seed (`seed:forum`) reliably produces multi-page topics.

## Open Questions

- Whether profile activity / notifications should pre-bake the page into links
  for even faster SSR (current design recomputes; acceptable given low QPS of
  the resolve endpoint).
