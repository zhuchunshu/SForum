# Decision: Public Feed Loads By Appending Only (No Pager)

## Status

Accepted and implemented 2026-10-23.

## Context

The public topic feed (`/` and `/search`, both rendered by
`apps/web/app/components/forum/SFHomePage.vue`) carried **two competing
loading models at once**:

1. `IntersectionObserver` on a bottom sentinel appended pages into
   `loadedTopics` (M5 keyset `after` cursor when available, page fallback for
   search).
2. A `SFPagination` block rendered `?page=N` links.

They were hidden from each other only by CSS ownership luck:

- `apps/web/app/assets/css/sforum-home.css` set
  `.sforum-home__pagination { display: none; }`.
- `extensions/builtin/themes/sforum-default/assets/theme.css` set the same
  class to `display: flex`.

The active theme skin stylesheet can be attached after the host stylesheet at
runtime, so `display: flex` won and the pager became visible underneath an
already infinitely-scrolling list (operator screenshot: `‹ 1 2 … 61 ›` under
the feed). Two state machines also disagreed: `currentPage` (the URL anchor)
drove the pager highlight, while `loadedTopics` held an ever-growing window.
Clicking a page link replaced the window and kept the old scroll offset.

## Decision

- **Public feed = append only.** `/` and `/search` render no page-number
  control; the bottom sentinel/status area is the only loading affordance.
- **`?page=N` stays an anchor.** SSR still renders page N, the client continues
  from N+1, and deep links keep working. The URL is not rewritten while
  appending.
- **Archive pages keep the pager.** Category (`/c/*`), tag (`/tags/*`), topic
  comments, moderation, and admin lists are untouched: they have no
  auto-append, and "jump to page N" is their stable navigation.
- **No backend, contract, or cursor changes.** `hasMore` / `nextCursor` /
  `page` semantics stay as documented in `contracts/openapi/schemas/forum.yaml`.
- **No IntersectionObserver → manual button.** The feed falls back to a
  "load more" button instead of becoming unpageable now that the pager is gone.
- **Built-in theme artifact is not rebuilt in this change.** The
  `display: flex` rule in the theme source is now dead (no matching element).
  It is left for the next theme artifact rebuild so this change does not
  require a digest rebuild, API restart, and admin re-activation. The contract
  test prevents re-adding `SFPagination` to the feed in the meantime.

## Consequences

- Users can no longer jump to an arbitrary feed page; they scroll. Deep links
  and `?page=N` remain valid for anything that already points at them.
- Feed page 2+ is no longer internally linked, so crawlers discover older
  topics through the sitemap (`/t/*` entries) and topic-to-topic links instead
  of list pagination. `useSForumSeo` rules are unchanged.
- The removed `totalPages` / `homePageTo` code paths shrink `SFHomePage.vue`;
  no architecture baseline moved.
- A future "load mode" preference (auto / pager / both) would now be a new
  product decision, not a revert: the two models must not share `currentPage`
  again.
