# 2026-07-24 Profile Reply Comment Page Links

## Changed

- Profile activity comment items now expose `commentPage` (flat-view page for
  the reply inside its topic).
- Store SQL counts active comments ordered before `(path_key, id)` (same
  contract as `CountActiveCommentsBefore` / `ResolveCommentPage`) and divides
  by `forum.pagination.comments_per_page`.
- Frontend `profileActivityLink` builds `/t/.../page/N#comment-{id}` when
  `commentPage > 1` (page 1 stays without `/page/1`).
- OpenAPI `ProfileActivity.commentPage` documented; Go + web unit tests updated.

## Decisions

- Pre-bake page into profile reply links. Hash-only anchors cannot be
  SSR-resolved because fragments never reach the server; open-in-new-tab and
  hard refresh always painted page 1 before this fix.
- Page may drift after later inserts/deletes until the profile list is
  reloaded; topic page still has client/SSR resolve as a safety net when the
  baked page is wrong.

## Next

- Manual QA: user with a reply on comment page 2+ → profile Replies tab →
  link should include `/page/2#comment-…` and open that page scrolled to the
  comment (new tab and SPA click).
- Optional follow-up: notifications deep links still hash-only; same pre-bake
  pattern if operators hit the same issue there.

## Open Questions

- None for profile; notifications pre-bake deferred until requested.
