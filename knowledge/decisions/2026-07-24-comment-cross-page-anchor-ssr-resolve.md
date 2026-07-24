# Decision: Comment Cross-Page Anchor Via SSR Page Resolve

## Status

Accepted

## Context

Public profile activity, notifications, and copied links produce
`/t/<id>/<slug>#comment-<id>` anchors. When the target comment sits on page 2+,
the topic detail page defaulted to page 1, so the target comment was absent
from the DOM and the anchor could not resolve. The page also never consumed
`route.hash` for scrolling, so even same-page anchors were unreliable under
SSR hydration timing.

Two independent fixes were possible: resolve the correct page for the anchor,
and add hash-scroll handling. Resolving the page client-side after first paint
would cause a visible page-1 → blank → page-2 flash plus scroll jumps, which
the product owner explicitly rejected.

## Decision

Resolve the comment page **server-side** and SSR-render that page so the
target comment is present in first-paint HTML.

- New backend endpoint `GET /topics/:topicID/comments/:commentID/page` returns
  `{ page, perPage }` for flat-view pagination. The service reuses
  `GetCommentSummary` and counts active comments ordered before
  `(path_key, id)`, strictly aligned with `listCommentsFlat`. Only active
  comments resolve; soft-deleted, cross-topic, or hidden-topic targets return
  404 without leaking status. The endpoint is not cached (primary-key lookup
  plus indexed COUNT; caching would drift on delete).
- `SFTopicShowPage` runs the resolve as a `useAsyncData` that the SSR path
  awaits before the comment list fetch, so `commentPage` is correct at first
  paint. `commentsAsync` drops `lazy` only when a target anchor is present, so
  non-anchor visits keep fast first paint.
- Comment pagination moved from `?page=N` to a path segment `/page/N`
  (`forumTopicPath`/`parseTopicPath`/`topicPathLookupCandidates` share one
  parser). Old `?page=N` links still resolve and normalize client-side after
  hydration — never via an SSR redirect, which would re-trigger the flash.
- URL normalization splits into two paths: slug/mode mismatch still 301/replace
  (the lookup key changed), but page-segment normalization is client-only
  replace (data is already correct; an SSR redirect would discard the resolved
  first paint).

## Consequences

- Zero flash on anchor deep-links: first-paint HTML already contains the
  target comment, and the browser scrolls natively before hydration.
- SSR TTFB for anchor visits adds one lightweight resolve call (primary-key
  lookup + indexed COUNT). Non-anchor visits are unaffected.
- Lazy-generation links that only emit `#comment-<id>` still work for SPA
  navigation (client resolve), but **full page loads cannot SSR-resolve from
  hash** because fragments are never sent to the server. Profile activity
  therefore pre-bakes `commentPage` into `/page/N#comment-<id>` links (see
  amendment below and session `2026-07-24-profile-reply-comment-page-links`).
- `commentView` is hard-pinned to `flat` on the frontend, so tree-view
  cross-page anchoring is intentionally unsupported until a tree UI ships.
- Flat ordering (`path_key ASC, id ASC`) is now a load-bearing contract for
  both list pagination and page resolve; the `CountActiveCommentsBefore` SQL
  must stay aligned with `listCommentsFlat` or page numbers will drift.

## Amendment (2026-07-24): profile activity pre-bakes page

Profile public-activity reply items now include `commentPage` (flat-view page
computed with the same `active_before / commentsPerPage + 1` formula as
`ResolveCommentPage`). The web client builds:

`/t/<topic>/page/N#comment-<id>` (page segment omitted when N=1).

Rationale: open-in-new-tab, shared URLs, and hard refresh never deliver the
fragment to SSR; without `/page/N` the first paint is always page 1. Baking
page at list time is acceptable for profile QPS; drift after later inserts/
deletes is corrected on next profile load (list recomputes) and by the
existing topic-page resolve when the baked page is stale.
